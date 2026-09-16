package usage

import (
	"time"

	"github.com/KageRyo/netquota/internal/model"
)

// Period describes the accounting period containing a sample.
type Period struct {
	Key       string
	Start     time.Time
	NextReset time.Time
}

// CurrentPeriod resolves a billing cycle using calendar dates in location.
// It deliberately uses AddDate/time.Date instead of fixed durations so DST
// transitions do not move a reset away from local midnight.
func CurrentPeriod(now time.Time, cycle model.BillingCycle, location *time.Location) Period {
	if location == nil {
		location = time.Local
	}
	cycle = cycle.Normalized()
	localNow := now.In(location)

	switch cycle.Kind {
	case model.BillingCycleMonthly:
		start := time.Date(localNow.Year(), localNow.Month(), 1, 0, 0, 0, 0, location)
		return periodFrom(start, start.AddDate(0, 1, 0))
	case model.BillingCycleCustom:
		return customMonthlyPeriod(localNow, int(cycle.ResetDay), location)
	default:
		start := midnight(localNow, location)
		return periodFrom(start, start.AddDate(0, 0, 1))
	}
}

func customMonthlyPeriod(now time.Time, resetDay int, location *time.Location) Period {
	if resetDay < 1 || resetDay > 31 {
		resetDay = 1
	}
	candidate := resetDate(now.Year(), now.Month(), resetDay, location)
	if now.Before(candidate) {
		previousMonth := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, location)
		start := resetDate(previousMonth.Year(), previousMonth.Month(), resetDay, location)
		return periodFrom(start, candidate)
	}
	nextMonth := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, location)
	next := resetDate(nextMonth.Year(), nextMonth.Month(), resetDay, location)
	return periodFrom(candidate, next)
}

func periodFrom(start, nextReset time.Time) Period {
	return Period{
		Key:       start.Format(DateLayout),
		Start:     start,
		NextReset: nextReset,
	}
}

func midnight(value time.Time, location *time.Location) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, location)
}

func resetDate(year int, month time.Month, resetDay int, location *time.Location) time.Time {
	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, location).Day()
	if resetDay > lastDay {
		resetDay = lastDay
	}
	return time.Date(year, month, resetDay, 0, 0, 0, 0, location)
}
