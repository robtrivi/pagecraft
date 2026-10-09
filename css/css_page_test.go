package css

import (
	"testing"
)

func TestCSS_ParsePageRulesPreservesSelectorsAndDeclarationOrder(t *testing.T) {
	rules := ParsePageRules(`@page { size: A4; margin: 12pt } @page report:left { size: A3 landscape; margin-left: 24pt } @page :first { margin-top: 30pt }`)
	if len(rules) != 3 {
		t.Fatalf("got %d page rules, want 3", len(rules))
	}
	if rules[0].Selector.Name != "" || rules[0].Selector.Pseudo != "" {
		t.Errorf("default selector: got %+v", rules[0].Selector)
	}
	if rules[1].Selector.Name != "report" || rules[1].Selector.Pseudo != "left" {
		t.Errorf("named left selector: got %+v", rules[1].Selector)
	}
	if rules[1].Style.Width != 1190 || rules[1].Style.Height != 842 || rules[1].Style.Margins.Left != 24 {
		t.Errorf("named rule style: got %+v", rules[1].Style)
	}
	if rules[2].Selector.Pseudo != "first" || rules[2].Style.Margins.Top != 30 {
		t.Errorf("first-page rule: got %+v", rules[2])
	}
}

func TestCSS_ResolvePageStyleCascadesMatchingPageSelectors(t *testing.T) {
	rules := ParsePageRules(`@page { size: A4; margin: 10pt; margin-top: 8pt } @page report:left { margin-left: 24pt } @page :first { margin-top: 30pt } @page { margin-top: 2pt }`)
	style := ResolvePageStyle(rules, "report", 1)
	if style.Width != 595 || style.Height != 842 || style.Margins.Left != 10 || style.Margins.Top != 30 {
		t.Errorf("resolved style: %+v", style)
	}
	style = ResolvePageStyle(rules, "report", 2)
	if style.Margins.Left != 24 || style.Margins.Top != 2 {
		t.Errorf("left page selector should override matching side: %+v", style.Margins)
	}
}

func TestCSS_ParsePageStyleReadsBleedAndMarks(t *testing.T) {
	rules := ParsePageRules(`@page { bleed: 3mm; marks: crop cross }`)
	if len(rules) != 1 || !pageValueNear(rules[0].Style.Bleed, 3*72/25.4) || rules[0].Style.Marks != PageMarkCrop|PageMarkCross {
		t.Fatalf("got %+v, want 3mm bleed and crop+cross marks", rules)
	}
}

func TestCSS_ParsePageRulesRetainsAllMarginBoxDeclarations(t *testing.T) {
	rules := ParsePageRules(`@page { @top-left-corner { content: "TL"; color: red } @top-left { content: "L" } @top-center { content: "C" } @top-right { content: "R" } @top-right-corner { content: "TR" } @right-top { content: "RT" } @right-middle { content: "RM" } @right-bottom { content: "RB" } @bottom-right-corner { content: "BR" } @bottom-right { content: "B" } @bottom-center { content: "BC" } @bottom-left { content: "BL" } @bottom-left-corner { content: "BLc" } @left-bottom { content: "LB" } @left-middle { content: "LM" } @left-top { content: "LT" } }`)
	if len(rules) != 1 || len(rules[0].MarginBoxes) != 16 {
		t.Fatalf("got page rule margin boxes: %+v", rules)
	}
	if got := rules[0].MarginBoxes["top-left-corner"]["content"]; got != `"TL"` {
		t.Errorf("top-left-corner content = %q, want quoted literal", got)
	}
	if got := rules[0].MarginBoxes["top-left-corner"]["color"]; got != "red" {
		t.Errorf("top-left-corner color = %q, want red", got)
	}
}

func TestCSS_ResolvePageRuleCascadesMarginBoxDeclarations(t *testing.T) {
	rules := ParsePageRules(`@page { @bottom-center { content: counter(page); color: red } } @page :first { @bottom-center { color: blue } }`)
	page := ResolvePageRule(rules, "", 1)
	if got := page.MarginBoxes["bottom-center"]["content"]; got != "counter(page)" {
		t.Errorf("content = %q, want counter(page)", got)
	}
	if got := page.MarginBoxes["bottom-center"]["color"]; got != "blue" {
		t.Errorf("color = %q, want blue from more specific rule", got)
	}
}

func TestCSS_ResolvePageDefaultsIgnoresPageSpecificSelectors(t *testing.T) {
	rules := ParsePageRules(`@page { size: A4; margin: 10pt } @page :first { size: Letter; margin-top: 40pt } @page :left { margin-left: 30pt }`)
	style := ResolvePageDefaults(rules)
	if style.Width != 595 || style.Height != 842 || style.Margins.Top != 10 || style.Margins.Left != 10 {
		t.Errorf("default page style included a pseudo-page override: %+v", style)
	}
}

// Page step 1: @page size (keyword and landscape) and simple margins.
// Units: pt, px (1:1 in v1), cm, mm, and in.
func TestCSS_ParsePageRule_ResolvesPageSizeAndMargin(t *testing.T) {
	cases := []struct {
		name, css     string
		width, height float64
		margin        float64
		valid         bool
	}{
		{"A4 landscape and margin", `@page { size: A4 landscape; margin: 2cm }`, 842, 595, 2 * 72 / 2.54, true},
		{"Letter default margin", `@page { size: Letter }`, 612, 792, -1, true},
		{"margin preserves page size", `@page { margin: 2cm }`, 0, 0, 2 * 72 / 2.54, true},
		{"A6 portrait", `@page { size: A6 }`, 105 * 72 / 25.4, 148 * 72 / 25.4, -1, true},
		{"stylesheet without page rule", `p { color: red }`, 0, 0, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			width, height, margin, ok := ParsePageRule(tc.css)
			if ok != tc.valid {
				t.Fatalf("valid: got %v, want %v", ok, tc.valid)
			}
			if !ok {
				return
			}
			if !pageValueNear(width, tc.width) || !pageValueNear(height, tc.height) || !pageValueNear(margin, tc.margin) {
				t.Errorf("got %vx%v margin=%v, want %vx%v margin=%v", width, height, margin, tc.width, tc.height, tc.margin)
			}
		})
	}
}

func TestCSS_ParsePageStyle_ExpandsMarginShorthandAndLonghands(t *testing.T) {
	page, ok := ParsePageStyle(`@page { size: A4; margin: 18mm 16mm 20mm; margin-left: 12mm; margin-top: 17mm }`)
	if !ok {
		t.Fatal("ParsePageStyle returned no page rule")
	}
	want := PageMargins{
		Top: 17 * 72 / 25.4, Right: 16 * 72 / 25.4,
		Bottom: 20 * 72 / 25.4, Left: 12 * 72 / 25.4,
		HasTop: true, HasRight: true, HasBottom: true, HasLeft: true,
	}
	if !pageMarginsNear(page.Margins, want) {
		t.Errorf("margins: got %+v, want %+v", page.Margins, want)
	}
	if page.Width != 595 || page.Height != 842 || !page.HasSize {
		t.Errorf("size: got %gx%g (has=%v), want 595x842 (has=true)", page.Width, page.Height, page.HasSize)
	}
}

func TestCSS_ParsePageStyleExpandsOneAndTwoValueMargins(t *testing.T) {
	cases := []struct {
		name string
		css  string
		want PageMargins
	}{
		{"one value", `@page { margin: 7pt }`, PageMargins{Top: 7, Right: 7, Bottom: 7, Left: 7, HasTop: true, HasRight: true, HasBottom: true, HasLeft: true}},
		{"two values", `@page { margin: 3pt 9pt }`, PageMargins{Top: 3, Right: 9, Bottom: 3, Left: 9, HasTop: true, HasRight: true, HasBottom: true, HasLeft: true}},
		{"four values", `@page { margin: 1pt 2pt 3pt 4pt }`, PageMargins{Top: 1, Right: 2, Bottom: 3, Left: 4, HasTop: true, HasRight: true, HasBottom: true, HasLeft: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			page, ok := ParsePageStyle(tc.css)
			if !ok {
				t.Fatal("ParsePageStyle returned no page rule")
			}
			if !pageMarginsNear(page.Margins, tc.want) {
				t.Errorf("margins: got %+v, want %+v", page.Margins, tc.want)
			}
		})
	}
}

func TestCSS_ParsePageStylePreservesPercentageMargins(t *testing.T) {
	page, ok := ParsePageStyle(`@page { size: 400pt 600pt; margin: 10% 5% }`)
	if !ok {
		t.Fatal("ParsePageStyle returned no page rule")
	}
	want := PageMargins{
		Top: 10, Right: 5, Bottom: 10, Left: 5,
		TopPercent: true, RightPercent: true, BottomPercent: true, LeftPercent: true,
		HasTop: true, HasRight: true, HasBottom: true, HasLeft: true,
	}
	if !pageMarginsNear(page.Margins, want) {
		t.Errorf("margins: got %+v, want %+v", page.Margins, want)
	}
}

func TestCSS_ParsePageStyleSupportsScalableSquareAndPaperKeywordSizes(t *testing.T) {
	for _, tc := range []struct {
		name string
		css  string
		w, h float64
		has  bool
	}{
		{"automatic size", `@page { size: auto }`, 0, 0, false},
		{"square page", `@page { size: 300pt }`, 300, 300, true},
		{"orientation without paper name", `@page { size: landscape }`, 842, 595, true},
		{"ledger paper", `@page { size: ledger }`, 1224, 792, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			page, ok := ParsePageStyle(tc.css)
			if !ok {
				t.Fatal("ParsePageStyle returned no page rule")
			}
			if page.Width != tc.w || page.Height != tc.h || page.HasSize != tc.has {
				t.Errorf("size: got %gx%g (has=%v), want %gx%g (has=%v)", page.Width, page.Height, page.HasSize, tc.w, tc.h, tc.has)
			}
		})
	}
}

func TestCSS_ParsePageStyleReadsPageBackgroundColor(t *testing.T) {
	page, ok := ParsePageStyle(`@page { background-color: #123456 }`)
	if !ok {
		t.Fatal("ParsePageStyle returned no page rule")
	}
	if !page.HasBackground || page.Background != [4]float64{0x12 / 255.0, 0x34 / 255.0, 0x56 / 255.0, 1} {
		t.Errorf("background: got %v (has=%v), want #123456", page.Background, page.HasBackground)
	}
}

func TestCSS_ParsePageStyleHonorsDeclarationOrder(t *testing.T) {
	page, ok := ParsePageStyle(`@page { margin-left: 9pt; margin: 2pt 4pt }`)
	if !ok {
		t.Fatal("ParsePageStyle returned no page rule")
	}
	if page.Margins.Left != 4 || page.Margins.Top != 2 || page.Margins.Right != 4 || page.Margins.Bottom != 2 {
		t.Errorf("later shorthand did not override earlier longhand: %+v", page.Margins)
	}
}

func TestCSS_ParsePageStyleReadsOutputOrientation(t *testing.T) {
	page, ok := ParsePageStyle(`@page { page-orientation: rotate-left }`)
	if !ok {
		t.Fatal("ParsePageStyle returned no page rule")
	}
	if page.Rotation != 270 {
		t.Errorf("rotation: got %d, want 270 degrees counter-clockwise", page.Rotation)
	}
}

func pageMarginsNear(got, want PageMargins) bool {
	return pageValueNear(got.Top, want.Top) && pageValueNear(got.Right, want.Right) &&
		pageValueNear(got.Bottom, want.Bottom) && pageValueNear(got.Left, want.Left) &&
		got.TopPercent == want.TopPercent && got.RightPercent == want.RightPercent &&
		got.BottomPercent == want.BottomPercent && got.LeftPercent == want.LeftPercent &&
		got.HasTop == want.HasTop && got.HasRight == want.HasRight &&
		got.HasBottom == want.HasBottom && got.HasLeft == want.HasLeft
}

func pageValueNear(got, want float64) bool {
	const epsilon = 1e-9
	return got-want >= -epsilon && got-want <= epsilon
}
