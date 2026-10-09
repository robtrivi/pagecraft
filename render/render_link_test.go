package render

import (
	"bytes"
	"testing"
)

// Links step 3: a block with href produces a /Link annotation with a /URI action on its page.
// v1 supports block-level widthrs (direct children) only; nested inline widthrs
// require inline layout (pending, as documented in the roadmap).
func TestRender_Anchor_EmitsLinkAnnotation(t *testing.T) {
	got, err := RenderHTMLToPDF(`<a href="https://x.test/q">go</a>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"/Subtype /Link", "/S /URI", "/URI (https://x.test/q)", "/Annots"} {
		if !bytes.Contains(got, []byte(want)) {
			t.Errorf("the PDF does not contain %q", want)
		}
	}
}
