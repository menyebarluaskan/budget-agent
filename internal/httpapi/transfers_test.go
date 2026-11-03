package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"budget-agent/internal/transactions"
)

func postTransfer(t *testing.T, handler http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transfers", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestCreateTransferWritesBothLegs(t *testing.T) {
	handler, database := setupTransactionTest(t)

	rec := postTransfer(t, handler, `{"from_account_id":"w1","to_account_id":"w2","amount_minor":15000,"occurred_at":"2026-11-06T09:00:00Z","description":"Savings sweep","notes":"monthly"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var transfer transactions.Transfer
	if err := json.NewDecoder(rec.Body).Decode(&transfer); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if transfer.TransferID == "" || transfer.From.TransferID == nil || *transfer.From.TransferID != transfer.TransferID {
		t.Fatalf("transfer id mismatch: %+v", transfer)
	}
	if transfer.From.AmountMinor != -15000 || transfer.To.AmountMinor != 15000 {
		t.Errorf("legs = %d and %d, want -15000 and 15000", transfer.From.AmountMinor, transfer.To.AmountMinor)
	}
	if transfer.From.AccountID != "w1" || transfer.To.AccountID != "w2" {
		t.Errorf("accounts = %q and %q, want w1 and w2", transfer.From.AccountID, transfer.To.AccountID)
	}

	var legs int
	if err := database.QueryRow(`SELECT COUNT(*) FROM transactions WHERE transfer_id = ?`, transfer.TransferID).Scan(&legs); err != nil {
		t.Fatalf("count legs: %v", err)
	}
	if legs != 2 {
		t.Errorf("stored legs = %d, want 2", legs)
	}

	page := getTransactionPage(t, handler, "/api/v1/transactions?include_transfers=true&q=Savings+sweep")
	if len(page.Transactions) != 2 {
		t.Fatalf("listed transfer legs = %d, want 2", len(page.Transactions))
	}

	defaultPage := getTransactionPage(t, handler, "/api/v1/transactions?q=Savings+sweep")
	if len(defaultPage.Transactions) != 0 {
		t.Errorf("default list included transfer legs: %v", transactionIDs(defaultPage))
	}
}

func TestCreateTransferRejectsInvalidInput(t *testing.T) {
	handler, database := setupTransactionTest(t)
	if _, err := database.Exec(`INSERT INTO accounts (id, workspace_id, name, type, currency, opening_balance_minor, archived, created_at, updated_at)
		VALUES ('w3','default','Dollars','wallet','USD',0,0,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`); err != nil {
		t.Fatalf("seed usd account: %v", err)
	}

	tests := []struct {
		name   string
		body   string
		status int
	}{
		{
			name:   "missing destination",
			body:   `{"from_account_id":"w1","amount_minor":1000}`,
			status: http.StatusBadRequest,
		},
		{
			name:   "same account",
			body:   `{"from_account_id":"w1","to_account_id":"w1","amount_minor":1000}`,
			status: http.StatusBadRequest,
		},
		{
			name:   "zero amount",
			body:   `{"from_account_id":"w1","to_account_id":"w2","amount_minor":0}`,
			status: http.StatusBadRequest,
		},
		{
			name:   "negative amount",
			body:   `{"from_account_id":"w1","to_account_id":"w2","amount_minor":-100}`,
			status: http.StatusBadRequest,
		},
		{
			name:   "invalid timestamp",
			body:   `{"from_account_id":"w1","to_account_id":"w2","amount_minor":100,"occurred_at":"2026-11-06"}`,
			status: http.StatusBadRequest,
		},
		{
			name:   "unknown account",
			body:   `{"from_account_id":"missing","to_account_id":"w2","amount_minor":100}`,
			status: http.StatusNotFound,
		},
		{
			name:   "currency mismatch",
			body:   `{"from_account_id":"w1","to_account_id":"w3","amount_minor":100}`,
			status: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := postTransfer(t, handler, tt.body)
			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, tt.status, rec.Body.String())
			}
		})
	}
}
