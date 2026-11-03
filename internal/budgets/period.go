package budgets

import "time"

// Period is a half-open budget period: [Start, End).
type Period struct {
	Start time.Time
	End   time.Time
}

// PeriodsUpTo returns every period from the budget's start through the period
// containing asOf. It returns nil when the budget is not active on that date or
// has an unsupported period.
func PeriodsUpTo(b Budget, asOf time.Time) []Period {
	anchor, err := time.Parse("2006-01-02", b.StartDate)
	if err != nil {
		return nil
	}
	anchor = dateOnly(anchor)
	asOfDate := dateOnly(asOf)
	if asOfDate.Before(anchor) {
		return nil
	}
	if b.EndDate != nil {
		endDate, err := time.Parse("2006-01-02", *b.EndDate)
		if err != nil {
			return nil
		}
		if asOfDate.After(dateOnly(endDate)) {
			return nil
		}
	}

	var periods []Period
	switch b.Period {
	case "weekly":
		for start := anchor; !start.After(asOfDate); start = start.AddDate(0, 0, 7) {
			if b.EndDate != nil && periodStartsAfter(start, *b.EndDate) {
				break
			}
			periods = append(periods, Period{
				Start: start,
				End:   cappedEnd(start.AddDate(0, 0, 7), b.EndDate),
			})
		}
	case "monthly":
		anchorDay := anchor.Day()
		for start := anchor; !start.After(asOfDate); start = nextMonth(start, anchorDay) {
			if b.EndDate != nil && periodStartsAfter(start, *b.EndDate) {
				break
			}
			periods = append(periods, Period{
				Start: start,
				End:   cappedEnd(nextMonth(start, anchorDay), b.EndDate),
			})
		}
	default:
		return nil
	}
	if len(periods) == 0 {
		return nil
	}
	return periods
}

func dateOnly(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func nextMonth(start time.Time, anchorDay int) time.Time {
	firstOfNext := time.Date(start.Year(), start.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	lastDay := firstOfNext.AddDate(0, 1, -1).Day()
	day := anchorDay
	if day > lastDay {
		day = lastDay
	}
	return time.Date(firstOfNext.Year(), firstOfNext.Month(), day, 0, 0, 0, 0, time.UTC)
}

func periodStartsAfter(start time.Time, endDate string) bool {
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return true
	}
	return start.After(dateOnly(end))
}

// cappedEnd limits a period to the day after the budget's inclusive end date.
func cappedEnd(end time.Time, endDate *string) time.Time {
	if endDate == nil {
		return end
	}
	cutoff, err := time.Parse("2006-01-02", *endDate)
	if err != nil {
		return end
	}
	if cutoff = cutoff.AddDate(0, 0, 1); end.After(cutoff) {
		return cutoff
	}
	return end
}
