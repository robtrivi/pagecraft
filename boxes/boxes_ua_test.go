package boxes

import (
	"math"
	"testing"
)

// Heading levels h1 through h6 are bold and scaled by default; author styles take precedence.
// no UA margins in v1 (documented to avoid changing existing pagination).
func TestBoxes_UA_HeadingsScaled(t *testing.T) {
	root, err := Build(`<h1>a</h1><h2>b</h2><h4>c</h4><p>d</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	kids := root.Children
	if kids[0].Style.FontSize != 18 || !kids[0].Style.Bold {
		t.Errorf("h1: got %v bold=%v, want 18 true", kids[0].Style.FontSize, kids[0].Style.Bold)
	}
	if kids[1].Style.FontSize != 13.5 || !kids[1].Style.Bold {
		t.Errorf("h2: got %v bold=%v, want 13.5 true", kids[1].Style.FontSize, kids[1].Style.Bold)
	}
	if kids[2].Style.FontSize != 9 || !kids[2].Style.Bold {
		t.Errorf("h4: got %v bold=%v, want 9 true", kids[2].Style.FontSize, kids[2].Style.Bold)
	}
	if kids[3].Style.FontSize != 0 || kids[3].Style.Bold {
		t.Errorf("p: got %v bold=%v, want 0 false", kids[3].Style.FontSize, kids[3].Style.Bold)
	}
}

func TestBoxes_FontSizeRelativeKeywordsAndPercentagesUseInheritedSize(t *testing.T) {
	root, err := Build(`<div style="font-size:9pt"><small>UA</small><span style="font-size:smaller">SMALLER</span><i style="font-size:80%">PERCENT</i><b style="font-size:larger">LARGER</b><small style="font-size:11pt">OVERRIDE</small></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	got := make([]float64, len(root.Children[0].Children))
	for index, child := range root.Children[0].Children {
		got[index] = child.Style.FontSize
	}
	want := []float64{7.5, 7.5, 7.2, 10.8, 11}
	for index := range want {
		if math.Abs(got[index]-want[index]) > 0.001 {
			t.Errorf("%s font-size=%.3fpt, want %.3fpt", root.Children[0].Children[index].Tag, got[index], want[index])
		}
	}
}

func TestBoxes_UA_ParagraphMarginsScaleAndCanBeOverridden(t *testing.T) {
	cases := []struct {
		name, html string
		want       float64
	}{
		{"default", `<p>x</p>`, 12},
		{"relative_to_font_size", `<p style="font-size: 20px">x</p>`, 15},
		{"author_override", `<style>p { margin: 0 }</style><p>x</p>`, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, err := Build(tc.html)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			p := root.Children[0]
			if p.Style.MarginTop != tc.want || p.Style.MarginBottom != tc.want {
				t.Errorf("margins p=%v/%v, want %v/%v pt", p.Style.MarginTop, p.Style.MarginBottom, tc.want, tc.want)
			}
		})
	}
}

func TestBoxes_Author_BeatsUA(t *testing.T) {
	root, err := Build(`<style>h1 { font-size: 10px; font-weight: normal }</style><h1>x</h1>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := root.Children[0]
	if b.Style.FontSize != 7.5 || b.Style.Bold {
		t.Errorf("autor manda: got %v bold=%v", b.Style.FontSize, b.Style.Bold)
	}
}

func TestBoxes_UA_TableHeaderIsBold(t *testing.T) {
	root, err := Build(`<table><tr><th>Summary</th></tr></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	th := root.Children[0].Children[0].Children[0].Children[0]
	if !th.Style.Bold {
		t.Fatal("th must inherit font-weight: bold of the UA stylesheet")
	}
}

func TestBoxes_UA_TableCellsHaveOnePixelPaddingAndAllowOverride(t *testing.T) {
	root, err := Build(`<table><tr><td>default</td><th>header</th><td style="padding:0">reset</td></tr></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	cells := root.Children[0].Children[0].Children[0].Children
	for _, cell := range cells[:2] {
		if cell.Style.PaddingTop != 0.75 || cell.Style.PaddingRight != 0.75 || cell.Style.PaddingBottom != 0.75 || cell.Style.PaddingLeft != 0.75 {
			t.Errorf("cell %s default padding=%g/%g/%g/%gpt, want 0.75pt on every side", cell.Tag,
				cell.Style.PaddingTop, cell.Style.PaddingRight, cell.Style.PaddingBottom, cell.Style.PaddingLeft)
		}
	}
	reset := cells[2]
	if reset.Style.PaddingTop != 0 || reset.Style.PaddingRight != 0 || reset.Style.PaddingBottom != 0 || reset.Style.PaddingLeft != 0 {
		t.Errorf("author padding:0 did not override UA padding: %+v", reset.Style)
	}
}

func TestBoxes_TableCellPaddingComesFromContainingTable(t *testing.T) {
	root, err := Build(`<table cellpadding="4"><tr><td>table padding</td><td style="padding:2px">author override</td></tr></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	cells := root.Children[0].Children[0].Children[0].Children
	for side, got := range map[string]float64{
		"top": cells[0].Style.PaddingTop, "right": cells[0].Style.PaddingRight,
		"bottom": cells[0].Style.PaddingBottom, "left": cells[0].Style.PaddingLeft,
	} {
		if got != 3 {
			t.Errorf("table cellpadding=4 gives %s padding %gpt, want 3pt", side, got)
		}
	}
	for side, got := range map[string]float64{
		"top": cells[1].Style.PaddingTop, "right": cells[1].Style.PaddingRight,
		"bottom": cells[1].Style.PaddingBottom, "left": cells[1].Style.PaddingLeft,
	} {
		if got != 1.5 {
			t.Errorf("author padding overrides table cellpadding on %s: got %gpt, want 1.5pt", side, got)
		}
	}
}

func TestBoxes_UA_BodyMarginCanBeReset(t *testing.T) {
	withDefault, err := Build(`<html><body><p>text</p></body></html>`)
	if err != nil {
		t.Fatalf("Build default: %v", err)
	}
	if withDefault.Style.BodyMargin != 6 {
		t.Errorf("margin UA body: got %v pt, want 6", withDefault.Style.BodyMargin)
	}
	reset, err := Build(`<html><head><style>body { margin: 0 }</style></head><body><p>text</p></body></html>`)
	if err != nil {
		t.Fatalf("Build reset: %v", err)
	}
	if reset.Style.BodyMargin != 0 {
		t.Errorf("body margin:0 must remove margin, got %v", reset.Style.BodyMargin)
	}
}

func TestBoxes_UA_BodyMarginAppliesToImplicitBodyInDocument(t *testing.T) {
	root, err := Build(`<!doctype html><html><head><title>doc</title></head><p>text</p></html>`)
	if err != nil {
		t.Fatalf("Build document: %v", err)
	}
	if root.Style.BodyMargin != 6 {
		t.Errorf("implicit body UA margin: got %v pt, want 6", root.Style.BodyMargin)
	}
}

func TestBoxes_FragmentDoesNotCreateBodyMargin(t *testing.T) {
	root, err := Build(`<p>fragment</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if root.Style.BodyMargin != 0 {
		t.Errorf("fragment without body explicit must preserve zero margin, got %v", root.Style.BodyMargin)
	}
}

func TestBoxes_UA_HeadingMarginsScaleWithFontAndAuthorOverrides(t *testing.T) {
	root, err := Build(`<style>h2 { margin: 0 }</style><h1>one</h1><h2>two</h2>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	h1, h2 := root.Children[0], root.Children[1]
	if h1.Style.MarginTop != 12.06 || h1.Style.MarginBottom != 12.06 {
		t.Errorf("h1 margins: got %v/%v pt, want 12.06/12.06", h1.Style.MarginTop, h1.Style.MarginBottom)
	}
	if h2.Style.MarginTop != 0 || h2.Style.MarginBottom != 0 {
		t.Errorf("author CSS must be able to remove margins: got %v/%v", h2.Style.MarginTop, h2.Style.MarginBottom)
	}
}
