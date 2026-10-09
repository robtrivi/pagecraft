package render

import (
	"testing"
)

// CSS padding shifts inline content by its resolved left inset, regardless of
// whether the declaration came from a shorthand or longhand.
func TestRender_PaddingOffsetsTextByResolvedLeftInset(t *testing.T) {
	for _, tc := range []struct {
		name, style string
		leftInset   float64
	}{
		{"asymmetric shorthand", "padding: 0 10px 0 20px", 15},
		{"longhand in points", "padding-left: 12pt", 12},
		{"zero inset", "padding: 0", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pdf, err := RenderHTMLToPDF(`<p style="margin: 0; ` + tc.style + `">Hello</p>`)
			if err != nil {
				t.Fatalf("RenderHTMLToPDF: %v", err)
			}
			positions := tdAll([]byte(streamText(t, pdf)), "Hello")
			if len(positions) != 1 {
				t.Fatalf("text positions: got %v, want one rendered line", positions)
			}
			wantX := 72 + tc.leftInset
			if positions[0][0] != wantX {
				t.Errorf("text starts at x=%g, want page inset 72 + resolved left padding %g = %g", positions[0][0], tc.leftInset, wantX)
			}
		})
	}
}
