package budgets

import (
	"testing"
	"time"
)

func date(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		t.Fatalf("parse %s: %v", value, err)
	}
	return parsed
}

func TestPeriodsUpToWeekly(t *testing.T) {
	b := Budget{Period: "weekly", StartDate: "2026-11-02"}

	periods := PeriodsUpTo(b, date(t, "2026-11-18"))
	if len(periods) != 3 {
		t.Fatalf("periods = %d, want 3", len(periods))
	}
	current := periods[len(periods)-1]
	if got := current.Start.Format("2006-01-02"); got != "2026-11-16" {
		t.Errorf("start = %s, want 2026-11-16", got)
	}
	if got := current.End.Format("2006-01-02"); got != "2026-11-23" {
		t.Errorf("end = %s, want 2026-11-23", got)
	}
}

func TestPeriodsUpToMonthlyClampsShortMonths(t *testing.T) {
	b := Budget{Period: "monthly", StartDate: "2026-01-31"}

	periods := PeriodsUpTo(b, date(t, "2026-03-15"))
	if len(periods) != 2 {
		t.Fatalf("periods = %d, want 2", len(periods))
	}
	current := periods[len(periods)-1]
	if got := current.Start.Format("2006-01-02"); got != "2026-02-28" {
		t.Errorf("start = %s, want 2026-02-28", got)
	}
	if got := current.End.Format("2006-01-02"); got != "2026-03-31" {
		t.Errorf("end = %s, want 2026-03-31", got)
	}

	next := PeriodsUpTo(b, date(t, "2026-03-31"))
	if got := next[len(next)-1].Start.Format("2006-01-02"); got != "2026-03-31" {
		t.Errorf("next start = %s, want 2026-03-31", got)
	}
}

func TestPeriodsUpToRespectsEndDate(t *testing.T) {
	end := "2026-11-10"
	b := Budget{Period: "weekly", StartDate: "2026-11-02", EndDate: &end}

	if periods := PeriodsUpTo(b, date(t, "2026-11-20")); periods != nil {
		t.Fatalf("periods after end = %v, want nil", periods)
	}

	periods := PeriodsUpTo(b, date(t, "2026-11-09"))
	if len(periods) != 2 {
		t.Fatalf("periods = %d, want 2", len(periods))
	}
	if got := periods[1].End.Format("2006-01-02"); got != "2026-11-11" {
		t.Errorf("capped end = %s, want 2026-11-11", got)
	}
}

func TestPeriodsUpToInactiveAndInvalid(t *testing.T) {
	if periods := PeriodsUpTo(Budget{Period: "weekly", StartDate: "2026-12-01"}, date(t, "2026-11-20")); periods != nil {
		t.Fatalf("periods before start = %v, want nil", periods)
	}
	if periods := PeriodsUpTo(Budget{Period: "daily", StartDate: "2026-11-01"}, date(t, "2026-11-20")); periods != nil {
		t.Fatalf("periods with invalid period = %v, want nil", periods)
	}
}
