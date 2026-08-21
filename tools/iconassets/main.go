package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"github.com/sergeymakinen/go-ico"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

var iconSizes = []int{16, 32, 48, 64, 128, 256}

func main() {
	sourcePath := flag.String("source", "assets/icon.svg", "canonical SVG source")
	pngPath := flag.String("png", "assets/icon-256.png", "high-resolution PNG output")
	icoPath := flag.String("ico", "assets/windows/icon.ico", "Windows ICO output")
	flag.Parse()

	svg, err := os.ReadFile(*sourcePath)
	if err != nil {
		fail("read SVG source", err)
	}

	large, err := render(svg, 256)
	if err != nil {
		fail("render 256px PNG", err)
	}
	if err := writePNG(*pngPath, large); err != nil {
		fail("write PNG output", err)
	}

	icons := make([]image.Image, 0, len(iconSizes))
	for _, size := range iconSizes {
		icon, err := render(svg, size)
		if err != nil {
			fail(fmt.Sprintf("render %dpx ICO entry", size), err)
		}
		icons = append(icons, icon)
	}
	if err := writeICO(*icoPath, icons); err != nil {
		fail("write ICO output", err)
	}
}

func render(svg []byte, size int) (image.Image, error) {
	icon, err := oksvg.ReadIconStream(bytes.NewReader(svg), oksvg.StrictErrorMode)
	if err != nil {
		return nil, err
	}
	if icon.ViewBox.W <= 0 || icon.ViewBox.H <= 0 {
		return nil, fmt.Errorf("invalid SVG viewBox %gx%g", icon.ViewBox.W, icon.ViewBox.H)
	}

	icon.SetTarget(0, 0, float64(size), float64(size))
	image := image.NewRGBA(image.Rect(0, 0, size, size))
	scanner := rasterx.NewScannerGV(size, size, image, image.Bounds())
	raster := rasterx.NewDasher(size, size, scanner)
	icon.Draw(raster, 1)
	return image, nil
}

func writePNG(path string, image image.Image) error {
	file, err := createOutput(path)
	if err != nil {
		return err
	}
	if err := png.Encode(file, image); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func writeICO(path string, images []image.Image) error {
	file, err := createOutput(path)
	if err != nil {
		return err
	}
	if err := ico.EncodeAll(file, images); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func createOutput(path string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return os.Create(path)
}

func fail(action string, err error) {
	fmt.Fprintf(os.Stderr, "iconassets: %s: %v\n", action, err)
	os.Exit(1)
}
