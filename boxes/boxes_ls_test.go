package boxes

import (
	"testing"
)

// Letter spacing in px or pt defaults to zero.
func TestBoxes_ParsesLetterSpacing(t *testing.T) {
	root, err := Build(`<p style="letter-spacing: 2px">x</p><p style="letter-spacing: normal">y</p><p>z</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := root.Children[0].Style.LetterSpacing; got != 1.5 {
		t.Errorf("got %v, want 1.5", got)
	}
	if got := root.Children[1].Style.LetterSpacing; got != 0 {
		t.Errorf("normal: got %v, want 0", got)
	}
	if got := root.Children[2].Style.LetterSpacing; got != 0 {
		t.Errorf("default: got %v, want 0", got)
	}
}
