package boxes

import (
	"testing"
)

// Transform step 1: uppercase/lowercase/capitalize, inherited.
// All downstream operations (measurement, drawing, extraction) inherit it from the text.
func TestBoxes_TextTransform(t *testing.T) {
	root, err := Build(`<p style="text-transform: uppercase">Hello World</p><p style="text-transform: lowercase">Hello World</p><p style="text-transform: capitalize">hello world</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"HELLO WORLD", "hello world", "Hello World"}
	for i, w := range want {
		if got := root.Children[i].Text; got != w {
			t.Errorf("child %d: got %q, want %q", i, got, w)
		}
	}
}

func TestBoxes_TextTransform_Inherited(t *testing.T) {
	root, err := Build(`<div style="text-transform: uppercase"><p>Hello <b>world</b></p></div>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	p := root.Children[0].Children[0]
	if p.Text != "HELLO WORLD" {
		t.Errorf("inherited: got %q", p.Text)
	}
}
