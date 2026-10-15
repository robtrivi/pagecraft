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

func TestRender_PageFooter_UsesDeclaredFontSizeAndColor(t *testing.T) {
	html := `<style>@page { @bottom-center { content: "PÁGINA " counter(page); color: #53636b; font: 7pt Arial, sans-serif; } }</style><p>body</p>`
	got, err := RenderHTMLToPDF(html)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stream := streamText(t, got)
	if !strings.Contains(stream, "/F1 7 Tf") {
		t.Errorf("footer should use declared 7pt font size; content stream: %s", stream)
	}
	if !strings.Contains(stream, "0.3254901960784314 0.38823529411764707 0.4196078431372549 rg") {
		t.Errorf("footer should use declared #53636b color; content stream: %s", stream)
	}
}
