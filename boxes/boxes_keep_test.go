package boxes

import (
	"testing"
)

func TestBoxes_ComputedBreakModesUseTypedValues(t *testing.T) {
	root, err := Build(`<div style="break-before:page;break-inside:avoid">text</div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	style := root.Children[0].Style
	if style.BreakBefore != BreakBeforePage || style.BreakInside != BreakInsideAvoid {
		t.Fatalf("computed break modes = %+v", style)
	}
}

func TestBoxes_ComputedBreakAfterNormalizesModernAndLegacyAvoid(t *testing.T) {
	for _, property := range []string{"break-after: avoid", "page-break-after: avoid"} {
		t.Run(property, func(t *testing.T) {
			root, err := Build(`<p style="` + property + `">x</p><p>y</p>`)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			if got := root.Children[0].Style.BreakAfter; got != BreakAfterAvoid {
				t.Errorf("BreakAfter=%q, want %q", got, BreakAfterAvoid)
			}
			if got := root.Children[1].Style.BreakAfter; got != BreakAfterAuto {
				t.Errorf("initial value BreakAfter=%q, want auto", got)
			}
		})
	}
}

func TestBoxes_ParsesBreakInsideAvoid(t *testing.T) {
	cases := []struct {
		name, property string
	}{
		{name: "modern", property: "break-inside"},
		{name: "legacy", property: "page-break-inside"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, err := Build(`<p style="` + tc.property + `: avoid">text</p>`)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			if got := root.Children[0].Style.BreakInside; got != BreakInsideAvoid {
				t.Errorf("break-inside: got %q, want avoid", got)
			}
		})
	}
}

func TestBoxes_ParsesBreakBeforePage(t *testing.T) {
	cases := []struct {
		name, property string
	}{
		{name: "modern", property: "break-before: page"},
		{name: "legacy", property: "page-break-before: always"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, err := Build(`<p style="` + tc.property + `">text</p>`)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			if got := root.Children[0].Style.BreakBefore; got != BreakBeforePage {
				t.Errorf("break-before: got %q, want page", got)
			}
		})
	}
}

func TestBoxes_ParsesBreakAfterPage(t *testing.T) {
	for _, property := range []string{"break-after: page", "page-break-after: always"} {
		t.Run(property, func(t *testing.T) {
			root, err := Build(`<p style="` + property + `">x</p>`)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			if got := root.Children[0].Style.BreakAfter; got != BreakAfterPage {
				t.Fatalf("break-after=%q, want %q", got, BreakAfterPage)
			}
		})
	}
}

func TestBoxes_ModernBreakAfterAutoOverridesLegacyAlias(t *testing.T) {
	root, err := Build(`<p style="page-break-after:always;break-after:auto">x</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if got := root.Children[0].Style.BreakAfter; got != BreakAfterAuto {
		t.Fatalf("break-after=%q, want auto", got)
	}
}

func TestBoxes_ParsesAndInheritsWidowsOrphans(t *testing.T) {
	root, err := Build(`<div style="widows:3;orphans:4"><p>text</p></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	for name, box := range map[string]*Box{"wrapper": root.Children[0], "paragraph": root.Children[0].Children[0]} {
		if box.Style.Widows != 3 || box.Style.Orphans != 4 {
			t.Errorf("%s: widows/orphans=%d/%d, want 3/4", name, box.Style.Widows, box.Style.Orphans)
		}
	}
}
