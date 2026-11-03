// Package dashboard assembles the read model used by the web dashboard. It
// composes the account, budget, and transaction services so the UI does not
// duplicate money or budget-period logic.
package dashboard

import (
	"context"
	"database/sql"
	"time"

	"budget-agent/internal/accounts"
	"budget-agent/internal/budgets"
	"budget-agent/internal/transactions"
)

const recentTransactionLimit = 5

// Service builds dashboard summaries for one workspace.
type Service struct {
	accounts     *accounts.Service
	budgets      *budgets.Service
	transactions *transactions.Service
	currency     string
}

func NewService(db *sql.DB, defaultCurrency string) *Service {
	if defaultCurrency == "" {
		defaultCurrency = "IDR"
	}
	return &Service{
		accounts:     accounts.NewService(db),
		budgets:      budgets.NewService(db),
		transactions: transactions.NewService(db),
		currency:     defaultCurrency,
	}
}

// Result is the dashboard read model.
type Result struct {
	From               string                     `json:"from"`
	To                 string                     `json:"to"`
	Currency           string                     `json:"currency"`
	TotalBalanceMinor  int64                      `json:"total_balance_minor"`
	AccountCount       int                        `json:"account_count"`
	IncomeMinor        int64                      `json:"income_minor"`
	ExpenseMinor       int64                      `json:"expense_minor"`
	NetMinor           int64                      `json:"net_minor"`
	Budgets            []BudgetProgress           `json:"budgets"`
	RecentTransactions []transactions.Transaction `json:"recent_transactions"`
}

// BudgetProgress is the current-period progress for a budget.
type BudgetProgress struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Period         string `json:"period"`
	AmountMinor    int64  `json:"amount_minor"`
	SpentMinor     int64  `json:"spent_minor"`
	RemainingMinor int64  `json:"remaining_minor"`
	Rollover       bool   `json:"rollover"`
	PeriodStart    string `json:"period_start"`
	PeriodEnd      string `json:"period_end"`
}

// Build returns the dashboard for the inclusive date range [from, to].
func (s *Service) Build(ctx context.Context, workspaceID string, from, to time.Time) (Result, error) {
	from = startOfDay(from)
	to = startOfDay(to)
	toExclusive := to.AddDate(0, 0, 1)

	accountList, err := s.accounts.List(ctx, workspaceID)
	if err != nil {
		return Result{}, err
	}

	var totalBalance int64
	for _, account := range accountList {
		if !account.Archived {
			totalBalance += account.BalanceMinor
		}
	}

	budgetList, err := s.budgets.List(ctx, workspaceID)
	if err != nil {
		return Result{}, err
	}

	type activeBudget struct {
		budget  budgets.Budget
		periods []budgets.Period
	}
	active := make([]activeBudget, 0, len(budgetList))
	fetchFrom := from
	for _, budget := range budgetList {
		periods := budgets.PeriodsUpTo(budget, to)
		if len(periods) == 0 {
			continue
		}
		active = append(active, activeBudget{budget: budget, periods: periods})
		start := periods[len(periods)-1].Start
		if budget.Rollover {
			start = periods[0].Start
		}
		if start.Before(fetchFrom) {
			fetchFrom = start
		}
	}

	txList, err := s.transactions.List(ctx, workspaceID, transactions.Filter{
		From: fetchFrom.Format(time.RFC3339),
		To:   toExclusive.Add(-time.Second).Format(time.RFC3339),
	})
	if err != nil {
		return Result{}, err
	}

	result := Result{
		From:               from.Format("2006-01-02"),
		To:                 to.Format("2006-01-02"),
		Currency:           s.currency,
		TotalBalanceMinor:  totalBalance,
		AccountCount:       len(accountList),
		Budgets:            make([]BudgetProgress, 0, len(active)),
		RecentTransactions: []transactions.Transaction{},
	}

	for _, tx := range txList {
		occurred, err := time.Parse(time.RFC3339, tx.OccurredAt)
		if err != nil || occurred.Before(from) || !occurred.Before(toExclusive) {
			continue
		}
		// Transfers are shown in the recent list but never count as spending,
		// income, or budget progress.
		if len(result.RecentTransactions) < recentTransactionLimit {
			result.RecentTransactions = append(result.RecentTransactions, tx)
		}
		if tx.TransferID != nil {
			continue
		}
		result.NetMinor += tx.AmountMinor
		if tx.AmountMinor > 0 {
			result.IncomeMinor += tx.AmountMinor
		} else {
			result.ExpenseMinor += -tx.AmountMinor
		}
	}

	for _, entry := range active {
		current := entry.periods[len(entry.periods)-1]
		carry := int64(0)
		spent := int64(0)
		for i, period := range entry.periods {
			spent = spentInPeriod(entry.budget, period, txList)
			if i == len(entry.periods)-1 {
				break
			}
			if entry.budget.Rollover {
				carry += entry.budget.AmountMinor - spent
				if carry < 0 {
					carry = 0
				}
			}
		}
		result.Budgets = append(result.Budgets, BudgetProgress{
			ID:             entry.budget.ID,
			Name:           entry.budget.Name,
			Period:         entry.budget.Period,
			AmountMinor:    entry.budget.AmountMinor,
			SpentMinor:     spent,
			RemainingMinor: entry.budget.AmountMinor + carry - spent,
			Rollover:       entry.budget.Rollover,
			PeriodStart:    current.Start.Format("2006-01-02"),
			PeriodEnd:      current.End.AddDate(0, 0, -1).Format("2006-01-02"),
		})
	}

	return result, nil
}

func spentInPeriod(budget budgets.Budget, period budgets.Period, txList []transactions.Transaction) int64 {
	var sum int64
	for _, tx := range txList {
		if tx.TransferID != nil {
			continue
		}
		occurred, err := time.Parse(time.RFC3339, tx.OccurredAt)
		if err != nil || occurred.Before(period.Start) || !occurred.Before(period.End) {
			continue
		}
		if budget.AccountID != nil && *budget.AccountID != tx.AccountID {
			continue
		}
		if !hasAnyTag(tx.TagIDs, budget.TagIDs) {
			continue
		}
		sum += tx.AmountMinor
	}
	if sum >= 0 {
		return 0
	}
	return -sum
}

func hasAnyTag(transactionTags, budgetTags []string) bool {
	for _, transactionTag := range transactionTags {
		for _, budgetTag := range budgetTags {
			if transactionTag == budgetTag {
				return true
			}
		}
	}
	return false
}

func startOfDay(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
