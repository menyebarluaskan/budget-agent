package transactions

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// MaxListLimit is the largest page size the API will serve.
const MaxListLimit = 200

type Transaction struct {
	ID          string   `json:"id"`
	WorkspaceID string   `json:"workspace_id"`
	AccountID   string   `json:"account_id"`
	AmountMinor int64    `json:"amount_minor"`
	OccurredAt  string   `json:"occurred_at"`
	Description string   `json:"description"`
	Notes       string   `json:"notes"`
	Source      string   `json:"source"`
	TagIDs      []string `json:"tag_ids"`
	TransferID  *string  `json:"transfer_id,omitempty"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

type CreateInput struct {
	WorkspaceID string
	AccountID   string
	AmountMinor int64
	OccurredAt  string
	Description string
	Notes       string
	Source      string
	TagIDs      []string
}

type UpdateInput struct {
	AccountID   *string
	AmountMinor *int64
	OccurredAt  *string
	Description *string
	Notes       *string
	Source      *string
	TagIDs      *[]string
}

// Filter describes a transaction list query. Empty optional fields are ignored.
type Filter struct {
	AccountID        string
	TagIDs           []string
	From             string
	To               string
	MinAmountMinor   *int64
	MaxAmountMinor   *int64
	Query            string
	Source           string
	ExcludeTransfers bool
	Sort             string
	Limit            int
	Cursor           string
}

// Page is one cursor-paginated slice of transactions.
type Page struct {
	Transactions []Transaction `json:"transactions"`
	NextCursor   string        `json:"next_cursor,omitempty"`
}

type Service struct{ db *sql.DB }

func NewService(db *sql.DB) *Service { return &Service{db: db} }

func nowUTC() string { return time.Now().UTC().Format(time.RFC3339) }

var (
	ErrNotFound      = errors.New("not found")
	ErrInvalidFilter = errors.New("invalid filter")
	ErrInvalidSort   = errors.New("invalid sort")
	ErrInvalidCursor = errors.New("invalid cursor")
)

var validSources = map[string]bool{
	"manual":   true,
	"whatsapp": true,
	"mcp":      true,
	"receipt":  true,
	"import":   true,
}

type sortSpec struct {
	name      string
	column    string
	direction string
	numeric   bool
}

var sorts = map[string]sortSpec{
	"occurred_at":  {name: "occurred_at", column: "occurred_at", direction: "ASC"},
	"-occurred_at": {name: "-occurred_at", column: "occurred_at", direction: "DESC"},
	"amount_minor": {name: "amount_minor", column: "amount_minor", direction: "ASC", numeric: true},
	"-amount_minor": {
		name:      "-amount_minor",
		column:    "amount_minor",
		direction: "DESC",
		numeric:   true,
	},
}

// List returns every matching transaction. Prefer ListPage for user-facing
// lists, which enforce a bounded page size.
func (s *Service) List(ctx context.Context, workspaceID string, f Filter) ([]Transaction, error) {
	page, err := s.ListPage(ctx, workspaceID, f)
	if err != nil {
		return nil, err
	}
	return page.Transactions, nil
}

// ListPage returns a stable page of transactions ordered by the requested sort.
// Ties are broken by ID so the cursor cannot skip or repeat rows.
func (s *Service) ListPage(ctx context.Context, workspaceID string, f Filter) (Page, error) {
	spec, err := resolveSort(f.Sort)
	if err != nil {
		return Page{}, err
	}
	if f.Limit < 0 || f.Limit > MaxListLimit {
		return Page{}, fmt.Errorf("%w: limit must be between 1 and %d", ErrInvalidFilter, MaxListLimit)
	}
	if f.Cursor != "" && f.Limit == 0 {
		return Page{}, fmt.Errorf("%w: cursor requires a limit", ErrInvalidFilter)
	}
	if f.MinAmountMinor != nil && f.MaxAmountMinor != nil && *f.MinAmountMinor > *f.MaxAmountMinor {
		return Page{}, fmt.Errorf("%w: min_amount_minor must not exceed max_amount_minor", ErrInvalidFilter)
	}

	where, args := buildWhere(workspaceID, f)
	if f.Cursor != "" {
		condition, cursorArgs, err := cursorCondition(f.Cursor, spec)
		if err != nil {
			return Page{}, err
		}
		where += condition
		args = append(args, cursorArgs...)
	}

	q := `SELECT id, workspace_id, account_id, amount_minor, occurred_at, description, notes, source, transfer_id, created_at, updated_at FROM transactions` + where +
		` ORDER BY ` + spec.column + ` ` + spec.direction + `, id ` + spec.direction
	if f.Limit > 0 {
		q += " LIMIT ?"
		args = append(args, f.Limit+1)
	}

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return Page{}, err
	}
	defer rows.Close()

	var out []Transaction
	for rows.Next() {
		var t Transaction
		var transferID sql.NullString
		if err := rows.Scan(&t.ID, &t.WorkspaceID, &t.AccountID, &t.AmountMinor, &t.OccurredAt, &t.Description, &t.Notes, &t.Source, &transferID, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return Page{}, err
		}
		if transferID.Valid {
			t.TransferID = &transferID.String
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return Page{}, err
	}
	if err := rows.Close(); err != nil {
		return Page{}, err
	}

	page := Page{Transactions: out}
	if f.Limit > 0 && len(out) > f.Limit {
		page.Transactions = out[:f.Limit]
		page.NextCursor = encodeCursor(spec, page.Transactions[len(page.Transactions)-1])
	}
	if page.Transactions == nil {
		page.Transactions = []Transaction{}
	}

	for i := range page.Transactions {
		tags, err := s.tagIDs(ctx, page.Transactions[i].ID)
		if err != nil {
			return Page{}, err
		}
		page.Transactions[i].TagIDs = tags
	}
	return page, nil
}

func resolveSort(value string) (sortSpec, error) {
	if value == "" {
		value = "-occurred_at"
	}
	spec, ok := sorts[value]
	if !ok {
		return sortSpec{}, fmt.Errorf("%w: %s", ErrInvalidSort, value)
	}
	return spec, nil
}

func buildWhere(workspaceID string, f Filter) (string, []any) {
	q := ` WHERE workspace_id = ?`
	args := []any{workspaceID}
	if f.AccountID != "" {
		q += " AND account_id = ?"
		args = append(args, f.AccountID)
	}
	if f.From != "" {
		q += " AND occurred_at >= ?"
		args = append(args, f.From)
	}
	if f.To != "" {
		q += " AND occurred_at <= ?"
		args = append(args, f.To)
	}
	if len(f.TagIDs) > 0 {
		placeholders := make([]string, len(f.TagIDs))
		for i, tagID := range f.TagIDs {
			placeholders[i] = "?"
			args = append(args, tagID)
		}
		q += " AND id IN (SELECT transaction_id FROM transaction_tags WHERE tag_id IN (" +
			strings.Join(placeholders, ",") + "))"
	}
	if f.MinAmountMinor != nil {
		q += " AND amount_minor >= ?"
		args = append(args, *f.MinAmountMinor)
	}
	if f.MaxAmountMinor != nil {
		q += " AND amount_minor <= ?"
		args = append(args, *f.MaxAmountMinor)
	}
	if f.Query != "" {
		pattern := "%" + escapeLike(f.Query) + "%"
		q += ` AND (description LIKE ? ESCAPE '\' OR notes LIKE ? ESCAPE '\')`
		args = append(args, pattern, pattern)
	}
	if f.Source != "" {
		q += " AND source = ?"
		args = append(args, f.Source)
	}
	if f.ExcludeTransfers {
		q += " AND transfer_id IS NULL"
	}
	return q, args
}

func escapeLike(value string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(value)
}

type pageCursor struct {
	Sort  string `json:"sort"`
	Value string `json:"value"`
	ID    string `json:"id"`
}

func encodeCursor(spec sortSpec, t Transaction) string {
	value := t.OccurredAt
	if spec.numeric {
		value = strconv.FormatInt(t.AmountMinor, 10)
	}
	payload, _ := json.Marshal(pageCursor{Sort: spec.name, Value: value, ID: t.ID})
	return base64.RawURLEncoding.EncodeToString(payload)
}

func cursorCondition(raw string, spec sortSpec) (string, []any, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return "", nil, fmt.Errorf("%w: malformed", ErrInvalidCursor)
	}
	var cursor pageCursor
	if err := json.Unmarshal(decoded, &cursor); err != nil {
		return "", nil, fmt.Errorf("%w: malformed", ErrInvalidCursor)
	}
	if cursor.Sort != spec.name || cursor.ID == "" || cursor.Value == "" {
		return "", nil, fmt.Errorf("%w: does not match the requested sort", ErrInvalidCursor)
	}

	operator := "<"
	if spec.direction == "ASC" {
		operator = ">"
	}
	condition := fmt.Sprintf(
		" AND (%s %s ? OR (%s = ? AND id %s ?))",
		spec.column, operator, spec.column, operator,
	)

	if spec.numeric {
		value, err := strconv.ParseInt(cursor.Value, 10, 64)
		if err != nil {
			return "", nil, fmt.Errorf("%w: malformed amount", ErrInvalidCursor)
		}
		return condition, []any{value, value, cursor.ID}, nil
	}
	return condition, []any{cursor.Value, cursor.Value, cursor.ID}, nil
}

func (s *Service) tagIDs(ctx context.Context, txID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT tag_id FROM transaction_tags WHERE transaction_id = ?`, txID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (s *Service) Get(ctx context.Context, workspaceID, id string) (Transaction, error) {
	var t Transaction
	var transferID sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT id, workspace_id, account_id, amount_minor, occurred_at, description, notes, source, transfer_id, created_at, updated_at FROM transactions WHERE id = ? AND workspace_id = ?`, id, workspaceID).Scan(&t.ID, &t.WorkspaceID, &t.AccountID, &t.AmountMinor, &t.OccurredAt, &t.Description, &t.Notes, &t.Source, &transferID, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Transaction{}, ErrNotFound
		}
		return Transaction{}, err
	}
	if transferID.Valid {
		t.TransferID = &transferID.String
	}
	tags, _ := s.tagIDs(ctx, t.ID)
	t.TagIDs = tags
	if t.TagIDs == nil {
		t.TagIDs = []string{}
	}
	return t, nil
}

func (s *Service) Create(ctx context.Context, in CreateInput) (Transaction, error) {
	if in.AccountID == "" {
		return Transaction{}, fmt.Errorf("account_id required")
	}
	if in.OccurredAt == "" {
		in.OccurredAt = nowUTC()
	}
	if err := validateOccurredAt(in.OccurredAt); err != nil {
		return Transaction{}, err
	}
	if in.Source == "" {
		in.Source = "manual"
	}
	if !validSources[in.Source] {
		return Transaction{}, fmt.Errorf("source must be one of manual, whatsapp, mcp, receipt, import")
	}
	id := uuid.NewString()
	now := nowUTC()
	_, err := s.db.ExecContext(ctx, `INSERT INTO transactions (id, workspace_id, account_id, amount_minor, occurred_at, description, notes, source, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?,?)`, id, in.WorkspaceID, in.AccountID, in.AmountMinor, in.OccurredAt, in.Description, in.Notes, in.Source, now, now)
	if err != nil {
		return Transaction{}, err
	}
	if err := s.setTags(ctx, id, in.TagIDs); err != nil {
		return Transaction{}, err
	}
	return s.Get(ctx, in.WorkspaceID, id)
}

func validateOccurredAt(value string) error {
	if _, err := time.Parse(time.RFC3339, value); err != nil {
		return fmt.Errorf("occurred_at must be an RFC 3339 timestamp")
	}
	return nil
}

func (s *Service) setTags(ctx context.Context, txID string, tagIDs []string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM transaction_tags WHERE transaction_id = ?`, txID); err != nil {
		return err
	}
	for _, tid := range tagIDs {
		if _, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO transaction_tags (transaction_id, tag_id) VALUES (?,?)`, txID, tid); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) Update(ctx context.Context, workspaceID, id string, in UpdateInput) (Transaction, error) {
	t, err := s.Get(ctx, workspaceID, id)
	if err != nil {
		return Transaction{}, err
	}
	if in.AccountID != nil {
		if *in.AccountID == "" {
			return Transaction{}, fmt.Errorf("account_id required")
		}
		t.AccountID = *in.AccountID
	}
	if in.AmountMinor != nil {
		t.AmountMinor = *in.AmountMinor
	}
	if in.OccurredAt != nil {
		if err := validateOccurredAt(*in.OccurredAt); err != nil {
			return Transaction{}, err
		}
		t.OccurredAt = *in.OccurredAt
	}
	if in.Description != nil {
		t.Description = *in.Description
	}
	if in.Notes != nil {
		t.Notes = *in.Notes
	}
	if in.Source != nil {
		if !validSources[*in.Source] {
			return Transaction{}, fmt.Errorf("source must be one of manual, whatsapp, mcp, receipt, import")
		}
		t.Source = *in.Source
	}
	now := nowUTC()
	_, err = s.db.ExecContext(ctx, `UPDATE transactions SET account_id=?, amount_minor=?, occurred_at=?, description=?, notes=?, source=?, updated_at=? WHERE id=? AND workspace_id=?`, t.AccountID, t.AmountMinor, t.OccurredAt, t.Description, t.Notes, t.Source, now, id, workspaceID)
	if err != nil {
		return Transaction{}, err
	}
	if in.TagIDs != nil {
		if err := s.setTags(ctx, id, *in.TagIDs); err != nil {
			return Transaction{}, err
		}
	}
	return s.Get(ctx, workspaceID, id)
}

func (s *Service) Delete(ctx context.Context, workspaceID, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM transactions WHERE id = ? AND workspace_id = ?`, id, workspaceID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
