package assets

import (
	"bytes"
	"image"
	"image/png"
	"os"
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

func TestCJKFontsAreEmbedded(t *testing.T) {
	for name, font := range map[string][]byte{
		"Traditional Chinese": TraditionalChineseFont(),
		"Japanese":            JapaneseFont(),
	} {
		if len(font) < 1<<20 {
			t.Fatalf("%s font is unexpectedly small: %d bytes", name, len(font))
		}
		if got := string(font[:4]); got != "OTTO" {
			t.Fatalf("%s font signature = %q, want OpenType CFF signature", name, got)
		}
	}

	license, err := os.ReadFile("fonts/OFL.txt")
	if err != nil {
		t.Fatalf("read bundled font license: %v", err)
	}
	if !strings.Contains(string(license), "SIL OPEN FONT LICENSE") {
		t.Fatal("bundled font license is not the SIL Open Font License")
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
	assertTransparentCorners(t, img)
}

func assertTransparentCorners(t *testing.T, img image.Image) {
	t.Helper()
	bounds := img.Bounds()
	corners := []image.Point{
		bounds.Min,
		{X: bounds.Max.X - 1, Y: bounds.Min.Y},
		{X: bounds.Min.X, Y: bounds.Max.Y - 1},
		{X: bounds.Max.X - 1, Y: bounds.Max.Y - 1},
	}
	for _, point := range corners {
		_, _, _, alpha := img.At(point.X, point.Y).RGBA()
		if alpha != 0 {
			t.Fatalf("icon corner %v alpha = %d, want transparent", point, alpha)
		}
	}
}
