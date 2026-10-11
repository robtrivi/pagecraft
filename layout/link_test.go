package layout

import (
	"testing"

	"github.com/robtrivi/pagecraft/boxes"
)

// Links step 2: href propagates from the source box to the placed block.
func TestLayout_CarriesLink(t *testing.T) {
	root, err := boxes.Build(`<a href="https://x.test">go</a>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("blocks: got %d, want 1", len(blocks))
	}
	if blocks[0].Link != "https://x.test" {
		t.Errorf("link: got %q, want %q", blocks[0].Link, "https://x.test")
	}
}
