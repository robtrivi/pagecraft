package boxes

import (
	"testing"
)

// Opacity step 1: values from 0..1 with a flag (explicit 0 differs from unset).
func TestBoxes_ParsesOpacity(t *testing.T) {
	root, err := Build(`<p style="opacity: 0.5">x</p><p style="opacity: 0">y</p><p>z</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := root.Children[0]
	if !b.Style.HasOpacity || b.Style.Opacity != 0.5 {
		t.Errorf("got %v %v, want true 0.5", b.Style.HasOpacity, b.Style.Opacity)
	}
	if z := root.Children[1]; !z.Style.HasOpacity || z.Style.Opacity != 0 {
		t.Errorf("explicit zero: got %v %v", z.Style.HasOpacity, z.Style.Opacity)
	}
	if root.Children[2].Style.HasOpacity {
		t.Error("default is not marked")
	}
}

func TestBoxes_OpacityInvalid_Ignored(t *testing.T) {
	root, err := Build(`<p style="opacity: 2">x</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root.Children[0].Style.HasOpacity {
		t.Error("out-of-range value is ignored")
	}
}
