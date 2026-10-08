package boxes

import (
	"math"
	"testing"
)

// Longhands step 1: margin-top/... and padding-top/... override their
// shorthand (regardless of order in v1, as documented).
func TestBoxes_MarginPaddingLonghands(t *testing.T) {
	for _, tc := range []struct {
		name, declarations string
	}{
		{"longhands after shorthands", "margin: 0; margin-top: 5px; padding: 2px; padding-left: 9px"},
		{"longhands before shorthands", "margin-top: 5px; margin: 0; padding-left: 9px; padding: 2px"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, err := Build(`<p style="` + tc.declarations + `">x</p>`)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			b := root.Children[0]
			if b.Style.MarginTop != 3.75 || b.Style.MarginRight != 0 || b.Style.MarginBottom != 0 || b.Style.MarginLeft != 0 {
				t.Errorf("margin: got %v %v %v %v, want 3.75 0 0 0",
					b.Style.MarginTop, b.Style.MarginRight, b.Style.MarginBottom, b.Style.MarginLeft)
			}
			if b.Style.PaddingTop != 1.5 || b.Style.PaddingRight != 1.5 || b.Style.PaddingBottom != 1.5 || b.Style.PaddingLeft != 6.75 {
				t.Errorf("padding: got %v %v %v %v, want 1.5 1.5 1.5 6.75",
					b.Style.PaddingTop, b.Style.PaddingRight, b.Style.PaddingBottom, b.Style.PaddingLeft)
			}
		})
	}
}

func TestBoxes_LonghandInvalid_Ignored(t *testing.T) {
	root, err := Build(`<p style="margin: 4px; margin-top: bogus">x</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := root.Children[0].Style.MarginTop; got != 3 {
		t.Errorf("invalid is ignored: got %v, want 3", got)
	}
}

func TestBoxes_NegativeMarginIsParsed(t *testing.T) {
	root, err := Build(`<p style="margin-top: -10px">x</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if got := root.Children[0].Style.MarginTop; got != -7.5 {
		t.Errorf("negative margin-top: got %v pt, want -7.5", got)
	}
}

func TestBoxes_CollapsedMarginsUsePositiveAndNegativeExtremes(t *testing.T) {
	// CSS 2.1 section 8.3.1 combines a pair by adding the largest positive margin to
	// the most negative margin; this also covers pairs with only one sign.
	values := []float64{-12, -4, 0, 7, 19}
	for _, a := range values {
		for _, b := range values {
			want := math.Max(0, math.Max(a, b)) + math.Min(0, math.Min(a, b))
			if got := collapseMarginPair(a, b); got != want {
				t.Errorf("collapse(%g,%g)=%g, want positive extreme + negative extreme = %g", a, b, got, want)
			}
			if got := collapseMarginPair(b, a); got != want {
				t.Errorf("reversed collapse(%g,%g)=%g, want %g", b, a, got, want)
			}
		}
	}
}

func TestBoxes_BlockParentMarginCollapsesWithSingleChild(t *testing.T) {
	root, err := Build(`<div style="margin: 10px 0"><p style="margin: 20px 0">Anidado</p></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	parent := root.Children[0]
	if parent.Style.MarginTop != 15 || parent.Style.MarginBottom != 15 {
		t.Errorf("outer margins: got %v/%v pt, want 15/15", parent.Style.MarginTop, parent.Style.MarginBottom)
	}
}

func TestBoxes_ParentBorderOrPaddingStopsMarginCollapse(t *testing.T) {
	cases := []struct{ name, separator string }{
		{"border", `border-top: 1px solid black;`},
		{"padding", `padding-top: 1px;`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			html := `<div style="margin-top: 10px; ` + tc.separator + `"><p style="margin-top: 20px">Anidado</p></div>`
			root, err := Build(html)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			if got := root.Children[0].Style.MarginTop; got != 7.5 {
				t.Errorf("parent margin with separator: got %v, want 7.5", got)
			}
		})
	}
}
