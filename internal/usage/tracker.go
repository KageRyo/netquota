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
	Baseline      bool
	DownloadReset bool
	UploadReset   bool
}

type Tracker struct {
	state    model.State
	location *time.Location
}

func NewTracker(state model.State, location *time.Location) *Tracker {
	if location == nil {
		location = time.Local
	}
	state = state.Clone()
	if state.Version == 0 {
		state.Version = model.StateVersion
	}
	if state.AlertedThresholds == nil {
		state.AlertedThresholds = make(map[string]bool)
	}
	return &Tracker{state: state, location: location}
}

func (t *Tracker) Apply(now time.Time, current network.Counters) Result {
	now = now.In(t.location)
	today := now.Format(DateLayout)
	if t.state.Date == "" || t.state.Date != today {
		previous := t.state.Usage
		wasDifferentDay := t.state.Date != "" && t.state.Date != today
		t.state.Date = today
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
			NewDay:        wasDifferentDay,
			Baseline:      true,
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

// ResetForInterface starts a new baseline when the user changes the tracked
// interface. It avoids mixing counters from two different interfaces.
func (t *Tracker) ResetForInterface() {
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
