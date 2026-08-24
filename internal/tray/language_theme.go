package tray

import (
	"fyne.io/fyne/v2"

	"github.com/KageRyo/netquota/assets"
	"github.com/KageRyo/netquota/internal/i18n"
)

type languageTheme struct {
	fyne.Theme
	font fyne.Resource
}

func (theme *languageTheme) Font(fyne.TextStyle) fyne.Resource {
	return theme.font
}

func languageFont(language i18n.Language) fyne.Resource {
	switch language {
	case i18n.TraditionalChinese:
		return fyne.NewStaticResource("NotoSansCJKtc-Regular.otf", assets.TraditionalChineseFont())
	case i18n.Japanese:
		return fyne.NewStaticResource("NotoSansCJKjp-Regular.otf", assets.JapaneseFont())
	default:
		return nil
	}
}

func applyLanguageTheme(application fyne.App, base fyne.Theme, language i18n.Language) {
	font := languageFont(language)
	if font == nil {
		application.Settings().SetTheme(base)
		return
	}
	application.Settings().SetTheme(&languageTheme{Theme: base, font: font})
}
