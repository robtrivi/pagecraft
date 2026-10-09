package render

import (
	"strings"
	"testing"
)

// Per-side borders: the bottom border increases outer height by 2.25 pt.
func TestRender_DrawsSideBorder(t *testing.T) {
	got, err := RenderHTMLToPDF(`<p style="margin: 0; border-bottom: 3px solid red">x</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := streamText(t, got)
	if !strings.Contains(text, "72 754.406 451 2.25 re f") {
		t.Error("the PDF does not contain the bottom rectangle")
	}
	if strings.Contains(text, " re S") {
		t.Error("there must not be a uniform stroke")
	}
}
