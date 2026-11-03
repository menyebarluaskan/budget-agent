package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"budget-agent/internal/budgets"
)

func postBudget(t *testing.T, handler http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/budgets", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestCreateBudget(t *testing.T) {
	handler, _ := setupTransactionTest(t)

	rec := postBudget(t, handler, `{"name":"Groceries","amount_minor":500000,"period":"monthly","start_date":"2026-11-01","rollover":true,"tag_ids":["food"]}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var budget budgets.Budget
	if err := json.NewDecoder(rec.Body).Decode(&budget); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if budget.Name != "Groceries" || budget.AmountMinor != 500000 || budget.Period != "monthly" || !budget.Rollover {
		t.Errorf("budget = %+v, want Groceries monthly 500000 with rollover", budget)
	}
	if len(budget.TagIDs) != 1 || budget.TagIDs[0] != "food" {
		t.Errorf("tags = %v, want [food]", budget.TagIDs)
	}
}

func TestCreateBudgetRejectsInvalidInput(t *testing.T) {
	handler, _ := setupTransactionTest(t)

	tests := []struct {
		name string
		body string
	}{
		{name: "missing name", body: `{"amount_minor":1000,"period":"monthly","start_date":"2026-11-01"}`},
		{name: "zero amount", body: `{"name":"Food","amount_minor":0,"period":"monthly","start_date":"2026-11-01"}`},
		{name: "bad period", body: `{"name":"Food","amount_minor":1000,"period":"daily","start_date":"2026-11-01"}`},
		{name: "missing start date", body: `{"name":"Food","amount_minor":1000,"period":"monthly"}`},
		{name: "bad start date", body: `{"name":"Food","amount_minor":1000,"period":"monthly","start_date":"11/01/2026"}`},
		{name: "end before start", body: `{"name":"Food","amount_minor":1000,"period":"monthly","start_date":"2026-11-10","end_date":"2026-11-01"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := postBudget(t, handler, tt.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
		})
	}
}
