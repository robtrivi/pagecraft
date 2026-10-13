package boxes

import (
	"testing"
)

// Fixed step 1: position:fixed with top/left (layout repeats it on every page).
func TestBoxes_ParsesFixed(t *testing.T) {
	root, err := Build(`<div style="position: fixed; top: 300px; left: 100px">x</div>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := root.Children[0]
	if b.Style.Position != PositionFixed {
		t.Errorf("position: got %q, want fixed", b.Style.Position)
	}
	// 300px equals 225pt and 100px equals 75pt at a 0.75 scale.
	if b.Style.Top != 225 || b.Style.Left != 75 {
		t.Errorf("offsets: got %v %v, want 225 75", b.Style.Top, b.Style.Left)
	}
}

func TestBoxes_ParsesRelativePosition(t *testing.T) {
	root, err := Build(`<div style="position: relative">x</div>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := root.Children[0].Style.Position; got != PositionRelative {
		t.Errorf("position: got %q, want relative", got)
	}
}

func TestBoxes_ParsesColumnSpanAll(t *testing.T) {
	root, err := Build(`<div style="column-count:3"><h2 style="column-span:all">Summary</h2></div>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !root.Children[0].Children[0].Style.ColumnSpanAll {
		t.Error("column-span: all was not preserved")
	}
}

func TestBoxes_ParsesPositionedDimensionsAndRight(t *testing.T) {
	root, err := Build(`<div style="position: absolute; right: 10pt; width: 130pt; height: 80px">x</div>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := root.Children[0]
	if b.Style.Right != 10 || !b.Style.HasRight {
		t.Errorf("right: got %v (set=%v), want 10pt", b.Style.Right, b.Style.HasRight)
	}
	if b.Style.Sizing.Width.Value != 130 || b.Style.Sizing.Height.Value != 60 {
		t.Errorf("dimensions: got %vx%v, want 130x60 pt", b.Style.Sizing.Width.Value, b.Style.Sizing.Height.Value)
	}
}

func TestBoxes_AbsolutePositionTracksExplicitTopOffset(t *testing.T) {
	root, err := Build(`<div style="position:absolute;top:0">Stamp</div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !root.Children[0].Style.HasTop {
		t.Fatal("explicit top:0 must be distinguishable from an unset top offset")
	}
}
