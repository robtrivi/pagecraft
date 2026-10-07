package images

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

// Decode PNG/JPEG images to RGB rasters with their dimensions.
// Fixtures are generated in memory; no files are needed.
func makePNG(t *testing.T, w, h int, c color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestImages_DecodePNG_ReturnsDimsAndPixels(t *testing.T) {
	got, err := Decode(makePNG(t, 2, 1, color.RGBA{255, 0, 0, 255}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Format != "png" || got.Width != 2 || got.Height != 1 {
		t.Errorf("got %s %dx%d, want png 2x1", got.Format, got.Width, got.Height)
	}
	if len(got.Pixels) != 2*1*3 {
		t.Fatalf("pixels: got %d bytes, want 6", len(got.Pixels))
	}
	if got.Pixels[0] != 255 || got.Pixels[1] != 0 || got.Pixels[2] != 0 {
		t.Errorf("pixel 0: got %v, want red", got.Pixels[:3])
	}
}

func TestImages_DecodeTransparentPNG_FlattensOnWhite(t *testing.T) {
	got, err := Decode(makePNG(t, 1, 1, color.RGBA{0, 0, 0, 0}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i, v := range got.Pixels {
		if v != 255 {
			t.Errorf("byte %d: got %v, want 255 (white)", i, v)
		}
	}
}

func TestImages_DecodeJPEG_ReturnsDims(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 3, 2))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	got, err := Decode(buf.Bytes())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Format != "jpeg" || got.Width != 3 || got.Height != 2 {
		t.Errorf("got %s %dx%d, want jpeg 3x2", got.Format, got.Width, got.Height)
	}
	if got.Space != "DeviceRGB" {
		t.Errorf("color space: got %q, want DeviceRGB", got.Space)
	}
}

func TestImages_DecodeInvalid_ReturnsError(t *testing.T) {
	if _, err := Decode([]byte("not an image")); err == nil {
		t.Error("expected an error for invalid bytes")
	}
}
