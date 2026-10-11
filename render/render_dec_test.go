package render

import (
	"fmt"
	"strings"
	"testing"

	"github.com/robtrivi/pagecraft/boxes"
	"github.com/robtrivi/pagecraft/layout"
	"github.com/robtrivi/pagecraft/text"
)

func TestRender_PaintsInlineBoxDecorationAtItsMeasuredPosition(t *testing.T) {
	html := `<p style="margin:0;font:10pt/14pt Helvetica">before <span style="background:#ffee99;border:1pt solid #123456;padding:2pt 3pt">HIGHLIGHTED LABEL</span> after</p>`
	pdf, err := RenderHTMLToPDF(html)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	inflated := streamText(t, pdf)
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := layout.DefaultOptions()
	opts.Measure, opts.MeasureBold = text.Measure, text.MeasureBold
	pages := layout.Flow(root, opts)
	var decoration *layout.InlineBoxDecoration
	for _, page := range pages {
		for _, block := range page.Blocks {
			for _, line := range block.Runs {
				for _, run := range line {
					if run.Text == "HIGHLIGHTED" && len(run.InlineDecorations) > 0 {
						decoration = &run.InlineDecorations[0]
					}
				}
			}
		}
	}
	if decoration == nil {
		t.Fatal("layout did not produce the expected inline decoration")
	}
	wantColor := fmt.Sprintf("%g %g %g rg", decoration.Box.Bg[0], decoration.Box.Bg[1], decoration.Box.Bg[2])
	if !strings.Contains(inflated, wantColor) {
		t.Errorf("the PDF stream does not paint the inline background %q", wantColor)
	}
	wantRect := fmt.Sprintf("%g %g %g %g re f", decoration.Box.X, opts.PageHeight-decoration.Box.Y-decoration.Box.Height, decoration.Box.Width, decoration.Box.Height)
	if !strings.Contains(inflated, wantRect) {
		t.Errorf("the PDF stream does not paint the measured fragment %q", wantRect)
	}
}

// Decorations use the text run's width and color. Underline and strike differ
// only in their specified offset from the baseline.
func TestRender_TextDecorationsSpanMeasuredRunAtSpecifiedOffset(t *testing.T) {
	const fontSize, baseline = 12.0, 758.0
	for _, tc := range []struct {
		name, tag      string
		baselineOffset float64
	}{
		{"underline", "u", -0.1},
		{"strike", "s", 0.26},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := RenderHTMLToPDF("<" + tc.tag + ">Hi</" + tc.tag + ">")
			if err != nil {
				t.Fatalf("RenderHTMLToPDF: %v", err)
			}
			y := baseline + tc.baselineOffset*fontSize
			want := fmt.Sprintf("72 %g m %g %g l S", y, 72+text.Measure("Hi", fontSize), y)
			if !strings.Contains(streamText(t, got), want) {
				t.Errorf("PDF stream does not contain measured decoration segment %q", want)
			}
		})
	}
}
