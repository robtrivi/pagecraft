package boxes

import (
	"testing"
)

// Padding step 1: shorthand accepts one to four values, like margin.
func TestBoxes_ParsesPadding(t *testing.T) {
	root, err := Build(`<p style="padding: 5px 10px">x</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := root.Children[0]
	if b.Style.PaddingTop != 3.75 || b.Style.PaddingRight != 7.5 || b.Style.PaddingBottom != 3.75 || b.Style.PaddingLeft != 7.5 {
		t.Errorf("padding: got %v %v %v %v, want 3.75 7.5 3.75 7.5",
			b.Style.PaddingTop, b.Style.PaddingRight, b.Style.PaddingBottom, b.Style.PaddingLeft)
	}
}

func TestBoxes_NoPadding_Defaults(t *testing.T) {
	root, err := Build(`<p>x</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := root.Children[0]
	if b.Style.PaddingTop+b.Style.PaddingRight+b.Style.PaddingBottom+b.Style.PaddingLeft != 0 {
		t.Error("padding default must be 0")
	}
}
