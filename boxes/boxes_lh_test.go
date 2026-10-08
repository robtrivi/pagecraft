package boxes

import (
	"testing"
)

// Line-height step 1: a number (multiplier) or px/pt length.
// Inherited, like color/font-size.
func TestBoxes_ParsesLineHeight(t *testing.T) {
	root, err := Build(`<p style="line-height: 2">x</p><p style="line-height: 24px">y</p><p>z</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := root.Children[0].Style.LineHeight; got != 2 {
		t.Errorf("factor: got %v, want 2", got)
	}
	b := root.Children[1]
	if b.Style.LineHeightPt != 18 {
		t.Errorf("absolute: got %v, want 18", b.Style.LineHeightPt)
	}
	if root.Children[2].Style.LineHeight != 0 {
		t.Error("default should be 0 (resolved during layout)")
	}
}

func TestBoxes_LineHeight_Inherited(t *testing.T) {
	root, err := Build(`<div style="line-height: 2"><p>x</p></div>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := root.Children[0].Children[0].Style.LineHeight; got != 2 {
		t.Errorf("inherited: got %v, want 2", got)
	}
}
