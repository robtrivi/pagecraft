package render

import (
	"bytes"
	"fmt"
	"math"
	"testing"
)

// @page dimensions define the PDF MediaBox; the resolved margin offsets the
// content origin on both axes, and the top margin is converted to PDF baseline.
func TestRender_PageRuleComposesPhysicalSizeAndMargins(t *testing.T) {
	for _, tc := range []struct {
		name, pageCSS string
		width, height float64
		margin        float64
	}{
		{"named paper in inches", "size: Letter; margin: 1in", 612, 792, 72},
		{"explicit dimensions in points", "size: 400pt 500pt; margin: 20px", 400, 500, 15},
		{"landscape paper in centimeters", "size: A4 landscape; margin: 2cm", 842, 595, 2 * 72 / 2.54},
	} {
		t.Run(tc.name, func(t *testing.T) {
			html := `<style>@page { ` + tc.pageCSS + ` }</style><p style="margin: 0">Hello</p>`
			pdf, err := RenderHTMLToPDF(html)
			if err != nil {
				t.Fatalf("RenderHTMLToPDF: %v", err)
			}
			if box := fmt.Sprintf("/MediaBox [0 0 %g %g]", tc.width, tc.height); !bytes.Contains(pdf, []byte(box)) {
				t.Errorf("PDF MediaBox does not match @page size %q; want %s", tc.pageCSS, box)
			}
			positions := tdAll([]byte(streamText(t, pdf)), "Hello")
			if len(positions) != 1 {
				t.Fatalf("text positions: got %v, want one rendered line", positions)
			}
			wantY := tc.height - tc.margin - 12 // page height - top margin - font baseline offset
			if math.Abs(positions[0][0]-tc.margin) > 0.001 || math.Abs(positions[0][1]-wantY) > 0.001 {
				t.Errorf("text position=(%g,%g), want (%g,%g) from page margins and size", positions[0][0], positions[0][1], tc.margin, wantY)
			}
		})
	}
}
