package tray

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	monitorapp "github.com/KageRyo/netquota/internal/app"
	"github.com/KageRyo/netquota/internal/config"
	"github.com/KageRyo/netquota/internal/format"
	"github.com/KageRyo/netquota/internal/model"
	"github.com/KageRyo/netquota/internal/network"
	"github.com/KageRyo/netquota/internal/quota"
	"github.com/KageRyo/netquota/internal/startup"
	"github.com/KageRyo/netquota/internal/version"
)

func Run(ctx context.Context, monitor *monitorapp.Monitor, executable string) {
	application := fyneapp.NewWithID("com.kageryo.netquota")
	application.SetIcon(theme.ComputerIcon())
	window := application.NewWindow("NetQuota")
	ui := newUI(application, window, monitor, executable)
	window.SetContent(ui.dashboard())
	window.Resize(fyne.NewSize(430, 500))
	window.SetCloseIntercept(window.Hide)

	if desktopApp, ok := application.(desktop.App); ok {
		trayMenu := newTrayMenu(window, ui.showSettings)
		ui.trayMenu = trayMenu
		desktopApp.SetSystemTrayMenu(trayMenu.menu)
		desktopApp.SetSystemTrayWindow(window)
	}

	pollContext, cancel := context.WithCancel(ctx)
	application.Lifecycle().SetOnStopped(cancel)
	go ui.poll(pollContext)
	go func() {
		<-pollContext.Done()
		fyne.Do(func() {
			application.Quit()
		})
	}()
	window.Show()
	application.Run()
	cancel()
}

type trayMenu struct {
	menu         *fyne.Menu
	totalItem    *fyne.MenuItem
	downloadItem *fyne.MenuItem
	uploadItem   *fyne.MenuItem
}

func newTrayMenu(window fyne.Window, showSettings func()) *trayMenu {
	totalItem := newTrayMetricItem("Total")
	downloadItem := newTrayMetricItem("Download")
	uploadItem := newTrayMetricItem("Upload")
	menu := fyne.NewMenu("NetQuota",
		totalItem,
		downloadItem,
		uploadItem,
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Show window", func() {
			window.Show()
		}),
		fyne.NewMenuItem("Settings", showSettings),
	)
	return &trayMenu{
		menu:         menu,
		totalItem:    totalItem,
		downloadItem: downloadItem,
		uploadItem:   uploadItem,
	}
}

func newTrayMetricItem(name string) *fyne.MenuItem {
	item := fyne.NewMenuItem(name+": —", nil)
	item.Disabled = true
	return item
}

func (m *trayMenu) update(status quota.Status) {
	m.totalItem.Label = metricText("Total", status.Total.UsedBytes, status.Total)
	m.downloadItem.Label = metricText("Download", status.Download.UsedBytes, status.Download)
	m.uploadItem.Label = metricText("Upload", status.Upload.UsedBytes, status.Upload)
	m.menu.Refresh()
}

type ui struct {
	application fyne.App
	window      fyne.Window
	monitor     *monitorapp.Monitor
	executable  string

	interfaceLabel *widget.Label
	statusLabel    *widget.Label
	updatedLabel   *widget.Label
	downloadLabel  *widget.Label
	uploadLabel    *widget.Label
	totalLabel     *widget.Label
	remainingLabel *widget.Label
	trayMenu       *trayMenu
}

func newUI(application fyne.App, window fyne.Window, monitor *monitorapp.Monitor, executable string) *ui {
	return &ui{
		application:    application,
		window:         window,
		monitor:        monitor,
		executable:     executable,
		interfaceLabel: widget.NewLabel("Interface: waiting for discovery"),
		statusLabel:    widget.NewLabel("Waiting for the first sample…"),
		updatedLabel:   widget.NewLabel("Last sample: —"),
		downloadLabel:  widget.NewLabel("Download: —"),
		uploadLabel:    widget.NewLabel("Upload: —"),
		totalLabel:     widget.NewLabel("Total: —"),
		remainingLabel: widget.NewLabel("Remaining: —"),
	}
}

func (u *ui) dashboard() fyne.CanvasObject {
	settingsButton := widget.NewButtonWithIcon("Settings", theme.SettingsIcon(), u.showSettings)
	quitButton := widget.NewButtonWithIcon("Quit", theme.CancelIcon(), u.application.Quit)
	buttons := container.NewGridWithColumns(2, settingsButton, quitButton)
	content := container.NewVBox(
		widget.NewLabelWithStyle("NetQuota v"+version.Value, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		u.interfaceLabel,
		u.statusLabel,
		u.updatedLabel,
		widget.NewSeparator(),
		u.downloadLabel,
		u.uploadLabel,
		u.totalLabel,
		u.remainingLabel,
		widget.NewSeparator(),
		buttons,
	)
	return container.NewPadded(content)
}

func (u *ui) poll(ctx context.Context) {
	for {
		interval := time.Duration(u.monitor.Config().PollIntervalSeconds) * time.Second
		if interval < time.Second {
			interval = 2 * time.Second
		}
		ticker := time.NewTicker(interval)
		u.sample(ctx)
		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			ticker.Stop()
		}
	}
}

func (u *ui) sample(ctx context.Context) {
	sample, err := u.monitor.Sample(ctx, time.Now())
	fyne.Do(func() {
		if err != nil {
			u.statusLabel.SetText("Error: " + err.Error())
			u.statusLabel.Importance = widget.WarningImportance
			u.statusLabel.Refresh()
			return
		}
		u.statusLabel.Importance = widget.MediumImportance
		u.statusLabel.SetText("Monitoring active")
		u.interfaceLabel.SetText(interfaceText(sample.Interface))
		u.updatedLabel.SetText("Last sample: " + sample.At.Local().Format("15:04:05"))
		u.downloadLabel.SetText(metricText("Download", sample.Usage.DownloadBytes, sample.Quota.Download))
		u.uploadLabel.SetText(metricText("Upload", sample.Usage.UploadBytes, sample.Quota.Upload))
		u.totalLabel.SetText(metricText("Total", sample.Quota.Total.UsedBytes, sample.Quota.Total))
		if sample.Quota.Total.Enabled {
			u.remainingLabel.SetText(fmt.Sprintf("Remaining: %s", format.Bytes(sample.Quota.Total.RemainingBytes)))
		} else {
			u.remainingLabel.SetText("Remaining: total quota disabled")
		}
		if u.trayMenu != nil {
			u.trayMenu.update(sample.Quota)
		}
	})
}

func (u *ui) showSettings() {
	cfg := u.monitor.Config()
	interfaces, err := u.monitor.Interfaces(context.Background())
	if err != nil {
		dialog.ShowError(err, u.window)
		return
	}
	options := make([]string, 0, len(interfaces))
	byName := make(map[string]network.Interface, len(interfaces))
	for _, iface := range interfaces {
		options = append(options, iface.Name)
		byName[iface.Name] = iface
	}
	interfaceSelect := widget.NewSelect(options, nil)
	interfaceSelect.PlaceHolder = "Choose an interface"
	if cfg.Interface.Name != "" {
		interfaceSelect.SetSelected(cfg.Interface.Name)
	}

	totalQuota := widget.NewEntry()
	downloadQuota := widget.NewEntry()
	uploadQuota := widget.NewEntry()
	totalThresholds := widget.NewEntry()
	downloadThresholds := widget.NewEntry()
	uploadThresholds := widget.NewEntry()
	totalQuota.SetText(gibString(cfg.Quotas.Total.Bytes))
	downloadQuota.SetText(gibString(cfg.Quotas.Download.Bytes))
	uploadQuota.SetText(gibString(cfg.Quotas.Upload.Bytes))
	totalThresholds.SetText(thresholdString(cfg.Quotas.Total.AlertPercentages))
	downloadThresholds.SetText(thresholdString(cfg.Quotas.Download.AlertPercentages))
	uploadThresholds.SetText(thresholdString(cfg.Quotas.Upload.AlertPercentages))
	notifications := widget.NewCheck("Desktop notifications", nil)
	notifications.SetChecked(cfg.Notifications.Enabled)
	startOnLogin := widget.NewCheck("Start with the user session", nil)
	startOnLogin.SetChecked(cfg.StartOnLogin)

	form := widget.NewForm()
	form.SubmitText = "Save settings"
	form.CancelText = "Cancel"
	form.Append("Network interface", interfaceSelect)
	form.Append("Total quota (GiB)", totalQuota)
	form.Append("Total alerts (%)", totalThresholds)
	form.Append("Download quota (GiB)", downloadQuota)
	form.Append("Download alerts (%)", downloadThresholds)
	form.Append("Upload quota (GiB)", uploadQuota)
	form.Append("Upload alerts (%)", uploadThresholds)
	form.Append("Notifications", notifications)
	form.Append("Startup", startOnLogin)
	form.OnCancel = func() { u.window.SetContent(u.dashboard()) }
	form.OnSubmit = func() {
		updated, err := readSettings(cfg, interfaceSelect, byName, totalQuota, totalThresholds, downloadQuota, downloadThresholds, uploadQuota, uploadThresholds, notifications, startOnLogin)
		if err != nil {
			dialog.ShowError(err, u.window)
			return
		}
		if err := u.monitor.SetConfig(updated); err != nil {
			dialog.ShowError(err, u.window)
			return
		}
		if err := startup.Configure(updated.StartOnLogin, u.executable); err != nil {
			dialog.ShowError(err, u.window)
			return
		}
		u.window.SetContent(u.dashboard())
	}
	back := widget.NewButtonWithIcon("Back", theme.NavigateBackIcon(), func() { u.window.SetContent(u.dashboard()) })
	u.window.SetContent(container.NewBorder(back, nil, nil, nil, container.NewVScroll(form)))
	u.window.Show()
}

func readSettings(
	cfg model.Config,
	interfaceSelect *widget.Select,
	byName map[string]network.Interface,
	totalQuota, totalThresholds,
	downloadQuota, downloadThresholds,
	uploadQuota, uploadThresholds *widget.Entry,
	notifications, startOnLogin *widget.Check,
) (model.Config, error) {
	var err error
	if cfg.Quotas.Total, err = parseLimit(totalQuota.Text, totalThresholds.Text); err != nil {
		return model.Config{}, fmt.Errorf("total quota: %w", err)
	}
	if cfg.Quotas.Download, err = parseLimit(downloadQuota.Text, downloadThresholds.Text); err != nil {
		return model.Config{}, fmt.Errorf("download quota: %w", err)
	}
	if cfg.Quotas.Upload, err = parseLimit(uploadQuota.Text, uploadThresholds.Text); err != nil {
		return model.Config{}, fmt.Errorf("upload quota: %w", err)
	}
	if selected, ok := byName[interfaceSelect.Selected]; ok {
		cfg.Interface = model.InterfaceSelection{
			Name:            selected.Name,
			HardwareAddress: selected.HardwareAddress,
			IPv4:            selected.IPv4,
		}
	}
	cfg.Notifications.Enabled = notifications.Checked
	cfg.StartOnLogin = startOnLogin.Checked
	return cfg, nil
}

func parseLimit(quotaInput, thresholdInput string) (model.Limit, error) {
	bytes, err := format.ParseGiB(quotaInput)
	if err != nil {
		return model.Limit{}, err
	}
	thresholds, err := format.ParsePercentages(thresholdInput)
	if err != nil {
		return model.Limit{}, err
	}
	return model.Limit{Bytes: bytes, AlertPercentages: thresholds}, nil
}

func interfaceText(iface network.Interface) string {
	if iface.IPv4 == "" {
		return "Interface: " + iface.Name
	}
	return fmt.Sprintf("Interface: %s (%s)", iface.Name, iface.IPv4)
}

func metricText(name string, used uint64, metric quota.MetricStatus) string {
	if !metric.Enabled {
		return fmt.Sprintf("%s: %s (limit disabled)", name, format.Bytes(used))
	}
	return fmt.Sprintf("%s: %s / %s (%s)", name, format.Bytes(used), format.Bytes(metric.LimitBytes), format.Percent(metric.Percent))
}

func gibString(value uint64) string {
	if value == 0 {
		return "0"
	}
	return strconv.FormatFloat(float64(value)/float64(config.BytesPerGiB), 'f', 2, 64)
}

func thresholdString(values []uint8) string {
	if len(values) == 0 {
		return ""
	}
	result := ""
	for index, value := range values {
		if index > 0 {
			result += ","
		}
		result += strconv.Itoa(int(value))
	}
	return result
}
