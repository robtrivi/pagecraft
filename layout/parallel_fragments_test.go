package layout

import (
	"testing"

	"github.com/robtrivi/pagecraft-go/boxes"
)

func TestLayout_ParallelFlowsShareConstraintsAlignmentAndMerge(t *testing.T) {
	opts := testOptions()
	opts.PageWidth = 200
	opts.Margin = 10
	pages := []Page{{Number: 1}}
	var gotLeft, gotRight []float64
	items := []parallelFlowItem{
		{
			x: 20, width: 50, alignment: boxes.AlignFlexStart,
			layout: func(itemPages *[]Page, itemOpts Options) float64 {
				gotLeft = append(gotLeft, itemOpts.left())
				gotRight = append(gotRight, itemOpts.right())
				(*itemPages)[0].emitBlock(PlacedBox{X: 20, Y: 20, Width: 50, Height: 10})
				return 30
			},
		},
		{
			x: 80, width: 60, alignment: boxes.AlignCenter,
			layout: func(itemPages *[]Page, itemOpts Options) float64 {
				gotLeft = append(gotLeft, itemOpts.left())
				gotRight = append(gotRight, itemOpts.right())
				(*itemPages)[0].emitBlock(PlacedBox{X: 80, Y: 20, Width: 60, Height: 4})
				return 24
			},
		},
	}
	offset, endY := layoutParallelFlows(&pages, 20, 10, opts, items)

	if len(gotLeft) != 2 || gotLeft[0] != 20 || gotLeft[1] != 80 {
		t.Errorf("item left constraints=%v, want [20 80]", gotLeft)
	}
	if len(gotRight) != 2 || gotRight[0] != 130 || gotRight[1] != 60 {
		t.Errorf("item right constraints=%v, want [130 60]", gotRight)
	}
	if offset != 0 || endY != 30 {
		t.Errorf("merged extent=(%d, %g), want (0, 30)", offset, endY)
	}
	if len(pages) != 1 || pages[0].blockCount() != 2 {
		t.Fatalf("merged page blocks=%d across %d pages, want 2 on one page", pages[0].blockCount(), len(pages))
	}
	if got := pages[0].block(1).Y; got != 23 {
		t.Errorf("center-aligned parallel fragment Y=%g, want 23", got)
	}
}
