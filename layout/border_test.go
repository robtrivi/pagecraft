package layout

import (
	"testing"

	"github.com/robtrivi/pagecraft/boxes"
)

// Per-side borders step 2: borders reach the placed block for rendering.
func TestLayout_CarriesSideBorders(t *testing.T) {
	root, err := boxes.Build(`<p style="border-bottom: 3px solid red">x</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("blocks: got %d, want 1", len(blocks))
	}
	b := blocks[0]
	if b.BorderBottom.Width != 2.25 {
		t.Errorf("bottom: got %v, want 2.25", b.BorderBottom.Width)
	}
	if b.BorderTop.Width != 0 {
		t.Errorf("top: got %v, want 0", b.BorderTop.Width)
	}
}
