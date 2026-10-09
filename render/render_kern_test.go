package render

import (
	"strings"
	"testing"
)

// Kerning step 2: kerned pairs use a TJ array with an adjustment
// AFM kern -120 becomes TJ adjustment +120; without kerning, use classic Tj.
func TestRender_Kerning_EmitsTJAdjust(t *testing.T) {
	got, err := RenderHTMLToPDF(`<p>To</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := streamText(t, got)
	if !strings.Contains(text, "[(T)120(o)] TJ") {
		t.Error("the PDF does not contain the adjustment TJ of kerning")
	}
}

func TestRender_NoKerning_ClassicTj(t *testing.T) {
	got, err := RenderHTMLToPDF(`<p>Hi</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := streamText(t, got)
	if !strings.Contains(text, "(Hi) Tj ET") {
		t.Error("without kerning, classic Tj output should be used")
	}
}
