package transactions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

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
	AmountMinor *int64
	OccurredAt  *string
	Description *string
	Notes       *string
	TagIDs      *[]string
}

type Filter struct {
	AccountID string
	TagID     string
	From      string
	To        string
}

type Service struct{ db *sql.DB }

func NewService(db *sql.DB) *Service { return &Service{db: db} }

func nowUTC() string { return time.Now().UTC().Format(time.RFC3339) }

var ErrNotFound = errors.New("not found")

func (s *Service) List(ctx context.Context, workspaceID string, f Filter) ([]Transaction, error) {
	q := `SELECT id, workspace_id, account_id, amount_minor, occurred_at, description, notes, source, created_at, updated_at FROM transactions WHERE workspace_id = ?`
	args := []any{workspaceID}
	if f.AccountID != "" { q += " AND account_id = ?"; args = append(args, f.AccountID) }
	if f.From != "" { q += " AND occurred_at >= ?"; args = append(args, f.From) }
	if f.To != "" { q += " AND occurred_at <= ?"; args = append(args, f.To) }
	if f.TagID != "" { q += " AND id IN (SELECT transaction_id FROM transaction_tags WHERE tag_id = ?)"; args = append(args, f.TagID) }
	q += " ORDER BY occurred_at DESC"
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []Transaction
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.WorkspaceID, &t.AccountID, &t.AmountMinor, &t.OccurredAt, &t.Description, &t.Notes, &t.Source, &t.CreatedAt, &t.UpdatedAt); err != nil { return nil, err }
		tags, _ := s.tagIDs(ctx, t.ID)
		t.TagIDs = tags
		if t.TagIDs == nil { t.TagIDs = []string{} }
		out = append(out, t)
	}
	if err := rows.Err(); err != nil { return nil, err }
	if out == nil { out = []Transaction{} }
	return out, nil
}

func (s *Service) tagIDs(ctx context.Context, txID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT tag_id FROM transaction_tags WHERE transaction_id = ?`, txID)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil { return nil, err }
		out = append(out, id)
	}
	return out, rows.Err()
}

func (s *Service) Get(ctx context.Context, workspaceID, id string) (Transaction, error) {
	var t Transaction
	err := s.db.QueryRowContext(ctx, `SELECT id, workspace_id, account_id, amount_minor, occurred_at, description, notes, source, created_at, updated_at FROM transactions WHERE id = ? AND workspace_id = ?`, id, workspaceID).Scan(&t.ID, &t.WorkspaceID, &t.AccountID, &t.AmountMinor, &t.OccurredAt, &t.Description, &t.Notes, &t.Source, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) { return Transaction{}, ErrNotFound }
		return Transaction{}, err
	}
	tags, _ := s.tagIDs(ctx, t.ID)
	t.TagIDs = tags
	if t.TagIDs == nil { t.TagIDs = []string{} }
	return t, nil
}

func (s *Service) Create(ctx context.Context, in CreateInput) (Transaction, error) {
	if in.AccountID == "" { return Transaction{}, fmt.Errorf("account_id required") }
	if in.OccurredAt == "" { in.OccurredAt = nowUTC() }
	if in.Source == "" { in.Source = "manual" }
	id := uuid.NewString()
	now := nowUTC()
	_, err := s.db.ExecContext(ctx, `INSERT INTO transactions (id, workspace_id, account_id, amount_minor, occurred_at, description, notes, source, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?,?)`, id, in.WorkspaceID, in.AccountID, in.AmountMinor, in.OccurredAt, in.Description, in.Notes, in.Source, now, now)
	if err != nil { return Transaction{}, err }
	if err := s.setTags(ctx, id, in.TagIDs); err != nil { return Transaction{}, err }
	return s.Get(ctx, in.WorkspaceID, id)
}

func (s *Service) setTags(ctx context.Context, txID string, tagIDs []string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM transaction_tags WHERE transaction_id = ?`, txID); err != nil { return err }
	for _, tid := range tagIDs {
		if _, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO transaction_tags (transaction_id, tag_id) VALUES (?,?)`, txID, tid); err != nil { return err }
	}
	return nil
}

func (s *Service) Update(ctx context.Context, workspaceID, id string, in UpdateInput) (Transaction, error) {
	t, err := s.Get(ctx, workspaceID, id)
	if err != nil { return Transaction{}, err }
	if in.AmountMinor != nil { t.AmountMinor = *in.AmountMinor }
	if in.OccurredAt != nil { t.OccurredAt = *in.OccurredAt }
	if in.Description != nil { t.Description = *in.Description }
	if in.Notes != nil { t.Notes = *in.Notes }
	now := nowUTC()
	_, err = s.db.ExecContext(ctx, `UPDATE transactions SET amount_minor=?, occurred_at=?, description=?, notes=?, updated_at=? WHERE id=? AND workspace_id=?`, t.AmountMinor, t.OccurredAt, t.Description, t.Notes, now, id, workspaceID)
	if err != nil { return Transaction{}, err }
	if in.TagIDs != nil {
		if err := s.setTags(ctx, id, *in.TagIDs); err != nil { return Transaction{}, err }
	}
	return s.Get(ctx, workspaceID, id)
}

func (s *Service) Delete(ctx context.Context, workspaceID, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM transactions WHERE id = ? AND workspace_id = ?`, id, workspaceID)
	if err != nil { return err }
	n, _ := res.RowsAffected()
	if n == 0 { return ErrNotFound }
	return nil
}