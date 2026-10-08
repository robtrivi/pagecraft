package boxes

import "testing"

func TestBoxes_ParsesOverflowHidden(t *testing.T) {
	root, err := Build(`<div style="overflow:hidden">content</div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if got := root.Children[0].Style.Overflow; got != OverflowHidden {
		t.Fatalf("overflow=%q, want hidden", got)
	}
}

func TestBoxes_ParsesOverflowAxis(t *testing.T) {
	root, err := Build(`<div style="overflow-x:hidden;overflow-y:visible">content</div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	box := root.Children[0]
	if box.Style.OverflowX != OverflowHidden || box.Style.OverflowY != OverflowVisible {
		t.Fatalf("overflow axes = %q/%q, want hidden/visible", box.Style.OverflowX, box.Style.OverflowY)
	}
}

func TestBoxes_OverflowAxisOverridesShorthand(t *testing.T) {
	root, err := Build(`<div style="overflow:hidden;overflow-x:visible">content</div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	box := root.Children[0]
	if box.Style.Overflow != OverflowHidden || box.Style.OverflowX != OverflowVisible || box.Style.OverflowY != OverflowHidden {
		t.Fatalf("overflow=%q axes=%q/%q, want hidden shorthand with visible/hidden axes", box.Style.Overflow, box.Style.OverflowX, box.Style.OverflowY)
	}
}
