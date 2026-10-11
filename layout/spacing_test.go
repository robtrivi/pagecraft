package layout

import (
	"testing"

	"github.com/robtrivi/pagecraft/boxes"
)

// Spacing step 2: the measurer adds letter spacing per rune (fake space width: 6 pt).
// "Hi there" with ls=2: X[1] = 72 + (12+3) + (6+1.5) = 94.5.
func TestLayout_LetterSpacing_WidensMeasure(t *testing.T) {
	root, err := boxes.Build(`<p style="letter-spacing: 2px">Hi there</p>`)
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
	if r[1].X != 94.5 {
		t.Errorf("X[1]: got %v, want 94.5", r[1].X)
	}
}
