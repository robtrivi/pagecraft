package render

import (
	"strings"
	"testing"
)

func TestRender_TableCellTextRestoresDefaultColorAfterWhiteHeader(t *testing.T) {
	pdf, err := RenderHTMLToPDF(`<table><thead><tr><th style="color: white">Header</th></tr></thead><tbody><tr><td>Body</td></tr></tbody></table>`)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	stream := streamText(t, pdf)
	header := strings.Index(stream, "(Header) Tj")
	body := strings.Index(stream, "(Body) Tj")
	black := strings.Index(stream, "0 0 0 rg\n")
	if header < 0 || body < 0 {
		t.Fatalf("could not find both texts in the stream:\n%s", stream)
	}
	if black < header || black > body {
		t.Fatalf("cell text does not restore black between the white header and body:\n%s", stream)
	}
}
