package layout

import (
	"strings"
	"testing"

	"github.com/robtrivi/pagecraft-go/boxes"
)

// Thead step 1: in multipage tables, the header repeats at the top of
// each page as additional blocks containing the header text.
func TestLayout_Thead_RepeatsOnPages(t *testing.T) {
	html := `<table><thead><tr><th>name</th></tr></thead><tbody>` +
		strings.Repeat("<tr><td>filla</td></tr>", 60) + `</tbody></table>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pages := Flow(root, testOptions())
	if len(pages) < 2 {
		t.Fatalf("pages: got %d, want ≥2", len(pages))
	}
	for i := 1; i < len(pages); i++ {
		if pages[i].Blocks[0].Text != "name" {
			t.Errorf("page %d starts with %q, want header", i+1, pages[i].Blocks[0].Text)
		}
	}
	total := 0
	for _, p := range pages {
		total += len(p.Blocks)
	}
	if total != 60+len(pages) {
		t.Errorf("blocks: got %d, want 60+%d", total, len(pages))
	}
}
