package layout

import (
	"testing"

	"github.com/robtrivi/pagecraft-go/boxes"
)

// Superscript moves up and scales to 0.7; subscript moves down.
// At 12 pt and y=72, super has y below 72 and size near 8.4; sub has y above 72.
func TestLayout_VAlign_ShiftsRuns(t *testing.T) {
	root, err := boxes.Build(`<p style="margin: 0">a<sup>s</sup><sub>b</sub></p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	runs := Flow(root, testOptions())[0].Blocks[0].Runs[0]
	if len(runs) != 3 {
		t.Fatalf("corridas: got %d, want 3", len(runs))
	}
	if !(runs[1].Y < 72) || !approx(runs[1].FontSize, 8.4) {
		t.Errorf("super: got y=%v fs=%v, want y<72 fs=8.4", runs[1].Y, runs[1].FontSize)
	}
	if !(runs[2].Y > 72) || !approx(runs[2].FontSize, 8.4) {
		t.Errorf("sub: got y=%v fs=%v, want y>72 fs=8.4", runs[2].Y, runs[2].FontSize)
	}
	if runs[0].Y != 72 || runs[0].FontSize != 12 {
		t.Errorf("base: got y=%v fs=%v", runs[0].Y, runs[0].FontSize)
	}
}
