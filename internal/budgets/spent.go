package budgets

import (
	"time"

	"budget-agent/internal/transactions"
)

// SpentInPeriod sums the expenses that match a budget's account and tags inside
// a half-open period. Transfers and income never count as spending.
func SpentInPeriod(b Budget, period Period, txList []transactions.Transaction) int64 {
	var sum int64
	for _, tx := range txList {
		if tx.TransferID != nil {
			continue
		}
		occurred, err := time.Parse(time.RFC3339, tx.OccurredAt)
		if err != nil || occurred.Before(period.Start) || !occurred.Before(period.End) {
			continue
		}
		if b.AccountID != nil && *b.AccountID != tx.AccountID {
			continue
		}
		if !hasAnyTag(tx.TagIDs, b.TagIDs) {
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
