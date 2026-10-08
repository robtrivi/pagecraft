package layout

import (
	"strings"
	"testing"

	"github.com/robtrivi/pagecraft-go/boxes"
)

// Fixed positioning step 2: the box appears on EVERY page without moving flow.
func TestLayout_Fixed_RepeatsOnPages(t *testing.T) {
	html := `<div style="position: fixed; top: 10px; left: 10px">Sello</div>` +
		strings.Repeat(`<p style="margin: 0">Hello world</p>`, 100)
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pages := Flow(root, testOptions())
	if len(pages) != 3 {
		t.Fatalf("pages: got %d, want 3", len(pages))
	}
	for i, p := range pages {
		found := false
		for _, b := range p.Blocks {
			if b.Text == "Sello" {
				found = true
			}
		}
		if !found {
			t.Errorf("page %d without sello", i+1)
		}
	}
	// Flow is unchanged: all 100 paragraphs remain intact.
	n := 0
	for _, p := range pages {
		for _, b := range p.Blocks {
			if b.Text == "Hello world" {
				n++
			}
		}
	}
	if n != 100 {
		t.Errorf("paragraphs: got %d, want 100", n)
	}
}

func TestLayout_Fixed_BottomRightUsesPageEdgesAndRepeats(t *testing.T) {
	html := `<!doctype html><html><head><style>body{margin:8px}.badge{position:fixed;right:12pt;bottom:8pt;width:100pt;padding:5pt;border:1pt solid #ef4444}</style></head><body><div class="badge">Marca</div>` +
		strings.Repeat(`<p style="margin:0">Hello world</p>`, 100) + `</body></html>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	pages := Flow(root, testOptions())
	if len(pages) != 3 {
		t.Fatalf("pages: got %d, want 3", len(pages))
	}
	for i, page := range pages {
		var badge *PlacedBox
		for j := range page.Blocks {
			if page.Blocks[j].Text == "Marca" {
				badge = &page.Blocks[j]
				break
			}
		}
		if badge == nil {
			t.Errorf("page %d without a fixed marker", i+1)
			continue
		}
		if !approx(badge.X, 399) || !approx(badge.Y, 735.6) || !approx(badge.Width, 112) {
			t.Errorf("page %d marca: x=%v y=%v width=%v, want 399/735.6/112", i+1, badge.X, badge.Y, badge.Width)
		}
	}
}
