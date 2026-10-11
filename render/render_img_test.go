package render

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	pngstd "image/png"
	"strings"
	"testing"

	"github.com/robtrivi/pagecraft/images"
)

// Image step 4: <img> is embedded as an Image XObject and invoked with Do.
// PNG uses Flate RGB encoding; JPEG data can pass through as DCT.
func pngURI(t *testing.T, w, h int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.RGBA{0, 128, 255, 255})
		}
	}
	var buf bytes.Buffer
	if err := pngstd.Encode(&buf, img); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestRender_OpaqueRGBPNGUsesPDFPredictor(t *testing.T) {
	compressed := []byte{0x78, 0x9c, 0x01, 0x02, 0x03}
	got := imageStream(&images.Raster{Format: "png", Width: 4, Height: 2, FlateData: compressed})
	for _, want := range []string{"/Predictor 15 /Colors 3 /BitsPerComponent 8 /Columns 4", "/Length 5"} {
		if !strings.Contains(got, want) {
			t.Errorf("PNG PDF stream missing %q", want)
		}
	}
	if !strings.Contains(got, string(compressed)) {
		t.Error("PDF stream does not contain the preserved PNG data")
	}
}

func TestRender_RGBPNGFallsBackWhenPredictorStreamIsLarger(t *testing.T) {
	raster := &images.Raster{
		Format: "png", Width: 10, Height: 10,
		Pixels: bytes.Repeat([]byte{0}, 300), FlateData: bytes.Repeat([]byte{1}, 1000),
	}
	got := imageStream(raster)
	if strings.Contains(got, "/Predictor 15") {
		t.Fatal("larger source PNG stream must not increase the PDF image size")
	}
	if !strings.Contains(got, fmt.Sprintf("/Length %d", len(flateBytes(raster.Pixels)))) {
		t.Fatal("RGB fallback stream was not used")
	}
}

func TestRender_EmbedsImage_AsXObject(t *testing.T) {
	got, err := RenderHTMLToPDF(`<img src="` + pngURI(t, 4, 2) + `">`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"/Subtype /Image", "/Width 4", "/Height 2"} {
		if !bytes.Contains(got, []byte(want)) {
			t.Errorf("the PDF does not contain %q", want)
		}
	}
	if !strings.Contains(streamText(t, got), " Do") {
		t.Error("the image was not drawn in the PDF")
	}
}

func TestRender_AbsolutelyPositionedImageIsPaintedBehindFlowContent(t *testing.T) {
	html := `<img style="position:absolute;top:24pt;right:18pt;width:80pt;height:60pt;opacity:.14" src="` + pngURI(t, 4, 2) + `"><p>VISIBLE DOCUMENT TEXT</p>`
	pdf, err := RenderHTMLToPDF(html)
	if err != nil {
		t.Fatalf("render document with background image: %v", err)
	}
	if !bytes.Contains(pdf, []byte("/Subtype /Image")) {
		t.Fatal("the absolutely positioned image must be embedded in the PDF")
	}
	content := streamText(t, pdf)
	background := strings.Index(content, "/Fm1 Do")
	image := strings.Index(content, "/Im1 Do")
	text := strings.Index(content, "(VISIBLE)")
	if background < 0 || image < 0 || text < 0 || background > text {
		t.Errorf("background image must paint before flow text: form=%d image=%d text=%d", background, image, text)
	}
}

func TestRender_InlineImageInTablePreservesPaintOrder(t *testing.T) {
	pdf, err := RenderHTMLToPDF(`<style>body{margin:0}table{margin:0;width:220pt}td{font:10pt/12pt Helvetica;padding:0}</style><table><tr><td>BEFORE <img style="width:24pt;height:20pt" src="` + pngURI(t, 4, 2) + `"> AFTER</td></tr></table>`)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	if !bytes.Contains(pdf, []byte("/Subtype /Image")) || !bytes.Contains(pdf, []byte("/Width 4 /Height 2")) {
		t.Fatal("inline cell image must be registered as an XObject")
	}
	content := streamText(t, pdf)
	before := strings.Index(content, "(BEF)")
	image := strings.Index(content, "/Im1 Do")
	after := strings.Index(content, "(AFTER)")
	if before < 0 || image < 0 || after < 0 || !(before < image && image < after) {
		t.Errorf("incorrect inline paint order: BEFORE=%d image=%d AFTER=%d", before, image, after)
	}
}

func TestRender_InlineImageInBlockPreservesPaintOrder(t *testing.T) {
	pdf, err := RenderHTMLToPDF(`<p style="margin:0;font:10pt/12pt Helvetica">BEFORE <img style="width:24pt;height:20pt" src="` + pngURI(t, 4, 2) + `"> AFTER</p>`)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	content := streamText(t, pdf)
	t.Logf("stream: %q", content)
	before := strings.Index(content, "(BEF)")
	image := strings.Index(content, "/Im1 Do")
	after := strings.Index(content, "(AFTER)")
	if before < 0 || image < 0 || after < 0 || !(before < image && image < after) {
		t.Errorf("incorrect inline paint order: BEFORE=%d image=%d AFTER=%d", before, image, after)
	}
}
