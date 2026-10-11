package layout

import (
	"strings"
	"testing"

	"github.com/robtrivi/pagecraft/boxes"
)

// Justification is baked into X positions: non-final lines have extra word
// gaps (not Tw operator); the final line uses normal spacing.
func TestLayout_Justify_SpacesWords(t *testing.T) {
	root, err := boxes.Build(`<p style="text-align: justify">` + strings.Repeat("ab ", 30) + `</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("blocks: got %d, want 1", len(blocks))
	}
	runs := blocks[0].Runs
	if len(runs) < 2 {
		t.Fatalf("lines: got %d, want ≥2", len(runs))
	}
	// First line is justified: gap > the fake measurer's 6 pt space.
	gap := runs[0][1].X - runs[0][0].X - runs[0][0].Width
	if gap <= 6 {
		t.Errorf("gap justificado: got %v, want >6", gap)
	}
	// Final line remains left-aligned: gap equals the normal space.
	last := runs[len(runs)-1]
	if len(last) > 1 {
		gapLast := last[1].X - last[0].X - last[0].Width
		if gapLast != 6 {
			t.Errorf("gaplast line: got %v, want 6", gapLast)
		}
	}
}
