package render

import (
	"bytes"
	"strings"
	"testing"
)

// Non-ASCII (v0.5): with /WinAnsiEncoding, text is emitted as bytes
// WinAnsi literals (Latin e with acute uses byte \351). UTF-16BE exposed the BOM in poppler, and
// without /Encoding readers used StandardEncoding.
func TestRender_AccentedText_EncodedExtractable(t *testing.T) {
	got, err := RenderHTMLToPDF("<p>Café</p>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := streamText(t, got)
	if !strings.Contains(text, `(Caf\351)`) {
		t.Error("the PDF does not contain café in bytes WinAnsi")
	}
	if !bytes.Contains(got, []byte("/WinAnsiEncoding")) {
		t.Error("source does not declare /WinAnsiEncoding")
	}
}
