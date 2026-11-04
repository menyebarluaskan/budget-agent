// Package analytics builds read-only aggregates for the spending charts and
// budget simulations. It never writes data.
package analytics

import (
	"context"
	"database/sql"
	"sort"
	"time"

	"budget-agent/internal/budgets"
	"budget-agent/internal/tags"
	"budget-agent/internal/transactions"
)

const (
	// DefaultTrendMonths is the default number of months in a trend chart.
	DefaultTrendMonths = 6
	// MaxTrendMonths bounds a trend request.
	MaxTrendMonths = 24
)

// Service builds analytics for one workspace.
type Service struct {
	budgets      *budgets.Service
	tags         *tags.Service
	transactions *transactions.Service
	currency     string
}

func NewService(db *sql.DB, defaultCurrency string) *Service {
	if defaultCurrency == "" {
		defaultCurrency = "IDR"
	}
	return &Service{
		budgets:      budgets.NewService(db),
		tags:         tags.NewService(db),
		transactions: transactions.NewService(db),
		currency:     defaultCurrency,
	}
}

// TagSpending is one row in the spending-by-tag chart.
type TagSpending struct {
	TagID      string `json:"tag_id"`
	Name       string `json:"name"`
	Color      string `json:"color"`
	SpentMinor int64  `json:"spent_minor"`
}

// MonthPoint is one month in the income/expense trend.
type MonthPoint struct {
	Month        string `json:"month"`
	IncomeMinor  int64  `json:"income_minor"`
	ExpenseMinor int64  `json:"expense_minor"`
	NetMinor     int64  `json:"net_minor"`
}

// Spending is the read model behind the dashboard charts.
type Spending struct {
	From            string        `json:"from"`
	To              string        `json:"to"`
	Currency        string        `json:"currency"`
	TotalSpentMinor int64         `json:"total_spent_minor"`
	UntaggedMinor   int64         `json:"untagged_minor"`
	Tags            []TagSpending `json:"tags"`
	Trend           []MonthPoint  `json:"trend"`
}

// BudgetPeriodSpending is one budget period in the burn-down chart.
type BudgetPeriodSpending struct {
	Start          string `json:"start"`
	End            string `json:"end"`
	SpentMinor     int64  `json:"spent_minor"`
	AmountMinor    int64  `json:"amount_minor"`
	RemainingMinor int64  `json:"remaining_minor"`
}

// BudgetTrend is the spending history for a single budget.
type BudgetTrend struct {
	BudgetID    string                 `json:"budget_id"`
	Name        string                 `json:"name"`
	Period      string                 `json:"period"`
	Currency    string                 `json:"currency"`
	AmountMinor int64                  `json:"amount_minor"`
	Periods     []BudgetPeriodSpending `json:"periods"`
}

// Spending returns spending by tag for the inclusive date range [from, to] and
// an income/expense trend ending in the month that contains to.
func (s *Service) Spending(ctx context.Context, workspaceID string, from, to time.Time, months int) (Spending, error) {
	months = normalizeMonths(months)
	from = startOfDay(from)
	to = startOfDay(to)
	if from.After(to) {
		return Spending{}, ErrInvalidRange
	}
	toExclusive := to.AddDate(0, 0, 1)
	trendStart := time.Date(to.Year(), to.Month(), 1, 0, 0, 0, 0, time.UTC).
		AddDate(0, -(months - 1), 0)

	fetchFrom := from
	if trendStart.Before(fetchFrom) {
		fetchFrom = trendStart
	}
	txList, err := s.transactions.List(ctx, workspaceID, transactions.Filter{
		From:             fetchFrom.Format(time.RFC3339),
		To:               toExclusive.Add(-time.Second).Format(time.RFC3339),
		ExcludeTransfers: true,
	})
	if err != nil {
		return Spending{}, err
	}

	tagList, err := s.tags.List(ctx, workspaceID)
	if err != nil {
		return Spending{}, err
	}
	tagByID := make(map[string]tags.Tag, len(tagList))
	for _, tag := range tagList {
		tagByID[tag.ID] = tag
	}

	spentByTag := make(map[string]int64)
	result := Spending{
		From:     from.Format("2006-01-02"),
		To:       to.Format("2006-01-02"),
		Currency: s.currency,
		Tags:     []TagSpending{},
		Trend:    monthBuckets(trendStart, months),
	}
	monthIndex := make(map[string]int, len(result.Trend))
	for i, point := range result.Trend {
		monthIndex[point.Month] = i
	}

	for _, tx := range txList {
		occurred, err := time.Parse(time.RFC3339, tx.OccurredAt)
		if err != nil || occurred.Before(fetchFrom) || !occurred.Before(toExclusive) {
			continue
		}
		if tx.TransferID != nil {
			continue
		}

		if !occurred.Before(trendStart) {
			if index, ok := monthIndex[occurred.UTC().Format("2006-01")]; ok {
				result.Trend[index].NetMinor += tx.AmountMinor
				if tx.AmountMinor > 0 {
					result.Trend[index].IncomeMinor += tx.AmountMinor
				} else if tx.AmountMinor < 0 {
					result.Trend[index].ExpenseMinor += -tx.AmountMinor
				}
			}
		}

		if occurred.Before(from) || !occurred.Before(toExclusive) || tx.AmountMinor >= 0 {
			continue
		}
		amount := -tx.AmountMinor
		result.TotalSpentMinor += amount
		tagIDs := uniqueTagIDs(tx.TagIDs)
		if len(tagIDs) == 0 {
			result.UntaggedMinor += amount
			continue
		}
		for _, tagID := range tagIDs {
			spentByTag[tagID] += amount
		}
	}

	for tagID, spent := range spentByTag {
		tag := tagByID[tagID]
		name := tag.Name
		if name == "" {
			name = tagID
		}
		result.Tags = append(result.Tags, TagSpending{
			TagID:      tagID,
			Name:       name,
			Color:      tag.Color,
			SpentMinor: spent,
		})
	}
	sort.Slice(result.Tags, func(i, j int) bool {
		if result.Tags[i].SpentMinor != result.Tags[j].SpentMinor {
			return result.Tags[i].SpentMinor > result.Tags[j].SpentMinor
		}
		return result.Tags[i].Name < result.Tags[j].Name
	})

	return result, nil
}

// BudgetTrend returns spending for the most recent periods of a budget up to
// asOf. It never writes to the budget.
func (s *Service) BudgetTrend(ctx context.Context, workspaceID, budgetID string, asOf time.Time, periods int) (BudgetTrend, error) {
	periods = normalizeMonths(periods)
	b, err := s.budgets.Get(ctx, workspaceID, budgetID)
	if err != nil {
		return BudgetTrend{}, err
	}

	allPeriods := budgets.PeriodsUpTo(b, asOf)
	if len(allPeriods) > periods {
		allPeriods = allPeriods[len(allPeriods)-periods:]
	}
	result := BudgetTrend{
		BudgetID:    b.ID,
		Name:        b.Name,
		Period:      b.Period,
		Currency:    s.currency,
		AmountMinor: b.AmountMinor,
		Periods:     []BudgetPeriodSpending{},
	}
	if len(allPeriods) == 0 {
		return result, nil
	}

	first := allPeriods[0].Start
	last := allPeriods[len(allPeriods)-1].End
	txList, err := s.transactions.List(ctx, workspaceID, transactions.Filter{
		From:             first.Format(time.RFC3339),
		To:               last.Add(-time.Second).Format(time.RFC3339),
		ExcludeTransfers: true,
	})
	if err != nil {
		return BudgetTrend{}, err
	}

	for _, period := range allPeriods {
		spent := budgets.SpentInPeriod(b, period, txList)
		result.Periods = append(result.Periods, BudgetPeriodSpending{
			Start:          period.Start.Format("2006-01-02"),
			End:            period.End.AddDate(0, 0, -1).Format("2006-01-02"),
			SpentMinor:     spent,
			AmountMinor:    b.AmountMinor,
			RemainingMinor: b.AmountMinor - spent,
		})
	}
	return result, nil
}

// ErrInvalidRange is returned when a date range ends before it starts.
var ErrInvalidRange = errInvalidRange{}

type errInvalidRange struct{}

func (errInvalidRange) Error() string { return "from must not be after to" }

func normalizeMonths(months int) int {
	if months <= 0 {
		return DefaultTrendMonths
	}
	if months > MaxTrendMonths {
		return MaxTrendMonths
	}
	return months
}

func monthBuckets(start time.Time, count int) []MonthPoint {
	out := make([]MonthPoint, 0, count)
	for i := 0; i < count; i++ {
		out = append(out, MonthPoint{Month: start.AddDate(0, i, 0).Format("2006-01")})
	}
	return out
}

func uniqueTagIDs(ids []string) []string {
	seen := make(map[string]bool, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func startOfDay(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
