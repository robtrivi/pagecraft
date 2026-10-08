package layout

import (
	"testing"

	"github.com/robtrivi/pagecraft-go/boxes"
)

// Text-align step 2: alignment reaches the placed block (render computes
// each line's offset from text metrics).
func TestLayout_CarriesAlign(t *testing.T) {
	root, err := boxes.Build(`<p style="text-align: center">x</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("blocks: got %d, want 1", len(blocks))
	}
	if blocks[0].Align != "center" {
		t.Errorf("align: got %q, want center", blocks[0].Align)
	}
}
