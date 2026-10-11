package layout

import (
	"strings"
	"testing"

	"github.com/robtrivi/pagecraft/boxes"
)

func TestLayout_DecoratedTextBlockSlicesBordersAcrossPageFragments(t *testing.T) {
	html := `<p style="margin:0;padding:3pt;border:1pt solid #123456;background:#e0f2fe">` +
		strings.Repeat("following words ", 90) + `</p>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 260, 150, 20
	opts.FontSize, opts.LineHeight = 10, 1.2
	opts.Measure = func(value string, _ float64) float64 { return float64(len([]rune(value))) * 4.5 }
	pages := Flow(root, opts)
	if len(pages) < 2 {
		t.Fatalf("pages=%d, want decorated text to fragment", len(pages))
	}
	var fragments []PlacedBox
	for _, page := range pages {
		for _, block := range page.Blocks {
			if strings.Contains(block.Text, "following") && block.HasBg {
				fragments = append(fragments, block)
			}
		}
	}
	if len(fragments) != len(pages) {
		t.Fatalf("decorated fragments=%d, pages=%d", len(fragments), len(pages))
	}
	if fragments[0].BorderTop.Width != 1 || fragments[0].BorderBottom.Width != 0 {
		t.Errorf("first fragment borders top/bottom=%v/%v, want 1/0", fragments[0].BorderTop.Width, fragments[0].BorderBottom.Width)
	}
	if fragments[0].BorderWidth != 0 {
		t.Errorf("first fragment uniform border=%v, want 0 so only the slice edge is painted", fragments[0].BorderWidth)
	}
	for index, fragment := range fragments[1 : len(fragments)-1] {
		if fragment.BorderTop.Width != 0 || fragment.BorderBottom.Width != 0 || fragment.BorderWidth != 0 {
			t.Errorf("middle fragment %d borders top/bottom=%v/%v, want 0/0", index+1, fragment.BorderTop.Width, fragment.BorderBottom.Width)
		}
	}
	last := fragments[len(fragments)-1]
	if last.BorderTop.Width != 0 || last.BorderBottom.Width != 1 {
		t.Errorf("last fragment borders top/bottom=%v/%v, want 0/1", last.BorderTop.Width, last.BorderBottom.Width)
	}
	if last.BorderWidth != 0 {
		t.Errorf("last fragment uniform border=%v, want 0 so only the slice edge is painted", last.BorderWidth)
	}
}
