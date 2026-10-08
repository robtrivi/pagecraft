package layout

import (
	"testing"

	"github.com/robtrivi/pagecraft-go/boxes"
)

// Hanging step 2: with outside (the default), the marker hangs to the left
// of the content (relative X: 72-12=60 with the fake measurer).
func TestLayout_HangingMarker_Outside(t *testing.T) {
	root, err := boxes.Build(`<ul><li>ab</li></ul>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("blocks: got %d, want 1", len(blocks))
	}
	r := blocks[0].Runs[0]
	if len(r) != 2 {
		t.Fatalf("corridas: got %d, want 2", len(r))
	}
	// A 6 pt bullet and 6 pt space hang outside: X=60, content starts at 72.
	if r[0].X != 60 || r[1].X != 72 {
		t.Errorf("got X=%v %v, want 60 72", r[0].X, r[1].X)
	}
}

func TestLayout_HangingMarker_InsideKeeps(t *testing.T) {
	root, err := boxes.Build(`<ul style="list-style-position: inside"><li>ab</li></ul>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := Flow(root, testOptions())[0].Blocks[0].Runs[0]
	if r[0].X != 72 {
		t.Errorf("inside: got X=%v, want 72", r[0].X)
	}
}
