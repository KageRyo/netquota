package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/KageRyo/netquota/internal/i18n"
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

func TestMonitorDoesNotFallbackWhenSavedInterfaceUnavailable(t *testing.T) {
	t.Parallel()

	cfg := model.Config{
		Version:             model.ConfigVersion,
		PollIntervalSeconds: 1,
		Interface: model.InterfaceSelection{
			Name:            "Wi-Fi",
			HardwareAddress: "AA:BB:CC:DD:EE:FF",
		},
	}
	provider := &fakeProvider{interfaces: []network.Interface{{Name: "Ethernet", IPv4: "192.0.2.20"}}}
	stateSaver := &fakeStateSaver{}
	monitor := NewMonitor(cfg, model.State{}, provider, nil, stateSaver, nil, quietLogger())

	_, err := monitor.Sample(context.Background(), time.Date(2026, 8, 21, 8, 0, 0, 0, time.Local))
	if !errors.Is(err, network.ErrSelectedInterfaceUnavailable) {
		t.Fatalf("Sample error = %v, want ErrSelectedInterfaceUnavailable", err)
	}
	if len(stateSaver.states) != 0 {
		t.Fatalf("state saves = %d, want 0 while interface is unavailable", len(stateSaver.states))
	}
}

func TestMonitorRetainsAutomaticInterfaceWhenDefaultRouteChanges(t *testing.T) {
	t.Parallel()

	cfg := model.Config{Version: model.ConfigVersion, PollIntervalSeconds: 1}
	provider := &fakeProvider{interfaces: []network.Interface{
		{Name: "VPN", Index: 2, IPv4: "10.8.0.2"},
		{Name: "Wi-Fi", Index: 1, HardwareAddress: "AA", IPv4: "192.0.2.10", DefaultRoute: true},
	}}
	monitor := NewMonitor(cfg, model.State{}, provider, nil, nil, nil, quietLogger())
	when := time.Date(2026, 8, 21, 8, 0, 0, 0, time.Local)
	provider.counters = network.Counters{DownloadBytes: 100}
	first, err := monitor.Sample(context.Background(), when)
	if err != nil {
		t.Fatalf("first Sample: %v", err)
	}
	if first.Interface.Name != "Wi-Fi" || !first.Baseline {
		t.Fatalf("first sample = %+v, want Wi-Fi baseline", first)
	}

	provider.interfaces = []network.Interface{
		{Name: "VPN", Index: 2, IPv4: "10.8.0.2", DefaultRoute: true},
		{Name: "Wi-Fi", Index: 1, HardwareAddress: "AA", IPv4: "192.0.2.10"},
	}
	provider.counters = network.Counters{DownloadBytes: 150}
	second, err := monitor.Sample(context.Background(), when.Add(time.Minute))
	if err != nil {
		t.Fatalf("second Sample: %v", err)
	}
	if second.Interface.Name != "Wi-Fi" {
		t.Fatalf("second interface = %q, want Wi-Fi", second.Interface.Name)
	}
	if second.Baseline || second.Usage.DownloadBytes != 50 {
		t.Fatalf("second sample = %+v, want 50-byte continuation", second)
	}
}

func TestMonitorReappearingInterfaceTreatsLowerCounterAsReset(t *testing.T) {
	t.Parallel()

	cfg := model.Config{
		Version:             model.ConfigVersion,
		PollIntervalSeconds: 1,
		Interface:           model.InterfaceSelection{Name: "Wi-Fi", HardwareAddress: "AA"},
	}
	provider := &fakeProvider{interfaces: []network.Interface{{Name: "Wi-Fi", HardwareAddress: "AA", IPv4: "192.0.2.10"}}}
	monitor := NewMonitor(cfg, model.State{}, provider, nil, nil, nil, quietLogger())
	when := time.Date(2026, 8, 21, 8, 0, 0, 0, time.Local)
	provider.counters = network.Counters{DownloadBytes: 100}
	monitor.Sample(context.Background(), when)
	provider.counters = network.Counters{DownloadBytes: 150}
	monitor.Sample(context.Background(), when.Add(time.Minute))

	provider.interfaces = nil
	if _, err := monitor.Sample(context.Background(), when.Add(2*time.Minute)); !errors.Is(err, network.ErrSelectedInterfaceUnavailable) {
		t.Fatalf("missing-interface error = %v, want ErrSelectedInterfaceUnavailable", err)
	}

	provider.interfaces = []network.Interface{{Name: "Wi-Fi", HardwareAddress: "AA", IPv4: "192.0.2.10"}}
	provider.counters = network.Counters{DownloadBytes: 10}
	result, err := monitor.Sample(context.Background(), when.Add(3*time.Minute))
	if err != nil {
		t.Fatalf("reappearing Sample: %v", err)
	}
	if !result.DownloadReset || result.Usage.DownloadBytes != 50 {
		t.Fatalf("reappearing sample = %+v, want reset with unchanged usage", result)
	}
}

func TestMonitorLocalizesQuotaNotifications(t *testing.T) {
	t.Parallel()

	cfg := model.Config{
		Version:             model.ConfigVersion,
		Language:            i18n.TraditionalChinese,
		PollIntervalSeconds: 1,
		Quotas: model.Quotas{
			Total: model.Limit{Bytes: 100, AlertPercentages: []uint8{70}},
		},
		Notifications: model.NotificationConfig{Enabled: true},
	}
	provider := &fakeProvider{interfaces: []network.Interface{{Name: "Ethernet"}}}
	notifier := &fakeNotifier{}
	monitor := NewMonitor(cfg, model.State{}, provider, nil, nil, notifier, quietLogger())
	when := time.Date(2026, 8, 21, 8, 0, 0, 0, time.Local)
	provider.counters = network.Counters{DownloadBytes: 1}
	if _, err := monitor.Sample(context.Background(), when); err != nil {
		t.Fatalf("baseline sample: %v", err)
	}
	provider.counters = network.Counters{DownloadBytes: 71}
	if _, err := monitor.Sample(context.Background(), when.Add(time.Minute)); err != nil {
		t.Fatalf("alert sample: %v", err)
	}
	if len(notifier.notifications) != 1 {
		t.Fatalf("notifications = %d, want 1", len(notifier.notifications))
	}
	got := notifier.notifications[0]
	if got.title != "NetQuota 流量上限警示" || got.message != "總流量已達 70%（70 B / 100 B）・每日" {
		t.Fatalf("notification = (%q, %q)", got.title, got.message)
	}
}

func TestMonitorResetsMonthlyPeriodsAndDeduplicatesAlertsPerPeriod(t *testing.T) {
	t.Parallel()

	cfg := model.Config{
		Version:             model.ConfigVersion,
		BillingCycle:        model.BillingCycle{Kind: model.BillingCycleMonthly},
		PollIntervalSeconds: 1,
		Quotas: model.Quotas{
			Total: model.Limit{Bytes: 100, AlertPercentages: []uint8{70}},
		},
		Notifications: model.NotificationConfig{Enabled: true},
	}
	provider := &fakeProvider{interfaces: []network.Interface{{Name: "Ethernet"}}}
	stateSaver := &fakeStateSaver{}
	notifier := &fakeNotifier{}
	monitor := NewMonitor(cfg, model.State{}, provider, nil, stateSaver, notifier, quietLogger())

	before := time.Date(2026, 8, 31, 23, 0, 0, 0, time.Local)
	provider.counters = network.Counters{DownloadBytes: 1}
	first, err := monitor.Sample(context.Background(), before)
	if err != nil {
		t.Fatalf("first Sample: %v", err)
	}
	if !first.Baseline || first.NewPeriod || len(first.Alerts) != 0 {
		t.Fatalf("first sample = %+v", first)
	}

	provider.counters = network.Counters{DownloadBytes: 100}
	boundary, err := monitor.Sample(context.Background(), before.Add(2*time.Hour))
	if err != nil {
		t.Fatalf("boundary Sample: %v", err)
	}
	if !boundary.Baseline || !boundary.NewPeriod || len(boundary.Alerts) != 0 || len(notifier.notifications) != 0 {
		t.Fatalf("monthly boundary sample = %+v notifications=%d", boundary, len(notifier.notifications))
	}
	if len(stateSaver.states) != 2 || stateSaver.states[1].PeriodKey != "2026-09-01" {
		t.Fatalf("saved monthly state = %+v", stateSaver.states)
	}

	provider.counters = network.Counters{DownloadBytes: 170}
	alert, err := monitor.Sample(context.Background(), before.Add(3*time.Hour))
	if err != nil {
		t.Fatalf("post-boundary Sample: %v", err)
	}
	if len(alert.Alerts) != 1 || len(notifier.notifications) != 1 {
		t.Fatalf("post-boundary alerts = %+v notifications=%d", alert.Alerts, len(notifier.notifications))
	}
}

func TestMonitorChangingBillingCycleStartsNewBaseline(t *testing.T) {
	t.Parallel()

	cfg := model.Config{Version: model.ConfigVersion, PollIntervalSeconds: 1}
	provider := &fakeProvider{interfaces: []network.Interface{{Name: "Ethernet"}}, counters: network.Counters{DownloadBytes: 100}}
	stateSaver := &fakeStateSaver{}
	monitor := NewMonitor(cfg, model.State{}, provider, nil, stateSaver, nil, quietLogger())
	when := time.Date(2026, 8, 21, 8, 0, 0, 0, time.Local)
	if _, err := monitor.Sample(context.Background(), when); err != nil {
		t.Fatalf("baseline Sample: %v", err)
	}
	provider.counters = network.Counters{DownloadBytes: 200}
	if _, err := monitor.Sample(context.Background(), when.Add(time.Minute)); err != nil {
		t.Fatalf("second Sample: %v", err)
	}

	updated := cfg
	updated.BillingCycle = model.BillingCycle{Kind: model.BillingCycleMonthly}
	if err := monitor.SetConfig(updated); err != nil {
		t.Fatalf("SetConfig: %v", err)
	}
	provider.counters = network.Counters{DownloadBytes: 500}
	result, err := monitor.Sample(context.Background(), when.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("cycle-change Sample: %v", err)
	}
	if !result.Baseline || result.Usage != (model.Usage{}) {
		t.Fatalf("cycle-change sample = %+v, want fresh baseline", result)
	}
	if len(stateSaver.states) != 4 || stateSaver.states[3].BillingCycleKey != "monthly" {
		t.Fatalf("saved states after cycle change = %+v", stateSaver.states)
	}
}

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
