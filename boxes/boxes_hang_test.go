package boxes

import (
	"testing"
)

// Hanging step 1: outside (spec default) versus explicit inside.
func TestBoxes_ListPosition(t *testing.T) {
	root, err := Build(`<ul><li>a</li></ul><ul style="list-style-position: inside"><li>b</li></ul>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	first := root.Children[0].Children[0]
	second := root.Children[1].Children[0]
	if first.Style.ListInside {
		t.Error("default must be outside")
	}
	if !second.Style.ListInside {
		t.Error("inside explicit must be marked")
	}
}
