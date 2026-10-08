package boxes

import (
	"testing"
)

// Per-side borders step 1: border-top/etc. use the same grammar; the
// shorthand sets all four sides, and a specific side takes precedence.
func TestBoxes_ParsesSideBorders(t *testing.T) {
	root, err := Build(`<p style="border: 1px solid black; border-bottom: 3px solid red">x</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := root.Children[0]
	if b.Style.BorderTop.Width != 0.75 || b.Style.BorderLeft.Width != 0.75 {
		t.Errorf("sides not set: top=%v left=%v", b.Style.BorderTop.Width, b.Style.BorderLeft.Width)
	}
	if b.Style.BorderBottom.Width != 2.25 || b.Style.BorderBottom.Color != [4]float64{1, 0, 0, 1} {
		t.Errorf("bottom: got %v %v, want 2.25 [1 0 0]", b.Style.BorderBottom.Width, b.Style.BorderBottom.Color)
	}
}

func TestBoxes_SideBorderAlone(t *testing.T) {
	root, err := Build(`<p style="border-top: 2px dashed blue">x</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := root.Children[0]
	if b.Style.BorderTop.Width != 1.5 {
		t.Errorf("top: got %v, want 2", b.Style.BorderTop.Width)
	}
	if b.Style.BorderBottom.Width != 0 || b.Style.BorderWidth != 0 {
		t.Errorf("others must be 0: bottom=%v uniform=%v", b.Style.BorderBottom.Width, b.Style.BorderWidth)
	}
}
