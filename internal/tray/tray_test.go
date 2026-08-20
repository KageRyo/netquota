package tray

import (
	"testing"

	"fyne.io/fyne/v2/widget"

	"github.com/KageRyo/netquota/internal/config"
	"github.com/KageRyo/netquota/internal/model"
	"github.com/KageRyo/netquota/internal/network"
	"github.com/KageRyo/netquota/internal/quota"
)

func TestParseLimitSupportsIndependentGiBSettings(t *testing.T) {
	t.Parallel()

	limit, err := parseLimit("1.5", "95,70")
	if err != nil {
		t.Fatalf("parseLimit: %v", err)
	}
	if limit.Bytes != uint64(1.5*float64(config.BytesPerGiB)) {
		t.Fatalf("limit bytes = %d", limit.Bytes)
	}
	if got, want := limit.AlertPercentages, []uint8{70, 95}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("thresholds = %v, want %v", got, want)
	}
}

func TestMetricTextShowsDisabledAndEnabledLimits(t *testing.T) {
	t.Parallel()

	disabled := metricText("Upload", 1024, quota.MetricStatus{})
	if disabled != "Upload: 1.0 KiB (limit disabled)" {
		t.Fatalf("disabled metric = %q", disabled)
	}
	enabled := metricText("Total", 50, quota.MetricStatus{
		UsedBytes:  50,
		LimitBytes: 100,
		Percent:    50,
		Enabled:    true,
	})
	if enabled != "Total: 50 B / 100 B (50.0%)" {
		t.Fatalf("enabled metric = %q", enabled)
	}
}

func TestReadSettingsKeepsInterfaceIdentity(t *testing.T) {
	t.Parallel()

	selectWidget := newSelectForTest("Ethernet")
	updated, err := readSettings(
		model.Config{Version: model.ConfigVersion, PollIntervalSeconds: 2},
		selectWidget,
		map[string]network.Interface{"Ethernet": {Name: "Ethernet", HardwareAddress: "AA", IPv4: "192.0.2.10"}},
		entryForTest("100"), entryForTest("70,95,100"),
		entryForTest("10"), entryForTest("90"),
		entryForTest("5"), entryForTest("90"),
		checkForTest(true), checkForTest(false),
	)
	if err != nil {
		t.Fatalf("readSettings: %v", err)
	}
	if updated.Interface.Name != "Ethernet" || updated.Interface.HardwareAddress != "AA" || updated.Interface.IPv4 != "192.0.2.10" {
		t.Fatalf("interface = %+v", updated.Interface)
	}
	if !updated.Notifications.Enabled || updated.StartOnLogin {
		t.Fatalf("settings flags = %+v", updated)
	}
}

func newSelectForTest(selected string) *widget.Select {
	result := widget.NewSelect([]string{selected}, nil)
	result.Selected = selected
	return result
}

func entryForTest(value string) *widget.Entry {
	result := widget.NewEntry()
	result.Text = value
	return result
}

func checkForTest(value bool) *widget.Check {
	result := widget.NewCheck("", nil)
	result.Checked = value
	return result
}
