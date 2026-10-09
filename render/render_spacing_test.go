package render

import (
	"strings"
	"testing"
)

// Spacing step 3: emit Tc per block and reset it afterward.
func TestRender_LetterSpacing_EmitsTc(t *testing.T) {
	got, err := RenderHTMLToPDF(`<p style="letter-spacing: 2px">Hi</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := streamText(t, got)
	if !strings.Contains(text, "1.5 Tc\n") {
		t.Error("the PDF does not set Tc")
	}
	if !strings.Contains(text, "\n0 Tc\n") {
		t.Error("the PDF does not reset Tc")
	}
}
