package boxes

import (
	"testing"
)

// White-space step 1: pre/pre-wrap/pre-line/nowrap; other values become "".
// Inherited per spec.
func TestBoxes_ParsesWhiteSpace(t *testing.T) {
	root, err := Build(`<p style="white-space: pre">x</p><p style="white-space: nowrap">y</p><p style="white-space: pre-line">z</p><p>w</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"pre", "nowrap", "pre-line", ""}
	for i, w := range want {
		if got := root.Children[i].Style.WhiteSpace; got != w {
			t.Errorf("child %d: got %q, want %q", i, got, w)
		}
	}
}

func TestBoxes_WhiteSpace_Inherited(t *testing.T) {
	root, err := Build(`<div style="white-space: pre"><p>x</p></div>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := root.Children[0].Children[0].Style.WhiteSpace; got != "pre" {
		t.Errorf("inherited: got %q, want pre", got)
	}
}

func TestBoxes_PreTag_DefaultsToPre(t *testing.T) {
	root, err := Build(`<pre>x</pre>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := root.Children[0].Style.WhiteSpace; got != "pre" {
		t.Errorf("tag pre: got %q, want pre", got)
	}
}
