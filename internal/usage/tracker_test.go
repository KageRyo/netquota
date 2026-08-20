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
