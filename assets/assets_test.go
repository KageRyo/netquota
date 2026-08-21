package assets

import (
	"bytes"
	"image"
	"image/png"
	"strings"
	"testing"
)

func TestIconSVGIsEmbedded(t *testing.T) {
	content := string(IconSVG())
	if !strings.Contains(content, "<svg") {
		t.Fatal("embedded application icon is not an SVG document")
	}
	if !strings.Contains(content, "NetQuota icon") {
		t.Fatal("embedded application icon is missing its title")
	}
}

func TestTrayIconPNGIsEmbedded(t *testing.T) {
	img, err := png.Decode(bytes.NewReader(TrayIconPNG()))
	if err != nil {
		t.Fatalf("decode embedded tray icon: %v", err)
	}

	if got, want := img.Bounds().Size(), image.Pt(16, 16); got != want {
		t.Fatalf("tray icon size = %v, want %v", got, want)
	}
}
