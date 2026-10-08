package layout

import (
	"testing"

	"github.com/robtrivi/pagecraft-go/boxes"
)

// Break step 1: <br> breaks the line without adding text.
func TestLayout_Br_BreaksLine(t *testing.T) {
	root, err := boxes.Build(`<p>line1<br>line2</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("blocks: got %d, want 1", len(blocks))
	}
	lines := runLines(blocks[0])
	if len(lines) != 2 || lines[0] != "line1" || lines[1] != "line2" {
		t.Errorf("lines: got %q", runLines(blocks[0]))
	}
}

func TestLayout_Br_DoubleSkipsLine(t *testing.T) {
	root, err := boxes.Build(`<p>a<br><br>b</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(runLines(blocks[0])) != 3 {
		t.Errorf("lines: got %q, want 3", runLines(blocks[0]))
	}
}
