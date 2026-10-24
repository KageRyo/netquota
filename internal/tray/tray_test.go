package tray

import (
	"context"
	"errors"
	"image"
	"strings"
	"testing"

	"golang.org/x/image/font/sfnt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	fyneTest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	monitorapp "github.com/KageRyo/netquota/internal/app"
	"github.com/KageRyo/netquota/internal/config"
	"github.com/KageRyo/netquota/internal/i18n"
	"github.com/KageRyo/netquota/internal/model"
	"github.com/KageRyo/netquota/internal/network"
	"github.com/KageRyo/netquota/internal/quota"
	updateapp "github.com/KageRyo/netquota/internal/update"
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

	disabled := metricText(i18n.New(i18n.English), "metric.upload", 1024, quota.MetricStatus{})
	if disabled != "Upload: 1.0 KiB (limit disabled)" {
		t.Fatalf("disabled metric = %q", disabled)
	}
	enabled := metricText(i18n.New(i18n.English), "metric.total", 50, quota.MetricStatus{
		UsedBytes:  50,
		LimitBytes: 100,
		Percent:    50,
		Enabled:    true,
	})
	if enabled != "Total: 50 B / 100 B (50.0%)" {
		t.Fatalf("enabled metric = %q", enabled)
	}
}

func TestInterfaceTextUsesIPv6WhenIPv4IsMissing(t *testing.T) {
	t.Parallel()

	got := interfaceText(i18n.New(i18n.English), network.Interface{
		Name: "Tunnel",
		IPv6: "2001:db8::10",
	})
	if want := "Interface: Tunnel (2001:db8::10)"; got != want {
		t.Fatalf("interface text = %q, want %q", got, want)
	}
}

func TestInterfaceRecoveryMessagesAreLocalized(t *testing.T) {
	t.Parallel()
	keys := []string{"error.interface_unavailable", "dashboard.choose_interface", "settings.rebaseline.title", "settings.rebaseline.message"}
	args := map[string]any{"Interface": "audit-interface"}
	english := i18n.New(i18n.English)
	for _, language := range []i18n.Language{i18n.English, i18n.TraditionalChinese, i18n.Japanese} {
		translator := i18n.New(language)
		for _, key := range keys {
			got := translator.Text(key, args)
			if got == "" || got == key {
				t.Fatalf("%s missing recovery translation %s", language, key)
			}
			if language != i18n.English && got == english.Text(key, args) {
				t.Fatalf("%s recovery translation %s fell back to English", language, key)
			}
		}
		if got := translator.Text("settings.rebaseline.message", args); !strings.Contains(got, "audit-interface") {
			t.Fatalf("%s recovery confirmation omitted the selected interface", language)
		}
	}
}

func TestInterfaceSelectionChangeRequiresRebaselineConfirmation(t *testing.T) {
	t.Parallel()

	current := model.InterfaceSelection{Name: "Wi-Fi", Index: 7, HardwareAddress: "AA", IPv4: "192.0.2.10"}
	addressRefresh := current
	addressRefresh.IPv4 = "192.0.2.11"
	if interfaceSelectionChanged(current, addressRefresh) {
		t.Fatal("an address refresh should not require re-baselining")
	}
	if !interfaceSelectionChanged(current, model.InterfaceSelection{Name: "Ethernet", Index: 8, HardwareAddress: "BB"}) {
		t.Fatal("changing the interface should require re-baselining")
	}
}

func TestBillingCycleTextIncludesLocalizedCycleAndNextReset(t *testing.T) {
	t.Parallel()

	nextReset := time.Date(2026, 9, 30, 0, 0, 0, 0, time.Local)
	got := billingCycleText(i18n.New(i18n.English), model.BillingCycle{Kind: model.BillingCycleCustom, ResetDay: 31}, nextReset)
	if want := "Custom monthly (day 31) · reset 2026-09-30 00:00"; got != want {
		t.Fatalf("billing cycle text = %q, want %q", got, want)
	}
}

func TestReadSettingsKeepsInterfaceIdentity(t *testing.T) {
	t.Parallel()

	languageWidget := newSelectForTest("日本語")
	selectWidget := newSelectForTest("Ethernet")
	updated, err := readSettings(
		model.Config{Version: model.ConfigVersion, PollIntervalSeconds: 2},
		languageWidget,
		selectWidget,
		map[string]network.Interface{"Ethernet": {Name: "Ethernet", Index: 3, HardwareAddress: "AA", IPv4: "192.0.2.10", IPv6: "2001:db8::10"}},
		entryForTest("100"), entryForTest("70,95,100"),
		entryForTest("10"), entryForTest("90"),
		entryForTest("5"), entryForTest("90"),
		checkForTest(true), checkForTest(false),
	)
	if err != nil {
		t.Fatalf("readSettings: %v", err)
	}
	if updated.Interface.Name != "Ethernet" || updated.Interface.Index != 3 || updated.Interface.HardwareAddress != "AA" || updated.Interface.IPv4 != "192.0.2.10" || updated.Interface.IPv6 != "2001:db8::10" {
		t.Fatalf("interface = %+v", updated.Interface)
	}
	if !updated.Notifications.Enabled || updated.StartOnLogin {
		t.Fatalf("settings flags = %+v", updated)
	}
	if updated.Language != i18n.Japanese {
		t.Fatalf("language = %q, want %q", updated.Language, i18n.Japanese)
	}
}

func TestReadSettingsParsesCustomBillingCycle(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()
	view := newSettingsView(i18n.New(i18n.English), config.Default(), nil)
	view.billingCycle.SetSelected("Custom monthly (day 1)")
	view.customResetDay.SetText("31")
	updated, err := readSettingsWithCycle(
		config.Default(),
		view.language,
		view.interfacePick,
		view.byName,
		view.totalQuota, view.totalAlerts,
		view.downloadQuota, view.downloadAlerts,
		view.uploadQuota, view.uploadAlerts,
		view.notifications, view.startup,
		view.billingCycle, view.customResetDay,
	)
	if err != nil {
		t.Fatalf("readSettingsWithCycle: %v", err)
	}
	if updated.BillingCycle != (model.BillingCycle{Kind: model.BillingCycleCustom, ResetDay: 31}) {
		t.Fatalf("billing cycle = %+v, want custom day 31", updated.BillingCycle)
	}
}

func TestReadSettingsRejectsInvalidCustomBillingCycleDay(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()
	view := newSettingsView(i18n.New(i18n.English), config.Default(), nil)
	view.billingCycle.SetSelected("Custom monthly (day 1)")
	view.customResetDay.SetText("32")
	if _, err := readSettingsWithCycle(
		config.Default(),
		view.language,
		view.interfacePick,
		view.byName,
		view.totalQuota, view.totalAlerts,
		view.downloadQuota, view.downloadAlerts,
		view.uploadQuota, view.uploadAlerts,
		view.notifications, view.startup,
		view.billingCycle, view.customResetDay,
	); err == nil {
		t.Fatal("readSettingsWithCycle accepted custom reset day 32")
	}
}

func TestSettingsViewExposesFocusableInputsAndKeyboardActions(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	view := newSettingsView(i18n.New(i18n.English), config.Default(), []network.Interface{
		{Name: "Ethernet", Index: 1, IPv4: "192.0.2.10"},
		{Name: "Wi-Fi", Index: 2, IPv6: "2001:db8::10"},
	})
	for _, item := range view.form.Items {
		if _, ok := item.Widget.(fyne.Focusable); !ok {
			t.Fatalf("form item %q does not implement fyne.Focusable: %T", item.Text, item.Widget)
		}
	}
	if view.form.OnSubmit == nil || view.form.OnCancel == nil {
		t.Fatal("settings form must expose keyboard-action callbacks")
	}
	if view.keyboardHint.Text == "" {
		t.Fatal("settings form must expose a keyboard hint")
	}
}

func TestSettingsFocusTraversalVisitsEveryInput(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	view := newSettingsView(i18n.New(i18n.English), config.Default(), []network.Interface{{Name: "Ethernet"}})
	window := app.NewWindow("Settings")
	window.SetContent(container.NewVBox(view.keyboardHint, view.form))
	window.Resize(fyne.NewSize(700, 700))

	want := []fyne.Focusable{
		view.language,
		view.interfacePick,
		view.billingCycle,
		view.totalQuota,
		view.totalAlerts,
		view.downloadQuota,
		view.downloadAlerts,
		view.uploadQuota,
		view.uploadAlerts,
		view.notifications,
		view.startup,
	}
	for index, expected := range want {
		window.Canvas().FocusNext()
		if got := window.Canvas().Focused(); got != expected {
			t.Fatalf("focus %d = %T, want %T", index, got, expected)
		}
	}
}

func TestSettingsKeyboardTogglesCheckboxes(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	view := newSettingsView(i18n.New(i18n.English), config.Default(), nil)
	initialNotifications := view.notifications.Checked
	initialStartup := view.startup.Checked
	view.notifications.TypedRune(' ')
	view.startup.TypedRune(' ')
	if view.notifications.Checked == initialNotifications || view.startup.Checked == initialStartup {
		t.Fatalf("Space did not toggle checkboxes: notifications=%v startup=%v", view.notifications.Checked, view.startup.Checked)
	}
}

func TestSettingsEscapeUsesCancelHandlerAndRestoresPreviousHandler(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	window := app.NewWindow("Settings")
	forwarded := false
	previous := func(*fyne.KeyEvent) { forwarded = true }
	window.Canvas().SetOnTypedKey(previous)
	cancelled := false
	restore := installSettingsKeyboard(window.Canvas(), func() { cancelled = true })
	window.Canvas().OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyA})
	if !forwarded {
		t.Fatal("non-Escape key was not forwarded to the previous handler")
	}
	window.Canvas().OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyEscape})
	if !cancelled {
		t.Fatal("Escape did not invoke the cancel handler")
	}
	restore()
	forwarded = false
	window.Canvas().OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyA})
	if !forwarded {
		t.Fatal("restoring settings keyboard handler did not restore previous handler")
	}
}

func TestTrayMenuLeavesQuitToFyne(t *testing.T) {
	t.Parallel()
	translator := i18n.New(i18n.English)
	window := &showingWindow{}
	settingsCalls, updateCalls := 0, 0
	tray := newTrayMenu(translator, window, func() { settingsCalls++ }, func() { updateCalls++ })
	for _, item := range []*fyne.MenuItem{tray.totalItem, tray.downloadItem, tray.uploadItem} {
		if !item.Disabled {
			t.Fatalf("initial usage item %q should be disabled", item.Label)
		}
	}
	for _, item := range tray.menu.Items {
		if item.IsQuit {
			t.Fatalf("tray item %q should not be an app-provided quit item", item.Label)
		}
		switch item.Label {
		case translator.Text("tray.show_window"), translator.Text("dashboard.settings"), translator.Text("tray.check_updates"):
			item.Action()
		}
	}
	if !window.shown || settingsCalls != 1 || updateCalls != 1 {
		t.Fatalf("required tray actions did not run: shown=%v settings=%d updates=%d", window.shown, settingsCalls, updateCalls)
	}
}

type showingWindow struct {
	fyne.Window
	shown bool
}

func (w *showingWindow) Show() { w.shown = true }

func TestTrayMenuUpdatesUsage(t *testing.T) {
	fyneTest.NewApp()

	tray := newTrayMenu(i18n.New(i18n.English), nil, func() {}, func() {})
	tray.update(quota.Status{
		Total: quota.MetricStatus{
			UsedBytes:  1536,
			LimitBytes: 4096,
			Percent:    37.5,
			Enabled:    true,
		},
		Download: quota.MetricStatus{UsedBytes: 1024},
		Upload:   quota.MetricStatus{UsedBytes: 512},
	})

	if got, want := tray.totalItem.Label, "Total: 1.5 KiB / 4.0 KiB (37.5%)"; got != want {
		t.Fatalf("total item = %q, want %q", got, want)
	}
	if got, want := tray.downloadItem.Label, "Download: 1.0 KiB (limit disabled)"; got != want {
		t.Fatalf("download item = %q, want %q", got, want)
	}
	if got, want := tray.uploadItem.Label, "Upload: 512 B (limit disabled)"; got != want {
		t.Fatalf("upload item = %q, want %q", got, want)
	}
}

func TestTrayMenuShowsAvailableUpdate(t *testing.T) {
	fyneTest.NewApp()

	opened := false
	tray := newTrayMenu(i18n.New(i18n.English), nil, func() {}, func() {})
	tray.setUpdateAvailable("v0.2.0", func() { opened = true })

	if got, want := tray.updateItem.Label, "Update available: v0.2.0"; got != want {
		t.Fatalf("update item = %q, want %q", got, want)
	}
	if tray.updateItem.Disabled {
		t.Fatal("available update should be actionable")
	}
	tray.updateItem.Action()
	if !opened {
		t.Fatal("available update action was not invoked")
	}

	tray.setChecking()
	if !tray.updateItem.Disabled || tray.updateItem.Action != nil {
		t.Fatal("checking state should disable the update item")
	}
}

func TestReleasePageURLDoesNotUseDownloadAsset(t *testing.T) {
	release := updateapp.Release{
		PageURL:     "https://github.com/KageRyo/netquota/releases/tag/v0.2.0",
		DownloadURL: "https://github.com/KageRyo/netquota/releases/download/v0.2.0/netquota-windows-amd64-setup.exe",
	}
	got, err := releasePageURL(release)
	if err != nil {
		t.Fatalf("releasePageURL: %v", err)
	}
	if got.String() != release.PageURL {
		t.Fatalf("release page URL = %q, want %q", got, release.PageURL)
	}
}

func TestReleasePageURLRejectsNonGitHubPage(t *testing.T) {
	_, err := releasePageURL(updateapp.Release{PageURL: "https://example.test/release"})
	if err == nil {
		t.Fatal("releasePageURL accepted a non-GitHub page")
	}
}

func TestEnterInstallingStateRejectsCancellationBeforeTransition(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	status := widget.NewLabel("Downloading update…")
	cancelButton := widget.NewButton("Cancel", nil)
	if err := enterInstallingState(ctx, status, cancelButton, i18n.New(i18n.English)); !errors.Is(err, context.Canceled) {
		t.Fatalf("enterInstallingState error = %v, want context.Canceled", err)
	}
	if status.Text != "Downloading update…" {
		t.Fatalf("status = %q, want download status", status.Text)
	}
	if cancelButton.Disabled() {
		t.Fatal("cancel button should remain enabled when installation transition is rejected")
	}
}

func TestTrayMenuUsesSelectedLanguage(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	tray := newTrayMenu(i18n.New(i18n.TraditionalChinese), nil, func() {}, func() {})
	if got, want := tray.menu.Items[4].Label, "顯示視窗"; got != want {
		t.Fatalf("show item = %q, want %q", got, want)
	}
	tray.update(quota.Status{Total: quota.MetricStatus{UsedBytes: 1024}})
	if got, want := tray.totalItem.Label, "總計：1.0 KiB（未設定上限）"; got != want {
		t.Fatalf("total item = %q, want %q", got, want)
	}
}

func TestSetLanguageImmediatelyRefreshesDashboard(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	ui := &ui{
		application:    app,
		translator:     i18n.New(i18n.English),
		baseTheme:      app.Settings().Theme(),
		interfaceLabel: widget.NewLabel(""),
		statusLabel:    widget.NewLabel(""),
		updatedLabel:   widget.NewLabel(""),
		downloadLabel:  widget.NewLabel(""),
		uploadLabel:    widget.NewLabel(""),
		totalLabel:     widget.NewLabel(""),
		remainingLabel: widget.NewLabel(""),
		lastSample: &monitorapp.Sample{
			Interface: network.Interface{Name: "Ethernet", IPv4: "192.0.2.10"},
			Usage:     model.Usage{DownloadBytes: 1024},
			Quota: quota.Status{
				Download: quota.MetricStatus{UsedBytes: 1024},
			},
		},
	}
	ui.refreshDashboardLabels()
	ui.setLanguage(i18n.Japanese)

	if got, want := ui.statusLabel.Text, "監視中"; got != want {
		t.Fatalf("status = %q, want %q", got, want)
	}
	if got, want := ui.interfaceLabel.Text, "ネットワークインターフェース：Ethernet（192.0.2.10）"; got != want {
		t.Fatalf("interface = %q, want %q", got, want)
	}
	if got, want := ui.downloadLabel.Text, "ダウンロード：1.0 KiB（上限なし）"; got != want {
		t.Fatalf("download = %q, want %q", got, want)
	}
}

func TestBundledCJKFontsContainLocalizedGlyphs(t *testing.T) {
	for _, sample := range []struct {
		language i18n.Language
		text     string
	}{
		{i18n.TraditionalChinese, "正體中文"},
		{i18n.Japanese, "日本語"},
	} {
		resource := languageFont(sample.language)
		if resource == nil {
			t.Fatalf("%s has no bundled font", sample.language)
		}
		font, err := sfnt.Parse(resource.Content())
		if err != nil {
			t.Fatalf("parse %s font: %v", sample.language, err)
		}
		for _, character := range sample.text {
			glyph, err := font.GlyphIndex(nil, character)
			if err != nil || glyph == 0 {
				t.Fatalf("%s font lacks glyph %q: %v", sample.language, character, err)
			}
		}
	}
}

func hasVisiblePixels(image image.Image) bool {
	for y := image.Bounds().Min.Y; y < image.Bounds().Max.Y; y++ {
		for x := image.Bounds().Min.X; x < image.Bounds().Max.X; x++ {
			_, _, _, alpha := image.At(x, y).RGBA()
			if alpha != 0 {
				return true
			}
		}
	}
	return false
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
