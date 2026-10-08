package boxes

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
)

// Remote resources step 1: HTTP(S) is fetched (fetch is injectable); failures
// return alt text and a warning without aborting. Base resolves relative paths.
func stubPNG(t *testing.T, w, h int) []byte {
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

func TestBoxes_ImgRemote_FetchesAndResolves(t *testing.T) {
	var gotURL string
	opts := Options{
		Fetch: func(url string) ([]byte, error) {
			gotURL = url
			return stubPNG(t, 4, 2), nil
		},
	}
	root, warns, err := BuildWithOptions(`<img src="https://x.test/y.png" alt="z">`, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warns) != 0 {
		t.Errorf("warnings: got %v, want empty", warns)
	}
	if gotURL != "https://x.test/y.png" {
		t.Errorf("url: got %q", gotURL)
	}
	b := root.Children[0]
	if b.Image == nil || b.Image.Width != 4 {
		t.Errorf("image was not resolved: %+v", b.Image)
	}
}

func TestBoxes_ImgRemote_FailureUsesAlt(t *testing.T) {
	opts := Options{
		Fetch: func(url string) ([]byte, error) {
			return nil, errors.New("network failure")
		},
	}
	root, warns, err := BuildWithOptions(`<img src="https://x.test/y.png" alt="photo">`, opts)
	if err != nil {
		t.Fatalf("must not abort: %v", err)
	}
	if root.Children[0].Text != "photo" {
		t.Errorf("text: got %q, want photo", root.Children[0].Text)
	}
	if len(warns) != 1 {
		t.Errorf("warnings: got %v, want 1", warns)
	}
}

func TestBoxes_RelativePath_UsesBase(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/a.png", stubPNG(t, 3, 3), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	root, warns, err := BuildWithOptions(`<img src="a.png">`, Options{Base: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warns) != 0 {
		t.Errorf("warnings: got %v", warns)
	}
	if b := root.Children[0]; b.Image == nil || b.Image.Width != 3 {
		t.Errorf("image was not resolved with base: %+v", b.Image)
	}
}

func TestBoxes_BaseTag_Overrides(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/b.png", stubPNG(t, 2, 2), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	html := `<html><head><base href="` + dir + `"></head><body><img src="b.png"></body></html>`
	root, _, err := BuildWithOptions(html, Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if b := root.Children[0]; b.Image == nil {
		t.Error("<base href> did not resolve")
	}
}
