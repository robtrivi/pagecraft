package boxes

import (
	"testing"
)

// Text styling step 1: font-size in px/pt and color. Not inheritance,
// without relative units (%/em) or named sizes (v1).
func TestBoxes_ParsesFontSizeAndColor(t *testing.T) {
	root, err := Build(`<p style="font-size: 20px; color: blue">x</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := root.Children[0]
	if b.Style.FontSize != 15 {
		t.Errorf("font-size: got %v, want 15", b.Style.FontSize)
	}
	if !b.Style.HasColor || b.Style.Color != [4]float64{0, 0, 1, 1} {
		t.Errorf("color: got %v %v, want true [0 0 1]", b.Style.HasColor, b.Style.Color)
	}
}

func TestBoxes_FontSizePtAndDefaults(t *testing.T) {
	root, err := Build(`<p style="font-size: 15pt">x</p><p>y</p><p style="font-size: larger">z</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := root.Children[0].Style.FontSize; got != 15 {
		t.Errorf("pt: got %v, want 15", got)
	}
	if got := root.Children[1].Style.FontSize; got != 0 {
		t.Errorf("default: got %v, want 0 (resolved during layout)", got)
	}
	if got := root.Children[2].Style.FontSize; got != 14.4 {
		t.Errorf("larger: got %v, want 14.4pt relative to default 12pt", got)
	}
}
