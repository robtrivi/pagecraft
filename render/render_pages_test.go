package render

import (
	"strings"
	"testing"
)

// Real pipeline from boxes through layout to PDF with
// one PDF page per layout page. The default layout measurer places the
// paragraphs across multiple pages.
func TestRender_LongDocument_ProducesMultiplePages(t *testing.T) {
	got, err := RenderHTMLToPDF(strings.Repeat(`<p style="margin: 0">Hello world</p>`, 100))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := streamText(t, got)
	words := assertPDFHasWordCoordinates(t, got)
	var firstY, lastY float64
	for _, word := range words {
		if word.Text == "Hello" || word.Text == "world" {
			if firstY == 0 || word.Y < firstY {
				firstY = word.Y
			}
			if word.Y > lastY {
				lastY = word.Y
			}
		}
	}
	if lastY-firstY < 842 {
		t.Errorf("content coordinates span only %.2fpt; expected a flow at least one A4 page tall", lastY-firstY)
	}
	if !strings.Contains(text, "(Hello) Tj") || !strings.Contains(text, "[(wor)") || !strings.Contains(text, "(ld)] TJ") {
		t.Error("the PDF does not contain the text in extractable form")
	}
}
