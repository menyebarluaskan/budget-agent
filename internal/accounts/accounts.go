package accounts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Account struct {
	ID                  string `json:"id"`
	WorkspaceID         string `json:"workspace_id"`
	Name                string `json:"name"`
	Type                string `json:"type"`
	Currency            string `json:"currency"`
	OpeningBalanceMinor int64  `json:"opening_balance_minor"`
	Archived            bool   `json:"archived"`
	CreatedAt           string `json:"created_at"`
	UpdatedAt           string `json:"updated_at"`
	BalanceMinor        int64  `json:"balance_minor"`
}

type CreateInput struct {
	WorkspaceID         string
	Name                string
	Type                string
	Currency            string
	OpeningBalanceMinor int64
}

type UpdateInput struct {
	Name                *string
	Archived            *bool
	OpeningBalanceMinor *int64
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service { return &Service{db: db} }

func nowUTC() string { return time.Now().UTC().Format(time.RFC3339) }

func (s *Service) List(ctx context.Context, workspaceID string) ([]Account, error) {
	var out []Account
	rows2, err := s.db.QueryContext(ctx, `
		SELECT a.id, a.workspace_id, a.name, a.type, a.currency, a.opening_balance_minor, a.archived, a.created_at, a.updated_at
		FROM accounts a
		WHERE a.workspace_id = ?
		ORDER BY a.created_at`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()
	for rows2.Next() {
		var a Account
		var archived int
		if err := rows2.Scan(&a.ID, &a.WorkspaceID, &a.Name, &a.Type, &a.Currency, &a.OpeningBalanceMinor, &archived, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		a.Archived = archived == 1
		bal, err := s.balance(ctx, a.ID, a.OpeningBalanceMinor)
		if err != nil {
			return nil, err
		}
		a.BalanceMinor = bal
		out = append(out, a)
	}
	if err := rows2.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []Account{}
	}
	return out, nil
}

func (s *Service) balance(ctx context.Context, accountID string, opening int64) (int64, error) {
	var sum sql.NullInt64
	err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(amount_minor),0) FROM transactions WHERE account_id = ?`, accountID).Scan(&sum)
	if err != nil {
		return 0, err
	}
	v := int64(0)
	if sum.Valid {
		v = sum.Int64
	}
	return opening + v, nil
}

func (s *Service) Get(ctx context.Context, workspaceID, id string) (Account, error) {
	var a Account
	var archived int
	err := s.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, name, type, currency, opening_balance_minor, archived, created_at, updated_at
		FROM accounts WHERE id = ? AND workspace_id = ?`, id, workspaceID).Scan(
		&a.ID, &a.WorkspaceID, &a.Name, &a.Type, &a.Currency, &a.OpeningBalanceMinor, &archived, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Account{}, ErrNotFound
		}
		return Account{}, err
	}
	a.Archived = archived == 1
	bal, err := s.balance(ctx, a.ID, a.OpeningBalanceMinor)
	if err != nil {
		return Account{}, err
	}
	a.BalanceMinor = bal
	return a, nil
}

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("conflict")

func (s *Service) Create(ctx context.Context, in CreateInput) (Account, error) {
	if in.Name == "" {
		return Account{}, fmt.Errorf("name required")
	}
	if in.Type != "wallet" && in.Type != "savings" && in.Type != "credit" {
		return Account{}, fmt.Errorf("invalid type")
	}
	if len(in.Currency) != 3 {
		return Account{}, fmt.Errorf("invalid currency")
	}
	id := uuid.NewString()
	now := nowUTC()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO accounts (id, workspace_id, name, type, currency, opening_balance_minor, archived, created_at, updated_at)
		VALUES (?,?,?,?,?,?,0,?,?)`, id, in.WorkspaceID, in.Name, in.Type, in.Currency, in.OpeningBalanceMinor, now, now)
	if err != nil {
		return Account{}, err
	}
	return s.Get(ctx, in.WorkspaceID, id)
}

func (s *Service) Update(ctx context.Context, workspaceID, id string, in UpdateInput) (Account, error) {
	a, err := s.Get(ctx, workspaceID, id)
	if err != nil {
		return Account{}, err
	}
	if in.Name != nil {
		a.Name = *in.Name
	}
	if in.Archived != nil {
		a.Archived = *in.Archived
	}
	if in.OpeningBalanceMinor != nil {
		a.OpeningBalanceMinor = *in.OpeningBalanceMinor
	}
	archived := 0
	if a.Archived {
		archived = 1
	}
	now := nowUTC()
	_, err = s.db.ExecContext(ctx, `
		UPDATE accounts SET name=?, archived=?, opening_balance_minor=?, updated_at=? WHERE id=? AND workspace_id=?`,
		a.Name, archived, a.OpeningBalanceMinor, now, id, workspaceID)
	if err != nil {
		return Account{}, err
	}
	return s.Get(ctx, workspaceID, id)
}

func (s *Service) Delete(ctx context.Context, workspaceID, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM accounts WHERE id = ? AND workspace_id = ?`, id, workspaceID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
