package transactions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// TransferInput describes a move between two accounts. The amount is always
// positive; the service stores a negative leg on the source and a positive leg
// on the destination.
type TransferInput struct {
	WorkspaceID   string
	FromAccountID string
	ToAccountID   string
	AmountMinor   int64
	OccurredAt    string
	Description   string
	Notes         string
}

// Transfer is the pair of transaction rows that make up one transfer.
type Transfer struct {
	TransferID string      `json:"transfer_id"`
	From       Transaction `json:"from"`
	To         Transaction `json:"to"`
}

// CreateTransfer writes both legs in one database transaction. Cross-currency
// transfers are rejected because the ledger stores a single amount per leg.
func (s *Service) CreateTransfer(ctx context.Context, in TransferInput) (Transfer, error) {
	if in.FromAccountID == "" || in.ToAccountID == "" {
		return Transfer{}, fmt.Errorf("from_account_id and to_account_id are required")
	}
	if in.FromAccountID == in.ToAccountID {
		return Transfer{}, fmt.Errorf("from_account_id and to_account_id must differ")
	}
	if in.AmountMinor <= 0 {
		return Transfer{}, fmt.Errorf("amount_minor must be greater than zero")
	}
	if in.OccurredAt == "" {
		in.OccurredAt = nowUTC()
	}
	if err := validateOccurredAt(in.OccurredAt); err != nil {
		return Transfer{}, err
	}
	if in.Description == "" {
		in.Description = "Transfer"
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Transfer{}, err
	}
	defer func() { _ = tx.Rollback() }()

	fromCurrency, err := accountCurrency(ctx, tx, in.WorkspaceID, in.FromAccountID)
	if err != nil {
		return Transfer{}, err
	}
	toCurrency, err := accountCurrency(ctx, tx, in.WorkspaceID, in.ToAccountID)
	if err != nil {
		return Transfer{}, err
	}
	if fromCurrency != toCurrency {
		return Transfer{}, fmt.Errorf("accounts must use the same currency")
	}

	transferID := uuid.NewString()
	fromID := uuid.NewString()
	toID := uuid.NewString()
	now := nowUTC()
	const insert = `INSERT INTO transactions (id, workspace_id, account_id, amount_minor, occurred_at, description, notes, source, transfer_id, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`
	if _, err := tx.ExecContext(ctx, insert, fromID, in.WorkspaceID, in.FromAccountID, -in.AmountMinor, in.OccurredAt, in.Description, in.Notes, "manual", transferID, now, now); err != nil {
		return Transfer{}, err
	}
	if _, err := tx.ExecContext(ctx, insert, toID, in.WorkspaceID, in.ToAccountID, in.AmountMinor, in.OccurredAt, in.Description, in.Notes, "manual", transferID, now, now); err != nil {
		return Transfer{}, err
	}
	if err := tx.Commit(); err != nil {
		return Transfer{}, err
	}

	base := Transaction{
		WorkspaceID: in.WorkspaceID,
		OccurredAt:  in.OccurredAt,
		Description: in.Description,
		Notes:       in.Notes,
		Source:      "manual",
		TagIDs:      []string{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	from := base
	from.ID = fromID
	from.AccountID = in.FromAccountID
	from.AmountMinor = -in.AmountMinor
	from.TransferID = &transferID

	to := base
	to.ID = toID
	to.AccountID = in.ToAccountID
	to.AmountMinor = in.AmountMinor
	to.TransferID = &transferID

	return Transfer{TransferID: transferID, From: from, To: to}, nil
}

func accountCurrency(ctx context.Context, tx *sql.Tx, workspaceID, accountID string) (string, error) {
	var currency string
	err := tx.QueryRowContext(ctx, `SELECT currency FROM accounts WHERE id = ? AND workspace_id = ?`, accountID, workspaceID).Scan(&currency)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("%w: account %s", ErrNotFound, accountID)
		}
		return "", err
	}
	return currency, nil
}
