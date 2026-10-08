package boxes

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// Image step 2: <img> resolves data URIs and local files to Raster;
// remote and missing sources have defined behavior.
func makePNGBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.RGBA{0, 128, 255, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return buf.Bytes()
}

func TestBoxes_ImgDataURI_ResolvesRaster(t *testing.T) {
	uri := "data:image/png;base64," + base64.StdEncoding.EncodeToString(makePNGBytes(t, 4, 2))
	root, err := Build(`<img src="` + uri + `" alt="x">`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(root.Children) != 1 {
		t.Fatalf("children: got %d, want 1", len(root.Children))
	}
	b := root.Children[0]
	if b.Image == nil {
		t.Fatal("Image nil")
	}
	if b.Image.Width != 4 || b.Image.Height != 2 {
		t.Errorf("dims: got %dx%d, want 4x2", b.Image.Width, b.Image.Height)
	}
}

func TestBoxes_ImgLocalFile_ResolvesRaster(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.png")
	if err := os.WriteFile(path, makePNGBytes(t, 3, 3), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	root, err := Build(`<img src="` + path + `">`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root.Children[0].Image == nil {
		t.Fatal("Image nil")
	}
}

func TestBoxes_ImgRemote_UnreachableUsesAlt(t *testing.T) {
	// x.test does not resolve: the default fetch fails and returns alt without aborting.
	// This replaces the earlier remote-resource error contract for P0 resources.
	root, err := Build(`<img src="https://x.test/y.png" alt="photo">`)
	if err != nil {
		t.Fatalf("must not abort: %v", err)
	}
	if root.Children[0].Text != "photo" {
		t.Errorf("text: got %q, want photo", root.Children[0].Text)
	}
}

func TestBoxes_ImgWithoutSrc_UsesAlt(t *testing.T) {
	root, err := Build(`<img alt="missing image">`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root.Children[0].Text != "missing image" {
		t.Errorf("text: got %q, want %q", root.Children[0].Text, "missing image")
	}
}

func TestBoxes_ImgParsesCSSDimensions(t *testing.T) {
	root, err := Build(`<img style="width:80px;height:40pt" alt="photo">`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	img := root.Children[0]
	if img.Style.ImageWidth != 60 || img.Style.ImageHeight != 40 {
		t.Errorf("CSS dimensions: %gx%gpt, want 60x40pt", img.Style.ImageWidth, img.Style.ImageHeight)
	}
}

func TestBoxes_ImgParsesCSSPercentageDimensions(t *testing.T) {
	root, err := Build(`<img style="width:50%;height:25%" alt="photo">`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	img := root.Children[0]
	if img.Style.ImageWidthPct != 50 || img.Style.ImageHeightPct != 25 {
		t.Errorf("percentage CSS dimensions: %g%%x%g%%, want 50%%x25%%", img.Style.ImageWidthPct, img.Style.ImageHeightPct)
	}
}

func TestBoxes_InlineSVG_ParsesVectorDocument(t *testing.T) {
	root, err := Build(`<svg width="40" height="20"><rect width="40" height="20" fill="red"/></svg>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(root.Children) != 1 || root.Children[0].SVG == nil {
		t.Fatal("vector SVG was not preserved")
	}
	if len(root.Children[0].SVG.Shapes) != 1 || !root.Children[0].SVG.Shapes[0].HasFill {
		t.Fatalf("unexpected SVG content: %+v", root.Children[0].SVG)
	}
}

func TestBoxes_ImgDataURI_SVGResolvesVector(t *testing.T) {
	data := base64.StdEncoding.EncodeToString([]byte(`<svg width="12" height="6"><circle cx="3" cy="3" r="2"/></svg>`))
	root, err := Build(`<img src="data:image/svg+xml;base64,` + data + `">`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if root.Children[0].SVG == nil || len(root.Children[0].SVG.Shapes) != 1 {
		t.Fatalf("vector SVG was not resolved: %+v", root.Children[0])
	}
}
