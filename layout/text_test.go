package layout

import (
	"testing"

	"github.com/robtrivi/pagecraft/boxes"
)

// Text styling step 2: the element's font-size controls wrapping and height
// A 20px font creates a 24pt line; otherwise Options supplies the default.
func TestLayout_FontSize_AffectsHeight(t *testing.T) {
	root, err := boxes.Build(`<p style="font-size: 20px">Hello</p><p>Goodbye</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pages := Flow(root, testOptions())
	blocks := pages[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("blocks: got %d, want 2", len(blocks))
	}
	if !approx(blocks[0].Height, 18) || blocks[0].FontSize != 15 {
		t.Errorf("large: got h=%v fs=%v, want h=18 fs=15", blocks[0].Height, blocks[0].FontSize)
	}
	if !approx(blocks[1].Height, 14.4) || blocks[1].FontSize != 12 {
		t.Errorf("default: got h=%v fs=%v, want h=14.4 fs=12", blocks[1].Height, blocks[1].FontSize)
	}
}
