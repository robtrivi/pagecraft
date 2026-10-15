package layout

import (
	"math"
	"strings"
	"testing"

	"github.com/robtrivi/pagecraft/boxes"
)

func TestLayout_GridWrapsTallItemAcrossPages(t *testing.T) {
	content := "A1 alpha A2 bravo A3 charlie A4 delta A5 echo A6 foxtrot A7 golf A8 hotel A9 india A10 juliet A11 kilo A12 lima A13 mike A14 november A15 oscar A16 papa A17 quebec A18 romeo A19 sierra A20 tango A21 uniform A22 victor A23 whiskey A24 xray A25 yankee A26 zulu"
	root, err := boxes.Build(`<div style="height:75pt">BEFORE</div><div style="display:grid;grid-template-columns:150pt"><div style="width:150pt;font-size:9pt;line-height:10.5pt">` + content + `</div></div><p style="margin:0">AFTER GRID</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 225, 135, 9
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Errorf("pages=%d, want 2 as in Chromium", len(pages))
	}
	itemPages := make(map[int]bool)
	var rendered strings.Builder
	afterPage := 0
	for pageIndex, page := range pages {
		for _, block := range page.Blocks {
			if block.Width == 150 && block.FontSize == 9 && len(block.Runs) > 0 {
				itemPages[pageIndex+1] = true
				for _, line := range block.Runs {
					for _, run := range line {
						rendered.WriteString(run.Text)
						rendered.WriteByte(' ')
					}
				}
			}
			if block.Text == "AFTER GRID" {
				afterPage = pageIndex + 1
			}
		}
	}
	if !itemPages[1] || !itemPages[2] {
		t.Errorf("the tall item must start on page 1 and continue on page 2; pages=%v", itemPages)
	}
	if got, want := strings.Join(strings.Fields(rendered.String()), " "), strings.Join(strings.Fields(content), " "); got != want {
		t.Errorf("fragmented runs lost or repeated text: got %q want %q", got, want)
	}
	if afterPage != 2 {
		t.Errorf("AFTER GRID appears on page %d; must follow the fragmented content on page 2", afterPage)
	}
}

func TestLayout_GridKeepsFittingStructuredRowTogetherAtPageBoundary(t *testing.T) {
	html := `<style>@page{size:420pt 595pt;margin:22pt}body{margin:0;font:8pt/10.5pt Helvetica}.products{display:grid;grid-template-columns:1fr 1fr;gap:8pt}.product{border:1pt solid #cbd5c0;padding:7pt}.wide{grid-column:span 2;display:flex;gap:10pt}.swatch{height:48pt}.wide .swatch{width:72pt;height:55pt;flex:none}.order{break-before:page}</style><div class="products"><article class="product"><div class="swatch">NP / 01</div><h2>ITEM 01</h2><p>cup</p><b>$28</b></article><article class="product"><div class="swatch">NP / 02</div><h2>ITEM 02</h2><p>bowl</p><b>$46</b></article><article class="product wide"><div class="swatch">NP / 07</div><div><h2>ITEM 07 · HARBOR THROW</h2><p>Recycled wool, woven in a small coastal mill. Finished with a brushed fringe and presented in a reusable cotton bag.</p><span>$148</span></div></article><article class="product"><div class="swatch">NP / 08</div><h2>ITEM 08</h2><p>board</p><b>$72</b></article></div><section class="order">ORDER DETAILS</section>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = root.Page.Width, root.Page.Height, root.Page.Margin
	opts.Faces, opts.FontVariants = root.Fonts, root.FontVariants
	pages := Flow(root, opts)
	locations := map[string][]int{}
	titleX := math.NaN()
	for pageIndex, page := range pages {
		var pageText strings.Builder
		for _, block := range page.Blocks {
			if strings.Contains(block.Text, "ITEM 07") {
				titleX = block.X
			}
			pageText.WriteString(block.Text)
			for _, line := range block.Runs {
				for _, run := range line {
					pageText.WriteString(run.Text)
				}
			}
		}
		for _, marker := range []string{"NP / 07", "ITEM 07", "reusable cotton bag", "$148"} {
			if strings.Contains(pageText.String(), marker) {
				locations[marker] = append(locations[marker], pageIndex+1)
			}
		}
	}
	for _, marker := range []string{"NP / 07", "ITEM 07", "reusable cotton bag", "$148"} {
		if len(locations[marker]) == 0 {
			t.Errorf("the PDF does not contain the text %q", marker)
		}
	}
	for marker, markerPages := range locations {
		for _, page := range markerPages {
			if page != 1 {
				t.Errorf("%q appears on page %d; Chromium keeps complete the row that fits on the page 1", marker, page)
			}
		}
	}
	if math.IsNaN(titleX) || titleX < 100 || titleX > 125 {
		t.Errorf("card title at x=%.2f pt; Chromium places it near x=111.5 pt inside the Flex item", titleX)
	}
}

func TestLayout_GridItemWithBlockChildrenContinuesAtContentBreaks(t *testing.T) {
	html := `<div style="height:65pt">BEFORE</div><div style="display:grid;grid-template-columns:180pt 24pt;align-items:start"><section style="padding:4pt;border:1pt solid #123456;background:#eaf3ff;font-size:9pt;line-height:10.5pt"><p style="margin:0 0 3pt">P1: A1 alpha A2 bravo A3 charlie A4 delta A5 echo A6 foxtrot A7 golf A8 hotel A9 india A10 juliet A11 kilo A12 lima A13 mike</p><p style="margin:0">P2: A14 november A15 oscar A16 papa A17 quebec A18 romeo A19 sierra A20 tango A21 uniform A22 victor A23 whiskey A24 xray A25 yankee A26 zulu</p></section><div style="font-size:9pt;line-height:10.5pt">SIDE</div></div><p style="margin:0">AFTER GRID</p>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 225, 135, 9
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages=%d, want 2 as in Chromium", len(pages))
	}
	pageText := make([]string, len(pages))
	for pageIndex, page := range pages {
		var text strings.Builder
		for _, block := range page.Blocks {
			if len(block.Runs) == 0 {
				text.WriteString(block.Text)
			}
			for _, line := range block.Runs {
				for _, run := range line {
					text.WriteString(run.Text)
					text.WriteByte(' ')
				}
			}
		}
		pageText[pageIndex] = text.String()
	}
	if !strings.Contains(pageText[0], "P1:") || !strings.Contains(pageText[0], "SIDE") {
		t.Errorf("grid row cells must start in the remaining space on page 1: %q", pageText[0])
	}
	if !strings.Contains(pageText[1], "P2:") || !strings.Contains(pageText[1], "AFTER GRID") {
		t.Errorf("the cell continuation and following block must continue on page 2: %q", pageText[1])
	}
}

func TestLayout_GridFragmentsTextItemSpanningAutoRows(t *testing.T) {
	content := "A1 alpha A2 bravo A3 charlie A4 delta A5 echo A6 foxtrot A7 golf A8 hotel A9 india A10 juliet A11 kilo A12 lima A13 mike A14 november A15 oscar A16 papa A17 quebec A18 romeo A19 sierra A20 tango A21 uniform A22 victor A23 whiskey A24 xray A25 yankee A26 zulu"
	root, err := boxes.Build(`<div style="height:75pt">BEFORE</div><div style="display:grid;grid-template-columns:150pt;grid-auto-rows:auto"><div style="grid-row:span 2;font-size:9pt;line-height:10.5pt">` + content + `</div><div>AFTER GRID</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 225, 135, 9
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages=%d, want 2", len(pages))
	}
	itemPages := make(map[int]bool)
	var rendered strings.Builder
	afterPage := 0
	for pageIndex, page := range pages {
		for _, block := range page.Blocks {
			if block.FontSize == 9 && len(block.Runs) > 0 {
				itemPages[pageIndex+1] = true
				for _, line := range block.Runs {
					for _, run := range line {
						rendered.WriteString(run.Text)
						rendered.WriteByte(' ')
					}
				}
			}
			if strings.Contains(block.Text, "AFTER GRID") {
				afterPage = pageIndex + 1
			}
		}
	}
	if !itemPages[1] || !itemPages[2] {
		t.Errorf("the item spanning two rows must fragment across both pages; pages=%v", itemPages)
	}
	if got, want := strings.Join(strings.Fields(rendered.String()), " "), strings.Join(strings.Fields(content), " "); got != want {
		t.Errorf("fragmented runs lost or repeated text: got %q want %q", got, want)
	}
	if afterPage != 2 {
		t.Errorf("AFTER GRID appears on page %d; must follow the fragmented item on page 2", afterPage)
	}
}

func TestLayout_GridBreakInsideAvoidMovesItemThatFitsFreshPage(t *testing.T) {
	content := "A1 alpha A2 bravo A3 charlie A4 delta A5 echo A6 foxtrot A7 golf A8 hotel A9 india A10 juliet A11 kilo A12"
	root, err := boxes.Build(`<div style="height:90pt">BEFORE</div><div style="display:grid;grid-template-columns:150pt 50pt;grid-auto-rows:auto"><div style="font:9pt/10.5pt Helvetica;break-inside:avoid">` + content + `</div><div>SIDE</div><div>AFTER GRID</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 225, 135, 9
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages=%d, want 2", len(pages))
	}
	var itemPages []int
	afterPage := 0
	sidePage := 0
	var itemText strings.Builder
	for pageIndex, page := range pages {
		for _, block := range page.Blocks {
			if strings.Contains(block.Text, "A1") {
				itemPages = append(itemPages, pageIndex+1)
				itemText.WriteString(block.Text)
			}
			if strings.Contains(block.Text, "AFTER GRID") {
				afterPage = pageIndex + 1
			}
			if strings.Contains(block.Text, "SIDE") {
				sidePage = pageIndex + 1
			}
		}
	}
	if len(itemPages) == 0 || itemPages[0] != 2 {
		t.Errorf("Grid item with break-inside:avoid starts on page %v; must move complete to page 2", itemPages)
	}
	if got, want := strings.Join(strings.Fields(itemText.String()), " "), content; got != want {
		t.Errorf("item Grid with avoid was lost or duplicated text: got %q want %q", got, want)
	}
	if afterPage != 2 {
		t.Errorf("AFTER GRID appears on page %d; must follow when item complete on page 2", afterPage)
	}
	if sidePage != 2 {
		t.Errorf("the item SIDE of the row Grid appears on page %d; must remain with the row on page 2", sidePage)
	}
}

// A two-column grid with four
// children forms a 2x2 grid with equal columns, no mixed fr/px units, no gap,
// and no explicit placement (row auto-flow only).
func TestLayout_GridPlacesRowsAndColumns(t *testing.T) {
	root, err := boxes.Build(`<div style="display: grid; grid-template-columns: 1fr 1fr"><span>A</span><span>B</span><span>C</span><span>D</span></div>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pages := Flow(root, testOptions())
	if len(pages) != 1 {
		t.Fatalf("pages: got %d, want 1", len(pages))
	}
	blocks := pages[0].Blocks
	if len(blocks) != 4 {
		t.Fatalf("blocks: got %d, want 4", len(blocks))
	}
	if blocks[0].X != 72 || blocks[1].X != 297.5 || blocks[2].X != 72 || blocks[3].X != 297.5 {
		t.Errorf("columns: got %v %v %v %v, want 72 297.5 72 297.5",
			blocks[0].X, blocks[1].X, blocks[2].X, blocks[3].X)
	}
	if blocks[0].Y != 72 || blocks[1].Y != 72 {
		t.Errorf("row 1 y: got %v %v, want 72 72", blocks[0].Y, blocks[1].Y)
	}
	if blocks[2].Y <= blocks[0].Y || blocks[3].Y != blocks[2].Y {
		t.Errorf("row 2 y: got %v %v", blocks[2].Y, blocks[3].Y)
	}
}

func TestLayout_GridItemsPreserveNestedBlockFormatting(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;grid-template-columns:1fr 1fr;gap:8pt"><article style="padding:6pt;border:1pt solid #222;background-color:#f0f0f0"><div style="height:24pt;background-color:#dddddd">PHOTO</div><div><b>PRODUCT NAME</b><p style="color:#123456">Product description</p></div></article><article>SECOND CARD</article></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	texts := map[string]PlacedBox{}
	runs := map[string]TextRun{}
	var rendered strings.Builder
	boldProduct := false
	for _, block := range blocks {
		for _, line := range block.Runs {
			for _, run := range line {
				texts[run.Text] = block
				runs[run.Text] = run
				rendered.WriteString(run.Text)
				rendered.WriteByte(' ')
				if run.Text == "PRODUCT" && run.Bold {
					boldProduct = true
				}
			}
		}
	}
	for _, want := range []string{"PHOTO", "PRODUCT", "NAME", "Product", "description"} {
		if _, ok := texts[want]; !ok {
			t.Errorf("missing run %q; Grid items must preserve their descendants", want)
		}
	}
	if !strings.Contains(rendered.String(), "SECOND CARD") {
		t.Errorf("second Grid item is missing: %q", rendered.String())
	}
	if photo, ok := texts["PHOTO"]; ok && (!photo.HasBg || photo.X != 79 || photo.Y <= 72) {
		t.Errorf("the blockdecorated PHOTO lost background/position: %+v", photo)
	}
	if photo, ok := runs["PHOTO"]; ok && (photo.X < 79 || photo.Y <= 72) {
		t.Errorf("descendant text did not move with its cell: x=%v y=%v", photo.X, photo.Y)
	}
	if !boldProduct {
		t.Error("the product label lost its bold style")
	}
}

func TestLayout_GridItemPreservesNestedSVGAndCaption(t *testing.T) {
	root, err := boxes.Build(`<style>.art svg{width:100%;height:100%}</style><div style="display:grid;grid-template-columns:1fr"><article><div class="art" style="height:130pt;background:#f1f5f9"><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 300 160"><rect width="300" height="160" fill="#2563eb"/></svg></div><p style="margin:0">SVG CAPTION</p></article></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var hasSVG, hasCaption bool
	for _, block := range Flow(root, testOptions())[0].Blocks {
		hasSVG = hasSVG || block.SVG != nil
		if block.SVG != nil && (block.Width == 0 || math.Abs(block.Height-130) > 0.01) {
			t.Errorf("SVG with height:100%% must resolve the container height of 130pt; got %v×%v", block.Width, block.Height)
		}
		for _, line := range block.Runs {
			for _, run := range line {
				if run.SVG != nil {
					hasSVG = true
					if run.Width == 0 || math.Abs(run.SVGHeight-130) > 0.01 {
						t.Errorf("inline SVG with height:100%% must resolve the container height of 130pt; got %v×%v", run.Width, run.SVGHeight)
					}
				}
			}
		}
		hasCaption = hasCaption || strings.Contains(strings.Join(runLines(block), " "), "SVG CAPTION")
	}
	if !hasSVG || !hasCaption {
		t.Fatalf("the item Grid must preserve SVG and caption; svg=%v caption=%v", hasSVG, hasCaption)
	}
}

func TestLayout_GridNestedItemMovesIntactWhenItDoesNotFit(t *testing.T) {
	root, err := boxes.Build(`<div style="height:100pt">BEFORE</div><div style="display:grid;grid-template-columns:1fr"><article style="background-color:#eeeeee;padding:4pt"><p style="margin:0">FIRST PARAGRAPH</p><p style="margin:0">SECOND PARAGRAPH</p></article></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 225, 135, 9
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages=%d, want 2", len(pages))
	}
	for pageIndex, page := range pages {
		for _, block := range page.Blocks {
			for _, line := range block.Runs {
				for _, run := range line {
					if strings.Contains(run.Text, "PARAGRAPH") && pageIndex != 1 {
						t.Errorf("nested item content was fragmented on page %d", pageIndex+1)
					}
				}
			}
		}
	}
	var pageTwoText strings.Builder
	for _, block := range pages[1].Blocks {
		pageTwoText.WriteString(strings.Join(runLines(block), " "))
		pageTwoText.WriteByte(' ')
	}
	if !strings.Contains(pageTwoText.String(), "FIRST PARAGRAPH") || !strings.Contains(pageTwoText.String(), "SECOND PARAGRAPH") {
		t.Errorf(" descendants were lost when moving the item: %q", pageTwoText.String())
	}
}

func TestLayout_GridGapsSeparateTracksAndRows(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;grid-template-columns:1fr 1fr;gap:4px 8px"><span>A</span><span>B</span><span>C</span><span>D</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 4 {
		t.Fatalf("grid items: got %d, want 4", len(blocks))
	}
	firstWidth := blocks[0].Width
	if !approx(firstWidth, blocks[1].Width) || !approx(blocks[0].X, blocks[2].X) || !approx(blocks[1].X, blocks[3].X) {
		t.Errorf("two equal tracks should share x/width across rows: first=%+v second=%+v", blocks[:2], blocks[2:])
	}
	if !approx(blocks[1].X-blocks[0].X-firstWidth, 6) {
		t.Errorf("horizontal gap=%g, want 6pt from 8px", blocks[1].X-blocks[0].X-firstWidth)
	}
	if !approx(blocks[2].Y-blocks[0].Y-blocks[0].Height, 3) || !approx(blocks[3].Y-blocks[1].Y-blocks[1].Height, 3) {
		t.Errorf("vertical row gaps=%g/%g, want 3pt from 4px", blocks[2].Y-blocks[0].Y-blocks[0].Height, blocks[3].Y-blocks[1].Y-blocks[1].Height)
	}
	if !approx(firstWidth*2+6, 451) {
		t.Errorf("track widths and gap occupy %gpt, want available width 451pt", firstWidth*2+6)
	}
}

func TestLayout_GridDistributesFreeSpaceAcrossFractionalTracks(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid; grid-template-columns:100px 1fr 2fr; column-gap:8px"><span>A</span><span>B</span><span>C</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 3 {
		t.Fatalf("boxes grid: got %d, want 3", len(blocks))
	}
	wantX := []float64{72, 153, 280 + 1.0/3}
	wantWidth := []float64{75, 364.0 / 3, 728.0 / 3}
	for i, block := range blocks {
		if !approx(block.X, wantX[i]) || !approx(block.Width, wantWidth[i]) {
			t.Errorf("track %d: x=%v width=%v, want x=%v width=%v", i, block.X, block.Width, wantX[i], wantWidth[i])
		}
	}
}

func TestLayout_GridResolvesPercentTracksAgainstContainerWidth(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid; grid-template-columns:25% 1fr; column-gap:8px"><span>A</span><span>B</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("boxes grid: got %d, want 2", len(blocks))
	}
	if !approx(blocks[0].Width, 112.75) || !approx(blocks[0].X, 72) {
		t.Errorf("percentage track: x=%v width=%v, want x=72 width=112.75", blocks[0].X, blocks[0].Width)
	}
	if !approx(blocks[1].Width, 332.25) || !approx(blocks[1].X, 190.75) {
		t.Errorf("track fr: x=%v width=%v, want x=190.75 width=332.25", blocks[1].X, blocks[1].Width)
	}
}

func TestLayout_GridTemplateColumnsPercentMinmaxTracksRespectMinimums(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;width:250pt;grid-template-columns:100pt minmax(45%,1fr) minmax(0%,2fr);grid-template-rows:20pt;column-gap:8pt"><span>A</span><span>B</span><span>C</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth = 394
	blocks := Flow(root, opts)[0].Blocks
	wantX, wantWidth := []float64{72, 180, 300.5}, []float64{100, 112.5, 21.5}
	if len(blocks) != len(wantX) {
		t.Fatalf("blocks: got %d, want %d", len(blocks), len(wantX))
	}
	for i := range blocks {
		if !approx(blocks[i].X, wantX[i]) || !approx(blocks[i].Width, wantWidth[i]) {
			t.Errorf("block %s: x/width=%.5f/%.5f, want %.5f/%.5f", blocks[i].Text, blocks[i].X, blocks[i].Width, wantX[i], wantWidth[i])
		}
	}
}

func TestLayout_GridMinmaxTrackKeepsMinimumWhenFrSpaceIsSmall(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid; grid-template-columns:minmax(100px, 1fr) 1fr"><span>A</span><span>B</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth = 200
	opts.Margin = 50
	blocks := Flow(root, opts)[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("boxes grid: got %d, want 2", len(blocks))
	}
	if !approx(blocks[0].X, 50) || !approx(blocks[0].Width, 75) {
		t.Errorf("track minmax: x=%v width=%v, want x=50 width=75", blocks[0].X, blocks[0].Width)
	}
	if !approx(blocks[1].X, 125) || !approx(blocks[1].Width, 25) {
		t.Errorf("track fr: x=%v width=%v, want x=125 width=25", blocks[1].X, blocks[1].Width)
	}
}

func TestLayout_GridFixedZeroLengthRowDoesNotGrow(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;grid-template-columns:100pt;grid-template-rows:20pt 0pt 30pt;row-gap:8pt"><span>A</span><span>B</span><span>C</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	wantY := []float64{72, 100, 108}
	if len(blocks) != len(wantY) {
		t.Fatalf("blocks: got %d, want %d", len(blocks), len(wantY))
	}
	for i, want := range wantY {
		if !approx(blocks[i].Y, want) {
			t.Errorf("block %d y: got %.2f, want %.2f", i, blocks[i].Y, want)
		}
	}
}

func TestLayout_GridFixedRowsSetRowPositions(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid; grid-template-columns:1fr 1fr; grid-template-rows:30pt 40pt; row-gap:5pt"><span>A</span><span>B</span><span>C</span><span>D</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 4 {
		t.Fatalf("boxes grid: got %d, want 4", len(blocks))
	}
	if !approx(blocks[0].Y, 72) || !approx(blocks[1].Y, 72) || !approx(blocks[2].Y, 107) || !approx(blocks[3].Y, 107) {
		t.Errorf("rows: y=[%v %v %v %v], want [72 72 107 107]", blocks[0].Y, blocks[1].Y, blocks[2].Y, blocks[3].Y)
	}
}

func TestLayout_GridAutoRowsZeroPercentIsFixedTrack(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;grid-template-columns:100pt;grid-template-rows:20pt;grid-auto-rows:0%;row-gap:8pt"><span>A</span><span>B</span><span>C</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 3 {
		t.Fatalf("blocks: got %d, want 3", len(blocks))
	}
	wantY := []float64{72, 100, 108}
	for i, want := range wantY {
		if !approx(blocks[i].Y, want) {
			t.Errorf("block %d y: got %.2f, want %.2f", i, blocks[i].Y, want)
		}
	}
}

func TestLayout_GridAutoRowsZeroLengthIsFixedTrack(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;grid-template-columns:100pt;grid-template-rows:20pt;grid-auto-rows:0pt;row-gap:8pt"><span>A</span><span>B</span><span>C</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	wantY := []float64{72, 100, 108}
	if len(blocks) != len(wantY) {
		t.Fatalf("blocks: got %d, want %d", len(blocks), len(wantY))
	}
	for i, want := range wantY {
		if !approx(blocks[i].Y, want) {
			t.Errorf("block %d y: got %.2f, want %.2f", i, blocks[i].Y, want)
		}
	}
}

func TestLayout_GridAutoRowsSetsImplicitRowHeight(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid; grid-template-columns:1fr; grid-auto-rows:24pt; row-gap:5pt"><span>A</span><span>B</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("boxes grid: got %d, want 2", len(blocks))
	}
	if !approx(blocks[0].Height, 24) || !approx(blocks[1].Height, 24) || !approx(blocks[1].Y, 101) {
		t.Errorf("rows implicits:heights=%v,%v Y2=%v, want 24,24,101", blocks[0].Height, blocks[1].Height, blocks[1].Y)
	}
}

func TestLayout_GridAutoRowsRepeatsTrackSizeList(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid; grid-template-columns:1fr; grid-auto-rows:24pt 36pt; row-gap:5pt"><span>A</span><span>B</span><span>C</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 3 {
		t.Fatalf("boxes grid: got %d, want 3", len(blocks))
	}
	wantHeights := []float64{24, 36, 24}
	wantY := []float64{72, 101, 142}
	for i, block := range blocks {
		if !approx(block.Height, wantHeights[i]) || !approx(block.Y, wantY[i]) {
			t.Errorf("row %d:height=%v y=%v, wantheight=%v y=%v", i, block.Height, block.Y, wantHeights[i], wantY[i])
		}
	}
}

func TestLayout_GridAutoFlowDenseBackfillsEarlierCell(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid; grid-template-columns:1fr 1fr 1fr; grid-auto-flow:row dense"><span style="grid-column:3">A</span><span style="grid-column:1">B</span><span>C</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 3 {
		t.Fatalf("boxes grid: got %d, want 3", len(blocks))
	}
	for _, block := range blocks {
		if block.Text == "C" {
			if !approx(block.X, 222.33333333333334) || !approx(block.Y, 72) {
				t.Errorf("item auto: x=%v y=%v, want the free space in the first row x=222.333 y=72", block.X, block.Y)
			}
			return
		}
	}
	t.Fatal("block not found: C")
}

func TestLayout_GridAutoFlowColumnPlacesTopToBottom(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid; grid-template-columns:1fr 1fr; grid-template-rows:20pt 30pt; grid-auto-flow:column; gap:5pt 8pt"><span>A</span><span>B</span><span>C</span><span>D</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	want := map[string][4]float64{
		"A": {72, 72, 221.5, 20},
		"B": {72, 97, 221.5, 30},
		"C": {301.5, 72, 221.5, 20},
		"D": {301.5, 97, 221.5, 30},
	}
	if len(blocks) != len(want) {
		t.Fatalf("boxes grid: got %d, want %d", len(blocks), len(want))
	}
	for _, block := range blocks {
		position, ok := want[block.Text]
		if !ok {
			t.Errorf("block unexpected %q", block.Text)
			continue
		}
		got := [4]float64{block.X, block.Y, block.Width, block.Height}
		for i := range got {
			if !approx(got[i], position[i]) {
				t.Errorf("block %s: geometry %v, want %v", block.Text, got, position)
				break
			}
		}
	}
}

func TestLayout_GridAutoColumnsZeroPercentIsFixedTrack(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;width:451pt;grid-template-columns:100pt;grid-template-rows:20pt;grid-auto-flow:column;grid-auto-columns:0%;column-gap:8pt"><span>A</span><img src="` + imgURI(t, 200, 100) + `"><span>C</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 3 {
		t.Fatalf("blocks: got %d, want 3", len(blocks))
	}
	if !approx(blocks[2].X, 188) {
		t.Errorf("third block X: got %.2f, want 188", blocks[2].X)
	}
}

func TestLayout_GridAutoColumnsZeroLengthIsFixedTrack(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;width:451pt;grid-template-columns:100pt;grid-template-rows:20pt;grid-auto-flow:column;grid-auto-columns:0pt;column-gap:8pt"><span>A</span><img src="` + imgURI(t, 200, 100) + `"><span>C</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 3 {
		t.Fatalf("blocks: got %d, want 3", len(blocks))
	}
	if !approx(blocks[2].X, 188) {
		t.Errorf("third block X: got %.2f, want 188", blocks[2].X)
	}
}

func TestLayout_GridAutoColumnsResolvesPercentTracksAgainstContainerWidth(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;width:451pt;grid-template-columns:100pt;grid-template-rows:20pt;grid-auto-flow:column;grid-auto-columns:50% 25%;column-gap:8pt"><span>A</span><span>B</span><span>C</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	wantX := []float64{72, 180, 413.5}
	if len(blocks) != len(wantX) {
		t.Fatalf("blocks: got %d, want %d", len(blocks), len(wantX))
	}
	for i, want := range wantX {
		if !approx(blocks[i].X, want) {
			t.Errorf("block %d X: got %.2f, want %.2f", i, blocks[i].X, want)
		}
	}
}

func TestLayout_GridAutoColumnsDistributesImplicitFractionTracks(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;width:451pt;grid-template-columns:100pt;grid-template-rows:20pt;grid-auto-flow:column;grid-auto-columns:1fr 2fr;column-gap:8pt"><span>A</span><span>B</span><span>C</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	frUnit := (451.0 - 100 - 16) / 3
	wantX := []float64{72, 180, 180 + frUnit + 8}
	wantWidth := []float64{100, frUnit, 2 * frUnit}
	if len(blocks) != len(wantX) {
		t.Fatalf("blocks: got %d, want %d", len(blocks), len(wantX))
	}
	for i := range blocks {
		if !approx(blocks[i].X, wantX[i]) || !approx(blocks[i].Width, wantWidth[i]) {
			t.Errorf("block %s: x/width=%.5f/%.5f, want %.5f/%.5f", blocks[i].Text, blocks[i].X, blocks[i].Width, wantX[i], wantWidth[i])
		}
	}
}

func TestLayout_GridAutoColumnsFractionTracksRespectMinmaxMinimums(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;width:250pt;grid-template-columns:100pt;grid-template-rows:20pt;grid-auto-flow:column;grid-auto-columns:minmax(100pt,1fr) minmax(0pt,2fr);column-gap:8pt"><span>A</span><span>B</span><span>C</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth = 394 // usable width: 250pt
	blocks := Flow(root, opts)[0].Blocks
	wantX := []float64{72, 180, 288}
	wantWidth := []float64{100, 100, 34}
	if len(blocks) != len(wantX) {
		t.Fatalf("blocks: got %d, want %d", len(blocks), len(wantX))
	}
	for i := range blocks {
		if !approx(blocks[i].X, wantX[i]) || !approx(blocks[i].Width, wantWidth[i]) {
			t.Errorf("block %s: x/width=%.5f/%.5f, want %.5f/%.5f", blocks[i].Text, blocks[i].X, blocks[i].Width, wantX[i], wantWidth[i])
		}
	}
}

func TestLayout_GridAutoColumnsMinmaxMinimumsShareFrSpaceWithExplicitTracks(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;width:250pt;grid-template-columns:minmax(80pt,1fr);grid-template-rows:20pt;grid-auto-flow:column;grid-auto-columns:minmax(100pt,1fr) minmax(0pt,2fr);column-gap:8pt"><span>A</span><span>B</span><span>C</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth = 394
	blocks := Flow(root, opts)[0].Blocks
	wantX := []float64{72, 160, 268}
	wantWidth := []float64{80, 100, 54}
	if len(blocks) != len(wantX) {
		t.Fatalf("blocks: got %d, want %d", len(blocks), len(wantX))
	}
	for i := range blocks {
		if !approx(blocks[i].X, wantX[i]) || !approx(blocks[i].Width, wantWidth[i]) {
			t.Errorf("block %s: x/width=%.5f/%.5f, want %.5f/%.5f", blocks[i].Text, blocks[i].X, blocks[i].Width, wantX[i], wantWidth[i])
		}
	}
}

func TestLayout_GridAutoColumnsPercentMinmaxTracksRespectMinimums(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;width:250pt;grid-template-columns:100pt;grid-template-rows:20pt;grid-auto-flow:column;grid-auto-columns:minmax(45%,1fr) minmax(0%,2fr);column-gap:8pt"><span>A</span><span>B</span><span>C</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth = 394
	blocks := Flow(root, opts)[0].Blocks
	wantX := []float64{72, 180, 300.5}
	wantWidth := []float64{100, 112.5, 21.5}
	if len(blocks) != len(wantX) {
		t.Fatalf("blocks: got %d, want %d", len(blocks), len(wantX))
	}
	for i := range blocks {
		if !approx(blocks[i].X, wantX[i]) || !approx(blocks[i].Width, wantWidth[i]) {
			t.Errorf("block %s: x/width=%.5f/%.5f, want %.5f/%.5f", blocks[i].Text, blocks[i].X, blocks[i].Width, wantX[i], wantWidth[i])
		}
	}
}

func TestLayout_GridAutoColumnsFractionTracksRespectAutomaticMinimum(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;width:250pt;grid-template-columns:100pt;grid-template-rows:20pt;grid-auto-flow:column;grid-auto-columns:1fr 1fr;column-gap:8pt"><span>A</span><img src="` + imgURI(t, 200, 100) + `"><span>C</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth = 394
	blocks := Flow(root, opts)[0].Blocks
	if len(blocks) != 3 {
		t.Fatalf("blocks: got %d, want 3", len(blocks))
	}
	if !approx(blocks[2].X, 338) {
		t.Errorf("third block X: got %.2f, want 338", blocks[2].X)
	}
	if blocks[2].Width <= 0 {
		t.Errorf("width of the third block: got %.2f, want > 0 from its automatic minimum", blocks[2].Width)
	}
}

func TestLayout_GridTemplateFractionTracksRespectAutomaticMinimum(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;width:250pt;grid-template-columns:1fr 1fr;grid-template-rows:20pt;column-gap:8pt"><img src="` + imgURI(t, 200, 100) + `"><span>C</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth = 394
	blocks := Flow(root, opts)[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("blocks: got %d, want 2", len(blocks))
	}
	if !approx(blocks[1].X, 230) {
		t.Errorf("second block X: got %.2f, want 230", blocks[1].X)
	}
	if !approx(blocks[1].Width, 92) {
		t.Errorf("width of the second block: got %.2f, want 92", blocks[1].Width)
	}
}

func TestLayout_GridAutomaticMinimumIncludesItemBoxInsets(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;width:20pt;grid-template-columns:1fr 1fr"><span style="padding:2pt;border:1pt solid #000">ABCD</span><span>Z</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.Measure = func(text string, size float64) float64 {
		if text == "ABCD" {
			return 40
		}
		return float64(len([]rune(text))) * size * 0.5
	}
	blocks := Flow(root, opts)[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("blocks=%d, want two griditems", len(blocks))
	}
	if got, want := blocks[0].Width, 46.0; !approx(got, want) {
		t.Errorf("automatic minimum width=%gpt, want %gpt (40pt min-content + 6pt of padding/border)", got, want)
	}
}

func TestLayout_GridFractionAutomaticMinimumRespectsNoWrap(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;width:145pt;grid-template-columns:1fr 1fr;grid-template-rows:20pt;column-gap:8pt"><img style="width:100px" src="` + imgURI(t, 200, 100) + `"><span style="white-space:nowrap;font-family:monospace;font-size:12pt">alpha beta</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth = 289
	opts.Measure = func(text string, _ float64) float64 {
		switch text {
		case "alpha beta":
			return 72
		case "alpha":
			return 36
		case "beta":
			return 28
		default:
			return 0
		}
	}
	blocks := Flow(root, opts)[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("blocks: got %d, want 2", len(blocks))
	}
	if !approx(blocks[1].X, 155) || !approx(blocks[1].Width, 72) {
		t.Errorf("second block: x=%.2f width=%.2f, want x=155 width=72", blocks[1].X, blocks[1].Width)
	}
}

func TestLayout_GridAutoFractionAutomaticMinimumRespectsNoWrap(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;width:145pt;grid-template-columns:75pt;grid-template-rows:20pt;grid-auto-flow:column;grid-auto-columns:1fr;column-gap:8pt"><img style="width:100px" src="` + imgURI(t, 200, 100) + `"><span style="white-space:nowrap;font-family:monospace;font-size:12pt">alpha beta</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth = 289
	opts.Measure = func(text string, _ float64) float64 {
		switch text {
		case "alpha beta":
			return 72
		case "alpha":
			return 36
		case "beta":
			return 28
		default:
			return 0
		}
	}
	blocks := Flow(root, opts)[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("blocks: got %d, want 2", len(blocks))
	}
	if !approx(blocks[1].X, 155) || !approx(blocks[1].Width, 72) {
		t.Errorf("second block: x=%.2f width=%.2f, want x=155 width=72", blocks[1].X, blocks[1].Width)
	}
}

func TestLayout_GridFractionAutomaticMinimumRespectsPre(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;width:145pt;grid-template-columns:1fr 1fr;grid-template-rows:20pt;column-gap:8pt"><img style="width:100px" src="` + imgURI(t, 200, 100) + `"><span style="white-space:pre;font-family:monospace;font-size:12pt">alpha beta</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth = 289
	opts.Measure = func(text string, _ float64) float64 {
		switch text {
		case "alpha beta":
			return 72
		case "alpha":
			return 36
		case "beta":
			return 28
		default:
			return 0
		}
	}
	blocks := Flow(root, opts)[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("blocks: got %d, want 2", len(blocks))
	}
	if !approx(blocks[1].X, 155) || !approx(blocks[1].Width, 72) {
		t.Errorf("second block: x=%.2f width=%.2f, want x=155 width=72", blocks[1].X, blocks[1].Width)
	}
}

func TestLayout_GridFractionAutomaticMinimumExcludesHangingPreWrapSpaces(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;width:145pt;grid-template-columns:1fr 1fr;grid-template-rows:20pt;column-gap:8pt"><img style="width:100px" src="` + imgURI(t, 200, 100) + `"><span style="white-space:pre-wrap;font-family:monospace;font-size:12pt">alpha    beta</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth = 289
	opts.Measure = func(text string, _ float64) float64 {
		switch text {
		case "alpha    ":
			return 64.8
		case "alpha":
			return 36
		case "beta":
			return 28
		default:
			return 0
		}
	}
	blocks := Flow(root, opts)[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("blocks: got %d, want 2", len(blocks))
	}
	if !approx(blocks[1].Width, 62) {
		t.Errorf("width of the second block: got %.2f, want 62 excluding the preceding column gap", blocks[1].Width)
	}
}

func TestLayout_GridFractionAutomaticMinimumAllowsCJKLineBreaks(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;width:83pt;grid-template-columns:1fr 1fr;grid-template-rows:20pt;column-gap:8pt"><img style="width:100px" src="` + imgURI(t, 200, 100) + `"><span style="font-family:monospace;font-size:12pt">漢字</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth = 227
	opts.Measure = func(text string, _ float64) float64 {
		switch text {
		case "漢字":
			return 24
		case "漢", "字":
			return 12
		default:
			return 0
		}
	}
	blocks := Flow(root, opts)[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("blocks: got %d, want 2", len(blocks))
	}
	if !approx(blocks[1].Width, 12) {
		t.Errorf("width of the second block: got %.2f, want 12 from the CJK glyph width", blocks[1].Width)
	}
}

func TestLayout_GridAutomaticMinimumMeasuresStyledInlineDescendants(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;width:130pt;grid-template-columns:1fr 1fr"><div><span style="font-size:24pt">WIDELONGTOKEN</span></div><div>SMALL</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.Measure = func(value string, size float64) float64 { return float64(len([]rune(value))) * size / 2 }
	pages := Flow(root, opts)
	if len(pages) != 1 {
		t.Fatalf("pages=%d, want 1", len(pages))
	}
	item := root.Children[0].Children[0]
	words := buildInlineStream(item, opts)
	minimum := words[0].w + horizontalInsets(item)
	for _, block := range pages[0].Blocks {
		if block.source != item {
			continue
		}
		if block.Width+fragmentEpsilon < minimum {
			t.Errorf("automatic track minimum=%gpt, smaller than the measured token with its style (%gpt)", block.Width, minimum)
		}
		return
	}
	t.Fatal("the first grid item box was not found")
}

func TestLayout_GridAutoColumnsSizesImplicitColumnTracks(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid; grid-template-columns:1fr 1fr; grid-template-rows:20pt 30pt; grid-auto-flow:column; grid-auto-columns:50pt 60pt; column-gap:8pt"><span>A</span><span>B</span><span>C</span><span>D</span><span>E</span><span>F</span><span>G</span><span>H</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	want := map[string][3]float64{
		"A": {72, 72, 158.5},
		"B": {72, 92, 158.5},
		"C": {238.5, 72, 158.5},
		"D": {238.5, 92, 158.5},
		"E": {405, 72, 50},
		"F": {405, 92, 50},
		"G": {463, 72, 60},
		"H": {463, 92, 60},
	}
	seen := make(map[string]bool, len(want))
	for _, block := range blocks {
		if position, ok := want[block.Text]; ok {
			seen[block.Text] = true
			got := [3]float64{block.X, block.Y, block.Width}
			for i := range got {
				if !approx(got[i], position[i]) {
					t.Errorf("block %s: geometry %v, want %v", block.Text, got, position)
					break
				}
			}
		}
	}
	for text := range want {
		if !seen[text] {
			t.Errorf("block not found: %s", text)
		}
	}
}

func TestLayout_GridAutoColumnsSizesAutomaticTrackFromContent(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid; grid-template-columns:1fr 1fr; grid-template-rows:20pt 30pt; grid-auto-flow:column; grid-auto-columns:auto 60pt; column-gap:8pt"><span>A</span><span>B</span><span>C</span><span>D</span><span>E</span><span>F</span><span>G</span><span>H</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	fontWidths := map[string]float64{"E": 8.004, "F": 6.672, "G": 9.334, "H": 8.666}
	opts := testOptions()
	opts.Measure = func(text string, _ float64) float64 { return fontWidths[text] }
	blocks := Flow(root, opts)[0].Blocks
	want := map[string][3]float64{
		"A": {72, 72, 179.498},
		"B": {72, 92, 179.498},
		"C": {259.498, 72, 179.498},
		"D": {259.498, 92, 179.498},
		"E": {446.996, 72, 8.004},
		"F": {446.996, 92, 8.004},
		"G": {463, 72, 60},
		"H": {463, 92, 60},
	}
	seen := make(map[string]bool, len(want))
	for _, block := range blocks {
		if position, ok := want[block.Text]; ok {
			seen[block.Text] = true
			got := [3]float64{block.X, block.Y, block.Width}
			for i := range got {
				if !approx(got[i], position[i]) {
					t.Errorf("block %s: geometry %v, want %v", block.Text, got, position)
					break
				}
			}
		}
	}
	for text := range want {
		if !seen[text] {
			t.Errorf("block not found: %s", text)
		}
	}
}

func TestLayout_GridAutoColumnsSizesAutomaticTrackFromImageIntrinsicWidth(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;grid-template-columns:100pt;grid-template-rows:20pt 30pt;grid-auto-flow:column;grid-auto-columns:auto;column-gap:8pt"><span>A</span><span>B</span><img src="` + imgURI(t, 200, 100) + `"><span>C</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth = 252
	opts.Margin = 72
	blocks := Flow(root, opts)[0].Blocks
	seenImage, seenContent := false, false
	for _, block := range blocks {
		if block.Image != nil {
			seenImage = true
			if !approx(block.X, 180) || !approx(block.Width, 150) {
				t.Errorf("image in track auto: x=%g width=%g, want x=180 width=150", block.X, block.Width)
			}
		}
		if block.Text == "C" {
			seenContent = true
			if !approx(block.X, 180) || !approx(block.Width, 150) {
				t.Errorf("content in track auto: x=%g width=%g, want x=180 width=150", block.X, block.Width)
			}
		}
	}
	if !seenImage || !seenContent {
		t.Errorf("found blocks: image=%v content=%v; both should be in the implicit track", seenImage, seenContent)
	}
}

func TestLayout_GridAutoColumnsDistributesFreeSpaceAcrossImplicitAutoTracks(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;grid-template-columns:100pt;grid-template-rows:20pt 30pt;grid-auto-flow:column;grid-auto-columns:auto;column-gap:8pt"><span>A</span><span>A2</span><span>B</span><img src="` + imgURI(t, 200, 100) + `"><span>C</span><span>C2</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.Measure = func(string, float64) float64 { return 10 }
	blocks := Flow(root, opts)[0].Blocks
	want := map[string][2]float64{"B": {180, 237.5}, "C": {425.5, 97.5}}
	for _, block := range blocks {
		if position, ok := want[block.Text]; ok {
			if !approx(block.X, position[0]) || !approx(block.Width, position[1]) {
				t.Errorf("track auto of %q: x=%g width=%g, want x=%g width=%g", block.Text, block.X, block.Width, position[0], position[1])
			}
			delete(want, block.Text)
		}
	}
	for text := range want {
		t.Errorf("block not found: %q", text)
	}
}

func TestLayout_GridAutoColumnsSpanContributesIntrinsicWidth(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid; grid-template-columns:100pt; grid-template-rows:20pt 30pt; grid-auto-flow:column; grid-auto-columns:auto; column-gap:8pt; row-gap:5pt"><span>A</span><span>B</span><span style="grid-column:span 2">longword</span><span>D</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth = 244
	opts.Margin = 72
	opts.Measure = func(text string, _ float64) float64 {
		if text == "longword" {
			return 48.684
		}
		return 8
	}
	blocks := Flow(root, opts)[0].Blocks
	want := map[string][2]float64{"longword": {180, 48.684}, "D": {180, 24.342}}
	for _, block := range blocks {
		if position, ok := want[block.Text]; ok {
			if !approx(block.X, position[0]) || !approx(block.Width, position[1]) {
				t.Errorf("block %q: x=%v width=%v, want x=%v width=%v", block.Text, block.X, block.Width, position[0], position[1])
			}
			delete(want, block.Text)
		}
	}
	for text := range want {
		t.Errorf("block not found: %q", text)
	}
}

func TestLayout_GridAutoColumnsSpanContributesImageIntrinsicWidth(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;grid-template-columns:100pt;grid-template-rows:20pt 30pt;grid-auto-flow:column;grid-auto-columns:auto;column-gap:8pt"><img style="grid-column:span 2" src="` + imgURI(t, 200, 100) + `"><span>A</span><span>B</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth = 252
	opts.Margin = 72
	blocks := Flow(root, opts)[0].Blocks
	seenImage, seenImplicit := false, false
	for _, block := range blocks {
		if block.Image != nil {
			seenImage = true
			if !approx(block.X, 72) || !approx(block.Width, 150) {
				t.Errorf("image span: x=%g width=%g, want x=72 width=150", block.X, block.Width)
			}
		}
		if block.Text == "B" {
			seenImplicit = true
			if !approx(block.X, 180) || !approx(block.Width, 42) {
				t.Errorf("auto track below the span: x=%g width=%g, want x=180 width=42", block.X, block.Width)
			}
		}
	}
	if !seenImage || !seenImplicit {
		t.Errorf("blocksfounds: image=%v content implicit=%v", seenImage, seenImplicit)
	}
}

func TestLayout_GridAutoColumnsSpanContributesAcrossExplicitTrack(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid; grid-template-columns:100pt; grid-template-rows:20pt 30pt; grid-auto-flow:column; grid-auto-columns:auto; column-gap:8pt"><span style="grid-column:span 2">longword</span><span>B</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth = 244
	opts.Margin = 72
	opts.Measure = func(text string, _ float64) float64 {
		if text == "longword" {
			return 146.092
		}
		return 8
	}
	blocks := Flow(root, opts)[0].Blocks
	want := map[string][2]float64{"longword": {72, 146.092}, "B": {72, 100}}
	for _, block := range blocks {
		if position, ok := want[block.Text]; ok {
			if !approx(block.X, position[0]) || !approx(block.Width, position[1]) {
				t.Errorf("block %q: x=%v width=%v, want x=%v width=%v", block.Text, block.X, block.Width, position[0], position[1])
			}
			delete(want, block.Text)
		}
	}
	for text := range want {
		t.Errorf("block not found: %q", text)
	}
}

func TestLayout_GridAutoColumnsSpanContributesAcrossFlexibleTrack(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid; grid-template-columns:1fr; grid-template-rows:20pt 30pt; grid-auto-flow:column; grid-auto-columns:auto; column-gap:8pt"><span style="grid-column:span 2">longword</span><span>B</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth = 244
	opts.Margin = 72
	opts.Measure = func(text string, _ float64) float64 {
		if text == "longword" {
			return 146.092
		}
		return 8
	}
	blocks := Flow(root, opts)[0].Blocks
	for _, block := range blocks {
		if block.Text == "longword" {
			if !approx(block.Width, 146.092) {
				t.Errorf("span over fractional and auto tracks: width=%v, want 146.092", block.Width)
			}
			return
		}
	}
	t.Fatal("block not found: longword")
}

func TestLayout_GridAutoColumnsSpanDistributesAcrossFlexibleTracksByFraction(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid; grid-template-columns:1fr 2fr; grid-template-rows:20pt 30pt; grid-auto-flow:column; grid-auto-columns:auto; column-gap:8pt"><span style="grid-column:span 3">longword</span><span style="grid-column:1">A</span><span style="grid-column:2">B</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth = 244
	opts.Margin = 72
	opts.Measure = func(text string, _ float64) float64 {
		if text == "longword" {
			return 146.092
		}
		return 8
	}
	blocks := Flow(root, opts)[0].Blocks
	want := map[string]float64{"A": 43.364, "B": 86.728, "longword": 146.092}
	for _, block := range blocks {
		if width, ok := want[block.Text]; ok {
			if !approx(block.Width, width) {
				t.Errorf("block %q: width=%v, want %v", block.Text, block.Width, width)
			}
			delete(want, block.Text)
		}
	}
	for text := range want {
		t.Errorf("block not found: %q", text)
	}
}

func TestLayout_GridMixedAutomaticAndFixedRowsSizeIndependently(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid; grid-template-columns:1fr; grid-template-rows:auto 30pt; row-gap:5pt"><span>A</span><span>B</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("boxes grid: got %d, want 2", len(blocks))
	}
	if !approx(blocks[0].Height, 14.4) || !approx(blocks[1].Height, 30) || !approx(blocks[1].Y, 91.4) {
		t.Errorf("auto/fixed rows:heights=%v,%v Y2=%v, want 14.4,30,91.4", blocks[0].Height, blocks[1].Height, blocks[1].Y)
	}
}

func TestLayout_GridPercentColumnGapUsesContainerWidth(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid; grid-template-columns:1fr 1fr; column-gap:10%"><span>A</span><span>B</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("boxes grid: got %d, want 2", len(blocks))
	}
	if !approx(blocks[0].X, 72) || !approx(blocks[0].Width, 202.95) {
		t.Errorf("track 1: x=%v width=%v, want x=72 width=202.95", blocks[0].X, blocks[0].Width)
	}
	if !approx(blocks[1].X, 320.05) || !approx(blocks[1].Width, 202.95) {
		t.Errorf("track 2: x=%v width=%v, want x=320.05 width=202.95", blocks[1].X, blocks[1].Width)
	}
}

func TestLayout_GridPercentRowGapUsesContainerWidth(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid; grid-template-columns:1fr; row-gap:10%"><span>A</span><span>B</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("boxes grid: got %d, want 2", len(blocks))
	}
	if !approx(blocks[1].Y, 131.5) {
		t.Errorf("second row: y=%v, want 131.5", blocks[1].Y)
	}
}

func TestLayout_GridMinimumHeightContributesToAutomaticRowSizing(t *testing.T) {
	rowDelta := func(minHeight string) float64 {
		t.Helper()
		html := `<style>body{margin:0;font:10pt/12pt Helvetica}.fields{display:grid;grid-template-columns:1fr 1fr;row-gap:0}.field{min-height:` + minHeight + `}</style>` +
			`<div class="fields"><div class="field">FIRST A</div><div class="field">FIRST B</div><div class="field">SECOND A</div></div>`
		root, err := boxes.Build(html)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		blocks := Flow(root, testOptions())[0].Blocks
		var first, second *PlacedBox
		for i := range blocks {
			if strings.Contains(blocks[i].Text, "FIRST A") {
				first = &blocks[i]
			}
			if strings.Contains(blocks[i].Text, "SECOND A") {
				second = &blocks[i]
			}
		}
		if first == nil || second == nil {
			t.Fatalf("grid rows not laid out: %+v", blocks)
		}
		return second.Y - first.Y
	}
	if got := rowDelta("27pt") - rowDelta("0pt"); !approx(got, 15) {
		t.Errorf("min-height:27pt increases automatic row start by %gpt, want 15pt over the 12pt line", got)
	}
}

func TestLayout_GridColumnSpanMovesNextItemToNextRow(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid; grid-template-columns:1fr 1fr 1fr"><span>A</span><span style="grid-column:span 3">B</span><span>C</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 3 {
		t.Fatalf("boxes grid: got %d, want 3", len(blocks))
	}
	wantX := []float64{72, 72, 72}
	wantY := []float64{72, 86.4, 100.8}
	wantWidth := []float64{451.0 / 3, 451, 451.0 / 3}
	for i, block := range blocks {
		if !approx(block.X, wantX[i]) || !approx(block.Y, wantY[i]) || !approx(block.Width, wantWidth[i]) {
			t.Errorf("item %d: x=%v y=%v width=%v, want x=%v y=%v width=%v", i, block.X, block.Y, block.Width, wantX[i], wantY[i], wantWidth[i])
		}
	}
}

func TestLayout_GridRowSpanOccupiesExplicitFixedRows(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid; grid-template-columns:1fr 1fr; grid-template-rows:30pt 40pt; gap:5pt 8pt"><span style="grid-row:span 2">A</span><span>B</span><span>C</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 3 {
		t.Fatalf("boxes grid: got %d, want 3", len(blocks))
	}
	wantX := []float64{72, 301.5, 301.5}
	wantY := []float64{72, 72, 107}
	wantHeight := []float64{75, 30, 40}
	for i, block := range blocks {
		if !approx(block.X, wantX[i]) || !approx(block.Y, wantY[i]) || !approx(block.Height, wantHeight[i]) {
			t.Errorf("item %d: x=%v y=%v height=%v, want x=%v y=%v height=%v", i, block.X, block.Y, block.Height, wantX[i], wantY[i], wantHeight[i])
		}
	}
}

func TestLayout_GridRowSpanSizesImplicitRows(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid; grid-template-columns:1fr 1fr; column-gap:8pt"><span style="grid-row:span 2">A</span><span>B</span><span>C</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 3 {
		t.Fatalf("boxes grid: got %d, want 3", len(blocks))
	}
	wantX := []float64{72, 301.5, 301.5}
	wantY := []float64{72, 72, 86.4}
	wantHeight := []float64{28.8, 14.4, 14.4}
	for i, block := range blocks {
		if !approx(block.X, wantX[i]) || !approx(block.Y, wantY[i]) || !approx(block.Height, wantHeight[i]) {
			t.Errorf("item %d: x=%v y=%v height=%v, want x=%v y=%v height=%v", i, block.X, block.Y, block.Height, wantX[i], wantY[i], wantHeight[i])
		}
	}
}

func TestLayout_GridColumnLineAutoPlacesFollowingItems(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid; grid-template-columns:1fr 1fr; column-gap:8pt"><span>A</span><span style="grid-column:1">B</span><span>C</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 3 {
		t.Fatalf("boxes grid: got %d, want 3", len(blocks))
	}
	wantX := []float64{72, 72, 301.5}
	wantY := []float64{72, 86.4, 86.4}
	for i, block := range blocks {
		if !approx(block.X, wantX[i]) || !approx(block.Y, wantY[i]) {
			t.Errorf("item %d: x=%v y=%v, want x=%v y=%v", i, block.X, block.Y, wantX[i], wantY[i])
		}
	}
}

func TestLayout_GridColumnLineRangeSpansTracks(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid; grid-template-columns:1fr 1fr 1fr; column-gap:6pt"><span style="grid-column:1 / 3">A</span><span>B</span><span>C</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 3 {
		t.Fatalf("boxes grid: got %d, want 3", len(blocks))
	}
	trackWidth := 439.0 / 3
	wantX := []float64{72, 376.0 + 2.0/3, 72}
	wantY := []float64{72, 72, 86.4}
	wantWidth := []float64{2*trackWidth + 6, trackWidth, trackWidth}
	for i, block := range blocks {
		if !approx(block.X, wantX[i]) || !approx(block.Y, wantY[i]) || !approx(block.Width, wantWidth[i]) {
			t.Errorf("item %d: x=%v y=%v width=%v, want x=%v y=%v width=%v", i, block.X, block.Y, block.Width, wantX[i], wantY[i], wantWidth[i])
		}
	}
}

func TestLayout_GridNegativeColumnLinesCountBackFromEnd(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid; grid-template-columns:1fr 1fr 1fr; column-gap:6pt"><span style="grid-column:-3 / -1">A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("boxes grid: got %d, want 1", len(blocks))
	}
	trackWidth := 439.0 / 3
	if !approx(blocks[0].X, 72+trackWidth+6) || !approx(blocks[0].Width, 2*trackWidth+6) {
		t.Errorf("item: x=%v width=%v, want x=%v width=%v", blocks[0].X, blocks[0].Width, 72+trackWidth+6, 2*trackWidth+6)
	}
}

func TestLayout_GridRowLineRangePlacesAtDeclaredRows(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid; grid-template-columns:1fr 1fr; grid-template-rows:24pt 36pt 18pt; gap:5pt 8pt"><span style="grid-row:2 / 4; grid-column:2">A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("boxes grid: got %d, want 1", len(blocks))
	}
	if !approx(blocks[0].X, 301.5) || !approx(blocks[0].Y, 101) || !approx(blocks[0].Height, 59) {
		t.Errorf("item: x=%v y=%v height=%v, want x=301.5 y=101 height=59", blocks[0].X, blocks[0].Y, blocks[0].Height)
	}
}

func TestLayout_GridNegativeRowLinesCountBackFromEnd(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid; grid-template-columns:1fr 1fr; grid-template-rows:24pt 36pt 18pt; gap:5pt 8pt"><span style="grid-row:-2 / -1; grid-column:2">A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("boxes grid: got %d, want 1", len(blocks))
	}
	if !approx(blocks[0].X, 301.5) || !approx(blocks[0].Y, 142) || !approx(blocks[0].Height, 18) {
		t.Errorf("item: x=%v y=%v height=%v, want x=301.5 y=142 height=18", blocks[0].X, blocks[0].Y, blocks[0].Height)
	}
}

func TestLayout_GridNamedAreas_PlaceAndSizeItems(t *testing.T) {
	root, err := boxes.Build(`<div style='display:grid; grid-template-columns:1fr 1fr; grid-template-rows:30pt 40pt; grid-template-areas:"hero hero" "left right"; gap:5pt 8pt'><span style="grid-area:hero">A</span><span style="grid-area:left">B</span><span style="grid-area:right">C</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 3 {
		t.Fatalf("boxes grid: got %d, want 3", len(blocks))
	}
	wantX := []float64{72, 72, 301.5}
	wantY := []float64{72, 107, 107}
	wantWidth := []float64{451, 221.5, 221.5}
	wantHeight := []float64{30, 40, 40}
	for i, block := range blocks {
		if !approx(block.X, wantX[i]) || !approx(block.Y, wantY[i]) || !approx(block.Width, wantWidth[i]) || !approx(block.Height, wantHeight[i]) {
			t.Errorf("item %d: x=%v y=%v width=%v height=%v, want x=%v y=%v width=%v height=%v", i, block.X, block.Y, block.Width, block.Height, wantX[i], wantY[i], wantWidth[i], wantHeight[i])
		}
	}
}

func TestLayout_GridRepeatFixedTracksKeepsDeclaredWidths(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid; grid-template-columns:repeat(2, 100px)"><span>A</span><span>B</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("boxes grid: got %d, want 2", len(blocks))
	}
	if !approx(blocks[0].X, 72) || !approx(blocks[0].Width, 75) {
		t.Errorf("track 0: x=%v width=%v, want x=72 width=75", blocks[0].X, blocks[0].Width)
	}
	if !approx(blocks[1].X, 147) || !approx(blocks[1].Width, 75) {
		t.Errorf("track 1: x=%v width=%v, want x=147 width=75", blocks[1].X, blocks[1].Width)
	}
}

func TestLayout_GridAutoRowUsesExplicitItemHeight(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;grid-template-columns:repeat(3,1fr);gap:5pt"><i style="height:35pt"></i><i style="height:35pt"></i><i style="height:35pt"></i><i style="height:35pt"></i></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 4 {
		t.Fatalf("blocks grid=%d, want 4", len(blocks))
	}
	for index, block := range blocks {
		if !approx(block.Height, 35) {
			t.Errorf("height of the item %d=%v, want 35pt", index, block.Height)
		}
	}
	if !approx(blocks[3].Y-blocks[0].Y, 40) {
		t.Errorf("spacing between rows=%v, want 40pt", blocks[3].Y-blocks[0].Y)
	}
}

func TestLayout_GridMarginsCollapseWithAdjacentBlocks(t *testing.T) {
	root, err := boxes.Build(`<p style="margin:0 0 8pt">A</p><div style="display:grid;grid-template-columns:1fr;margin-top:12pt;margin-bottom:12pt"><span>B</span></div><p style="margin:8pt 0 0">C</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	root.Style.BodyMargin = 0
	opts := testOptions()
	opts.Margin = 0
	blocks := Flow(root, opts)[0].Blocks
	var positions = map[string]float64{}
	for _, block := range blocks {
		for _, line := range block.Runs {
			for _, run := range line {
				positions[run.Text] = run.Y
			}
		}
	}
	if !approx(positions["B"]-positions["A"], 26.4) {
		t.Errorf("distance A→B=%v, want 26.4pt with 12pt collapsed margin", positions["B"]-positions["A"])
	}
	if !approx(positions["C"]-positions["B"], 26.4) {
		t.Errorf("distance from B to C=%v, want 26.4pt with margin collapsed 12pt", positions["C"]-positions["B"])
	}
}

func TestLayout_GridContainerAppliesHeightConstraintsToTracksAndFlow(t *testing.T) {
	tests := []struct {
		name, sizing, itemHeight string
		wantContainerHeight      float64
		wantItemHeight           float64
	}{
		{name: "minimum height stretches automatic row", sizing: "min-height:100pt", itemHeight: "20pt", wantContainerHeight: 100, wantItemHeight: 100},
		{name: "maximum height limits flow but lets content overflow", sizing: "max-height:60pt", itemHeight: "80pt", wantContainerHeight: 60, wantItemHeight: 80},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := `<div style="display:grid;grid-template-columns:100pt;` + tt.sizing + `;background:#f00"><div style="height:` + tt.itemHeight + `">ITEM</div></div><p style="margin:0">AFTER</p>`
			root, err := boxes.Build(html)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			blocks := Flow(root, testOptions())[0].Blocks
			var container, item, after *PlacedBox
			for _, block := range blocks {
				if block.HasBg && block.Bg[0] > .9 && block.Bg[1] < .1 {
					container = &block
				}
				if block.Text == "ITEM" {
					item = &block
				}
				if block.Text == "AFTER" {
					after = &block
				}
			}
			if container == nil || !approx(container.Height, tt.wantContainerHeight) {
				t.Fatalf("grid container=%v, want height %v", container, tt.wantContainerHeight)
			}
			if item == nil || !approx(item.Height, tt.wantItemHeight) {
				t.Fatalf("grid item=%v, want height %v", item, tt.wantItemHeight)
			}
			if after == nil || after.Y < container.Y+container.Height {
				t.Fatalf("following block=%v, want at or after grid bottom %v", after, container.Y+container.Height)
			}
		})
	}
}

func TestLayout_GridTracksUseContentBoxInsidePaddingAndBorders(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;width:200pt;grid-template-columns:1fr 1fr;grid-template-rows:20pt;column-gap:10pt;padding:10pt;border:2pt solid;background:#f00"><span>A</span><span>B</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	var container, first, second *PlacedBox
	for i := range blocks {
		block := &blocks[i]
		if block.HasBg {
			container = block
		}
		if block.Text == "A" {
			first = block
		}
		if block.Text == "B" {
			second = block
		}
	}
	if container == nil || !approx(container.X, 72) || !approx(container.Y, 72) || !approx(container.Width, 224) || !approx(container.Height, 44) {
		t.Fatalf("grid container=%+v, want outer box at (72,72) of 224x44pt", container)
	}
	if first == nil || second == nil || !approx(first.X, 84) || !approx(first.Y, 84) || !approx(first.Width, 95) || !approx(second.X, 189) || !approx(second.Width, 95) {
		t.Fatalf("grid tracks first=%+v second=%+v, want content-box tracks at x=84/189 with width 95pt", first, second)
	}
}

func TestLayout_GridOverflowClipUsesPaddingBoxAfterInsets(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;height:30pt;grid-template-columns:100pt;grid-template-rows:60pt;padding:8pt 0;border:2pt solid;overflow:hidden"><span>OVERFLOW</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	for _, block := range blocks {
		if block.Text != "OVERFLOW" {
			continue
		}
		if len(block.ClipRects) != 1 {
			t.Fatalf("clip rects=%+v, want one padding-box clip", block.ClipRects)
		}
		clip := block.ClipRects[0]
		if !approx(clip.Y, 74) || !approx(clip.Height, 46) {
			t.Errorf("clip=%+v, want padding box y=74 height=46", clip)
		}
		return
	}
	t.Fatal("grid item missing")
}

func TestLayout_GridContainerDecorationContinuesAcrossPages(t *testing.T) {
	content := strings.Repeat("GRID FRAGMENT CONTENT ", 100)
	root, err := boxes.Build(`<div style="display:grid;grid-template-columns:100pt;background:#f00;border:2pt solid;padding:4pt"><div>` + content + `</div></div><p style="margin:0">AFTER GRID</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 225, 135, 9
	pages := Flow(root, opts)
	if len(pages) < 2 {
		t.Fatalf("pages=%d, want grid content to fragment", len(pages))
	}
	lastGridContentPage := 0
	for pageIndex, page := range pages {
		for _, block := range page.Blocks {
			if strings.Contains(block.Text, "GRID FRAGMENT CONTENT") {
				lastGridContentPage = pageIndex + 1
			}
		}
	}
	for pageIndex, page := range pages {
		var decoration *PlacedBox
		pageHasGridContent := false
		for i := range page.Blocks {
			pageHasGridContent = pageHasGridContent || strings.Contains(page.Blocks[i].Text, "GRID FRAGMENT CONTENT")
			if decoration == nil && page.Blocks[i].HasBg && page.Blocks[i].Bg[0] > .9 && page.Blocks[i].Bg[1] < .1 {
				decoration = &page.Blocks[i]
			}
		}
		if !pageHasGridContent {
			continue
		}
		if decoration == nil {
			t.Errorf("page %d lost the grid background", pageIndex+1)
			continue
		}
		if decoration.Height <= 0 || decoration.BorderLeft.Width != 2 || decoration.BorderRight.Width != 2 {
			t.Errorf("fragment %d has invalid box geometry: %+v", pageIndex+1, decoration)
		}
		if pageIndex == 0 && decoration.BorderTop.Width != 2 {
			t.Errorf("first fragment top border=%v, want 2pt", decoration.BorderTop.Width)
		}
		if pageIndex > 0 && decoration.BorderTop.Width != 0 {
			t.Errorf("continuation %d repeated top border %v", pageIndex+1, decoration.BorderTop.Width)
		}
		if pageIndex+1 < lastGridContentPage && decoration.BorderBottom.Width != 0 {
			t.Errorf("fragment %d repeated bottom border %v", pageIndex+1, decoration.BorderBottom.Width)
		}
		if pageIndex+1 == lastGridContentPage && decoration.BorderBottom.Width != 2 {
			t.Errorf("last fragment bottom border=%v, want 2pt", decoration.BorderBottom.Width)
		}
	}
	lastGridPage, afterPage := lastGridContentPage, 0
	var lastGridBottom float64
	var afterY float64
	for pageIndex, page := range pages {
		for _, block := range page.Blocks {
			if strings.Contains(block.Text, "GRID FRAGMENT CONTENT") {
				lastGridBottom = block.Y + block.Height
			}
			if block.Text == "AFTER GRID" {
				afterPage, afterY = pageIndex+1, block.Y
			}
		}
	}
	if afterPage == 0 || afterPage < lastGridPage || (afterPage == lastGridPage && afterY < lastGridBottom+6) {
		t.Errorf("following block page/y=%d/%.2f overlaps final grid fragment page/bottom=%d/%.2f with 6pt bottom insets", afterPage, afterY, lastGridPage, lastGridBottom)
	}
}

func TestLayout_GridRowFragmentsKeepContentBoxHorizontalOrigin(t *testing.T) {
	content := strings.Repeat("FRAGMENTED GRID CONTENT ", 80)
	root, err := boxes.Build(`<div style="display:grid;grid-template-columns:100pt;padding:4pt 0 4pt 8pt;border:2pt solid"><div>` + content + `</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 225, 135, 9
	pages := Flow(root, opts)
	if len(pages) < 2 {
		t.Fatalf("pages=%d, want fragmented grid", len(pages))
	}
	fragments := 0
	for _, page := range pages {
		for _, block := range page.Blocks {
			if strings.Contains(block.Text, "FRAGMENTED GRID") {
				fragments++
				if !approx(block.X, 19) {
					t.Errorf("fragment x=%v, want content-box origin 19pt", block.X)
				}
			}
		}
	}
	if fragments < 2 {
		t.Fatalf("found %d text fragments, want at least 2", fragments)
	}
}

func TestLayout_GridPlacesContentThatFitsBeforeTrackBoundary(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;grid-template-columns:100pt;grid-template-rows:80pt 35pt;row-gap:10pt;padding:4pt 0;border:2pt solid;background:#f00"><div>ROW ONE</div><div>ROW TWO</div></div><p style="margin:0">AFTER GRID</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 225, 135, 9
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages=%d, want 2", len(pages))
	}
	rowOnePage, rowTwoPage, afterPage := 0, 0, 0
	var continuation, after *PlacedBox
	for pageIndex, page := range pages {
		for blockIndex := range page.Blocks {
			block := &page.Blocks[blockIndex]
			switch block.Text {
			case "ROW ONE":
				rowOnePage = pageIndex + 1
			case "ROW TWO":
				rowTwoPage = pageIndex + 1
			case "AFTER GRID":
				afterPage = pageIndex + 1
				after = block
			}
			if pageIndex == 1 && block.HasBg && block.Bg[0] > .9 && block.Bg[1] < .1 {
				continuation = block
			}
		}
	}
	if rowOnePage != 1 || rowTwoPage != 1 || afterPage != 2 {
		t.Errorf("row pages=%d/%d, following page=%d; want rows on page 1 and following block on page 2", rowOnePage, rowTwoPage, afterPage)
	}
	if continuation == nil || !approx(continuation.Y, 9) || !approx(continuation.Height, 20) || continuation.BorderTop.Width != 0 || continuation.BorderBottom.Width != 2 {
		t.Errorf("grid continuation=%+v, want second-page fragment y=9 height=20 with bottom border only", continuation)
	}
	if after == nil || continuation == nil || after.Y < continuation.Y+continuation.Height {
		t.Errorf("following block=%+v overlaps grid continuation=%+v", after, continuation)
	}
}

func TestLayout_GridBreaksAfterRowGapAtFreshPageOrigin(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;grid-template-columns:100pt;grid-template-rows:100pt 10pt;row-gap:20pt;border:2pt solid;background:#f00"><div>ROW ONE</div><div>ROW TWO</div></div><p style="margin:0">AFTER GRID</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 225, 135, 9
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages=%d, want 2", len(pages))
	}
	rowOnePage, rowTwoPage, afterPage := 0, 0, 0
	var rowTwo, after *PlacedBox
	for pageIndex, page := range pages {
		for blockIndex := range page.Blocks {
			block := &page.Blocks[blockIndex]
			switch block.Text {
			case "ROW ONE":
				rowOnePage = pageIndex + 1
			case "ROW TWO":
				rowTwoPage, rowTwo = pageIndex+1, block
			case "AFTER GRID":
				afterPage, after = pageIndex+1, block
			}
		}
	}
	if rowOnePage != 1 || rowTwoPage != 2 {
		t.Errorf("row pages=%d/%d, want first row on page 1 and second row on page 2", rowOnePage, rowTwoPage)
	}
	if rowTwo == nil || !approx(rowTwo.Y, 9) {
		t.Errorf("second row=%+v, want fresh page content origin y=9 without carrying the preceding row gap", rowTwo)
	}
	if afterPage != 2 || after == nil || rowTwo == nil || after.Y < rowTwo.Y+rowTwo.Height {
		t.Errorf("following block page/y=%d/%v overlaps or precedes second row=%+v", afterPage, after, rowTwo)
	}
}

func TestLayout_GridOverflowAdvancesFollowingRowsAcrossFragmentainers(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;grid-template-columns:100pt;grid-template-rows:300pt;background:#f00;border:2pt solid;padding:4pt 0"><div>ROW ONE</div><div>ROW TWO</div></div><p style="margin:0">AFTER GRID</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 225, 135, 9
	pages := Flow(root, opts)
	if len(pages) != 3 {
		t.Fatalf("pages=%d, want the 300pt track to continue across three pages", len(pages))
	}
	rowOnePage, rowTwoPage, afterPage := 0, 0, 0
	decorations := make([]*PlacedBox, len(pages))
	var after *PlacedBox
	for pageIndex, page := range pages {
		for blockIndex := range page.Blocks {
			block := &page.Blocks[blockIndex]
			switch block.Text {
			case "ROW ONE":
				rowOnePage = pageIndex + 1
			case "ROW TWO":
				rowTwoPage = pageIndex + 1
			case "AFTER GRID":
				afterPage, after = pageIndex+1, block
			}
			if block.HasBg && block.Bg[0] > .9 && block.Bg[1] < .1 {
				decorations[pageIndex] = block
			}
		}
	}
	if rowOnePage != 1 || rowTwoPage != 3 || afterPage != 3 {
		t.Errorf("row pages=%d/%d, following page=%d; want 1/3/3", rowOnePage, rowTwoPage, afterPage)
	}
	wantHeights := []float64{117, 117, 92.4}
	for pageIndex, decoration := range decorations {
		if decoration == nil || !approx(decoration.Y, 9) || !approx(decoration.Height, wantHeights[pageIndex]) {
			t.Errorf("grid fragment page %d=%+v, want y=9 height=%.0f", pageIndex+1, decoration, wantHeights[pageIndex])
		}
	}
	if decorations[0] != nil && (decorations[0].BorderTop.Width != 2 || decorations[0].BorderBottom.Width != 0) {
		t.Errorf("first fragment borders top/bottom=%v/%v, want 2/0", decorations[0].BorderTop.Width, decorations[0].BorderBottom.Width)
	}
	if decorations[1] != nil && (decorations[1].BorderTop.Width != 0 || decorations[1].BorderBottom.Width != 0) {
		t.Errorf("middle fragment borders top/bottom=%v/%v, want 0/0", decorations[1].BorderTop.Width, decorations[1].BorderBottom.Width)
	}
	if decorations[2] != nil && (decorations[2].BorderTop.Width != 0 || decorations[2].BorderBottom.Width != 2) {
		t.Errorf("last fragment borders top/bottom=%v/%v, want 0/2", decorations[2].BorderTop.Width, decorations[2].BorderBottom.Width)
	}
	if after == nil || decorations[2] == nil || after.Y < decorations[2].Y+decorations[2].Height {
		t.Errorf("following block=%+v overlaps last grid fragment=%+v", after, decorations[2])
	}
}
