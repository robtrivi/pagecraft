package boxes

import (
	"testing"
)

// VAlign step 1: sub/super/baseline; <sub>/<sup> provide defaults.
func TestBoxes_ParsesVAlign(t *testing.T) {
	root, err := Build(`<p style="vertical-align: super">x</p><p>y</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := root.Children[0].Style.VAlign; got != "super" {
		t.Errorf("got %q, want super", got)
	}
	if got := root.Children[1].Style.VAlign; got != "" {
		t.Errorf("default: got %q, want empty", got)
	}
}

func TestBoxes_SubSupTags_Default(t *testing.T) {
	root, err := Build(`<sub>x</sub><sup>y</sup>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := root.Children[0].Style.VAlign; got != "sub" {
		t.Errorf("sub: got %q", got)
	}
	if got := root.Children[1].Style.VAlign; got != "super" {
		t.Errorf("sup: got %q", got)
	}
}

func TestBoxes_TableCellsDefaultMiddleAndAuthorCanOverride(t *testing.T) {
	root, err := Build(`<style>td { vertical-align: top }</style><table><tr><td>A</td><th>B</th></tr></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	td := root.Children[0].Children[0].Children[0].Children[0]
	th := root.Children[0].Children[0].Children[0].Children[1]
	if td.Style.VAlign != "top" {
		t.Errorf("author vertical-align must take precedence: td=%q", td.Style.VAlign)
	}
	if th.Style.VAlign != "middle" {
		t.Errorf("th must use vertical-align: middle from the UA stylesheet: %q", th.Style.VAlign)
	}
}
