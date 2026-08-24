package assets

import (
	_ "embed"
)

//go:embed icon.svg
var iconSVG []byte

//go:embed tray/icon-16.png
var trayIconPNG []byte

//go:embed fonts/NotoSansCJKtc-Regular.otf
var traditionalChineseFont []byte

//go:embed fonts/NotoSansCJKjp-Regular.otf
var japaneseFont []byte

// IconSVG returns a copy of the scalable application icon.
func IconSVG() []byte {
	return append([]byte(nil), iconSVG...)
}

// TrayIconPNG returns a copy of the 16px system-tray and notification icon.
func TrayIconPNG() []byte {
	return append([]byte(nil), trayIconPNG...)
}

// TraditionalChineseFont returns the bundled Noto Sans CJK TC font. The font
// is licensed under the SIL Open Font License 1.1 in fonts/OFL.txt.
func TraditionalChineseFont() []byte {
	return traditionalChineseFont
}

// JapaneseFont returns the bundled Noto Sans CJK JP font. The font is
// licensed under the SIL Open Font License 1.1 in fonts/OFL.txt.
func JapaneseFont() []byte {
	return japaneseFont
}
