package boxes

import (
	"testing"
)

// Cascade step 2: <style> with classes/IDs, specificity, and source order;
// inline styles override stylesheet rules.
func TestBoxes_Stylesheet_AppliesByClass(t *testing.T) {
	root, err := Build(`<html><head><style>.a { color: blue; }</style></head><body><p class="a">x</p></body></html>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := root.Children[0]
	if !b.Style.HasColor || b.Style.Color != [4]float64{0, 0, 1, 1} {
		t.Errorf("color: got %v %v, want true [0 0 1]", b.Style.HasColor, b.Style.Color)
	}
}

func TestBoxes_Specificity_IDBeatsClass(t *testing.T) {
	root, err := Build(`<style>.a { color: blue; } #i { color: red; }</style><p id="i" class="a">x</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c := root.Children[0].Style.Color; c != [4]float64{1, 0, 0, 1} {
		t.Errorf("color: got %v, want [1 0 0]", c)
	}
}

func TestBoxes_Inline_BeatsStylesheet(t *testing.T) {
	root, err := Build(`<style>p { color: red; }</style><p style="color: green">x</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c := root.Children[0].Style.Color; c != [4]float64{0, 0.5, 0, 1} {
		t.Errorf("color: got %v, want green [0 0.5 0]", c)
	}
}
