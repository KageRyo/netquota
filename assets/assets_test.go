package assets

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"strings"
	"testing"

	"github.com/sergeymakinen/go-ico"
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

func TestPackagingPNGIsGeneratedAtHighResolution(t *testing.T) {
	file, err := os.Open("icon-256.png")
	if err != nil {
		t.Fatalf("open packaging PNG: %v", err)
	}
	defer file.Close()

	img, err := png.Decode(file)
	if err != nil {
		t.Fatalf("decode packaging PNG: %v", err)
	}
	if got, want := img.Bounds().Size(), image.Pt(256, 256); got != want {
		t.Fatalf("packaging PNG size = %v, want %v", got, want)
	}
}

func TestWindowsICOContainsExpectedSizes(t *testing.T) {
	file, err := os.Open("windows/icon.ico")
	if err != nil {
		t.Fatalf("open Windows ICO: %v", err)
	}
	defer file.Close()

	icons, err := ico.DecodeAll(file)
	if err != nil {
		t.Fatalf("decode Windows ICO: %v", err)
	}
	want := []image.Point{
		image.Pt(16, 16),
		image.Pt(32, 32),
		image.Pt(48, 48),
		image.Pt(64, 64),
		image.Pt(128, 128),
		image.Pt(256, 256),
	}
	if len(icons) != len(want) {
		t.Fatalf("Windows ICO entries = %d, want %d", len(icons), len(want))
	}
	for i, icon := range icons {
		if got := icon.Bounds().Size(); got != want[i] {
			t.Fatalf("Windows ICO entry %d size = %v, want %v", i, got, want[i])
		}
	}
}
