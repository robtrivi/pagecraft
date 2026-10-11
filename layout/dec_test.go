package layout

import (
	"testing"

	"github.com/robtrivi/pagecraft/boxes"
)

// Decoration step 2: decorations reach the placed block for rendering.
func TestLayout_CarriesDecoration(t *testing.T) {
	root, err := boxes.Build(`<u>x</u>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("blocks: got %d, want 1", len(blocks))
	}
	if !blocks[0].Underline {
		t.Error("Underline was not carried over")
	}
}
