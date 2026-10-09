package render

import (
	"bytes"
	"testing"
)

// Bold/italic step 3: each variant uses its base-14 font (F1 regular, F2 bold,
// F3 oblique, F4 bold-oblique). Custom fonts use F5+.
func TestRender_BoldItalic_SelectBase14(t *testing.T) {
	got, err := RenderHTMLToPDF(`<b>bold</b><i>italic</i><p style="font-weight: bold; font-style: italic">both</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"/BaseFont /Helvetica-Bold", "/BaseFont /Helvetica-Oblique", "/BaseFont /Helvetica-BoldOblique", "/F2", "/F3", "/F4"} {
		if !bytes.Contains(got, []byte(want)) {
			t.Errorf("the PDF does not contain %q", want)
		}
	}
}
