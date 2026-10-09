// Package images decodes PNG and JPEG images for inclusion in PDF files.
package images

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png"
)

// Raster contains either RGB pixels or original JPEG bytes for PDF passthrough.
type Raster struct {
	Format    string // "png" | "jpeg"
	Space     string // "DeviceRGB" | "DeviceGray" | "" (pass-through JPEG only)
	Width     int
	Height    int
	Pixels    []byte // RGB, len = W*H*3
	Raw       []byte // bytes originales (passthrough JPEG)
	FlateData []byte // original PNG RGB scanlines with PNG row filters, zlib-compressed
}

// Decode converts PNG bytes to RGB and preserves supported JPEG data for passthrough.
func Decode(data []byte) (*Raster, error) {
	if config, err := jpeg.DecodeConfig(bytes.NewReader(data)); err == nil {
		if config.Width > 0 && config.Height > 0 {
			switch config.ColorModel.Convert(color.Black).(type) {
			case color.Gray:
				return &Raster{Format: "jpeg", Space: "DeviceGray", Width: config.Width, Height: config.Height, Raw: data}, nil
			case color.YCbCr:
				return &Raster{Format: "jpeg", Space: "DeviceRGB", Width: config.Width, Height: config.Height, Raw: data}, nil
			}
		}
	}
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
	return &Raster{Format: format, Space: space, Width: w, Height: h, Pixels: px, Raw: data, FlateData: pngRGBFlateData(data)}, nil
}

func pngRGBFlateData(data []byte) []byte {
	if len(data) < 8 || !bytes.Equal(data[:8], []byte{137, 80, 78, 71, 13, 10, 26, 10}) {
		return nil
	}
	var out bytes.Buffer
	var width uint32
	validRGB := false
	transparent := false
	for offset := 8; offset+12 <= len(data); {
		size := binary.BigEndian.Uint32(data[offset : offset+4])
		end := uint64(offset) + 12 + uint64(size)
		if end > uint64(len(data)) {
			return nil
		}
		kind := string(data[offset+4 : offset+8])
		chunk := data[offset+8 : offset+8+int(size)]
		switch kind {
		case "IHDR":
			if len(chunk) != 13 || chunk[8] != 8 || chunk[9] != 2 || chunk[12] != 0 {
				return nil
			}
			width = binary.BigEndian.Uint32(chunk[:4])
			validRGB = width > 0
		case "tRNS":
			transparent = true
		case "IDAT":
			if validRGB && !transparent {
				out.Write(chunk)
			}
		case "IEND":
			if validRGB && !transparent && out.Len() > 0 {
				return out.Bytes()
			}
			return nil
		}
		offset = int(end)
	}
	return nil
}
