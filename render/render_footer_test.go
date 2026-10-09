package render

import (
	"strings"
	"testing"
)

// Footer step 3: "N of M" on every page (two pages of 50 paragraphs).
func TestRender_PageNumbers_DrawFooters(t *testing.T) {
	html := `<style>@page { @bottom-center { content: counter(page) " of " counter(pages); } }</style>` +
		strings.Repeat("<p>Hello world</p>", 50)
	got, err := RenderHTMLToPDF(html)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := streamText(t, got)
	// Fifty one-line paragraphs at 14.4 pt occupy 48 lines on page one and 2 on page two.
	for _, want := range []string{"(1 of 2) Tj ET", "(2 of 2) Tj ET"} {
		if !strings.Contains(text, want) {
			t.Errorf("the PDF does not contain %q", want)
		}
	}
}
