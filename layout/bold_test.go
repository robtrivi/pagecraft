package layout

import (
	"strings"
	"testing"

	"github.com/robtrivi/pagecraft/boxes"
	"github.com/robtrivi/pagecraft/text"
)

// Bold step 2: the block carries Bold and Flow measures with bold widths.
// Thirty-three copies of "m ": regular is 440 pt (one line), bold is 462 pt (two lines).
func TestLayout_Bold_MeasuresBold(t *testing.T) {
	root, err := boxes.Build(`<p style="font-weight: bold">` + strings.Repeat("m ", 33) + `</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	opts := testOptions()
	opts.Measure = text.Measure
	opts.MeasureBold = text.MeasureBold
	blocks := Flow(root, opts)[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("blocks: got %d, want 1", len(blocks))
	}
	if !blocks[0].Bold {
		t.Error("Bold was not carried into the block")
	}
	if n := len(blocks[0].Runs); n != 2 {
		t.Errorf("lines: got %d, want 2 (bold measurement)", n)
	}
}
