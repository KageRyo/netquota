package usage

import (
	"time"

	"github.com/KageRyo/netquota/internal/model"
	"github.com/KageRyo/netquota/internal/network"
)

const DateLayout = "2006-01-02"

type Result struct {
	PreviousUsage model.Usage
	Usage         model.Usage
	Delta         model.Usage
	NewDay        bool
	NewPeriod     bool
	Baseline      bool
	DownloadReset bool
	UploadReset   bool
	Period        Period
}

type Tracker struct {
	state    model.State
	cycle    model.BillingCycle
	location *time.Location
}

func NewTracker(state model.State, location *time.Location) *Tracker {
	return NewTrackerWithCycle(state, model.BillingCycle{Kind: model.BillingCycleDaily}, location)
}

func NewTrackerWithCycle(state model.State, cycle model.BillingCycle, location *time.Location) *Tracker {
	if location == nil {
		location = time.Local
	}
	cycle = cycle.Normalized()
	state = state.Clone()
	if state.Version == 0 || state.Version < model.StateVersion {
		state.Version = model.StateVersion
	}
	if state.PeriodKey == "" {
		state.PeriodKey = state.Date
	}
	if state.BillingCycleKey == "" {
		state.BillingCycleKey = model.BillingCycle{Kind: model.BillingCycleDaily}.Identity()
	}
	if state.AlertedThresholds == nil {
		state.AlertedThresholds = make(map[string]bool)
	}
	return &Tracker{state: state, cycle: cycle, location: location}
}

func (t *Tracker) Apply(now time.Time, current network.Counters) Result {
	now = now.In(t.location)
	period := CurrentPeriod(now, t.cycle, t.location)
	previousPeriodKey := t.state.PeriodKey
	cycleChanged := t.state.BillingCycleKey != t.cycle.Identity()
	periodChanged := previousPeriodKey != "" && previousPeriodKey != period.Key
	if previousPeriodKey == "" || periodChanged || cycleChanged {
		previous := t.state.Usage
		newPeriod := previousPeriodKey != "" && (periodChanged || cycleChanged)
		t.state.Date = period.Key
		t.state.PeriodKey = period.Key
		t.state.BillingCycleKey = t.cycle.Identity()
		t.state.Usage = model.Usage{}
		t.state.Counters = model.Counters{
			DownloadBytes: current.DownloadBytes,
			UploadBytes:   current.UploadBytes,
			Initialized:   true,
		}
		t.state.AlertedThresholds = make(map[string]bool)
		t.state.UpdatedAt = now
		return Result{
			PreviousUsage: previous,
			Usage:         t.state.Usage,
			NewDay:        newPeriod,
			NewPeriod:     newPeriod,
			Baseline:      true,
			Period:        period,
		}
	}

	if !t.state.Counters.Initialized {
		t.state.Counters = model.Counters{
			DownloadBytes: current.DownloadBytes,
			UploadBytes:   current.UploadBytes,
			Initialized:   true,
		}
		t.state.UpdatedAt = now
		return Result{
			PreviousUsage: t.state.Usage,
			Usage:         t.state.Usage,
			Baseline:      true,
			Period:        period,
		}
	}

	downloadDelta, downloadReset := counterDelta(current.DownloadBytes, t.state.Counters.DownloadBytes)
	uploadDelta, uploadReset := counterDelta(current.UploadBytes, t.state.Counters.UploadBytes)
	previous := t.state.Usage
	t.state.Usage.DownloadBytes = saturatingAdd(t.state.Usage.DownloadBytes, downloadDelta)
	t.state.Usage.UploadBytes = saturatingAdd(t.state.Usage.UploadBytes, uploadDelta)
	t.state.Counters = model.Counters{
		DownloadBytes: current.DownloadBytes,
		UploadBytes:   current.UploadBytes,
		Initialized:   true,
	}
	t.state.UpdatedAt = now
	return Result{
		PreviousUsage: previous,
		Usage:         t.state.Usage,
		Delta: model.Usage{
			DownloadBytes: downloadDelta,
			UploadBytes:   uploadDelta,
		},
		DownloadReset: downloadReset,
		UploadReset:   uploadReset,
		Period:        period,
	}
}

func (t *Tracker) State() model.State {
	return t.state.Clone()
}

func (t *Tracker) MarkAlert(key string) {
	if t.state.AlertedThresholds == nil {
		t.state.AlertedThresholds = make(map[string]bool)
	}
	t.state.AlertedThresholds[key] = true
}

// SetLocation refreshes the timezone used for the next calendar resolution.
// The monitor calls this before every sample so an operating-system timezone
// change is evaluated without reallocating historical usage.
func (t *Tracker) SetLocation(location *time.Location) {
	if location != nil {
		t.location = location
	}
}

func (t *Tracker) SetCycle(cycle model.BillingCycle) {
	t.cycle = cycle.Normalized()
}

// ResetForInterface starts a new baseline when the user changes the tracked
// interface. It avoids mixing counters from two different interfaces.
func (t *Tracker) ResetForInterface() {
	t.resetAccounting()
}

// ResetForBillingCycle starts a new baseline after an explicit cycle change.
func (t *Tracker) ResetForBillingCycle() {
	t.resetAccounting()
}

func (t *Tracker) resetAccounting() {
	t.state.Date = ""
	t.state.PeriodKey = ""
	t.state.BillingCycleKey = t.cycle.Identity()
	t.state.Usage = model.Usage{}
	t.state.Counters = model.Counters{}
	t.state.AlertedThresholds = make(map[string]bool)
}

func counterDelta(current, previous uint64) (uint64, bool) {
	if current < previous {
		return 0, true
	}
	return current - previous, false
}

func saturatingAdd(left, right uint64) uint64 {
	if ^uint64(0)-left < right {
		return ^uint64(0)
	}
	return left + right
}
