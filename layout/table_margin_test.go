package layout

import (
	"math"
	"testing"

	"github.com/robtrivi/pagecraft/boxes"
)

func TestLayout_TableAutoMarginsAlignDeclaredWidth(t *testing.T) {
	for _, tc := range []struct {
		name, margins string
		wantX         float64
	}{
		{name: "left auto aligns right", margins: "margin-left:auto", wantX: 423},
		{name: "right auto stays left", margins: "margin-right:auto", wantX: 72},
		{name: "both auto center", margins: "margin-left:auto;margin-right:auto", wantX: 247.5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, err := boxes.Build(`<table style="width:100pt;` + tc.margins + `"><tr><td>total</td></tr></table>`)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			blocks := Flow(root, testOptions())[0].Blocks
			if len(blocks) != 1 {
				t.Fatalf("table blocks: got %d, want 1", len(blocks))
			}
			if got := blocks[0].X; !approx(got, tc.wantX) {
				t.Errorf("table x = %.2fpt, want %.2fpt", got, tc.wantX)
			}
		})
	}
}

func TestLayout_TableCellRightAlignedTextHonorsPadding(t *testing.T) {
	root, err := boxes.Build(`<table style="width:200pt"><tr><td style="width:68%;padding:3mm">label</td><td style="width:32%;padding:3mm;text-align:right">FACTURA</td></tr></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 || len(blocks[1].Runs) != 1 || len(blocks[1].Runs[0]) != 1 {
		t.Fatalf("right table cell runs: %+v", blocks)
	}
	cell, run := blocks[1], blocks[1].Runs[0][0]
	got := cell.X + cell.Width - run.X - run.Width
	want := 3 * 72 / 25.4
	if math.Abs(got-want) > 0.01 {
		t.Errorf("right text inset = %.3fpt, want padding %.3fpt", got, want)
	}
}
