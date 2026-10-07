// Package images decodes PNG and JPEG images for inclusion in PDF files.
package images

import (
	"bytes"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
)

// Raster contains a decoded RGB image.
type Raster struct {
	Format string // "png" | "jpeg"
	Space  string // "DeviceRGB" | "DeviceGray" | "" (pass-through JPEG only)
	Width  int
	Height int
	Pixels []byte // RGB, len = W*H*3
	Raw    []byte // bytes originales (passthrough JPEG)
}

// Decode converts PNG or JPEG bytes into an RGB image.
func Decode(data []byte) (*Raster, error) {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if format != "png" && format != "jpeg" {
		return nil, errors.New("images: unsupported format: " + format)
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return nil, errors.New("images: invalid dimensions")
	}
	space := ""
	if format == "jpeg" {
		switch img.(type) {
		case *image.Gray:
			space = "DeviceGray"
		case *image.YCbCr:
			space = "DeviceRGB"
		}
	}
	px := make([]byte, 0, w*h*3)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := img.At(x, y).RGBA()
			if a == 0xFFFF {
				px = append(px, byte(r>>8), byte(g>>8), byte(bl>>8))
				continue
			}
			// Composite over white.
			af := float64(a) / 0xFFFF
			px = append(px,
				byte(float64(r>>8)*af+255*(1-af)),
				byte(float64(g>>8)*af+255*(1-af)),
				byte(float64(bl>>8)*af+255*(1-af)),
			)
		}
	}
	return &Raster{Format: format, Space: space, Width: w, Height: h, Pixels: px, Raw: data}, nil
}
