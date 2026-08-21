package assets

import (
	_ "embed"
)

//go:embed icon.svg
var iconSVG []byte

//go:embed tray/icon-16.png
var trayIconPNG []byte

// IconSVG returns a copy of the scalable application icon.
func IconSVG() []byte {
	return append([]byte(nil), iconSVG...)
}

// TrayIconPNG returns a copy of the 16px system-tray and notification icon.
func TrayIconPNG() []byte {
	return append([]byte(nil), trayIconPNG...)
}
