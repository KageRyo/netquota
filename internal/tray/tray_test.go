package tray

import (
	"context"
	"errors"
	"image"
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	canvaspkg "fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/software"
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

	for _, test := range []struct {
		language    i18n.Language
		unavailable string
		choose      string
		title       string
		message     string
	}{
		{
			language:    i18n.English,
			unavailable: "Selected network interface is unavailable. Choose another interface in Settings.",
			choose:      "Choose interface",
			title:       "Change monitored interface?",
			message:     "Traffic from the previous interface cannot be recovered. Reset the baseline and monitor Wi-Fi?",
		},
		{
			language:    i18n.TraditionalChinese,
			unavailable: "選取的網路介面目前無法使用。請在設定中選擇其他介面。",
			choose:      "選擇網路介面",
			title:       "要變更監測的網路介面嗎？",
			message:     "無法復原先前介面的流量。要重設基準並監測 Wi-Fi 嗎？",
		},
		{
			language:    i18n.Japanese,
			unavailable: "選択したネットワークインターフェースは利用できません。設定で別のインターフェースを選択してください。",
			choose:      "インターフェースを選択",
			title:       "監視するインターフェースを変更しますか？",
			message:     "以前のインターフェースの通信量は復元できません。基準値をリセットして Wi-Fi を監視しますか？",
		},
	} {
		translator := i18n.New(test.language)
		if got := translator.Text("error.interface_unavailable"); got != test.unavailable {
			t.Fatalf("%s unavailable text = %q, want %q", test.language, got, test.unavailable)
		}
		if got := translator.Text("dashboard.choose_interface"); got != test.choose {
			t.Fatalf("%s choose text = %q, want %q", test.language, got, test.choose)
		}
		if got := translator.Text("settings.rebaseline.title"); got != test.title {
			t.Fatalf("%s rebaseline title = %q, want %q", test.language, got, test.title)
		}
		if got := translator.Text("settings.rebaseline.message", map[string]any{"Interface": "Wi-Fi"}); got != test.message {
			t.Fatalf("%s rebaseline message = %q, want %q", test.language, got, test.message)
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

func TestSettingsViewHasDeterministicFocusableFormOrder(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	view := newSettingsView(i18n.New(i18n.English), config.Default(), []network.Interface{
		{Name: "Ethernet", Index: 1, IPv4: "192.0.2.10"},
		{Name: "Wi-Fi", Index: 2, IPv6: "2001:db8::10"},
	})
	got := make([]string, 0, len(view.form.Items))
	for _, item := range view.form.Items {
		got = append(got, item.Text)
		if _, ok := item.Widget.(fyne.Focusable); !ok {
			t.Fatalf("form item %q does not implement fyne.Focusable: %T", item.Text, item.Widget)
		}
	}
	want := []string{
		"Language",
		"Network interface",
		"Total quota (GiB)",
		"Total alerts (%)",
		"Download quota (GiB)",
		"Download alerts (%)",
		"Upload quota (GiB)",
		"Upload alerts (%)",
		"Notifications",
		"Startup",
	}
	if len(got) != len(want) {
		t.Fatalf("form item count = %d, want %d (%v)", len(got), len(want), got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("form item %d = %q, want %q", index, got[index], want[index])
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

	tray := newTrayMenu(i18n.New(i18n.English), nil, func() {}, func() {})
	if len(tray.menu.Items) != 7 {
		t.Fatalf("tray menu item count = %d, want 7", len(tray.menu.Items))
	}
	for _, item := range []*struct {
		item *fyne.MenuItem
		want string
	}{
		{tray.totalItem, "Total: —"},
		{tray.downloadItem, "Download: —"},
		{tray.uploadItem, "Upload: —"},
	} {
		if item.item.Label != item.want {
			t.Fatalf("initial tray item = %q, want %q", item.item.Label, item.want)
		}
		if !item.item.Disabled {
			t.Fatalf("initial usage item %q should be disabled", item.item.Label)
		}
	}
	if !tray.menu.Items[3].IsSeparator {
		t.Fatal("tray menu should separate usage from actions")
	}
	if got, want := tray.menu.Items[4].Label, "Show window"; got != want {
		t.Fatalf("show item = %q, want %q", got, want)
	}
	if got, want := tray.menu.Items[5].Label, "Settings"; got != want {
		t.Fatalf("settings item = %q, want %q", got, want)
	}
	if got, want := tray.menu.Items[6].Label, "Check for updates"; got != want {
		t.Fatalf("update item = %q, want %q", got, want)
	}
	for _, item := range tray.menu.Items {
		if item.IsQuit {
			t.Fatalf("tray item %q should not be an app-provided quit item", item.Label)
		}
	}
}

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

func TestBundledCJKFontsRenderLocalizedText(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	for _, test := range []struct {
		language i18n.Language
		text     string
	}{
		{language: i18n.TraditionalChinese, text: "正體中文"},
		{language: i18n.Japanese, text: "日本語"},
	} {
		canvas := software.NewTransparentCanvas()
		canvas.Resize(fyne.NewSize(200, 80))
		label := canvaspkg.NewText(test.text, color.Black)
		label.FontSource = languageFont(test.language)
		label.TextSize = 32
		canvas.SetContent(label)
		if !hasVisiblePixels(canvas.Capture()) {
			t.Fatalf("%s font did not render %q", test.language, test.text)
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
