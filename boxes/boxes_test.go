package boxes

import (
	"testing"
)

// Build a block box from a paragraph with inline style
// inline styles and margins computed from px values.
func TestBoxes_BlockWithMargin_ComputesBox(t *testing.T) {
	root, err := Build(`<p style="display: block; margin: 10px">Hello</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(root.Children) != 1 {
		t.Fatalf("expected 1 child box, got %d", len(root.Children))
	}
	b := root.Children[0]
	if b.Style.Display != DisplayBlock {
		t.Errorf("display: got %q, want %q", b.Style.Display, "block")
	}
	for _, m := range []float64{b.Style.MarginTop, b.Style.MarginRight, b.Style.MarginBottom, b.Style.MarginLeft} {
		if m != 7.5 {
			t.Errorf("margin: got %v, want 7.5", m)
		}
	}
	if b.Text != "Hello" {
		t.Errorf("text: got %q, want %q", b.Text, "Hello")
	}
}

func TestBoxes_MarginShorthand_Expands(t *testing.T) {
	cases := []struct {
		name                     string
		in                       string
		top, right, bottom, left float64
	}{
		{name: "1 value", in: "10px", top: 7.5, right: 7.5, bottom: 7.5, left: 7.5},
		{name: "2 values", in: "5px 10px", top: 3.75, right: 7.5, bottom: 3.75, left: 7.5},
		{name: "3 values", in: "1px 2px 3px", top: 0.75, right: 1.5, bottom: 2.25, left: 1.5},
		{name: "4 values", in: "1px 2px 3px 4px", top: 0.75, right: 1.5, bottom: 2.25, left: 3},
		{name: "zero without unit", in: "0", top: 0, right: 0, bottom: 0, left: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, err := Build(`<p style="margin: ` + tc.in + `">x</p>`)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			b := root.Children[0]
			if b.Style.MarginTop != tc.top || b.Style.MarginRight != tc.right ||
				b.Style.MarginBottom != tc.bottom || b.Style.MarginLeft != tc.left {
				t.Errorf("got %v %v %v %v, want %v %v %v %v",
					b.Style.MarginTop, b.Style.MarginRight, b.Style.MarginBottom, b.Style.MarginLeft,
					tc.top, tc.right, tc.bottom, tc.left)
			}
		})
	}
}
