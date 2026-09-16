package tray

import (
	"image/color"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/software"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/KageRyo/netquota/internal/config"
	"github.com/KageRyo/netquota/internal/i18n"
	"github.com/KageRyo/netquota/internal/model"
	"github.com/KageRyo/netquota/internal/network"
)

func TestDashboardAccessibleControlsUseLocalizedLabels(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	for _, language := range i18n.SupportedLanguages() {
		ui := newAccessibilityUI(app, language)
		accessible := collectAccessible(ui.dashboard())
		labels := make(map[string]fyne.Accessible, len(accessible))
		for _, object := range accessible {
			labels[object.AccessibilityLabel()] = object
		}
		for _, key := range []string{"dashboard.settings", "dashboard.quit"} {
			label := ui.translator.Text(key)
			object, ok := labels[label]
			if !ok {
				t.Fatalf("%s dashboard missing accessible label %q", language, label)
			}
			if object.AccessibilityRole() != fyne.AccessibleRoleButton {
				t.Fatalf("%s label %q has role %q, want button", language, label, object.AccessibilityRole())
			}
		}
	}
}

func TestLocalizedDashboardAndSettingsRenderWithinProductionWidths(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	for _, language := range i18n.SupportedLanguages() {
		ui := newAccessibilityUI(app, language)
		applyLanguageTheme(app, ui.baseTheme, language)
		dashboardCanvas := software.NewTransparentCanvas()
		dashboardCanvas.Resize(fyne.NewSize(430, 500))
		dashboard := ui.dashboard()
		if dashboard.MinSize().Width > 430 {
			t.Fatalf("%s dashboard minimum width = %v, exceeds 430", language, dashboard.MinSize())
		}
		dashboardCanvas.SetContent(dashboard)
		if !hasVisiblePixels(dashboardCanvas.Capture()) {
			t.Fatalf("%s dashboard rendered no visible pixels", language)
		}

		settingsConfig := config.Default()
		settingsConfig.BillingCycle = model.BillingCycle{Kind: model.BillingCycleCustom, ResetDay: 31}
		view := newSettingsView(ui.translator, settingsConfig, []network.Interface{
			{Name: "Ethernet", IPv4: "192.0.2.10"},
			{Name: "Wi-Fi", IPv6: "2001:db8::10"},
		})
		settingsCanvas := software.NewTransparentCanvas()
		settingsCanvas.Resize(fyne.NewSize(700, 700))
		settings := container.NewVBox(view.keyboardHint, view.form)
		if settings.MinSize().Width > 700 {
			t.Fatalf("%s settings minimum width = %v, exceeds 700", language, settings.MinSize())
		}
		settingsCanvas.SetContent(settings)
		if !hasVisiblePixels(settingsCanvas.Capture()) {
			t.Fatalf("%s settings rendered no visible pixels", language)
		}
	}
}

func TestLocalizedBillingCycleTextIsVisible(t *testing.T) {
	t.Parallel()

	nextReset := time.Date(2026, 9, 30, 0, 0, 0, 0, time.Local)
	for _, language := range i18n.SupportedLanguages() {
		text := billingCycleText(i18n.New(language), model.BillingCycle{Kind: model.BillingCycleCustom, ResetDay: 31}, nextReset)
		if text == "" || strings.Contains(text, "dashboard.billing_cycle") || strings.Contains(text, "settings.cycle_custom") {
			t.Fatalf("%s billing cycle text = %q, want rendered localized text", language, text)
		}
	}
}

func TestThemeFocusColorIsDistinctFromBackground(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name    string
		theme   fyne.Theme
		variant fyne.ThemeVariant
	}{
		{name: "light", theme: theme.LightTheme(), variant: theme.VariantLight},
		{name: "dark", theme: theme.DarkTheme(), variant: theme.VariantDark},
	} {
		background := color.NRGBAModel.Convert(test.theme.Color(theme.ColorNameBackground, test.variant)).(color.NRGBA)
		focus := color.NRGBAModel.Convert(test.theme.Color(theme.ColorNameFocus, test.variant)).(color.NRGBA)
		if background == focus {
			t.Fatalf("%s theme focus color equals background: %#v", test.name, background)
		}
	}
}

func collectAccessible(object fyne.CanvasObject) []fyne.Accessible {
	result := make([]fyne.Accessible, 0)
	if accessible, ok := object.(fyne.Accessible); ok {
		result = append(result, accessible)
	}
	if container, ok := object.(*fyne.Container); ok {
		for _, child := range container.Objects {
			result = append(result, collectAccessible(child)...)
		}
	}
	return result
}

func newAccessibilityUI(app fyne.App, language i18n.Language) *ui {
	return &ui{
		application:    app,
		window:         app.NewWindow("NetQuota"),
		translator:     i18n.New(language),
		baseTheme:      app.Settings().Theme(),
		interfaceLabel: widget.NewLabel(""),
		statusLabel:    widget.NewLabel(""),
		updatedLabel:   widget.NewLabel(""),
		downloadLabel:  widget.NewLabel(""),
		uploadLabel:    widget.NewLabel(""),
		totalLabel:     widget.NewLabel(""),
		remainingLabel: widget.NewLabel(""),
	}
}
