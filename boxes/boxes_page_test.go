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

func TestBoxes_PageRulesRetainSelectorsForLayoutResolution(t *testing.T) {
	root, err := Build(`<style>@page { margin: 10pt } @page :first { margin-top: 20pt } @page report:left { margin-left: 30pt }</style><p>x</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(root.PageRules) != 3 {
		t.Fatalf("got %d page rules, want 3", len(root.PageRules))
	}
	if root.PageRules[2].Selector.Name != "report" || root.PageRules[2].Selector.Pseudo != "left" {
		t.Fatalf("selector was discarded: %+v", root.PageRules[2].Selector)
	}
}

func TestBoxes_PagePrintDescriptorsReachRootConfiguration(t *testing.T) {
	root, err := Build(`<style>@page { bleed: 3mm; marks: crop cross }</style><p>x</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !root.Page.HasBleed || root.Page.Bleed-3*72/25.4 > 1e-9 || 3*72/25.4-root.Page.Bleed > 1e-9 {
		t.Errorf("bleed: %+v", root.Page)
	}
	if !root.Page.HasMarks || root.Page.Marks != 3 {
		t.Errorf("marks: %+v", root.Page)
	}
}
