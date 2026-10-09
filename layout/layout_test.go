package layout

import (
	"math"
	"strings"
	"testing"

	"github.com/robtrivi/pagecraft-go/boxes"
	"github.com/robtrivi/pagecraft-go/text"
)

func TestLayout_PageContextResolvesMarginsAfterExplicitPageBreak(t *testing.T) {
	root, err := boxes.Build(`<style>@page { size: 200pt 200pt; margin: 10pt } @page :left { size: 250pt 230pt; margin-left: 30pt; margin-top: 25pt } body { margin: 0 } p { margin: 0 }</style><p>A</p><p style="break-before:page">B</p>`)
	if err != nil {
		t.Fatalf("boxes.Build: %v", err)
	}
	opts := DefaultOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 200, 200, 10
	opts.Measure, opts.MeasureBold = text.Measure, text.MeasureBold
	opts.PageMargins = root.Page.Margins
	opts.PageRules = root.PageRules
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2", len(pages))
	}
	if got := pages[1].Blocks[0].X; got != 30 {
		t.Errorf("second page x = %g, want left page margin 30", got)
	}
	if got := pages[1].Blocks[0].Y; got != 25 {
		t.Errorf("second page y = %g, want left page top margin 25", got)
	}
	if pages[1].Geometry.Width != 250 || pages[1].Geometry.Height != 230 {
		t.Errorf("second page geometry = %gx%g, want 250x230", pages[1].Geometry.Width, pages[1].Geometry.Height)
	}
}

func TestLayout_ParagraphContinuationUsesDestinationPageCapacity(t *testing.T) {
	root, err := boxes.Build(`<style>@page { size: 200pt 60pt; margin: 10pt } @page :left { size: 200pt 100pt; margin: 20pt 10pt 10pt } body { margin: 0 } p { margin: 0; font: 10pt/10pt Helvetica }</style><p>A<br>B<br>C<br>D<br>E<br>F</p>`)
	if err != nil {
		t.Fatalf("boxes.Build: %v", err)
	}
	opts := DefaultOptions()
	opts.Measure, opts.MeasureBold = text.Measure, text.MeasureBold
	opts.PageWidth, opts.PageHeight, opts.Margin = 200, 60, 10
	opts.PageMargins, opts.PageRules = root.Page.Margins, root.PageRules
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2", len(pages))
	}
	if pages[0].Blocks[0].Text != "A\nB\nC\nD" {
		t.Errorf("first fragment text=%q", pages[0].Blocks[0].Text)
	}
	if pages[1].Blocks[0].Text != "E\nF" {
		t.Errorf("second fragment text=%q", pages[1].Blocks[0].Text)
	}
	if pages[1].Blocks[0].Y != 20 || pages[1].Geometry.Height != 100 {
		t.Errorf("second fragment page geometry=%+v block=%+v", pages[1].Geometry, pages[1].Blocks[0])
	}
}

func TestLayout_ParagraphContinuationReflowsForDestinationPageWidth(t *testing.T) {
	root, err := boxes.Build(`<style>@page { size: 100pt 35pt; margin: 5pt } @page :left { size: 60pt 130pt; margin: 5pt } body { margin: 0 } p { margin: 0; font: 10pt/10pt Helvetica }</style><p>alpha beta gamma delta epsilon zeta eta theta iota kappa lambda mu nu xi omicron pi rho sigma tau</p>`)
	if err != nil {
		t.Fatalf("boxes.Build: %v", err)
	}
	opts := DefaultOptions()
	opts.Measure, opts.MeasureBold = text.Measure, text.MeasureBold
	opts.PageWidth, opts.PageHeight, opts.Margin = 100, 35, 5
	opts.PageMargins, opts.PageRules = root.Page.Margins, root.PageRules
	pages := Flow(root, opts)
	if len(pages) < 2 {
		t.Fatalf("got %d pages, want a continuation", len(pages))
	}
	if pages[1].Geometry.Width != 60 || pages[1].Blocks[0].Width > 50+fragmentEpsilon {
		t.Errorf("continuation did not use the narrower content width: geometry=%+v block=%+v", pages[1].Geometry, pages[1].Blocks[0])
	}
	text := pages[0].Blocks[0].Text + " " + pages[1].Blocks[0].Text
	for _, word := range []string{"alpha", "lambda", "tau"} {
		if !strings.Contains(text, word) {
			t.Errorf("continuation lost %q: %q", word, text)
		}
	}
}

func TestLayout_FlexHeaderDoesNotAddExtraGapBeforeFollowingBlock(t *testing.T) {
	html := `<style>@page{size:612pt 792pt;margin:34pt}body{margin:0;font:9pt/13pt Helvetica}.mast{display:flex;justify-content:space-between;border-bottom:2pt solid #175b65;padding-bottom:8pt}.mast strong{font-size:14pt}.mast span{font-size:8pt}.eyebrow{font-size:7pt;margin-top:14pt}</style><header class="mast"><strong>MERIDIAN</strong><span>FIELD SERVICES / 2026.09</span></header><div class="eyebrow">INTERNAL REVIEW · REPORT 24-091</div>`
	root, err := boxes.Build(string(html))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := DefaultOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = root.Page.Width, root.Page.Height, root.Page.Margin
	opts.Measure, opts.MeasureBold = text.Measure, text.MeasureBold
	opts.Faces, opts.FontVariants = root.Fonts, root.FontVariants
	var mastheadY, eyebrowY float64
	for _, page := range Flow(root, opts) {
		for _, block := range page.Blocks {
			for _, line := range block.Runs {
				for _, run := range line {
					switch run.Text {
					case "MERIDIAN":
						mastheadY = run.Y
					case "INTERNAL":
						eyebrowY = run.Y
					}
				}
			}
		}
	}
	if mastheadY == 0 || eyebrowY == 0 {
		t.Fatalf("missing coordinates of the header: masthead=%.2f eyebrow=%.2f", mastheadY, eyebrowY)
	}
	if got := eyebrowY - mastheadY; math.Abs(got-36) > 2 {
		t.Errorf("vertical distance between masthead and eyebrow=%.2f pt; Chromium places those lines ~36 pt apart", got)
	}
}

func TestLayout_RelativeSmallFontKeepsFeeCalloutOnItsMeasuredLine(t *testing.T) {
	html := `<style>@page{size:595pt 842pt;margin:36pt}body{margin:0;font:9pt/13pt Helvetica}.investment{float:right;width:165pt;padding:10pt;margin:4pt 0 8pt 12pt;border:1pt solid #99f6e4}.investment b{font-size:15pt}table{border-collapse:collapse;margin-top:14pt;width:100%}td,th{padding:6pt;border:1pt solid #cbd5e1}</style><h2>PROGRAM AND FEES</h2><div class="investment"><span>PROPOSED FEE</span><br><b>$48,600</b><br>Fixed professional fee<br><small>Expenses billed at cost, capped at $1,200</small></div><table><tr><th>Stage</th><th>Fee</th></tr><tr><td>Discover</td><td>$12,400</td></tr></table>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := DefaultOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = root.Page.Width, root.Page.Height, root.Page.Margin
	opts.Measure, opts.MeasureBold = text.Measure, text.MeasureBold
	opts.Faces, opts.FontVariants = root.Fonts, root.FontVariants
	var expensesAndPriceShareLine bool
	var expensePriceY, tableHeaderY float64
	for _, page := range Flow(root, opts) {
		for _, block := range page.Blocks {
			for _, line := range block.Runs {
				var text strings.Builder
				for _, run := range line {
					text.WriteString(run.Text)
					if run.Text == "$1,200" {
						expensePriceY = run.Y
					}
					if run.Text == "Fee" && run.source != nil && run.source.Tag == "th" {
						tableHeaderY = run.Y
					}
				}
				if strings.Contains(text.String(), "Expenses") && strings.Contains(text.String(), "$1,200") {
					expensesAndPriceShareLine = true
				}
			}
		}
	}
	if !expensesAndPriceShareLine {
		t.Error("the expense amount must remain on the same line as the text, as in Chromium and WeasyPrint")
	}
	if expensePriceY == 0 || tableHeaderY == 0 {
		t.Fatalf("missing callout/table coordinates: amount=%.2f table=%.2f", expensePriceY, tableHeaderY)
	}
	if got := tableHeaderY - expensePriceY; math.Abs(got-38) > 3 {
		t.Errorf("distance between the callout amount and table header=%.2f pt; Chromium lays them out ~38 pt apart", got)
	}
}

// A long sequence of paragraphs produces multiple A4 pages.
// Uses a deterministic fake measurer (6 pt per rune); real measurement arrived in v0.5.
func testOptions() Options {
	return Options{
		PageWidth: 595, PageHeight: 842, // A4 en pt
		Margin:   72,
		FontSize: 12, LineHeight: 1.2,
		Measure: func(s string, _ float64) float64 { return float64(len([]rune(s)) * 6) },
	}
}

func TestLayout_LongDocument_SplitsPages(t *testing.T) {
	root, err := boxes.Build(strings.Repeat(`<p style="margin: 0">Hello world</p>`, 100))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pages := Flow(root, testOptions())
	// Ten runes at 6 pt each fit on one 14.4 pt line within the available width.
	// The 698 pt content area fits 48 blocks per page, leaving 48, 48, and 4.
	if len(pages) != 3 {
		t.Fatalf("pages: got %d, want 3", len(pages))
	}
	total := 0
	for _, p := range pages {
		total += len(p.Blocks)
	}
	if total != 100 {
		t.Errorf("total blocks: got %d, want 100", total)
	}
	if len(pages[0].Blocks) != 48 || len(pages[1].Blocks) != 48 || len(pages[2].Blocks) != 4 {
		t.Errorf("distribution: got %d+%d+%d, want 48+48+4",
			len(pages[0].Blocks), len(pages[1].Blocks), len(pages[2].Blocks))
	}
	prev := -1.0
	for _, b := range pages[0].Blocks {
		if b.Y <= prev {
			t.Fatalf("y did not increase: %v after %v", b.Y, prev)
		}
		prev = b.Y
	}
}

func TestLayout_ShortDocument_SinglePage(t *testing.T) {
	root, err := boxes.Build(`<p style="margin: 0">Hello</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pages := Flow(root, testOptions())
	if len(pages) != 1 {
		t.Fatalf("pages: got %d, want 1", len(pages))
	}
	b := pages[0].Blocks[0]
	if b.X != 72 || b.Y != 72 {
		t.Errorf("origin: got (%v,%v), want (72,72)", b.X, b.Y)
	}
	if b.Text != "Hello" {
		t.Errorf("text: got %q, want %q", b.Text, "Hello")
	}
}

func TestLayout_ParagraphUAMarginsSeparateParagraphs(t *testing.T) {
	root, err := boxes.Build(`<p>A</p><p>B</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("blocks: got %d, want 2", len(blocks))
	}
	if blocks[0].Y != 84 || blocks[1].Y != 110.4 {
		t.Errorf("paragraph and coordinates: got %v/%v, want 84/110.4", blocks[0].Y, blocks[1].Y)
	}
}

func TestLayout_BodyDefaultMarginOffsetsContent(t *testing.T) {
	root, err := boxes.Build(`<html><body><p style="margin: 0">text</p></body></html>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("blocks: got %d, want 1", len(blocks))
	}
	if blocks[0].X != 78 || blocks[0].Y != 78 || blocks[0].Width != 439 {
		t.Errorf("applied body margin: got x=%v y=%v w=%v, want 78 78 439", blocks[0].X, blocks[0].Y, blocks[0].Width)
	}
}

func TestLayout_HeadingMarginsAffectBlockPositions(t *testing.T) {
	root, err := boxes.Build(`<h1>Title</h1><p>Text</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("blocks: got %d, want 2", len(blocks))
	}
	if blocks[0].Y != 84.06 {
		t.Errorf("h1 start: got %v, want 84.06", blocks[0].Y)
	}
	if blocks[1].Y != 117.72 {
		t.Errorf("following block start: got %v, want 117.72", blocks[1].Y)
	}
}

func TestLayout_AdjacentVerticalMarginsCollapseToLargest(t *testing.T) {
	root, err := boxes.Build(`<div style="margin-bottom: 10px">A</div><div style="margin-top: 20px">B</div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("blocks: got %d, want 2", len(blocks))
	}
	if blocks[1].Y != 101.4 {
		t.Errorf("second block y=%v, want 101.4 (20px of collapsed margin)", blocks[1].Y)
	}
}

func TestLayout_FirstHeadingMarginCollapsesWithBody(t *testing.T) {
	root, err := boxes.Build(`<html><body><h1>Title</h1></body></html>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("blocks: got %d, want 1", len(blocks))
	}
	if blocks[0].Y != 84.06 {
		t.Errorf("first h1 y=%v, want 84.06 (max(body 6pt, heading 12.06pt))", blocks[0].Y)
	}
}

func TestLayout_CollapsesNegativeMarginsByCSSRule(t *testing.T) {
	cases := []struct {
		name, html string
		wantY      float64
	}{
		{"positive_and_negative", `<div style="margin-bottom: 20px">A</div><div style="margin-top: -5px">B</div>`, 97.65},
		{"both_negative", `<div style="margin-bottom: -10px">A</div><div style="margin-top: -20px">B</div>`, 71.4},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, err := boxes.Build(tc.html)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			blocks := Flow(root, testOptions())[0].Blocks
			if len(blocks) != 2 {
				t.Fatalf("blocks: got %d, want 2", len(blocks))
			}
			if blocks[1].Y != tc.wantY {
				t.Errorf("second block y=%v, want %v", blocks[1].Y, tc.wantY)
			}
		})
	}
}

func TestLayout_NestedBlockParentChildMarginsCollapse(t *testing.T) {
	root, err := boxes.Build(`<div style="margin-top: 10px"><p style="margin-top: 20px">Anidado</p></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("blocks: got %d, want 1", len(blocks))
	}
	if blocks[0].Y != 87 {
		t.Errorf("content child y=%v, want 87 (maximum of margins 10px and 20px)", blocks[0].Y)
	}
}

func TestLayout_NestedBlockSiblingsKeepBoxesAndCollapseMargins(t *testing.T) {
	root, err := boxes.Build(`<div style="margin-top: 10px"><p style="margin: 0 0 20px">A</p><p style="margin: 10px 0 0">B</p></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("blocks nested: got %d, want 2", len(blocks))
	}
	if blocks[0].Text != "A" || blocks[1].Text != "B" {
		t.Errorf("text of blocks: got %q/%q, want A/B", blocks[0].Text, blocks[1].Text)
	}
	if blocks[0].Y != 79.5 || blocks[1].Y != 108.9 {
		t.Errorf("y positions of nested siblings: got %v/%v, want 79.5/108.9", blocks[0].Y, blocks[1].Y)
	}
}

func TestLayout_NestedStyledWrapperKeepsBlocksAndInheritedStyle(t *testing.T) {
	root, err := boxes.Build(`<div style="font-weight: bold"><p>A</p><p>B</p></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("boxes: got %d, want two paragraphs", len(blocks))
	}
	for i, block := range blocks {
		if strings.TrimSpace(strings.Join(runLines(block), " ")) != []string{"A", "B"}[i] {
			t.Errorf("text block %d=%q, want %q", i, block.Text, []string{"A", "B"}[i])
		}
		if !block.Bold || len(block.Runs) == 0 || len(block.Runs[0]) == 0 || !block.Runs[0][0].Bold {
			t.Errorf("block %d must preserve inherited bold styling: %#v", i, block)
		}
	}
}

func TestLayout_DecoratedNestedBlockKeepsOuterBoxAndChildBoxes(t *testing.T) {
	root, err := boxes.Build(`<div style="background-color: #ff0000; padding: 5px; border: 1px solid black"><p style="margin: 0">A</p><p style="margin: 0">B</p></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 3 {
		t.Fatalf("boxes: got %d, want a container and two children", len(blocks))
	}
	outer, first, second := blocks[0], blocks[1], blocks[2]
	if !outer.HasBg || outer.BorderWidth != 0.75 || math.Abs(outer.Height-37.8) > 1e-9 {
		t.Errorf("outer box: background=%v border=%v height=%v, want true 0.75 37.8", outer.HasBg, outer.BorderWidth, outer.Height)
	}
	if first.Text != "A" || second.Text != "B" {
		t.Errorf("child text: got %q/%q, want A/B", first.Text, second.Text)
	}
	if first.X != 76.5 || first.Y != 76.5 || second.X != 76.5 || second.Y != 90.9 {
		t.Errorf("child positions: got (%v,%v) / (%v,%v), want (76.5,76.5) / (76.5,90.9)", first.X, first.Y, second.X, second.Y)
	}
}

func TestLayout_DecoratedWrapperPreservesGridChildLayout(t *testing.T) {
	root, err := boxes.Build(`<div style="padding:8pt;background-color:#eeeeee"><div style="display:grid;grid-template-columns:1fr 1fr;gap:8pt"><div style="background-color:#dddddd">FIRST CARD</div><div style="background-color:#cccccc">SECOND CARD</div></div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 3 {
		t.Fatalf("boxes=%d, want wrapper, grid, and two items", len(blocks))
	}
	if !blocks[0].HasBg {
		t.Error("background of the decorated wrapper was lost")
	}
	first, second := blocks[1], blocks[2]
	if !first.HasBg || !strings.Contains(strings.Join(runLines(first), " "), "FIRST CARD") {
		t.Errorf("first grid item lost background or text: %+v", first)
	}
	if !second.HasBg || !strings.Contains(strings.Join(runLines(second), " "), "SECOND CARD") || second.X <= first.X {
		t.Errorf("second item grid lost its style or column: %+v", second)
	}
}

func TestLayout_WrapperBreakBeforeAppliesOnceBeforeInlineTail(t *testing.T) {
	root, err := boxes.Build(`<p>BEFORE</p><section style="break-before:page"><h2>NEW SECTION</h2><p>Body text</p><small>Section note</small></section>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	pages := Flow(root, testOptions())
	if len(pages) != 2 {
		t.Fatalf("pages=%d, want 2; break-before of the wrapper must not repeat in its content inline", len(pages))
	}
	var tailPage int
	for pageIndex, page := range pages {
		for _, block := range page.Blocks {
			if strings.Contains(strings.Join(runLines(block), " "), "Section note") {
				tailPage = pageIndex + 1
			}
		}
	}
	if tailPage != 2 {
		t.Errorf("wrapper note appears on page %d, want 2", tailPage)
	}
}

func TestLayout_DecoratedNestedBlockHonorsAsymmetricInsets(t *testing.T) {
	root, err := boxes.Build(`<div style="background-color: #ff0000; padding: 2px 8px 6px 10px; border: 1px solid black"><p style="margin: 0">A</p><p style="margin: 0">B</p></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 3 {
		t.Fatalf("boxes: got %d, want a container and two children", len(blocks))
	}
	outer, first, second := blocks[0], blocks[1], blocks[2]
	if math.Abs(outer.Height-36.3) > 1e-9 {
		t.Errorf("height outer: got %v, want 36.3", outer.Height)
	}
	if first.X != 80.25 || first.Y != 74.25 || first.Width != 436 {
		t.Errorf("first child: x=%v y=%v w=%v, want 80.25 74.25 436", first.X, first.Y, first.Width)
	}
	if second.X != 80.25 || second.Y != 88.65 || second.Width != 436 {
		t.Errorf("second child: x=%v y=%v w=%v, want 80.25 88.65 436", second.X, second.Y, second.Width)
	}
}

func TestLayout_DecoratedContainerCollapsesChildMarginsWhenEdgesAllow(t *testing.T) {
	cases := []struct {
		name, html      string
		wantBackground  bool
		wantOpacity     bool
		wantOuterY      float64
		wantOuterHeight float64
		wantChildY      float64
		wantNextY       float64
	}{
		{
			name:           "background_allows_collapse_through",
			html:           `<div style="background-color: #ff0000; margin: 10px 0 8px"><p style="margin: 20px 0 15px">inside</p></div><p style="margin: 10px 0 0">next</p>`,
			wantBackground: true,
			wantOuterY:     87, wantOuterHeight: 14.4, wantChildY: 87, wantNextY: 112.65,
		},
		{
			name:        "opacity_allows_collapse_through",
			html:        `<div style="opacity: 0.5; margin: 10px 0 8px"><p style="margin: 20px 0 15px">inside</p></div><p style="margin: 10px 0 0">next</p>`,
			wantOpacity: true,
			wantOuterY:  87, wantOuterHeight: 14.4, wantChildY: 87, wantNextY: 112.65,
		},
		{
			name:           "border_stops_top_collapse",
			html:           `<div style="background-color: #ff0000; border-top: 1px solid; margin-top: 10px"><p style="margin: 20px 0 0">inside</p></div>`,
			wantBackground: true,
			wantOuterY:     79.5, wantOuterHeight: 30.15, wantChildY: 95.25,
		},
		{
			name:           "padding_stops_top_collapse",
			html:           `<div style="background-color: #ff0000; padding-top: 1px; margin-top: 10px"><p style="margin: 20px 0 0">inside</p></div>`,
			wantBackground: true,
			wantOuterY:     79.5, wantOuterHeight: 30.15, wantChildY: 95.25,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, err := boxes.Build(tc.html)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			blocks := Flow(root, testOptions())[0].Blocks
			wantBlocks := 2
			if tc.wantNextY != 0 {
				wantBlocks++
			}
			if len(blocks) != wantBlocks {
				t.Fatalf("boxes: got %d, want %d", len(blocks), wantBlocks)
			}
			outer, child := blocks[0], blocks[1]
			if outer.HasBg != tc.wantBackground || math.Abs(outer.Y-tc.wantOuterY) > 1e-9 || math.Abs(outer.Height-tc.wantOuterHeight) > 1e-9 {
				t.Errorf("wrapper: bg=%v y=%v height=%v, want %v %v %v", outer.HasBg, outer.Y, outer.Height, tc.wantBackground, tc.wantOuterY, tc.wantOuterHeight)
			}
			if gotOpacity := len(outer.OpacityGroups) == 1; gotOpacity != tc.wantOpacity {
				t.Errorf("opacity group present=%v, want %v", gotOpacity, tc.wantOpacity)
			}
			if child.Text != "inside" || math.Abs(child.Y-tc.wantChildY) > 1e-9 {
				t.Errorf("child: text=%q y=%v, want inside %v", child.Text, child.Y, tc.wantChildY)
			}
			if tc.wantNextY != 0 && (blocks[2].Text != "next" || math.Abs(blocks[2].Y-tc.wantNextY) > 1e-9) {
				t.Errorf("following sibling: text=%q y=%v, want next %v", blocks[2].Text, blocks[2].Y, tc.wantNextY)
			}
		})
	}
}

func TestLayout_OpacityNestedBlockKeepsChildrenAcrossPages(t *testing.T) {
	root, err := boxes.Build(`<div style="opacity: 0.5; background-color: #ff0000; padding: 5px"><p style="margin: 0">` + strings.Repeat("word ", 35) + `</p><p style="margin: 0">` + strings.Repeat("another ", 35) + `</p></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageHeight = 100
	opts.Margin = 10
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages: got %d, want 2", len(pages))
	}
	for i, page := range pages {
		if len(page.Blocks) < 2 || page.Blocks[0].Text != "" {
			t.Fatalf("page %d: must preserve wrapper and paragraph boxes: %#v", i+1, page.Blocks)
		}
		outer := page.Blocks[0]
		if !outer.HasBg || outer.HasOpacity || outer.Height <= 0 {
			t.Errorf("page %d: wrapper must preserve background/segment without individual opacity: %#v", i+1, outer)
		}
		for _, child := range page.Blocks[1:] {
			if child.Text == "" || child.HasBg {
				t.Errorf("page %d: must preserve boxes of paragraph with text: %#v", i+1, child)
			}
		}
	}
}

func TestLayout_OpacityWrapperPreservesInlineAndBlockOrder(t *testing.T) {
	root, err := boxes.Build(`<div style="opacity: 0.5; background-color: #ff0000; padding: 5px">before <p style="margin: 0">middle</p>after</div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 4 {
		t.Fatalf("boxes: got %d, want wrapper, inline, paragraph e inline", len(blocks))
	}
	want := []string{"", "before", "middle", "after"}
	for i, block := range blocks {
		got := strings.Join(runLines(block), " ")
		if i == 0 {
			if !block.HasBg || len(block.OpacityGroups) != 1 {
				t.Errorf("wrapper must preserve background and opacity group: %#v", block)
			}
			continue
		}
		if strings.TrimSpace(got) != want[i] {
			t.Errorf("text box %d=%q, want %q", i, got, want[i])
		}
		if len(block.OpacityGroups) != 1 || block.OpacityGroups[0].Opacity != 0.5 {
			t.Errorf("box %d must belong to the opacity group: %#v", i, block.OpacityGroups)
		}
	}
}

func TestLayout_TransparentWrapperPreservesInlineAndBlockOrder(t *testing.T) {
	root, err := boxes.Build(`<div>before <p style="margin: 0">middle</p>after</div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 3 {
		t.Fatalf("boxes: got %d, want inline, paragraph e inline", len(blocks))
	}
	want := []string{"before", "middle", "after"}
	for i, block := range blocks {
		got := strings.TrimSpace(strings.Join(runLines(block), " "))
		if got != want[i] {
			t.Errorf("text box %d=%q, want %q", i, got, want[i])
		}
	}
}

func TestLayout_InheritedStyleWrapperPreservesMixedInlineBlockFlow(t *testing.T) {
	cases := []struct {
		name, style string
		wantBold    bool
		wantItalic  bool
		wantAlign   string
	}{
		{"font_weight", "font-weight: bold", true, false, "left"},
		{"font_style", "font-style: italic", false, true, "left"},
		{"text_align", "text-align: center", false, false, "center"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			html := `<div style="` + tc.style + `">before <p style="margin: 0">middle</p>after</div>`
			root, err := boxes.Build(html)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			blocks := Flow(root, testOptions())[0].Blocks
			if len(blocks) != 3 {
				t.Fatalf("boxes: got %d, want inline, paragraph e inline", len(blocks))
			}
			wantText := []string{"before", "middle", "after"}
			for i, block := range blocks {
				gotText := strings.TrimSpace(strings.Join(runLines(block), " "))
				if gotText != wantText[i] {
					t.Errorf("text box %d=%q, want %q", i, gotText, wantText[i])
				}
				if block.Align != tc.wantAlign {
					t.Errorf("box %d alignment=%q, want %q", i, block.Align, tc.wantAlign)
				}
				for _, line := range block.Runs {
					for _, run := range line {
						if run.Bold != tc.wantBold || run.Italic != tc.wantItalic {
							t.Errorf("box %d style bold/italic=%v/%v, want %v/%v", i, run.Bold, run.Italic, tc.wantBold, tc.wantItalic)
						}
					}
				}
			}
		})
	}
}

func TestLayout_DecorationWrapperPreservesMixedInlineBlockFlow(t *testing.T) {
	cases := []struct {
		name, decoration string
		wantUnderline    bool
		wantStrike       bool
	}{
		{"underline", "underline", true, false},
		{"strike", "line-through", false, true},
		{"combined", "underline line-through", true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			html := `<div style="text-decoration: ` + tc.decoration + `">before <p style="margin: 0">middle</p>after</div>`
			root, err := boxes.Build(html)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			blocks := Flow(root, testOptions())[0].Blocks
			if len(blocks) != 3 {
				t.Fatalf("boxes: got %d, want inline, paragraph e inline", len(blocks))
			}
			wantText := []string{"before", "middle", "after"}
			for i, block := range blocks {
				gotText := strings.TrimSpace(strings.Join(runLines(block), " "))
				if gotText != wantText[i] {
					t.Errorf("text box %d=%q, want %q", i, gotText, wantText[i])
				}
				for _, line := range block.Runs {
					for _, run := range line {
						if run.Underline != tc.wantUnderline || run.Strike != tc.wantStrike {
							t.Errorf("decoration box %d underline/strike=%v/%v, want %v/%v", i, run.Underline, run.Strike, tc.wantUnderline, tc.wantStrike)
						}
					}
				}
			}
		})
	}
}

func TestLayout_DecorationAndBackgroundWrapperPreservesMixedFlow(t *testing.T) {
	cases := []struct {
		name, decoration string
		wantUnderline    bool
		wantStrike       bool
	}{
		{"underline", "underline", true, false},
		{"strike", "line-through", false, true},
		{"combined", "underline line-through", true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			html := `<div style="background-color: #ff0000; padding: 5px; text-decoration: ` + tc.decoration + `">before <p style="margin: 0">middle</p>after</div>`
			root, err := boxes.Build(html)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			blocks := Flow(root, testOptions())[0].Blocks
			if len(blocks) != 4 {
				t.Fatalf("boxes: got %d, want wrapper, inline, paragraph e inline", len(blocks))
			}
			if !blocks[0].HasBg || blocks[0].Text != "" {
				t.Errorf("wrapper must preserve its background: %#v", blocks[0])
			}
			wantText := []string{"", "before", "middle", "after"}
			for i, block := range blocks[1:] {
				gotText := strings.TrimSpace(strings.Join(runLines(block), " "))
				if gotText != wantText[i+1] {
					t.Errorf("text box %d=%q, want %q", i+1, gotText, wantText[i+1])
				}
				for _, line := range block.Runs {
					for _, run := range line {
						if run.Underline != tc.wantUnderline || run.Strike != tc.wantStrike {
							t.Errorf("decoration box %d underline/strike=%v/%v, want %v/%v", i+1, run.Underline, run.Strike, tc.wantUnderline, tc.wantStrike)
						}
					}
				}
			}
		})
	}
}

func TestLayout_DecoratedNestedBlockRepeatsBackgroundAcrossPages(t *testing.T) {
	root, err := boxes.Build(`<div style="background-color: #ff0000; padding: 5px"><p style="margin: 0">` + strings.Repeat("word ", 35) + `</p><p style="margin: 0">` + strings.Repeat("another ", 35) + `</p></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageHeight = 100
	opts.Margin = 10
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages: got %d, want 2", len(pages))
	}
	for i, page := range pages {
		if len(page.Blocks) < 2 || !page.Blocks[0].HasBg {
			t.Errorf("page %d must start with the container background: %#v", i+1, page.Blocks)
		}
	}
}

func TestLayout_DecoratedNestedBlockHonorsNegativeStartMargin(t *testing.T) {
	root, err := boxes.Build(`<div style="margin-top: -20px; background-color: #ff0000; padding: 5px"><p style="margin: 0">A</p></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.Margin = 10
	blocks := Flow(root, opts)[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("boxes: got %d, want outer echild", len(blocks))
	}
	if blocks[0].Y != -5 || blocks[1].Y != -1.25 {
		t.Errorf("positions with negative margin: got %v/%v, want -5/-1.25", blocks[0].Y, blocks[1].Y)
	}
}
