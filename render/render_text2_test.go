package render

import (
	"strings"
	"testing"
)

// Text styling step 3: per-block size and color in the PDF.
func TestRender_DrawsSizedColoredText(t *testing.T) {
	got, err := RenderHTMLToPDF(`<p style="font-size: 20px; color: blue">Hello</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := streamText(t, got)
	for _, want := range []string{"/F1 15 Tf", "0 0 1 rg"} {
		if !strings.Contains(text, want) {
			t.Errorf("the PDF does not contain %q", want)
		}
	}
}
