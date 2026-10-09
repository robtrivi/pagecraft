package images

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"hash/crc32"
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

func makeRGBPNG(t *testing.T, w, h int) ([]byte, []byte) {
	t.Helper()
	rows := make([]byte, h*(w*3+1))
	for y := 0; y < h; y++ {
		row := y * (w*3 + 1)
		rows[row] = 1 // PNG filter: Sub.
		for x := 0; x < w; x++ {
			pixel := row + 1 + x*3
			red, green := byte(x*20), byte(y*30)
			if x == 0 {
				rows[pixel], rows[pixel+1], rows[pixel+2] = red, green, 200
			} else {
				rows[pixel], rows[pixel+1], rows[pixel+2] = 20, 0, 0
			}
		}
	}
	var compressed bytes.Buffer
	zw := zlib.NewWriter(&compressed)
	if _, err := zw.Write(rows); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	out.Write([]byte{137, 80, 78, 71, 13, 10, 26, 10})
	writeChunk := func(kind string, data []byte) {
		_ = binary.Write(&out, binary.BigEndian, uint32(len(data)))
		out.WriteString(kind)
		out.Write(data)
		checksum := crc32.ChecksumIEEE(append([]byte(kind), data...))
		_ = binary.Write(&out, binary.BigEndian, checksum)
	}
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr, uint32(w))
	binary.BigEndian.PutUint32(ihdr[4:], uint32(h))
	ihdr[8], ihdr[9] = 8, 2 // 8-bit truecolor RGB.
	writeChunk("IHDR", ihdr)
	writeChunk("IDAT", compressed.Bytes())
	writeChunk("IEND", nil)
	return out.Bytes(), compressed.Bytes()
}

func TestImages_DecodeOpaqueRGBPNG_PreservesFilteredStream(t *testing.T) {
	source, compressed := makeRGBPNG(t, 64, 64)
	got, err := Decode(source)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if !bytes.Equal(got.FlateData, compressed) {
		t.Fatal("the original filtered PNG stream was not preserved")
	}
	var rgbCompressed bytes.Buffer
	rgbWriter := zlib.NewWriter(&rgbCompressed)
	if _, err := rgbWriter.Write(got.Pixels); err != nil {
		t.Fatal(err)
	}
	if err := rgbWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if len(got.FlateData) >= rgbCompressed.Len() {
		t.Fatalf("test PNG predictor stream (%d bytes) should beat raw RGB stream (%d bytes)", len(got.FlateData), rgbCompressed.Len())
	}
}

func TestImages_DecodeRGBAImage_UsesFlattenedPixels(t *testing.T) {
	got, err := Decode(makePNG(t, 1, 1, color.RGBA{R: 200, A: 128}))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(got.FlateData) != 0 {
		t.Fatal("an image with alpha must use the existing white-flattened RGB representation")
	}
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
	if len(got.Pixels) != 0 {
		t.Errorf("JPEG pixels: got %d bytes, want no decoded RGB buffer", len(got.Pixels))
	}
	if !bytes.Equal(got.Raw, buf.Bytes()) {
		t.Error("JPEG source bytes were not preserved for PDF passthrough")
	}
}

func TestImages_DecodeGrayscaleJPEG_PreservesDeviceGray(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 3, 2))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	got, err := Decode(buf.Bytes())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Format != "jpeg" || got.Space != "DeviceGray" {
		t.Errorf("got format=%q space=%q, want jpeg DeviceGray", got.Format, got.Space)
	}
	if len(got.Pixels) != 0 {
		t.Errorf("JPEG pixels: got %d bytes, want no decoded RGB buffer", len(got.Pixels))
	}
	if !bytes.Equal(got.Raw, buf.Bytes()) {
		t.Error("JPEG source bytes were not preserved for PDF passthrough")
	}
}

func TestImages_DecodeInvalid_ReturnsError(t *testing.T) {
	if _, err := Decode([]byte("not an image")); err == nil {
		t.Error("expected an error for invalid bytes")
	}
}
