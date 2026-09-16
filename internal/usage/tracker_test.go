package usage

import (
	"math"
	"testing"
	"time"

	"github.com/KageRyo/netquota/internal/model"
	"github.com/KageRyo/netquota/internal/network"
)

func TestTrackerUsesFirstSampleAsBaseline(t *testing.T) {
	t.Parallel()

	tracker := NewTracker(model.State{}, time.UTC)
	result := tracker.Apply(time.Date(2026, 8, 21, 8, 0, 0, 0, time.UTC), network.Counters{
		DownloadBytes: 1000,
		UploadBytes:   200,
	})
	if !result.Baseline || result.Delta != (model.Usage{}) {
		t.Fatalf("first sample = %+v, want baseline with zero delta", result)
	}
	if got := tracker.State().Date; got != "2026-08-21" {
		t.Fatalf("state date = %q, want 2026-08-21", got)
	}
}

func TestTrackerAccumulatesDownloadAndUploadDeltas(t *testing.T) {
	t.Parallel()

	tracker := NewTracker(model.State{}, time.UTC)
	when := time.Date(2026, 8, 21, 8, 0, 0, 0, time.UTC)
	tracker.Apply(when, network.Counters{DownloadBytes: 1000, UploadBytes: 200})
	result := tracker.Apply(when.Add(time.Minute), network.Counters{DownloadBytes: 1800, UploadBytes: 500})
	if result.Delta != (model.Usage{DownloadBytes: 800, UploadBytes: 300}) {
		t.Fatalf("delta = %+v", result.Delta)
	}
	if result.Usage != (model.Usage{DownloadBytes: 800, UploadBytes: 300}) {
		t.Fatalf("usage = %+v", result.Usage)
	}
}

func TestTrackerTreatsCounterDecreaseAsReset(t *testing.T) {
	t.Parallel()

	tracker := NewTracker(model.State{}, time.UTC)
	when := time.Date(2026, 8, 21, 8, 0, 0, 0, time.UTC)
	tracker.Apply(when, network.Counters{DownloadBytes: 1000, UploadBytes: 200})
	result := tracker.Apply(when.Add(time.Minute), network.Counters{DownloadBytes: 100, UploadBytes: 260})
	if !result.DownloadReset || result.UploadReset {
		t.Fatalf("reset flags = download:%v upload:%v", result.DownloadReset, result.UploadReset)
	}
	if result.Delta.DownloadBytes != 0 || result.Delta.UploadBytes != 60 {
		t.Fatalf("delta after reset = %+v", result.Delta)
	}
	if result.Usage != (model.Usage{DownloadBytes: 0, UploadBytes: 60}) {
		t.Fatalf("usage after reset = %+v", result.Usage)
	}
}

func TestTrackerResetsAtLocalDateBoundary(t *testing.T) {
	t.Parallel()

	tracker := NewTracker(model.State{}, time.UTC)
	previousDay := time.Date(2026, 8, 21, 22, 0, 0, 0, time.UTC)
	tracker.Apply(previousDay, network.Counters{DownloadBytes: 1000, UploadBytes: 200})
	tracker.Apply(previousDay.Add(time.Hour), network.Counters{DownloadBytes: 1500, UploadBytes: 300})
	result := tracker.Apply(previousDay.Add(4*time.Hour), network.Counters{DownloadBytes: 2000, UploadBytes: 400})
	if !result.NewDay || !result.Baseline {
		t.Fatalf("date-boundary result = %+v", result)
	}
	if result.Usage != (model.Usage{}) {
		t.Fatalf("new-day usage = %+v", result.Usage)
	}
	if got := tracker.State().Date; got != "2026-08-22" {
		t.Fatalf("state date = %q, want 2026-08-22", got)
	}
}

func TestTrackerSaturatesUsageOnOverflow(t *testing.T) {
	t.Parallel()

	state := model.State{
		Date:     "2026-08-21",
		Usage:    model.Usage{DownloadBytes: math.MaxUint64 - 5},
		Counters: model.Counters{DownloadBytes: 10, Initialized: true},
	}
	tracker := NewTracker(state, time.UTC)
	result := tracker.Apply(time.Date(2026, 8, 21, 8, 0, 0, 0, time.UTC), network.Counters{DownloadBytes: 20})
	if result.Usage.DownloadBytes != math.MaxUint64 {
		t.Fatalf("saturated usage = %d, want %d", result.Usage.DownloadBytes, uint64(math.MaxUint64))
	}
}

func TestTrackerPersistsAlertMarksInState(t *testing.T) {
	t.Parallel()

	tracker := NewTracker(model.State{}, time.UTC)
	tracker.MarkAlert("total:100:70")
	if !tracker.State().AlertedThresholds["total:100:70"] {
		t.Fatal("alert mark was not retained in tracker state")
	}
}

func TestTrackerResetsAtMonthlyBoundaryAndBaselinesCounters(t *testing.T) {
	t.Parallel()

	tracker := NewTrackerWithCycle(model.State{}, model.BillingCycle{Kind: model.BillingCycleMonthly}, time.UTC)
	before := time.Date(2026, 8, 31, 23, 0, 0, 0, time.UTC)
	tracker.Apply(before, network.Counters{DownloadBytes: 100})
	result := tracker.Apply(before.Add(2*time.Hour), network.Counters{DownloadBytes: 500})
	if !result.NewPeriod || !result.Baseline {
		t.Fatalf("monthly boundary result = %+v, want new-period baseline", result)
	}
	if result.Usage != (model.Usage{}) || result.Delta != (model.Usage{}) {
		t.Fatalf("monthly boundary counted missed bytes: %+v", result)
	}
	if got, want := tracker.State().PeriodKey, "2026-09-01"; got != want {
		t.Fatalf("period key = %q, want %q", got, want)
	}

	result = tracker.Apply(before.Add(3*time.Hour), network.Counters{DownloadBytes: 550})
	if result.Baseline || result.Delta.DownloadBytes != 50 || result.Usage.DownloadBytes != 50 {
		t.Fatalf("sample after monthly baseline = %+v", result)
	}
}

func TestTrackerClearsAlertsAtCustomPeriodBoundary(t *testing.T) {
	t.Parallel()

	state := model.State{
		Date:              "2026-08-15",
		PeriodKey:         "2026-08-15",
		BillingCycleKey:   "custom:15",
		Usage:             model.Usage{DownloadBytes: 80},
		Counters:          model.Counters{DownloadBytes: 100, Initialized: true},
		AlertedThresholds: map[string]bool{"total:100:70": true},
	}
	tracker := NewTrackerWithCycle(state, model.BillingCycle{Kind: model.BillingCycleCustom, ResetDay: 15}, time.UTC)
	result := tracker.Apply(time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC), network.Counters{DownloadBytes: 1000})
	if !result.NewPeriod || !result.Baseline {
		t.Fatalf("custom boundary result = %+v", result)
	}
	if len(tracker.State().AlertedThresholds) != 0 {
		t.Fatalf("alert marks survived custom boundary: %v", tracker.State().AlertedThresholds)
	}
}

func TestTrackerTreatsTimezoneChangeAsNewPeriodWhenKeyChanges(t *testing.T) {
	t.Parallel()

	newYork, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("timezone database unavailable: %v", err)
	}
	tracker := NewTrackerWithCycle(model.State{}, model.BillingCycle{Kind: model.BillingCycleDaily}, newYork)
	instant := time.Date(2026, 9, 1, 0, 30, 0, 0, time.UTC)
	tracker.Apply(instant, network.Counters{DownloadBytes: 100})
	tracker.SetLocation(time.UTC)
	result := tracker.Apply(instant.Add(10*time.Minute), network.Counters{DownloadBytes: 500})
	if !result.NewPeriod || !result.Baseline || result.Usage != (model.Usage{}) {
		t.Fatalf("timezone-change result = %+v, want a fresh baseline", result)
	}
}
