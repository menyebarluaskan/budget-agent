package analytics

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"budget-agent/internal/accounts"
	"budget-agent/internal/budgets"
	"budget-agent/internal/db"
)

func setupAnalytics(t *testing.T) (*Service, *sql.DB) {
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
		`INSERT INTO accounts (id, workspace_id, name, type, currency, opening_balance_minor, archived, created_at, updated_at)
		 VALUES ('w1','default','Wallet','wallet','IDR',0,0,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`,
		`INSERT INTO tags (id, workspace_id, name, color, created_at) VALUES
		 ('food','default','Food','#f97316','2026-01-01T00:00:00Z'),
		 ('transport','default','Transport','#0ea5e9','2026-01-01T00:00:00Z')`,
		`INSERT INTO transactions (id, workspace_id, account_id, amount_minor, occurred_at, description, notes, source, transfer_id, created_at, updated_at) VALUES
		 ('t1','default','w1',-5000,'2026-09-15T10:00:00Z','September groceries','','manual',NULL,'2026-09-15T10:00:00Z','2026-09-15T10:00:00Z'),
		 ('t2','default','w1',-12000,'2026-10-10T10:00:00Z','October groceries','','manual',NULL,'2026-10-10T10:00:00Z','2026-10-10T10:00:00Z'),
		 ('t3','default','w1',-10000,'2026-11-01T10:00:00Z','Coffee','','manual',NULL,'2026-11-01T10:00:00Z','2026-11-01T10:00:00Z'),
		 ('t4','default','w1',-25000,'2026-11-02T10:00:00Z','Groceries','','manual',NULL,'2026-11-02T10:00:00Z','2026-11-02T10:00:00Z'),
		 ('t5','default','w1',50000,'2026-11-03T10:00:00Z','Salary','','import',NULL,'2026-11-03T10:00:00Z','2026-11-03T10:00:00Z'),
		 ('t6','default','w1',-5000,'2026-11-04T10:00:00Z','Taxi','','manual',NULL,'2026-11-04T10:00:00Z','2026-11-04T10:00:00Z'),
		 ('t7','default','w1',-20000,'2026-11-05T10:00:00Z','Move to savings','','manual','tr1','2026-11-05T10:00:00Z','2026-11-05T10:00:00Z'),
		 ('t8','default','w1',20000,'2026-11-06T10:00:00Z','Move from savings','','manual','tr1','2026-11-06T10:00:00Z','2026-11-06T10:00:00Z'),
		 ('t9','default','w1',-7000,'2026-11-07T10:00:00Z','Untagged snack','','manual',NULL,'2026-11-07T10:00:00Z','2026-11-07T10:00:00Z')`,
		`INSERT INTO transaction_tags (transaction_id, tag_id) VALUES
		 ('t1','food'),('t2','food'),('t3','food'),('t4','food'),('t6','transport'),('t7','transport'),('t8','transport')`,
	}
	for _, statement := range seed {
		if _, err := database.Exec(statement); err != nil {
			t.Fatalf("seed database: %v", err)
		}
	}
	return NewService(database, "IDR"), database
}

func TestSpendingAggregatesTagsAndTrend(t *testing.T) {
	svc, _ := setupAnalytics(t)
	from := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC)

	got, err := svc.Spending(context.Background(), "default", from, to, 3)
	if err != nil {
		t.Fatalf("spending: %v", err)
	}

	if got.From != "2026-11-01" || got.To != "2026-11-30" || got.Currency != "IDR" {
		t.Errorf("range = %s..%s %s, want 2026-11-01..2026-11-30 IDR", got.From, got.To, got.Currency)
	}
	if got.TotalSpentMinor != 47000 {
		t.Errorf("total spent = %d, want 47000", got.TotalSpentMinor)
	}
	if got.UntaggedMinor != 7000 {
		t.Errorf("untagged = %d, want 7000", got.UntaggedMinor)
	}
	if len(got.Tags) != 2 {
		t.Fatalf("tags = %+v, want two rows", got.Tags)
	}
	if got.Tags[0].TagID != "food" || got.Tags[0].SpentMinor != 35000 || got.Tags[0].Name != "Food" {
		t.Errorf("first tag = %+v, want Food 35000", got.Tags[0])
	}
	if got.Tags[1].TagID != "transport" || got.Tags[1].SpentMinor != 5000 {
		t.Errorf("second tag = %+v, want Transport 5000", got.Tags[1])
	}

	if len(got.Trend) != 3 {
		t.Fatalf("trend = %+v, want three months", got.Trend)
	}
	want := []MonthPoint{
		{Month: "2026-09", IncomeMinor: 0, ExpenseMinor: 5000, NetMinor: -5000},
		{Month: "2026-10", IncomeMinor: 0, ExpenseMinor: 12000, NetMinor: -12000},
		{Month: "2026-11", IncomeMinor: 50000, ExpenseMinor: 47000, NetMinor: 3000},
	}
	for i, point := range want {
		if got.Trend[i] != point {
			t.Errorf("trend[%d] = %+v, want %+v", i, got.Trend[i], point)
		}
	}
}

func TestSpendingRangeExcludesEarlierMonthsFromTags(t *testing.T) {
	svc, _ := setupAnalytics(t)
	from := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC)

	got, err := svc.Spending(context.Background(), "default", from, to, 1)
	if err != nil {
		t.Fatalf("spending: %v", err)
	}
	if got.TotalSpentMinor != 37000 {
		t.Errorf("total spent = %d, want 37000", got.TotalSpentMinor)
	}
	if len(got.Trend) != 1 || got.Trend[0].Month != "2026-11" || got.Trend[0].ExpenseMinor != 47000 {
		t.Errorf("trend = %+v, want only November with the full-month 47000 expenses", got.Trend)
	}
}

func TestBudgetTrendUsesMatchingExpensesOnly(t *testing.T) {
	svc, database := setupAnalytics(t)
	budget, err := budgets.NewService(database).Create(context.Background(), budgets.CreateInput{
		WorkspaceID: "default",
		Name:        "Food",
		AmountMinor: 30000,
		Period:      "monthly",
		StartDate:   "2026-10-01",
		Rollover:    false,
		TagIDs:      []string{"food"},
	})
	if err != nil {
		t.Fatalf("create budget: %v", err)
	}

	got, err := svc.BudgetTrend(
		context.Background(),
		"default",
		budget.ID,
		time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC),
		6,
	)
	if err != nil {
		t.Fatalf("budget trend: %v", err)
	}
	if got.Name != "Food" || got.Period != "monthly" || got.AmountMinor != 30000 {
		t.Errorf("budget = %+v, want Food monthly 30000", got)
	}
	if len(got.Periods) != 2 {
		t.Fatalf("periods = %+v, want two", got.Periods)
	}
	if got.Periods[0].Start != "2026-10-01" || got.Periods[0].End != "2026-10-31" || got.Periods[0].SpentMinor != 12000 {
		t.Errorf("October = %+v, want spent 12000", got.Periods[0])
	}
	if got.Periods[0].RemainingMinor != 18000 {
		t.Errorf("October remaining = %d, want 18000", got.Periods[0].RemainingMinor)
	}
	if got.Periods[1].Start != "2026-11-01" || got.Periods[1].End != "2026-11-30" || got.Periods[1].SpentMinor != 35000 {
		t.Errorf("November = %+v, want spent 35000", got.Periods[1])
	}
	if got.Periods[1].RemainingMinor != -5000 {
		t.Errorf("November remaining = %d, want -5000", got.Periods[1].RemainingMinor)
	}
}

func TestBudgetTrendTruncatesToRequestedPeriods(t *testing.T) {
	svc, database := setupAnalytics(t)
	budget, err := budgets.NewService(database).Create(context.Background(), budgets.CreateInput{
		WorkspaceID: "default",
		Name:        "Food",
		AmountMinor: 30000,
		Period:      "monthly",
		StartDate:   "2026-10-01",
		TagIDs:      []string{"food"},
	})
	if err != nil {
		t.Fatalf("create budget: %v", err)
	}

	got, err := svc.BudgetTrend(
		context.Background(),
		"default",
		budget.ID,
		time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC),
		1,
	)
	if err != nil {
		t.Fatalf("budget trend: %v", err)
	}
	if len(got.Periods) != 1 || got.Periods[0].Start != "2026-11-01" {
		t.Errorf("periods = %+v, want only November", got.Periods)
	}
}

func TestBudgetTrendUnknownBudget(t *testing.T) {
	svc, _ := setupAnalytics(t)
	_, err := svc.BudgetTrend(context.Background(), "default", "missing", time.Now(), 6)
	if !errors.Is(err, budgets.ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}
