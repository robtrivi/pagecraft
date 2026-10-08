package boxes

import (
	"testing"
)

// Decoration step 1: underline and line-through can be combined; neither or
// absent means not decoration. <u>/<s>/<strike> provide defaults.
func TestBoxes_ParsesDecoration(t *testing.T) {
	root, err := Build(`<p style="text-decoration: underline line-through">x</p><p style="text-decoration: none">y</p><p>z</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := root.Children[0]
	if !b.Style.Underline || !b.Style.Strike {
		t.Errorf("got underline=%v strike=%v, want true true", b.Style.Underline, b.Style.Strike)
	}
	if root.Children[1].Style.Underline || root.Children[1].Style.Strike {
		t.Error("none must disable decorations")
	}
	if root.Children[2].Style.Underline || root.Children[2].Style.Strike {
		t.Error("default must be false")
	}
}

func TestBoxes_DecorationTags_Default(t *testing.T) {
	root, err := Build(`<u>x</u><s>y</s>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !root.Children[0].Style.Underline {
		t.Error("<u> must underline")
	}
	if !root.Children[1].Style.Strike {
		t.Error("<s> must strike through")
	}
}

func TestBoxes_TextDecorationPropagatesThroughDescendants(t *testing.T) {
	cases := []struct {
		name, html    string
		wantUnderline bool
		wantStrike    bool
	}{
		{"underline", `<div style="text-decoration: underline"><p>x</p></div>`, true, false},
		{"strike_survives_child_none", `<div style="text-decoration: line-through"><p style="text-decoration: none">x</p></div>`, false, true},
		{"child_decoration_adds_to_parent", `<div style="text-decoration: underline"><p style="text-decoration: line-through">x</p></div>`, true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, err := Build(tc.html)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			p := root.Children[0].Children[0]
			if p.Style.Underline != tc.wantUnderline || p.Style.Strike != tc.wantStrike {
				t.Errorf("decoration efectiva underline/strike=%v/%v, want %v/%v", p.Style.Underline, p.Style.Strike, tc.wantUnderline, tc.wantStrike)
			}
		})
	}
}
