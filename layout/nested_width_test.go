package layout

import (
	"testing"

	"github.com/robtrivi/pagecraft-go/boxes"
)

func TestLayout_NestedBlockFillsFixedWidthContainerContentArea(t *testing.T) {
	root, err := boxes.Build(`<div style="width:100pt;padding:10pt;border:1pt solid black;background-color:#eee"><div style="padding:2pt;background-color:#f00">child panel</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	pages := Flow(root, testOptions())
	var child *PlacedBox
	for _, block := range pages[0].Blocks {
		if block.HasBg && block.Bg[0] == 1 {
			child = &block
		}
	}
	if child == nil {
		t.Fatalf("panel child not found: %+v", pages[0].Blocks)
	}
	if child.Width != 100 {
		t.Errorf("panel child width=%gpt, want 100pt from the parent content box", child.Width)
	}
}

func TestLayout_InlineBlockUsesDeclaredContentWidth(t *testing.T) {
	root, err := boxes.Build(`<div style="width:220pt;padding:12pt;border:2pt solid #1d4ed8;background-color:#dbeafe;border-radius:50%">content inline</div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	pages := Flow(root, testOptions())
	if len(pages) != 1 || len(pages[0].Blocks) != 1 {
		t.Fatalf("layout unexpected: %+v", pages)
	}
	if got := pages[0].Blocks[0].Width; got != 248 {
		t.Errorf("width of box=%gpt, want 248pt (220pt of content plus padding and border)", got)
	}
}
