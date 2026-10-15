package render

import (
	"strings"
	"testing"

	"github.com/robtrivi/pagecraft/layout"
)

// Spacing step 3: emit Tc per block and reset it afterward.
func TestRender_LetterSpacing_EmitsTc(t *testing.T) {
	got, err := RenderHTMLToPDF(`<p style="letter-spacing: 2px">Hi</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := streamText(t, got)
	if !strings.Contains(text, "1.5 Tc") {
		t.Error("the PDF does not set Tc")
	}
	if !strings.Contains(text, "\n0 Tc\n") {
		t.Error("the PDF does not reset Tc")
	}
}

func TestRender_LetterSpacing_InlineRunEmitsTc(t *testing.T) {
	got, err := RenderHTMLToPDF(`<table><tr><td style="text-align:right"><div style="font-size:26pt;font-weight:bold;letter-spacing:-1px">Factura</div></td></tr></table>`)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	stream := streamText(t, got)
	if !strings.Contains(stream, "-0.75 Tc") {
		t.Error("the PDF painter omitted letter spacing from the inline text run")
	}
}

func TestRender_LetterSpacing_RightAlignmentUsesPaintedAdvance(t *testing.T) {
	var stream strings.Builder
	block := layout.PlacedBox{
		Align: "right",
		Runs: [][]layout.TextRun{{{
			Text: "Factura", X: 100, Width: 88.662, FontSize: 26, Bold: true, LetterSpacing: -0.75,
		}}},
	}
	drawRuns(&stream, block, 842, nil, nil, nil, nil, nil)
	if !strings.Contains(stream.String(), "-0.75 Tc 99.25 ") {
		t.Errorf("right-aligned text was not shifted to match its painted width: %q", stream.String())
	}
}
