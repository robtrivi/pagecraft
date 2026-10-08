package boxes

import (
	"testing"
)

// Bold/italic step 1: font-weight bold/700+ and font-style italic/oblique;
// <b>/<strong> and <i>/<em> provide presentational defaults.
func TestBoxes_ParsesBoldItalic(t *testing.T) {
	root, err := Build(`<p style="font-weight: bold; font-style: italic">x</p><p style="font-weight: 700">y</p><p>z</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := root.Children[0]
	if !b.Style.Bold || !b.Style.Italic {
		t.Errorf("bold+italic: got %v %v, want true true", b.Style.Bold, b.Style.Italic)
	}
	if !root.Children[1].Style.Bold {
		t.Error("700 must be bold")
	}
	if root.Children[2].Style.Bold || root.Children[2].Style.Italic {
		t.Error("defaults must be false")
	}
}

func TestBoxes_BoldItalicTags_Default(t *testing.T) {
	root, err := Build(`<b>x</b><i>y</i>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !root.Children[0].Style.Bold {
		t.Error("<b> must be bold")
	}
	if !root.Children[1].Style.Italic {
		t.Error("<i> must be italic")
	}
}
