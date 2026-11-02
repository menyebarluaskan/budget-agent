package budgets

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Budget struct {
	ID          string  `json:"id"`
	WorkspaceID string  `json:"workspace_id"`
	Name        string  `json:"name"`
	AmountMinor int64   `json:"amount_minor"`
	Period      string  `json:"period"`
	StartDate   string  `json:"start_date"`
	EndDate     *string `json:"end_date,omitempty"`
	Rollover    bool    `json:"rollover"`
	AccountID   *string `json:"account_id,omitempty"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type CreateInput struct {
	WorkspaceID string
	Name        string
	AmountMinor int64
	Period      string
	StartDate   string
	EndDate     *string
	Rollover    bool
	AccountID   *string
}

type UpdateInput struct {
	Name        *string
	AmountMinor *int64
	Period      *string
	StartDate   *string
	EndDate     *string
	Rollover    *bool
	AccountID   *string
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service { return &Service{db: db} }

func (s *Service) List(ctx context.Context, workspaceID string) ([]Budget, error) {
	var out []Budget
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, workspace_id, name, amount_minor, period, start_date, end_date, rollover, account_id, created_at, updated_at
		FROM budgets
		WHERE workspace_id = ?
		ORDER BY created_at DESC`,
		workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var b Budget
		var endDate sql.NullString
		if err := rows.Scan(&b.ID, &b.WorkspaceID, &b.Name, &b.AmountMinor, &b.Period, &b.StartDate, &endDate, &b.Rollover, &b.AccountID, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		if endDate.Valid {
			b.EndDate = &endDate.String
		} else {
			b.EndDate = nil
		}
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []Budget{}
	}
	return out, nil
}

func (s *Service) Get(ctx context.Context, workspaceID, id string) (Budget, error) {
	var b Budget
	var endDate sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, name, amount_minor, period, start_date, end_date, rollover, account_id, created_at, updated_at
		FROM budgets WHERE id = ? AND workspace_id = ?`,
		id, workspaceID).Scan(&b.ID, &b.WorkspaceID, &b.Name, &b.AmountMinor, &b.Period, &b.StartDate, &endDate, &b.Rollover, &b.AccountID, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Budget{}, ErrNotFound
		}
		return Budget{}, err
	}
	if endDate.Valid {
		b.EndDate = &endDate.String
	} else {
		b.EndDate = nil
	}
	return b, nil
}

func (s *Service) Create(ctx context.Context, in CreateInput) (Budget, error) {
	if in.Name == "" {
		return Budget{}, fmt.Errorf("name required")
	}
	if in.Period != "weekly" && in.Period != "monthly" {
		return Budget{}, fmt.Errorf("invalid period, must be 'weekly' or 'monthly'")
	}
	id := uuid.NewString()
	now := nowUTC()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO budgets (id, workspace_id, name, amount_minor, period, start_date, end_date, rollover, account_id, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		id, in.WorkspaceID, in.Name, in.AmountMinor, in.Period, in.StartDate, in.EndDate, in.Rollover, in.AccountID, now, now)
	if err != nil {
		return Budget{}, err
	}
	return s.Get(ctx, in.WorkspaceID, id)
}

func (s *Service) Update(ctx context.Context, workspaceID, id string, in UpdateInput) (Budget, error) {
	b, err := s.Get(ctx, workspaceID, id)
	if err != nil {
		return Budget{}, err
	}
	if in.Name != nil {
		b.Name = *in.Name
	}
	if in.AmountMinor != nil {
		b.AmountMinor = *in.AmountMinor
	}
	if in.Period != nil {
		b.Period = *in.Period
	}
	if in.StartDate != nil {
		b.StartDate = *in.StartDate
	}
	if in.EndDate != nil {
		b.EndDate = in.EndDate
	}
	if in.Rollover != nil {
		b.Rollover = *in.Rollover
	}
	if in.AccountID != nil {
		b.AccountID = in.AccountID
	}
	now := nowUTC()
	_, err = s.db.ExecContext(ctx, `
		UPDATE budgets SET name=?, amount_minor=?, period=?, start_date=?, end_date=?, rollover=?, account_id=?, updated_at=? WHERE id=? AND workspace_id=?`,
		b.Name, b.AmountMinor, b.Period, b.StartDate, b.EndDate, b.Rollover, b.AccountID, now, id, workspaceID)
	if err != nil {
		return Budget{}, err
	}
	return s.Get(ctx, workspaceID, id)
}

func (s *Service) Delete(ctx context.Context, workspaceID, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM budgets WHERE id = ? AND workspace_id = ?`, id, workspaceID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// SummaryOptions configures the budget summary query.
type SummaryOptions struct {
	TagID string
}

// Summarize returns tag-based budget summaries showing spent amounts per tag.
func (s *Service) Summarize(ctx context.Context, workspaceID string, opts SummaryOptions) ([]BudgetSummary, error) {
	q := `
		SELECT t.id, t.name, t.color,
		       COALESCE(SUM(tx.amount_minor), 0) AS spent_minor
		FROM tags t
		JOIN budget_tags bt ON bt.tag_id = t.id
		JOIN budgets b ON b.id = bt.budget_id AND b.workspace_id = t.workspace_id
		JOIN transactions tx ON tx.workspace_id = t.workspace_id
		WHERE b.workspace_id = ? AND b.period = 'monthly'`
	args := []any{workspaceID}

	if opts.TagID != "" {
		q += " AND t.id = ?"
		args = append(args, opts.TagID)
	}

	q += " GROUP BY t.id, t.name, t.color ORDER BY t.name"

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type tagRow struct {
		TagID   string
		TagName string
		Color   string
		Spent   int64
	}
	var out []BudgetSummary
	for rows.Next() {
		var tr tagRow
		if err := rows.Scan(&tr.TagID, &tr.TagName, &tr.Color, &tr.Spent); err != nil {
			return nil, err
		}
		out = append(out, BudgetSummary{
			TagID:      tr.TagID,
			TagName:    tr.TagName,
			Color:      tr.Color,
			SpentMinor: tr.Spent,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []BudgetSummary{}
	}
	return out, nil
}

type BudgetSummary struct {
	TagID      string `json:"tag_id"`
	TagName    string `json:"tag_name"`
	Color      string `json:"color"`
	SpentMinor int64  `json:"spent_minor"`
}

var (
	ErrNotFound = errors.New("not found")
)

func nowUTC() string { return time.Now().UTC().Format(time.RFC3339) }
