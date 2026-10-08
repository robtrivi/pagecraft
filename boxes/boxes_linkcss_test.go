package boxes

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// Link step 1: <link rel=stylesheet href=file> applies in document order;
// Remote resources use fetch; a missing local file returns an error, while a failed remote request adds a warning.
func TestBoxes_LinkStylesheet_Applies(t *testing.T) {
	css := `p { color: blue; }`
	path := filepath.Join(t.TempDir(), "a.css")
	if err := os.WriteFile(path, []byte(css), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	root, err := Build(`<html><head><link rel="stylesheet" href="` + path + `"></head><body><p>x</p></body></html>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c := root.Children[0].Style.Color; c != [4]float64{0, 0, 1, 1} {
		t.Errorf("color: got %v, want blue", c)
	}
}

func TestBoxes_LinkStylesheet_OrderMatters(t *testing.T) {
	css := `p { color: blue; }`
	path := filepath.Join(t.TempDir(), "b.css")
	if err := os.WriteFile(path, []byte(css), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	// <link> before <style>: the style rule takes precedence (document order).
	html := `<html><head><link rel="stylesheet" href="` + path + `"><style>p { color: red; }</style></head><body><p>x</p></body></html>`
	root, err := Build(html)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c := root.Children[0].Style.Color; c != [4]float64{1, 0, 0, 1} {
		t.Errorf("color: got %v, want red", c)
	}
}

func TestBoxes_LinkRemote_FallsBackWithWarning(t *testing.T) {
	opts := Options{
		Fetch: func(url string) ([]byte, error) {
			return nil, errors.New("network failure")
		},
	}
	root, warns, err := BuildWithOptions(`<html><head><link rel="stylesheet" href="https://x.test/a.css"></head><body><p>x</p></body></html>`, opts)
	if err != nil {
		t.Fatalf("must not abort: %v", err)
	}
	if len(warns) != 1 {
		t.Errorf("warnings: got %v, want 1", warns)
	}
	if root.Children[0].Style.HasColor {
		t.Error("no color without CSS")
	}
}
