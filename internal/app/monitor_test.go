package app

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/KageRyo/netquota/internal/model"
	"github.com/KageRyo/netquota/internal/network"
)

type fakeProvider struct {
	interfaces []network.Interface
	counters   network.Counters
}

func (f *fakeProvider) Interfaces(context.Context) ([]network.Interface, error) {
	return f.interfaces, nil
}

func (f *fakeProvider) Counters(context.Context, string) (network.Counters, error) {
	return f.counters, nil
}

type fakeStateSaver struct {
	states []model.State
}

func (f *fakeStateSaver) SaveState(state model.State) error {
	f.states = append(f.states, state)
	return nil
}

type fakeNotifier struct {
	notifications []struct {
		title   string
		message string
	}
}

func (f *fakeNotifier) Notify(title, message string) error {
	f.notifications = append(f.notifications, struct {
		title   string
		message string
	}{title: title, message: message})
	return nil
}

func TestMonitorPersistsSamplesAndNotifiesTotalAndSeparateLimits(t *testing.T) {
	t.Parallel()

	cfg := model.Config{
		Version:             model.ConfigVersion,
		PollIntervalSeconds: 1,
		Quotas: model.Quotas{
			Total:    model.Limit{Bytes: 100, AlertPercentages: []uint8{70}},
			Download: model.Limit{Bytes: 80, AlertPercentages: []uint8{50}},
		},
		Notifications: model.NotificationConfig{Enabled: true},
	}
	provider := &fakeProvider{interfaces: []network.Interface{{Name: "Ethernet", IPv4: "192.0.2.10"}}}
	stateSaver := &fakeStateSaver{}
	notifier := &fakeNotifier{}
	monitor := NewMonitor(cfg, model.State{}, provider, nil, stateSaver, notifier, quietLogger())
	when := time.Date(2026, 8, 21, 8, 0, 0, 0, time.Local)

	provider.counters = network.Counters{DownloadBytes: 100, UploadBytes: 10}
	first, err := monitor.Sample(context.Background(), when)
	if err != nil {
		t.Fatalf("first Sample: %v", err)
	}
	if !first.Baseline || len(first.Alerts) != 0 {
		t.Fatalf("first sample = %+v, want baseline without alerts", first)
	}

	provider.counters = network.Counters{DownloadBytes: 170, UploadBytes: 20}
	second, err := monitor.Sample(context.Background(), when.Add(time.Minute))
	if err != nil {
		t.Fatalf("second Sample: %v", err)
	}
	if second.Usage != (model.Usage{DownloadBytes: 70, UploadBytes: 10}) {
		t.Fatalf("usage = %+v", second.Usage)
	}
	if len(second.Alerts) != 2 {
		t.Fatalf("got %d alerts, want total and download", len(second.Alerts))
	}
	if len(notifier.notifications) != 2 {
		t.Fatalf("got %d notifications, want 2", len(notifier.notifications))
	}
	if len(stateSaver.states) != 2 || !stateSaver.states[1].AlertedThresholds[second.Alerts[0].Key] {
		t.Fatalf("saved states = %+v", stateSaver.states)
	}

	provider.counters = network.Counters{DownloadBytes: 180, UploadBytes: 25}
	third, err := monitor.Sample(context.Background(), when.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("third Sample: %v", err)
	}
	if len(third.Alerts) != 0 || len(notifier.notifications) != 2 {
		t.Fatalf("threshold repeated: alerts=%+v notifications=%d", third.Alerts, len(notifier.notifications))
	}
}

func TestMonitorChangingInterfaceStartsNewBaseline(t *testing.T) {
	t.Parallel()

	cfg := model.Config{Version: model.ConfigVersion, PollIntervalSeconds: 1}
	provider := &fakeProvider{interfaces: []network.Interface{{Name: "Ethernet"}}}
	monitor := NewMonitor(cfg, model.State{}, provider, nil, nil, nil, quietLogger())
	when := time.Date(2026, 8, 21, 8, 0, 0, 0, time.Local)
	provider.counters = network.Counters{DownloadBytes: 100}
	monitor.Sample(context.Background(), when)
	provider.counters = network.Counters{DownloadBytes: 200}
	monitor.Sample(context.Background(), when.Add(time.Minute))

	updated := cfg
	updated.Interface = model.InterfaceSelection{Name: "Wi-Fi"}
	if err := monitor.SetConfig(updated); err != nil {
		t.Fatalf("SetConfig: %v", err)
	}
	provider.interfaces = []network.Interface{{Name: "Wi-Fi"}}
	provider.counters = network.Counters{DownloadBytes: 500}
	result, err := monitor.Sample(context.Background(), when.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("Sample after interface change: %v", err)
	}
	if !result.Baseline || result.Usage != (model.Usage{}) {
		t.Fatalf("sample after interface change = %+v", result)
	}
}

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
