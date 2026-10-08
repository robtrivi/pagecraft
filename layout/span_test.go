package layout

import (
	"strings"
	"testing"

	"github.com/robtrivi/pagecraft-go/boxes"
)

// A spanning cell covers multiple columns and reserves rows.
// Percentage widths are not covered here (columns are equal).
func TestLayout_TableColspan_SpansColumns(t *testing.T) {
	root, err := boxes.Build(`<table style="width:451pt"><tr><td colspan="2">A</td></tr><tr><td>B</td><td>C</td></tr></table>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 3 {
		t.Fatalf("blocks: got %d, want 3", len(blocks))
	}
	if blocks[0].X != 72 || blocks[0].Width != 451 {
		t.Errorf("span: got X=%v W=%v, want 72 451", blocks[0].X, blocks[0].Width)
	}
	if blocks[1].X != 72 || blocks[2].X != 297.5 {
		t.Errorf("row 2: got %v %v, want 72 297.5", blocks[1].X, blocks[2].X)
	}
}

func TestLayout_TableRowspan_ReservesRows(t *testing.T) {
	root, err := boxes.Build(`<table style="width:451pt"><tr><td rowspan="2">A</td><td>B</td></tr><tr><td>C</td></tr></table>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 3 {
		t.Fatalf("blocks: got %d, want 3", len(blocks))
	}
	// C is placed in column 1 (To occupies column 0).
	if blocks[2].X != 297.5 {
		t.Errorf("C: got X=%v, want 297.5", blocks[2].X)
	}
	if blocks[2].Y <= blocks[0].Y {
		t.Errorf("C must be in row 2: got %v after %v", blocks[2].Y, blocks[0].Y)
	}
}

func TestLayout_TableRowspan_CentersContentAcrossRows(t *testing.T) {
	root, err := boxes.Build(`<table style="width:451pt"><tr><td rowspan="2">A</td><td>one</td></tr><tr><td>` + strings.Repeat("word ", 70) + `</td></tr></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 3 {
		t.Fatalf("blocks: got %d, want 3", len(blocks))
	}
	cell := blocks[0]
	if len(cell.Runs) != 1 || len(cell.Runs[0]) != 1 {
		t.Fatalf("runs of rowspan: %#v", cell.Runs)
	}
	wantRowsHeight := blocks[1].Height + blocks[2].Height
	if blocks[2].Height <= blocks[1].Height {
		t.Fatalf("fixture must produce a taller second row: %v <= %v", blocks[2].Height, blocks[1].Height)
	}
	wantY := cell.Y + (wantRowsHeight-cell.Runs[0][0].LineH)/2
	if cell.Runs[0][0].Y != wantY {
		t.Errorf("rowspan text y=%v, want it centered at %v (combined height %v)", cell.Runs[0][0].Y, wantY, wantRowsHeight)
	}
	if cell.Height != wantRowsHeight {
		t.Errorf("height rowspan: got %v, want %v", cell.Height, wantRowsHeight)
	}
}
