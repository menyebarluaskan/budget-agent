package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"budget-agent/internal/accounts"
	"budget-agent/internal/dashboard"
	"budget-agent/internal/db"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestHealthz(t *testing.T) {
	handler := New(Options{Logger: testLogger()})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	if got := rec.Header().Get(requestIDHeader); got == "" {
		t.Error("request ID header is empty")
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
	}

	assertStatusBody(t, rec, "ok")
}

func TestReadyzReady(t *testing.T) {
	handler := New(Options{
		Logger: testLogger(),
		Ready:  func(context.Context) error { return nil },
	})
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	assertStatusBody(t, rec, "ready")
}

func TestReadyzNotReady(t *testing.T) {
	handler := New(Options{
		Logger: testLogger(),
		Ready:  func(context.Context) error { return errors.New("database unavailable") },
	})
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	assertStatusBody(t, rec, "not ready")
}

func TestMethodsNotAllowed(t *testing.T) {
	handler := New(Options{Logger: testLogger()})
	req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestUnknownRoute(t *testing.T) {
	handler := New(Options{Logger: testLogger()})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/does-not-exist", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestDashboardBuildsReadModel(t *testing.T) {
	ctx := context.Background()
	database, err := db.Open(filepath.Join(t.TempDir(), "budget.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer database.Close()
	if err := db.Migrate(database); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := accounts.EnsureDefaultWorkspace(ctx, database); err != nil {
		t.Fatalf("bootstrap workspace: %v", err)
	}
	if _, err := database.Exec(`
		INSERT INTO accounts (id, workspace_id, name, type, currency, opening_balance_minor, archived, created_at, updated_at)
		VALUES ('w1','default','Wallet','wallet','IDR',100000,0,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`); err != nil {
		t.Fatalf("seed account: %v", err)
	}
	if _, err := database.Exec(`
		INSERT INTO transactions (id, workspace_id, account_id, amount_minor, occurred_at, description, notes, source, transfer_id, created_at, updated_at)
		VALUES ('t1','default','w1',-25000,'2026-11-05T10:00:00Z','Groceries','','manual',NULL,'2026-11-05T10:00:00Z','2026-11-05T10:00:00Z')`); err != nil {
		t.Fatalf("seed transaction: %v", err)
	}

	handler := New(Options{Logger: testLogger(), DB: database, DefaultCurrency: "IDR"})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard?from=2026-11-01&to=2026-11-30", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var result dashboard.Result
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.Currency != "IDR" {
		t.Errorf("currency = %q, want IDR", result.Currency)
	}
	if result.TotalBalanceMinor != 75000 {
		t.Errorf("total balance = %d, want 75000", result.TotalBalanceMinor)
	}
	if result.ExpenseMinor != 25000 {
		t.Errorf("expense = %d, want 25000", result.ExpenseMinor)
	}
	if len(result.RecentTransactions) != 1 || result.RecentTransactions[0].ID != "t1" {
		t.Errorf("recent transactions = %+v, want t1", result.RecentTransactions)
	}
}

func TestDashboardRejectsInvalidRange(t *testing.T) {
	database, err := db.Open(filepath.Join(t.TempDir(), "budget.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer database.Close()
	if err := db.Migrate(database); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	handler := New(Options{Logger: testLogger(), DB: database})

	cases := []string{
		"/api/v1/dashboard?from=2026-11-01",
		"/api/v1/dashboard?to=2026-11-30",
		"/api/v1/dashboard?from=2026-11-30&to=2026-11-01",
		"/api/v1/dashboard?from=11-01-2026&to=2026-11-30",
	}
	for _, target := range cases {
		t.Run(target, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, target, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestPanicRecovery(t *testing.T) {
	handler := withMiddleware(testLogger(), http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", got)
	}
}

func assertStatusBody(t *testing.T, rec *httptest.ResponseRecorder, want string) {
	t.Helper()

	var body statusResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != want {
		t.Errorf("status body = %q, want %q", body.Status, want)
	}
}
