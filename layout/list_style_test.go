package layout

import (
	"testing"

	"github.com/robtrivi/pagecraft-go/boxes"
)

func TestLayout_DecoratedListPreservesContainerBackgroundAndPadding(t *testing.T) {
	root, err := boxes.Build(`<ul style="list-style-position:inside;background-color:#f1f5f9;padding:8pt"><li>item</li></ul>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	pages := Flow(root, testOptions())
	var container, item *PlacedBox
	for i := range pages[0].Blocks {
		b := &pages[0].Blocks[i]
		if b.HasBg && b.Bg[0] > 0.94 && b.Bg[1] > 0.96 {
			container = b
		}
		if b.Text == "• item" {
			item = b
		}
	}
	if container == nil {
		t.Fatalf("the ul background was not painted: %+v", pages[0].Blocks)
	}
	if item == nil {
		t.Fatalf("inside marker was lost: %+v", pages[0].Blocks)
	}
	if item.X < container.X+8 {
		t.Errorf("item x=%g does not respect the ul padding-left of 8pt at x=%g", item.X, container.X)
	}
}
