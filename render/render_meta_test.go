package render

import (
	"bytes"
	"testing"
)

// Metadata step 2: title/author go in the Info dictionary referenced by the trailer.
func TestRender_InfoDict_HasMetadata(t *testing.T) {
	got, err := RenderHTMLToPDF(`<html><head><title>My Document</title><meta name="author" content="Anne"></head><body><p>x</p></body></html>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"/Title (My Document)", "/Author (Anne)", "/Info"} {
		if !bytes.Contains(got, []byte(want)) {
			t.Errorf("the PDF does not contain %q", want)
		}
	}
}

func TestRender_InfoDict_EmptyWithoutMeta(t *testing.T) {
	got, err := RenderHTMLToPDF(`<p>x</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bytes.Contains(got, []byte("/Info")) {
		t.Error("without metadata, /Info must not be present")
	}
}
