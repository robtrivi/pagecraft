package boxes

import (
	"testing"
)

// Page step 2: @page reaches the root (size and margin); without @page,
// zero values (render uses its defaults).
func TestBoxes_PageRule_ReachesRoot(t *testing.T) {
	root, err := Build(`<style>@page { size: A4 landscape; margin: 2cm }</style><p>x</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root.Page.Width != 842 || root.Page.Height != 595 {
		t.Errorf("size: got %vx%v, want 842x595", root.Page.Width, root.Page.Height)
	}
	if diff := root.Page.Margin - 2*72/2.54; diff < -1e-9 || diff > 1e-9 {
		t.Errorf("margin: got %v", root.Page.Margin)
	}
}

func TestBoxes_NoPageRule_Zero(t *testing.T) {
	root, err := Build(`<p>x</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root.Page != (PageConfig{}) {
		t.Errorf("without @page: got %+v", root.Page)
	}
}
