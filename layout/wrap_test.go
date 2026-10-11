package layout

import (
	"strings"
	"testing"

	"github.com/robtrivi/pagecraft/boxes"
)

// Inline flow shares line opportunities with floats: one line can use the
// available side space, and later lines can return to the full width.
func TestLayout_FloatWrap_PlacesBeside(t *testing.T) {
	root, err := boxes.Build(`<div style="float: left">L</div><p style="margin: 0">Hi</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("blocks: got %d, want 2", len(blocks))
	}
	// The block retains its containing block; text uses the available space
	// beside the float.
	if blocks[1].X != 72 || blocks[1].Y != 72 || blocks[1].Runs[0][0].X != 78 {
		t.Errorf("beside float: box=(%v,%v), first run=%+v", blocks[1].X, blocks[1].Y, blocks[1].Runs)
	}
}

func TestLayout_FloatWrap_TallBlockGoesBelow(t *testing.T) {
	// Flow wraps the paragraph in the reduced space beside the float, then
	// restores full width when the line no longer intersects the exclusion.
	root, err := boxes.Build(`<div style="float: left">FLOTANTE</div><p style="margin: 0">` + strings.Repeat("ab ", 23) + `</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("blocks: got %d, want 2", len(blocks))
	}
	if blocks[1].X != 72 || blocks[1].Y != 72 || len(blocks[1].Runs) < 2 {
		t.Fatalf("the paragraph must preserve its box and create lines around the float: %+v", blocks[1])
	}
	if blocks[1].Runs[0][0].X <= blocks[1].X || blocks[1].Runs[1][0].X != blocks[1].X || blocks[1].Runs[1][0].Y <= blocks[1].Runs[0][0].Y {
		t.Errorf("lines do not adapt to the float: %+v", blocks[1].Runs)
	}
}
