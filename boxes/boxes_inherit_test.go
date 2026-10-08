package boxes

import (
	"testing"
)

// Inheritance step 1: color and font-size inherit; the element's style takes precedence.
// Not stylesheet (<style>/classes) yet; only the inline style chain is covered.
func TestBoxes_InheritsColorAndFontSize(t *testing.T) {
	root, err := Build(`<div style="color: red; font-size: 20px"><p>x</p></div>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	p := root.Children[0].Children[0]
	if !p.Style.HasColor || p.Style.Color != [4]float64{1, 0, 0, 1} {
		t.Errorf("inherited color: got %v %v, want true [1 0 0]", p.Style.HasColor, p.Style.Color)
	}
	if p.Style.FontSize != 15 {
		t.Errorf("inherited font-size: got %v, want 15", p.Style.FontSize)
	}
}

func TestBoxes_FontShorthandInheritsSizeFamilyAndStyle(t *testing.T) {
	root, err := Build(`<div style="font: italic bold 10pt Helvetica"><p>x</p></div>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	parent, child := root.Children[0], root.Children[0].Children[0]
	if parent.Style.FontSize != 10 || parent.Style.FontFamily != "Helvetica" || !parent.Style.Bold || !parent.Style.Italic {
		t.Errorf("shorthand of the parent: size=%v family=%q bold=%v italic=%v", parent.Style.FontSize, parent.Style.FontFamily, parent.Style.Bold, parent.Style.Italic)
	}
	if child.Style.FontSize != 10 || child.Style.FontFamily != "Helvetica" || !child.Style.Bold || !child.Style.Italic {
		t.Errorf("inherited shorthand: size=%v family=%q bold=%v italic=%v", child.Style.FontSize, child.Style.FontFamily, child.Style.Bold, child.Style.Italic)
	}
}

func TestBoxes_BodyFontShorthandInheritsIntoChildren(t *testing.T) {
	root, err := Build(`<!doctype html><html><head><style>body { font: 10pt Helvetica; }</style></head><body><p>x</p></body></html>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	p := root.Children[0]
	if p.Style.FontSize != 10 || p.Style.FontFamily != "Helvetica" {
		t.Errorf("font inherited from body: size=%v family=%q, want 10 Helvetica", p.Style.FontSize, p.Style.FontFamily)
	}
}

func TestBoxes_OwnStyle_OverridesInherited(t *testing.T) {
	root, err := Build(`<div style="color: red; font-size: 20px"><p style="color: blue">x</p></div>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	p := root.Children[0].Children[0]
	if !p.Style.HasColor || p.Style.Color != [4]float64{0, 0, 1, 1} {
		t.Errorf("color own: got %v %v, want true [0 0 1]", p.Style.HasColor, p.Style.Color)
	}
	if p.Style.FontSize != 15 {
		t.Errorf("inherited font-size: got %v, want 15", p.Style.FontSize)
	}
}

func TestBoxes_NonInheritable_NotPassedDown(t *testing.T) {
	root, err := Build(`<div style="margin: 10px; display: flex"><p>x</p></div>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	p := root.Children[0].Children[0]
	if p.Style.Display != DisplayBlock {
		t.Errorf("display was not inherited: got %q", p.Style.Display)
	}
	if p.Style.MarginTop != 12 {
		t.Errorf("the UA margin for p applies without inheriting the parent's margin: got %v, want 12", p.Style.MarginTop)
	}
}
