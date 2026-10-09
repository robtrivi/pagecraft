package render

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	jpegstd "image/jpeg"
	"testing"
)

// DCT step 2: JPEG with a known color space is embedded as-is (/DCTDecode),
// without recompressing it with Flate.
func jpegURI(t *testing.T, w, h int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.RGBA{200, 30, 30, 255})
		}
	}
	var buf bytes.Buffer
	if err := jpegstd.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestRender_JPEG_PassthroughDCT(t *testing.T) {
	got, err := RenderHTMLToPDF(`<img src="` + jpegURI(t, 8, 4) + `">`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"/Filter /DCTDecode", "/Width 8", "/Height 4"} {
		if !bytes.Contains(got, []byte(want)) {
			t.Errorf("the PDF does not contain %q", want)
		}
	}
}
