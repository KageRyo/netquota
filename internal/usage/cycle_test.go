package usage

import (
	"testing"
	"time"

	"github.com/KageRyo/netquota/internal/model"
)

func TestCurrentPeriodDailyUsesLocalMidnight(t *testing.T) {
	t.Parallel()

	period := CurrentPeriod(time.Date(2026, 8, 21, 8, 0, 0, 0, time.UTC), model.BillingCycle{Kind: model.BillingCycleDaily}, time.UTC)
	if got, want := period.Key, "2026-08-21"; got != want {
		t.Fatalf("period key = %q, want %q", got, want)
	}
	if got, want := period.Start, time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("period start = %v, want %v", got, want)
	}
	if got, want := period.NextReset, time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("next reset = %v, want %v", got, want)
	}
}

func TestCurrentPeriodMonthlyUsesCalendarMonth(t *testing.T) {
	t.Parallel()

	period := CurrentPeriod(time.Date(2026, 8, 31, 23, 59, 0, 0, time.UTC), model.BillingCycle{Kind: model.BillingCycleMonthly}, time.UTC)
	if got, want := period.Key, "2026-08-01"; got != want {
		t.Fatalf("period key = %q, want %q", got, want)
	}
	if got, want := period.NextReset, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("next reset = %v, want %v", got, want)
	}

	period = CurrentPeriod(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), model.BillingCycle{Kind: model.BillingCycleMonthly}, time.UTC)
	if got, want := period.Key, "2026-09-01"; got != want {
		t.Fatalf("exact-boundary period key = %q, want %q", got, want)
	}
}

func TestCurrentPeriodCustomDayClampsShortMonths(t *testing.T) {
	t.Parallel()

	cycle := model.BillingCycle{Kind: model.BillingCycleCustom, ResetDay: 31}
	period := CurrentPeriod(time.Date(2027, 2, 15, 12, 0, 0, 0, time.UTC), cycle, time.UTC)
	if got, want := period.Key, "2027-01-31"; got != want {
		t.Fatalf("February period key = %q, want %q", got, want)
	}
	if got, want := period.NextReset, time.Date(2027, 2, 28, 0, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("February next reset = %v, want %v", got, want)
	}

	period = CurrentPeriod(time.Date(2028, 2, 29, 0, 0, 0, 0, time.UTC), cycle, time.UTC)
	if got, want := period.Key, "2028-02-29"; got != want {
		t.Fatalf("leap-day period key = %q, want %q", got, want)
	}
	if got, want := period.NextReset, time.Date(2028, 3, 31, 0, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("leap-day next reset = %v, want %v", got, want)
	}
}

func TestCurrentPeriodUsesCalendarArithmeticAcrossDST(t *testing.T) {
	t.Parallel()

	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("timezone database unavailable: %v", err)
	}
	period := CurrentPeriod(time.Date(2026, 3, 8, 3, 30, 0, 0, location), model.BillingCycle{Kind: model.BillingCycleDaily}, location)
	if got, want := period.NextReset.Format("2006-01-02 15:04 MST"), "2026-03-09 00:00 EDT"; got != want {
		t.Fatalf("DST next reset = %q, want %q", got, want)
	}
	if period.NextReset.Sub(period.Start) == 24*time.Hour {
		t.Fatal("DST period used a fixed 24-hour duration")
	}
}
