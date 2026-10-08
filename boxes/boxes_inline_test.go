package boxes

import (
	"testing"
)

// Inline step 1: the box preserves anonymous-text/inline order
// ("Hello " + <b> + " end"). The box text summary remains intact.
func TestBoxes_PreservesInlineOrder(t *testing.T) {
	root, err := Build(`<p>Hello <b>world</b> end</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	p := root.Children[0]
	if len(p.Parts) != 3 {
		t.Fatalf("partes: got %d, want 3", len(p.Parts))
	}
	if p.Parts[0].Child != nil || p.Parts[0].Text != "Hello " {
		t.Errorf("parte 0: got %+v", p.Parts[0])
	}
	if p.Parts[1].Child == nil || p.Parts[1].Child.Tag != "b" {
		t.Errorf("parte 1: got %+v", p.Parts[1])
	}
	if p.Parts[2].Child != nil || p.Parts[2].Text != " end" {
		t.Errorf("parte 2: got %+v", p.Parts[2])
	}
	if p.Text != "Hello world end" {
		t.Errorf("text: got %q", p.Text)
	}
	if p.Children[0].Style.Bold != true {
		t.Error("the <b> child must follow parsed text")
	}
}

func TestBoxes_StrongUsesElementDefaultOverInheritedNormalWeight(t *testing.T) {
	tests := []struct {
		name string
		html string
		want bool
	}{
		{name: "semantic weight", html: `<div style="font-weight:normal"><strong>x</strong></div>`, want: true},
		{name: "child author override", html: `<div style="font-weight:normal"><strong style="font-weight:normal">x</strong></div>`, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root, err := Build(test.html)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			strong := root.Children[0].Children[0]
			if strong.Style.Bold != test.want {
				t.Errorf("strong.Bold=%v, want %v", strong.Style.Bold, test.want)
			}
		})
	}
}
