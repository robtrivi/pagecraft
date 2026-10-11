package layout

import (
	"testing"

	"github.com/robtrivi/pagecraft/boxes"
)

// All flows emit Runs (one run per line).
func TestLayout_AllFlows_EmitRuns(t *testing.T) {
	html := `<div style="display: flex"><span>A</span></div>` +
		`<table><tr><td>B</td></tr></table>` +
		`<div style="display: grid; grid-template-columns: 1fr"><span>C</span></div>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 3 {
		t.Fatalf("blocks: got %d, want 3", len(blocks))
	}
	for i, b := range blocks {
		if len(b.Runs) == 0 || len(b.Runs[0]) == 0 {
			t.Errorf("block %d without corridas", i)
		}
	}
}
