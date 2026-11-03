package dashboard

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"budget-agent/internal/accounts"
	"budget-agent/internal/db"
)

func TestBuildDashboard(t *testing.T) {
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

	seed := []string{
		`INSERT INTO accounts (id, workspace_id, name, type, currency, opening_balance_minor, archived, created_at, updated_at) VALUES
			('w1','default','Wallet','wallet','IDR',100000,0,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`,
		`INSERT INTO accounts (id, workspace_id, name, type, currency, opening_balance_minor, archived, created_at, updated_at) VALUES
			('c1','default','Credit','credit','IDR',-50000,0,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`,
		`INSERT INTO accounts (id, workspace_id, name, type, currency, opening_balance_minor, archived, created_at, updated_at) VALUES
			('a1','default','Old','wallet','IDR',123456,1,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`,
		`INSERT INTO tags (id, workspace_id, name, color, created_at) VALUES ('food','default','Food','#f00','2026-01-01T00:00:00Z')`,
		`INSERT INTO tags (id, workspace_id, name, color, created_at) VALUES ('travel','default','Travel','#0f0','2026-01-01T00:00:00Z')`,
		`INSERT INTO budgets (id, workspace_id, name, amount_minor, period, start_date, end_date, rollover, account_id, created_at, updated_at) VALUES
			('b1','default','Food','50000','monthly','2026-11-01',NULL,0,NULL,'2026-10-01T00:00:00Z','2026-10-01T00:00:00Z')`,
		`INSERT INTO budget_tags (budget_id, tag_id) VALUES ('b1','food')`,
		`INSERT INTO budgets (id, workspace_id, name, amount_minor, period, start_date, end_date, rollover, account_id, created_at, updated_at) VALUES
			('b2','default','Travel','20000','monthly','2026-11-01',NULL,0,'c1','2026-10-01T00:00:00Z','2026-10-01T00:00:00Z')`,
		`INSERT INTO budget_tags (budget_id, tag_id) VALUES ('b2','travel')`,
		`INSERT INTO budgets (id, workspace_id, name, amount_minor, period, start_date, end_date, rollover, account_id, created_at, updated_at) VALUES
			('b3','default','Later','10000','monthly','2026-12-01',NULL,0,NULL,'2026-10-01T00:00:00Z','2026-10-01T00:00:00Z')`,
		`INSERT INTO transactions (id, workspace_id, account_id, amount_minor, occurred_at, description, notes, source, transfer_id, created_at, updated_at) VALUES
			('t1','default','w1',-30000,'2026-11-05T10:00:00Z','Groceries','','manual',NULL,'2026-11-05T10:00:00Z','2026-11-05T10:00:00Z')`,
		`INSERT INTO transaction_tags (transaction_id, tag_id) VALUES ('t1','food')`,
		`INSERT INTO transactions (id, workspace_id, account_id, amount_minor, occurred_at, description, notes, source, transfer_id, created_at, updated_at) VALUES
			('t2','default','w1',10000,'2026-11-10T10:00:00Z','Refund','','manual',NULL,'2026-11-10T10:00:00Z','2026-11-10T10:00:00Z')`,
		`INSERT INTO transaction_tags (transaction_id, tag_id) VALUES ('t2','food')`,
		`INSERT INTO transactions (id, workspace_id, account_id, amount_minor, occurred_at, description, notes, source, transfer_id, created_at, updated_at) VALUES
			('t3','default','c1',-5000,'2026-11-15T10:00:00Z','Train','','manual',NULL,'2026-11-15T10:00:00Z','2026-11-15T10:00:00Z')`,
		`INSERT INTO transaction_tags (transaction_id, tag_id) VALUES ('t3','travel')`,
		`INSERT INTO transactions (id, workspace_id, account_id, amount_minor, occurred_at, description, notes, source, transfer_id, created_at, updated_at) VALUES
			('t4','default','w1',-99999,'2026-10-20T10:00:00Z','Old groceries','','manual',NULL,'2026-10-20T10:00:00Z','2026-10-20T10:00:00Z')`,
		`INSERT INTO transaction_tags (transaction_id, tag_id) VALUES ('t4','food')`,
		`INSERT INTO transactions (id, workspace_id, account_id, amount_minor, occurred_at, description, notes, source, transfer_id, created_at, updated_at) VALUES
			('t5','default','w1',-7000,'2026-11-20T10:00:00Z','Transfer to savings','','manual','transfer-1','2026-11-20T10:00:00Z','2026-11-20T10:00:00Z')`,
		`INSERT INTO transaction_tags (transaction_id, tag_id) VALUES ('t5','food')`,
	}
	for _, statement := range seed {
		if _, err := database.Exec(statement); err != nil {
			t.Fatalf("seed %q: %v", statement, err)
		}
	}

	svc := NewService(database, "IDR")
	from, _ := time.Parse("2006-01-02", "2026-11-01")
	to, _ := time.Parse("2006-01-02", "2026-11-30")
	result, err := svc.Build(ctx, "default", from, to)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if result.Currency != "IDR" {
		t.Errorf("currency = %q, want IDR", result.Currency)
	}
	if result.AccountCount != 3 {
		t.Errorf("account count = %d, want 3", result.AccountCount)
	}
	if result.TotalBalanceMinor != -81999 {
		t.Errorf("total balance = %d, want -81999", result.TotalBalanceMinor)
	}
	if result.IncomeMinor != 10000 {
		t.Errorf("income = %d, want 10000", result.IncomeMinor)
	}
	if result.ExpenseMinor != 35000 {
		t.Errorf("expense = %d, want 35000", result.ExpenseMinor)
	}
	if result.NetMinor != -25000 {
		t.Errorf("net = %d, want -25000", result.NetMinor)
	}
	if len(result.RecentTransactions) != 4 {
		t.Fatalf("recent transactions = %d, want 4", len(result.RecentTransactions))
	}
	if result.RecentTransactions[0].ID != "t5" {
		t.Errorf("most recent = %s, want t5", result.RecentTransactions[0].ID)
	}

	if len(result.Budgets) != 2 {
		t.Fatalf("budgets = %d, want 2", len(result.Budgets))
	}
	food := findBudget(t, result.Budgets, "b1")
	if food.SpentMinor != 20000 {
		t.Errorf("food spent = %d, want 20000", food.SpentMinor)
	}
	if food.RemainingMinor != 30000 {
		t.Errorf("food remaining = %d, want 30000", food.RemainingMinor)
	}
	if food.PeriodStart != "2026-11-01" || food.PeriodEnd != "2026-11-30" {
		t.Errorf("food period = %s..%s, want 2026-11-01..2026-11-30", food.PeriodStart, food.PeriodEnd)
	}
	travel := findBudget(t, result.Budgets, "b2")
	if travel.SpentMinor != 5000 || travel.RemainingMinor != 15000 {
		t.Errorf("travel = spent %d remaining %d, want 5000 and 15000", travel.SpentMinor, travel.RemainingMinor)
	}
}

func findBudget(t *testing.T, list []BudgetProgress, id string) BudgetProgress {
	t.Helper()
	for _, budget := range list {
		if budget.ID == id {
			return budget
		}
	}
	t.Fatalf("budget %s not found", id)
	return BudgetProgress{}
}
