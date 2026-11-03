package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"budget-agent/internal/accounts"
	"budget-agent/internal/db"
	"budget-agent/internal/transactions"
)

type transactionPageResponse struct {
	Transactions []transactions.Transaction `json:"transactions"`
	NextCursor   string                     `json:"next_cursor"`
}

func setupTransactionTest(t *testing.T) (http.Handler, *sql.DB) {
	t.Helper()
	ctx := context.Background()
	database, err := db.Open(filepath.Join(t.TempDir(), "budget.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := db.Migrate(database); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := accounts.EnsureDefaultWorkspace(ctx, database); err != nil {
		t.Fatalf("bootstrap workspace: %v", err)
	}
	seed := []string{
		`INSERT INTO accounts (id, workspace_id, name, type, currency, opening_balance_minor, archived, created_at, updated_at) VALUES
		 ('w1','default','Wallet','wallet','IDR',0,0,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z'),
		 ('w2','default','Savings','savings','IDR',0,0,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`,
		`INSERT INTO tags (id, workspace_id, name, color, created_at) VALUES
		 ('food','default','Food','#f97316','2026-01-01T00:00:00Z'),
		 ('transport','default','Transport','#0ea5e9','2026-01-01T00:00:00Z')`,
		`INSERT INTO transactions (id, workspace_id, account_id, amount_minor, occurred_at, description, notes, source, transfer_id, created_at, updated_at) VALUES
		 ('t1','default','w1',-10000,'2026-11-01T10:00:00Z','Coffee','morning','manual',NULL,'2026-11-01T10:00:00Z','2026-11-01T10:00:00Z'),
		 ('t2','default','w1',-25000,'2026-11-02T10:00:00Z','Groceries','','manual',NULL,'2026-11-02T10:00:00Z','2026-11-02T10:00:00Z'),
		 ('t3','default','w2',50000,'2026-11-03T10:00:00Z','Salary','','import',NULL,'2026-11-03T10:00:00Z','2026-11-03T10:00:00Z'),
		 ('t4','default','w1',-5000,'2026-11-04T10:00:00Z','Taxi','airport','whatsapp',NULL,'2026-11-04T10:00:00Z','2026-11-04T10:00:00Z'),
		 ('t5','default','w1',-20000,'2026-11-05T10:00:00Z','Move to savings','','manual','tr1','2026-11-05T10:00:00Z','2026-11-05T10:00:00Z'),
		 ('t6','default','w2',20000,'2026-11-05T10:00:00Z','Move from wallet','','manual','tr1','2026-11-05T10:00:00Z','2026-11-05T10:00:00Z')`,
		`INSERT INTO transaction_tags (transaction_id, tag_id) VALUES
		 ('t1','food'),('t2','food'),('t4','transport'),('t5','transport'),('t6','transport')`,
	}
	for _, statement := range seed {
		if _, err := database.Exec(statement); err != nil {
			t.Fatalf("seed database: %v", err)
		}
	}
	return New(Options{Logger: testLogger(), DB: database, DefaultCurrency: "IDR"}), database
}

func getTransactionPage(t *testing.T, handler http.Handler, target string) transactionPageResponse {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("%s status = %d, want %d; body = %s", target, rec.Code, http.StatusOK, rec.Body.String())
	}
	var page transactionPageResponse
	if err := json.NewDecoder(rec.Body).Decode(&page); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return page
}

func transactionIDs(page transactionPageResponse) []string {
	ids := make([]string, len(page.Transactions))
	for i, transaction := range page.Transactions {
		ids[i] = transaction.ID
	}
	return ids
}

func TestListTransactionsFilters(t *testing.T) {
	handler, _ := setupTransactionTest(t)

	tests := []struct {
		name  string
		query string
		want  []string
	}{
		{
			name:  "transfers are excluded by default",
			query: "/api/v1/transactions",
			want:  []string{"t4", "t3", "t2", "t1"},
		},
		{
			name:  "include transfers",
			query: "/api/v1/transactions?include_transfers=true",
			want:  []string{"t6", "t5", "t4", "t3", "t2", "t1"},
		},
		{
			name:  "account",
			query: "/api/v1/transactions?account_id=w1",
			want:  []string{"t4", "t2", "t1"},
		},
		{
			name:  "repeated tags are ORed",
			query: "/api/v1/transactions?tag=food&tag=transport",
			want:  []string{"t4", "t2", "t1"},
		},
		{
			name:  "search is case-insensitive",
			query: "/api/v1/transactions?q=TAXI",
			want:  []string{"t4"},
		},
		{
			name:  "search matches notes",
			query: "/api/v1/transactions?q=morning",
			want:  []string{"t1"},
		},
		{
			name:  "amount range",
			query: "/api/v1/transactions?min_amount_minor=-20000&max_amount_minor=-10000",
			want:  []string{"t1"},
		},
		{
			name:  "source",
			query: "/api/v1/transactions?source=whatsapp",
			want:  []string{"t4"},
		},
		{
			name:  "date-only bounds include the whole day",
			query: "/api/v1/transactions?from=2026-11-03&to=2026-11-04",
			want:  []string{"t4", "t3"},
		},
		{
			name:  "ascending sort",
			query: "/api/v1/transactions?sort=occurred_at",
			want:  []string{"t1", "t2", "t3", "t4"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := transactionIDs(getTransactionPage(t, handler, tt.query))
			if strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Errorf("ids = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListTransactionsPaginates(t *testing.T) {
	handler, _ := setupTransactionTest(t)

	first := getTransactionPage(t, handler, "/api/v1/transactions?sort=occurred_at&limit=2&include_transfers=true")
	if got, want := strings.Join(transactionIDs(first), ","), "t1,t2"; got != want {
		t.Fatalf("first page = %s, want %s", got, want)
	}
	if first.NextCursor == "" {
		t.Fatal("first page has no next cursor")
	}

	second := getTransactionPage(t, handler, "/api/v1/transactions?sort=occurred_at&limit=2&include_transfers=true&cursor="+first.NextCursor)
	if got, want := strings.Join(transactionIDs(second), ","), "t3,t4"; got != want {
		t.Fatalf("second page = %s, want %s", got, want)
	}

	third := getTransactionPage(t, handler, "/api/v1/transactions?sort=occurred_at&limit=2&include_transfers=true&cursor="+second.NextCursor)
	if got, want := strings.Join(transactionIDs(third), ","), "t5,t6"; got != want {
		t.Fatalf("third page = %s, want %s", got, want)
	}
	if third.NextCursor != "" {
		t.Errorf("third page has next cursor %q, want empty", third.NextCursor)
	}
}

func TestListTransactionsRejectsInvalidQueries(t *testing.T) {
	handler, _ := setupTransactionTest(t)

	targets := []string{
		"/api/v1/transactions?sort=name",
		"/api/v1/transactions?limit=0",
		"/api/v1/transactions?limit=201",
		"/api/v1/transactions?cursor=not-a-cursor",
		"/api/v1/transactions?min_amount_minor=-100&max_amount_minor=-200",
		"/api/v1/transactions?include_transfers=maybe",
		"/api/v1/transactions?from=2026-11-30&to=2026-11-01",
		"/api/v1/transactions?from=11-01-2026",
	}
	for _, target := range targets {
		t.Run(target, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, target, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
		})
	}
}

func TestUpdateTransactionCanChangeAccountAndSource(t *testing.T) {
	handler, _ := setupTransactionTest(t)

	body := strings.NewReader(`{"account_id":"w2","source":"mcp","description":"Moved"}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v1/transactions/t1", body)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var transaction transactions.Transaction
	if err := json.NewDecoder(rec.Body).Decode(&transaction); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if transaction.AccountID != "w2" || transaction.Source != "mcp" || transaction.Description != "Moved" {
		t.Errorf("updated transaction = %+v, want account w2, source mcp, description Moved", transaction)
	}
}
