package tray

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/KageRyo/netquota/assets"
	monitorapp "github.com/KageRyo/netquota/internal/app"
	"github.com/KageRyo/netquota/internal/config"
	"github.com/KageRyo/netquota/internal/format"
	"github.com/KageRyo/netquota/internal/i18n"
	"github.com/KageRyo/netquota/internal/model"
	"github.com/KageRyo/netquota/internal/network"
	"github.com/KageRyo/netquota/internal/quota"
	"github.com/KageRyo/netquota/internal/startup"
	updateapp "github.com/KageRyo/netquota/internal/update"
	"github.com/KageRyo/netquota/internal/version"
)

func Run(ctx context.Context, monitor *monitorapp.Monitor, executable string) {
	_ = updateapp.CleanupStaleDownloads()
	application := fyneapp.NewWithID("com.kageryo.netquota")
	application.SetIcon(fyne.NewStaticResource("icon.svg", assets.IconSVG()))
	window := application.NewWindow("NetQuota")
	ui := newUI(application, window, monitor, executable)
	window.SetContent(ui.dashboard())
	window.Resize(fyne.NewSize(430, 500))
	window.SetCloseIntercept(window.Hide)

	if desktopApp, ok := application.(desktop.App); ok {
		desktopApp.SetSystemTrayIcon(fyne.NewStaticResource("icon-16.png", assets.TrayIconPNG()))
		ui.setDesktopApp(desktopApp)
	}

	pollContext, cancel := context.WithCancel(ctx)
	application.Lifecycle().SetOnStopped(cancel)
	go ui.poll(pollContext)
	if ui.trayMenu != nil {
		go ui.checkForUpdatesInBackground(pollContext)
	}
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
	translator   i18n.Translator
	menu         *fyne.Menu
	totalItem    *fyne.MenuItem
	downloadItem *fyne.MenuItem
	uploadItem   *fyne.MenuItem
	updateItem   *fyne.MenuItem
}

func newTrayMenu(translator i18n.Translator, window fyne.Window, showSettings, checkForUpdates func()) *trayMenu {
	totalItem := newTrayMetricItem(translator.Text("metric.total"))
	downloadItem := newTrayMetricItem(translator.Text("metric.download"))
	uploadItem := newTrayMetricItem(translator.Text("metric.upload"))
	updateItem := fyne.NewMenuItem(translator.Text("tray.check_updates"), checkForUpdates)
	menu := fyne.NewMenu("NetQuota",
		totalItem,
		downloadItem,
		uploadItem,
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem(translator.Text("tray.show_window"), func() {
			window.Show()
		}),
		fyne.NewMenuItem(translator.Text("dashboard.settings"), showSettings),
		updateItem,
	)
	return &trayMenu{
		translator:   translator,
		menu:         menu,
		totalItem:    totalItem,
		downloadItem: downloadItem,
		uploadItem:   uploadItem,
		updateItem:   updateItem,
	}
}

func newTrayMetricItem(name string) *fyne.MenuItem {
	item := fyne.NewMenuItem(name+": —", nil)
	item.Disabled = true
	return item
}

func (m *trayMenu) update(status quota.Status) {
	m.totalItem.Label = metricText(m.translator, "metric.total", status.Total.UsedBytes, status.Total)
	m.downloadItem.Label = metricText(m.translator, "metric.download", status.Download.UsedBytes, status.Download)
	m.uploadItem.Label = metricText(m.translator, "metric.upload", status.Upload.UsedBytes, status.Upload)
	m.menu.Refresh()
}

func (m *trayMenu) setChecking() {
	m.updateItem.Label = m.translator.Text("tray.checking_updates")
	m.updateItem.Action = nil
	m.updateItem.Disabled = true
	m.menu.Refresh()
}

func (m *trayMenu) setReady(checkForUpdates func()) {
	m.updateItem.Label = m.translator.Text("tray.check_updates")
	m.updateItem.Action = checkForUpdates
	m.updateItem.Disabled = false
	m.menu.Refresh()
}

func (m *trayMenu) setUpdateAvailable(tag string, openRelease func()) {
	m.updateItem.Label = m.translator.Text("tray.update_available", map[string]any{"Version": tag})
	m.updateItem.Action = openRelease
	m.updateItem.Disabled = false
	m.menu.Refresh()
}

func (m *trayMenu) setUpdating() {
	m.updateItem.Label = m.translator.Text("tray.downloading_update")
	m.updateItem.Action = nil
	m.updateItem.Disabled = true
	m.menu.Refresh()
}

type ui struct {
	application fyne.App
	window      fyne.Window
	monitor     *monitorapp.Monitor
	executable  string
	translator  i18n.Translator
	desktopApp  desktop.App
	baseTheme   fyne.Theme
	lastSample  *monitorapp.Sample
	lastError   error

	interfaceLabel *widget.Label
	statusLabel    *widget.Label
	updatedLabel   *widget.Label
	downloadLabel  *widget.Label
	uploadLabel    *widget.Label
	totalLabel     *widget.Label
	remainingLabel *widget.Label
	reselectButton *widget.Button
	trayMenu       *trayMenu
	updateChecker  updateapp.Checker
}

func newUI(application fyne.App, window fyne.Window, monitor *monitorapp.Monitor, executable string) *ui {
	ui := &ui{
		application:    application,
		window:         window,
		monitor:        monitor,
		executable:     executable,
		translator:     i18n.New(monitor.Config().Language),
		baseTheme:      application.Settings().Theme(),
		interfaceLabel: widget.NewLabel(""),
		statusLabel:    widget.NewLabel(""),
		updatedLabel:   widget.NewLabel(""),
		downloadLabel:  widget.NewLabel(""),
		uploadLabel:    widget.NewLabel(""),
		totalLabel:     widget.NewLabel(""),
		remainingLabel: widget.NewLabel(""),
		updateChecker:  updateapp.NewChecker(),
	}
	applyLanguageTheme(application, ui.baseTheme, ui.translator.Language())
	ui.refreshDashboardLabels()
	return ui
}

func (u *ui) setDesktopApp(application desktop.App) {
	u.desktopApp = application
	u.resetTrayMenu()
	application.SetSystemTrayWindow(u.window)
}

func (u *ui) resetTrayMenu() {
	if u.desktopApp == nil {
		return
	}
	u.trayMenu = newTrayMenu(u.translator, u.window, u.showSettings, u.checkForUpdates)
	if u.lastSample != nil {
		u.trayMenu.update(u.lastSample.Quota)
	}
	u.desktopApp.SetSystemTrayMenu(u.trayMenu.menu)
}

func (u *ui) setLanguage(language i18n.Language) {
	u.translator = i18n.New(language)
	applyLanguageTheme(u.application, u.baseTheme, u.translator.Language())
	u.refreshDashboardLabels()
	u.resetTrayMenu()
}

func (u *ui) dashboard() fyne.CanvasObject {
	settingsButton := widget.NewButtonWithIcon(u.translator.Text("dashboard.settings"), theme.SettingsIcon(), u.showSettings)
	quitButton := widget.NewButtonWithIcon(u.translator.Text("dashboard.quit"), theme.CancelIcon(), u.application.Quit)
	reselectButton := widget.NewButtonWithIcon(u.translator.Text("dashboard.choose_interface"), theme.SettingsIcon(), u.showSettings)
	reselectButton.Hide()
	u.reselectButton = reselectButton
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
		reselectButton,
		buttons,
	)
	u.refreshDashboardLabels()
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
			u.lastSample = nil
			if errors.Is(err, network.ErrSelectedInterfaceUnavailable) {
				u.lastError = i18n.WrapError("error.interface_unavailable", err, nil)
			} else {
				u.lastError = i18n.WrapError("error.monitoring_failed", err, nil)
			}
			u.refreshDashboardLabels()
			return
		}
		u.lastSample = &sample
		u.lastError = nil
		u.refreshDashboardLabels()
	})
}

func (u *ui) refreshDashboardLabels() {
	if u.lastError != nil {
		if u.reselectButton != nil {
			if errors.Is(u.lastError, network.ErrSelectedInterfaceUnavailable) {
				u.reselectButton.Show()
			} else {
				u.reselectButton.Hide()
			}
		}
		if errors.Is(u.lastError, network.ErrSelectedInterfaceUnavailable) {
			u.interfaceLabel.SetText(u.translator.Text("dashboard.interface.unavailable"))
		}
		u.statusLabel.SetText(u.translator.Text("status.error", map[string]any{"Error": u.translator.ErrorText(u.lastError)}))
		u.statusLabel.Importance = widget.WarningImportance
		u.statusLabel.Refresh()
		return
	}
	if u.reselectButton != nil {
		u.reselectButton.Hide()
	}
	if u.lastSample == nil {
		u.interfaceLabel.SetText(u.translator.Text("dashboard.interface.waiting"))
		u.statusLabel.SetText(u.translator.Text("dashboard.waiting"))
		u.statusLabel.Importance = widget.MediumImportance
		u.updatedLabel.SetText(u.translator.Text("dashboard.last_sample.none"))
		u.downloadLabel.SetText(u.translator.Text("dashboard.metric.download.none"))
		u.uploadLabel.SetText(u.translator.Text("dashboard.metric.upload.none"))
		u.totalLabel.SetText(u.translator.Text("dashboard.metric.total.none"))
		u.remainingLabel.SetText(u.translator.Text("dashboard.remaining.none"))
		return
	}
	sample := *u.lastSample
	u.statusLabel.Importance = widget.MediumImportance
	u.statusLabel.SetText(u.translator.Text("status.monitoring_active"))
	u.interfaceLabel.SetText(interfaceText(u.translator, sample.Interface))
	u.updatedLabel.SetText(u.translator.Text("status.last_sample", map[string]any{"Time": sample.At.Local().Format("15:04:05")}))
	u.downloadLabel.SetText(metricText(u.translator, "metric.download", sample.Usage.DownloadBytes, sample.Quota.Download))
	u.uploadLabel.SetText(metricText(u.translator, "metric.upload", sample.Usage.UploadBytes, sample.Quota.Upload))
	u.totalLabel.SetText(metricText(u.translator, "metric.total", sample.Quota.Total.UsedBytes, sample.Quota.Total))
	if sample.Quota.Total.Enabled {
		u.remainingLabel.SetText(u.translator.Text("metric.remaining", map[string]any{"Bytes": format.Bytes(sample.Quota.Total.RemainingBytes)}))
	} else {
		u.remainingLabel.SetText(u.translator.Text("metric.remaining.disabled"))
	}
	if u.trayMenu != nil {
		u.trayMenu.update(sample.Quota)
	}
}

func (u *ui) checkForUpdates() {
	u.startUpdateCheck(context.Background(), true)
}

func (u *ui) checkForUpdatesInBackground(ctx context.Context) {
	u.startUpdateCheck(ctx, false)
}

func (u *ui) showError(err error) {
	content := widget.NewLabel(u.translator.ErrorText(err))
	content.Wrapping = fyne.TextWrapWord
	dialog.NewCustom(u.translator.Text("app.error"), u.translator.Text("app.close"), content, u.window).Show()
}

func (u *ui) showInformation(title, message string) {
	content := widget.NewLabel(message)
	content.Wrapping = fyne.TextWrapWord
	dialog.NewCustom(title, u.translator.Text("app.close"), content, u.window).Show()
}

func (u *ui) showConfirm(title, message string, callback func(bool)) {
	content := widget.NewLabel(message)
	content.Wrapping = fyne.TextWrapWord
	dialog.NewCustomConfirm(title, u.translator.Text("app.confirm"), u.translator.Text("app.cancel"), content, callback, u.window).Show()
}

func (u *ui) startUpdateCheck(ctx context.Context, interactive bool) {
	if u.trayMenu == nil {
		return
	}
	fyne.Do(func() {
		if u.trayMenu != nil {
			u.trayMenu.setChecking()
		}
	})
	go func() {
		release, available, err := u.updateChecker.Check(ctx, version.Value)
		fyne.Do(func() {
			if u.trayMenu == nil {
				return
			}
			if err != nil {
				u.trayMenu.setReady(u.checkForUpdates)
				if interactive {
					u.showError(i18n.WrapError("update.check_failed", err, nil))
				}
				return
			}
			if !available {
				u.trayMenu.setReady(u.checkForUpdates)
				if interactive {
					u.showInformation(u.translator.Text("update.up_to_date.title"), u.translator.Text("update.up_to_date.message", map[string]any{"Version": version.Value}))
				}
				return
			}
			u.trayMenu.setUpdateAvailable(release.TagName, func() {
				u.confirmUpdate(release)
			})
			if interactive {
				u.confirmUpdate(release)
			}
		})
	}()
}

func (u *ui) confirmUpdate(release updateapp.Release) {
	if release.DownloadURL == "" {
		u.showInformation(u.translator.Text("update.available.title"), u.translator.Text("update.available.no_package", map[string]any{"Version": release.TagName}))
		u.openReleasePage(release)
		return
	}
	if release.ChecksumURL == "" {
		u.showConfirm(u.translator.Text("update.open_release.title"), u.translator.Text("update.open_release.no_checksum"), func(open bool) {
			if open {
				u.openReleasePage(release)
			}
		})
		return
	}
	u.showConfirm(u.translator.Text("update.install.title"), u.translator.Text("update.install.message", map[string]any{"Version": release.TagName}), func(confirmed bool) {
		if confirmed {
			u.downloadAndInstall(release)
		}
	})
}

func (u *ui) downloadAndInstall(release updateapp.Release) {
	if u.trayMenu != nil {
		u.trayMenu.setUpdating()
	}

	ctx, cancel := context.WithCancel(context.Background())
	progress := widget.NewProgressBar()
	status := widget.NewLabel(u.translator.Text("update.preparing_download"))
	content := container.NewVBox(status, progress)
	var progressDialog *dialog.CustomDialog
	progressDialog = dialog.NewCustomWithoutButtons(u.translator.Text("update.downloading.title"), content, u.window)
	cancelButton := widget.NewButton(u.translator.Text("app.cancel"), func() {
		cancel()
		progressDialog.Hide()
		u.restoreUpdateAction(release)
	})
	progressDialog.SetButtons([]fyne.CanvasObject{cancelButton})
	progressDialog.Show()

	go func() {
		updateDirectory, err := updateapp.NewDownloadDirectory()
		if err != nil {
			u.finishUpdateDownload(progressDialog, release, cancel, i18n.WrapError("update.prepare_failed", err, nil))
			return
		}
		path, err := u.updateChecker.Download(ctx, release, updateDirectory, func(written, total int64) {
			fyne.Do(func() {
				if total > 0 {
					progress.SetValue(float64(written) / float64(total))
					status.SetText(u.translator.Text("update.downloaded", map[string]any{"Written": format.Bytes(uint64(written)), "Total": format.Bytes(uint64(total))}))
				} else {
					status.SetText(u.translator.Text("update.downloaded_unknown_total", map[string]any{"Written": format.Bytes(uint64(written))}))
				}
			})
		})
		if err != nil {
			_ = os.RemoveAll(updateDirectory)
			u.finishUpdateDownload(progressDialog, release, cancel, err)
			return
		}
		if err := ctx.Err(); err != nil {
			_ = os.RemoveAll(updateDirectory)
			u.finishUpdateDownload(progressDialog, release, cancel, err)
			return
		}

		if err := enterInstallingState(ctx, status, cancelButton, u.translator); err != nil {
			_ = os.RemoveAll(updateDirectory)
			u.finishUpdateDownload(progressDialog, release, cancel, err)
			return
		}
		installErr := updateapp.Install(context.Background(), path, u.executable)
		if runtime.GOOS != "windows" || installErr != nil {
			_ = os.RemoveAll(updateDirectory)
		}
		if installErr != nil {
			u.finishUpdateDownload(progressDialog, release, cancel, installErr)
			return
		}
		fyne.Do(func() {
			cancel()
			progressDialog.Hide()
			u.application.Quit()
		})
	}()
}

func enterInstallingState(ctx context.Context, status *widget.Label, cancelButton *widget.Button, translator i18n.Translator) error {
	var transitionErr error
	fyne.DoAndWait(func() {
		if err := ctx.Err(); err != nil {
			transitionErr = err
			return
		}
		status.SetText(translator.Text("update.installing"))
		cancelButton.Disable()
	})
	return transitionErr
}

func (u *ui) finishUpdateDownload(progressDialog *dialog.CustomDialog, release updateapp.Release, cancel context.CancelFunc, err error) {
	fyne.Do(func() {
		cancel()
		progressDialog.Hide()
		if errors.Is(err, context.Canceled) {
			u.restoreUpdateAction(release)
			return
		}
		u.restoreUpdateAction(release)
		if errors.Is(err, updateapp.ErrUnsupportedPlatform) || errors.Is(err, os.ErrPermission) {
			u.showConfirm(u.translator.Text("update.automatic_unavailable.title"), u.translator.Text("update.automatic_unavailable.message", map[string]any{"Error": u.translator.ErrorText(err)}), func(open bool) {
				if open {
					u.openReleasePage(release)
				}
			})
			return
		}
		u.showError(i18n.WrapError("update.install_failed", err, nil))
	})
}

func (u *ui) restoreUpdateAction(release updateapp.Release) {
	if u.trayMenu == nil {
		return
	}
	u.trayMenu.setUpdateAvailable(release.TagName, func() {
		u.confirmUpdate(release)
	})
}

func (u *ui) openReleasePage(release updateapp.Release) {
	parsed, err := releasePageURL(release)
	if err != nil {
		u.showError(i18n.WrapError("update.open_failed", err, nil))
		return
	}
	if err := u.application.OpenURL(parsed); err != nil {
		u.showError(i18n.WrapError("update.open_failed", err, nil))
	}
}

func releasePageURL(release updateapp.Release) (*url.URL, error) {
	parsed, err := url.Parse(release.PageURL)
	if err != nil {
		return nil, err
	}
	if parsed.Scheme != "https" || parsed.User != nil || !strings.EqualFold(parsed.Host, "github.com") {
		return nil, fmt.Errorf("unexpected release URL %q", release.PageURL)
	}
	return parsed, nil
}

type settingsView struct {
	form           *widget.Form
	keyboardHint   *widget.Label
	language       *widget.Select
	interfacePick  *widget.Select
	totalQuota     *widget.Entry
	totalAlerts    *widget.Entry
	downloadQuota  *widget.Entry
	downloadAlerts *widget.Entry
	uploadQuota    *widget.Entry
	uploadAlerts   *widget.Entry
	notifications  *widget.Check
	startup        *widget.Check
	byName         map[string]network.Interface
}

func newSettingsView(translator i18n.Translator, cfg model.Config, interfaces []network.Interface) *settingsView {
	options := make([]string, 0, len(interfaces)+1)
	byName := make(map[string]network.Interface, len(interfaces)+1)
	for _, iface := range interfaces {
		options = append(options, iface.Name)
		byName[iface.Name] = iface
	}
	if cfg.Interface.Name != "" {
		if _, ok := byName[cfg.Interface.Name]; !ok {
			options = append(options, cfg.Interface.Name)
			byName[cfg.Interface.Name] = network.Interface{
				Name:            cfg.Interface.Name,
				Index:           cfg.Interface.Index,
				HardwareAddress: cfg.Interface.HardwareAddress,
				IPv4:            cfg.Interface.IPv4,
				IPv6:            cfg.Interface.IPv6,
			}
		}
	}

	interfacePick := widget.NewSelect(options, nil)
	interfacePick.PlaceHolder = translator.Text("settings.choose_interface")
	if cfg.Interface.Name != "" {
		interfacePick.SetSelected(cfg.Interface.Name)
	}
	language := widget.NewSelect(i18n.DisplayNames(), nil)
	language.SetSelected(i18n.DisplayName(cfg.Language))
	totalQuota := widget.NewEntry()
	downloadQuota := widget.NewEntry()
	uploadQuota := widget.NewEntry()
	totalAlerts := widget.NewEntry()
	downloadAlerts := widget.NewEntry()
	uploadAlerts := widget.NewEntry()
	totalQuota.SetText(gibString(cfg.Quotas.Total.Bytes))
	downloadQuota.SetText(gibString(cfg.Quotas.Download.Bytes))
	uploadQuota.SetText(gibString(cfg.Quotas.Upload.Bytes))
	totalAlerts.SetText(thresholdString(cfg.Quotas.Total.AlertPercentages))
	downloadAlerts.SetText(thresholdString(cfg.Quotas.Download.AlertPercentages))
	uploadAlerts.SetText(thresholdString(cfg.Quotas.Upload.AlertPercentages))
	notifications := widget.NewCheck(translator.Text("settings.desktop_notifications"), nil)
	notifications.SetChecked(cfg.Notifications.Enabled)
	startup := widget.NewCheck(translator.Text("settings.start_on_login"), nil)
	startup.SetChecked(cfg.StartOnLogin)

	form := widget.NewForm()
	form.SubmitText = translator.Text("settings.save")
	form.CancelText = translator.Text("app.cancel")
	form.Append(translator.Text("settings.language"), language)
	form.Append(translator.Text("settings.network_interface"), interfacePick)
	form.Append(translator.Text("settings.total_quota"), totalQuota)
	form.Append(translator.Text("settings.total_alerts"), totalAlerts)
	form.Append(translator.Text("settings.download_quota"), downloadQuota)
	form.Append(translator.Text("settings.download_alerts"), downloadAlerts)
	form.Append(translator.Text("settings.upload_quota"), uploadQuota)
	form.Append(translator.Text("settings.upload_alerts"), uploadAlerts)
	form.Append(translator.Text("settings.notifications"), notifications)
	form.Append(translator.Text("settings.startup"), startup)
	form.OnSubmit = func() {}
	form.OnCancel = func() {}

	return &settingsView{
		form:           form,
		keyboardHint:   widget.NewLabel(translator.Text("settings.keyboard_hint")),
		language:       language,
		interfacePick:  interfacePick,
		totalQuota:     totalQuota,
		totalAlerts:    totalAlerts,
		downloadQuota:  downloadQuota,
		downloadAlerts: downloadAlerts,
		uploadQuota:    uploadQuota,
		uploadAlerts:   uploadAlerts,
		notifications:  notifications,
		startup:        startup,
		byName:         byName,
	}
}

func (u *ui) showSettings() {
	cfg := u.monitor.Config()
	interfaces, err := u.monitor.Interfaces(context.Background())
	if err != nil {
		u.showError(i18n.WrapError("settings.load_interfaces_failed", err, nil))
		return
	}
	view := newSettingsView(u.translator, cfg, interfaces)
	form := view.form
	languageSelect := view.language
	interfaceSelect := view.interfacePick
	totalQuota := view.totalQuota
	totalThresholds := view.totalAlerts
	downloadQuota := view.downloadQuota
	downloadThresholds := view.downloadAlerts
	uploadQuota := view.uploadQuota
	uploadThresholds := view.uploadAlerts
	notifications := view.notifications
	startOnLogin := view.startup
	byName := view.byName
	form.OnCancel = func() { u.window.SetContent(u.dashboard()) }
	saveSettings := func(updated model.Config) {
		if err := u.monitor.SetConfig(updated); err != nil {
			u.showError(i18n.WrapError("settings.save_failed", err, nil))
			return
		}
		if err := startup.Configure(updated.StartOnLogin, u.executable); err != nil {
			u.showError(i18n.WrapError("settings.startup_failed", err, nil))
			return
		}
		u.setLanguage(updated.Language)
		u.window.SetContent(u.dashboard())
	}
	form.OnSubmit = func() {
		updated, err := readSettings(cfg, languageSelect, interfaceSelect, byName, totalQuota, totalThresholds, downloadQuota, downloadThresholds, uploadQuota, uploadThresholds, notifications, startOnLogin)
		if err != nil {
			u.showError(err)
			return
		}
		if interfaceSelectionChanged(cfg.Interface, updated.Interface) {
			u.showConfirm(
				u.translator.Text("settings.rebaseline.title"),
				u.translator.Text("settings.rebaseline.message", map[string]any{"Interface": updated.Interface.Name}),
				func(confirmed bool) {
					if confirmed {
						saveSettings(updated)
					}
				},
			)
			return
		}
		saveSettings(updated)
	}
	back := widget.NewButtonWithIcon(u.translator.Text("app.back"), theme.NavigateBackIcon(), func() { u.window.SetContent(u.dashboard()) })
	u.window.SetContent(container.NewBorder(back, nil, nil, nil, container.NewVScroll(container.NewVBox(view.keyboardHint, form))))
	u.window.Show()
}

func interfaceSelectionChanged(current, updated model.InterfaceSelection) bool {
	return !network.SameSelection(current, updated)
}

func readSettings(
	cfg model.Config,
	languageSelect,
	interfaceSelect *widget.Select,
	byName map[string]network.Interface,
	totalQuota, totalThresholds,
	downloadQuota, downloadThresholds,
	uploadQuota, uploadThresholds *widget.Entry,
	notifications, startOnLogin *widget.Check,
) (model.Config, error) {
	var err error
	language, ok := i18n.ParseDisplayName(languageSelect.Selected)
	if !ok {
		return model.Config{}, i18n.NewError("error.language_invalid", nil)
	}
	cfg.Language = language
	if cfg.Quotas.Total, err = parseLimit(totalQuota.Text, totalThresholds.Text); err != nil {
		return model.Config{}, i18n.WrapError("error.settings.total_quota", err, nil)
	}
	if cfg.Quotas.Download, err = parseLimit(downloadQuota.Text, downloadThresholds.Text); err != nil {
		return model.Config{}, i18n.WrapError("error.settings.download_quota", err, nil)
	}
	if cfg.Quotas.Upload, err = parseLimit(uploadQuota.Text, uploadThresholds.Text); err != nil {
		return model.Config{}, i18n.WrapError("error.settings.upload_quota", err, nil)
	}
	if selected, ok := byName[interfaceSelect.Selected]; ok {
		cfg.Interface = model.InterfaceSelection{
			Name:            selected.Name,
			Index:           selected.Index,
			HardwareAddress: selected.HardwareAddress,
			IPv4:            selected.IPv4,
			IPv6:            selected.IPv6,
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

func interfaceText(translator i18n.Translator, iface network.Interface) string {
	if iface.IPv4 != "" {
		return translator.Text("metric.interface_ipv4", map[string]any{"Name": iface.Name, "IPv4": iface.IPv4})
	}
	if iface.IPv6 != "" {
		return translator.Text("metric.interface_ipv6", map[string]any{"Name": iface.Name, "IPv6": iface.IPv6})
	}
	return translator.Text("metric.interface", map[string]any{"Name": iface.Name})
}

func metricText(translator i18n.Translator, nameKey string, used uint64, metric quota.MetricStatus) string {
	name := translator.Text(nameKey)
	if !metric.Enabled {
		return translator.Text("metric.disabled", map[string]any{"Name": name, "Used": format.Bytes(used)})
	}
	return translator.Text("metric.enabled", map[string]any{
		"Name":    name,
		"Used":    format.Bytes(used),
		"Limit":   format.Bytes(metric.LimitBytes),
		"Percent": format.Percent(metric.Percent),
	})
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
