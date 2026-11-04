package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"budget-agent/internal/analytics"
	"budget-agent/internal/budgets"
)

func TestSpendingAnalyticsEndpoint(t *testing.T) {
	handler, _ := setupTransactionTest(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/spending?from=2026-11-01&to=2026-11-30&months=3", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var result analytics.Spending
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.TotalSpentMinor != 40000 {
		t.Errorf("total spent = %d, want 40000", result.TotalSpentMinor)
	}
	if len(result.Tags) != 2 || result.Tags[0].TagID != "food" || result.Tags[0].SpentMinor != 35000 {
		t.Errorf("tags = %+v, want Food 35000 first", result.Tags)
	}
	if len(result.Trend) != 3 || result.Trend[2].Month != "2026-11" {
		t.Fatalf("trend = %+v, want three months ending in November", result.Trend)
	}
	if result.Trend[2].IncomeMinor != 50000 || result.Trend[2].ExpenseMinor != 40000 || result.Trend[2].NetMinor != 10000 {
		t.Errorf("November trend = %+v, want income 50000, expense 40000, net 10000", result.Trend[2])
	}
}

func TestBudgetTrendEndpoint(t *testing.T) {
	handler, _ := setupTransactionTest(t)

	rec := postBudget(t, handler, `{"name":"Groceries","amount_minor":500000,"period":"monthly","start_date":"2026-10-01","rollover":false,"tag_ids":["food"]}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create budget status = %d; body = %s", rec.Code, rec.Body.String())
	}
	var budget budgets.Budget
	if err := json.NewDecoder(rec.Body).Decode(&budget); err != nil {
		t.Fatalf("decode budget: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/budgets/"+budget.ID+"/trend?to=2026-11-30&periods=2", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var result analytics.BudgetTrend
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result.BudgetID != budget.ID || result.AmountMinor != 500000 {
		t.Errorf("budget = %+v, want id %s amount 500000", result, budget.ID)
	}
	if len(result.Periods) != 2 {
		t.Fatalf("periods = %+v, want two", result.Periods)
	}
	if result.Periods[0].Start != "2026-10-01" || result.Periods[0].SpentMinor != 0 {
		t.Errorf("October = %+v, want zero spent", result.Periods[0])
	}
	if result.Periods[1].Start != "2026-11-01" || result.Periods[1].SpentMinor != 35000 {
		t.Errorf("November = %+v, want spent 35000", result.Periods[1])
	}
}

func TestAnalyticsEndpointsRejectInvalidInput(t *testing.T) {
	handler, _ := setupTransactionTest(t)

	cases := []string{
		"/api/v1/analytics/spending?from=2026-11-01&to=2026-11-30&months=0",
		"/api/v1/analytics/spending?from=2026-11-01&to=2026-11-30&months=25",
		"/api/v1/analytics/spending?from=2026-11-30&to=2026-11-01",
		"/api/v1/budgets/missing/trend?to=30-11-2026",
		"/api/v1/budgets/missing/trend?periods=0",
	}
	for _, target := range cases {
		t.Run(target, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, target, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
		})
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/budgets/missing/trend?to=2026-11-30", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}
