package layout

import (
	"testing"

	"github.com/robtrivi/pagecraft-go/boxes"
)

// White-space step 2: pre preserves spaces and \n without wrapping;
// nowrap collapses whitespace and does not wrap (one line, even if it overflows).
func TestLayout_Pre_PreservesSpacesAndBreaks(t *testing.T) {
	root, err := boxes.Build("<p style=\"white-space: pre\">a  b\nc</p>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("blocks: got %d, want 1", len(blocks))
	}
	if len(blocks[0].Runs) != 2 {
		t.Fatalf("lines: got %d, want 2", len(blocks[0].Runs))
	}
	r0 := blocks[0].Runs[0]
	if len(r0) != 2 || r0[0].Text != "a" || r0[1].Text != "b" {
		t.Errorf("line 0: got %+v", r0)
	}
	// Double spaces are preserved with a 12pt gap using the 6pt test measurer.
	if gap := r0[1].X - r0[0].X - r0[0].Width; gap != 12 {
		t.Errorf("gap: got %v, want 12", gap)
	}
	if len(blocks[0].Runs) != 2 || len(blocks[0].Runs[1]) != 1 || blocks[0].Runs[1][0].Text != "c" {
		t.Errorf("line 1: got %+v", blocks[0].Runs)
	}
	if x := blocks[0].Runs[1][0].X; x != 72 {
		t.Errorf("without indent: got X=%v", x)
	}
}

func TestLayout_Nowrap_SingleLine(t *testing.T) {
	root, err := boxes.Build(`<p style="white-space: nowrap">` + "ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab ab" + `</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks[0].Runs) != 1 {
		t.Errorf("nowrap: got %d lines, want 1", len(blocks[0].Runs))
	}
}
