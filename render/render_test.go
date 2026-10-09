package render

import (
	"bytes"
	"strings"
	"testing"
)

// Public rendering contract:
// Empty input returns an error; valid input returns a PDF with the %PDF- header.
func TestRenderHTMLToPDF_EmptyInput_ReturnsError(t *testing.T) {
	for _, in := range []string{"", "   ", "\n\t "} {
		if _, err := RenderHTMLToPDF(in); err == nil {
			t.Errorf("input %q: expected error, got nil", in)
		}
	}
}

func TestRenderHTMLToPDF_MinimalParagraph_ReturnsPDFWithHeader(t *testing.T) {
	got, err := RenderHTMLToPDF("<p>Hello</p>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("PDF empty")
	}
	if !bytes.HasPrefix(got, []byte("%PDF-")) {
		t.Errorf("missing %%PDF- header; first bytes: %q", got[:min(8, len(got))])
	}
	if !bytes.Contains(got, []byte("%%EOF")) {
		t.Error("missing the final EOF marker")
	}
}

func TestRender_ParagraphWithText_EmbedsText(t *testing.T) {
	got, err := RenderHTMLToPDF("<p>Hello world</p>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := streamText(t, got)
	// The PDF text stream must retain both words.
	if !strings.Contains(text, "(Hello) Tj") || !strings.Contains(text, "[(wor)") || !strings.Contains(text, "(ld)] TJ") {
		t.Error("the PDF does not contain paragraph text in extractable form")
	}
}

func TestRender_BodyBackgroundCoversPrintedPage(t *testing.T) {
	got, err := RenderHTMLToPDF(`<!doctype html><style>@page{size:200pt 300pt;margin:0}body{margin:0;background:#f5f3ff}</style><p>Ticket</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content := streamText(t, got)
	if !strings.Contains(content, "0.9607843137254902 0.9529411764705882 1 rg") || !strings.Contains(content, "0 0 200 300 re f") {
		t.Fatalf("body background must paint the entire page canvas, stream: %q", content)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
