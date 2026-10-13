package layout

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/robtrivi/pagecraft/boxes"
	"github.com/robtrivi/pagecraft/text"
)

func TestLayout_TableRowsUseDestinationPageCapacity(t *testing.T) {
	root, err := boxes.Build(`<style>@page { size: 200pt 50pt; margin: 5pt } @page :left { size: 200pt 100pt; margin: 20pt 5pt 5pt } body { margin: 0 } table { margin: 0; width: 100pt } td { padding: 5pt; font: 10pt/10pt Helvetica }</style><table><tr><td>row-1</td></tr><tr><td>row-2</td></tr><tr><td>row-3</td></tr><tr><td>row-4</td></tr><tr><td>row-5</td></tr></table>`)
	if err != nil {
		t.Fatalf("boxes.Build: %v", err)
	}
	opts := DefaultOptions()
	opts.Measure, opts.MeasureBold = text.Measure, text.MeasureBold
	opts.PageWidth, opts.PageHeight, opts.Margin = 200, 50, 5
	opts.PageMargins, opts.PageRules = root.Page.Margins, root.PageRules
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("got %d pages, want 2 with larger continuation capacity", len(pages))
	}
	if pages[1].Geometry.Height != 100 {
		t.Errorf("second page height=%g, want 100", pages[1].Geometry.Height)
	}
	found := false
	for _, block := range pages[1].Blocks {
		found = found || block.Text == "row-5"
	}
	if !found {
		t.Error("last row was not placed on the larger second page")
	}
}

// A simple 2 by 2 table creates four blocks,
// equal columns, rows with increasing y. Not colspan/rowspan or nested tables.
func TestLayout_SimpleTable_RendersRows(t *testing.T) {
	root, err := boxes.Build(`<table style="width:451pt"><tbody><tr><td>A1</td><td>B1</td></tr><tr><td>A2</td><td>B2</td></tr></tbody></table>`)
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
	// Browser-default 2px border spacing leaves 446.5pt for tracks and 1.5pt at each edge.
	if blocks[0].X != 73.5 || blocks[1].X != 298.25 {
		t.Errorf("columns row 1: got (%v,%v), want (73.5,298.25)", blocks[0].X, blocks[1].X)
	}
	if blocks[0].Y != 73.5 {
		t.Errorf("row 1 y: got %v, want 73.5", blocks[0].Y)
	}
	if blocks[2].Y <= blocks[0].Y {
		t.Errorf("row 2 did not advance: %v after %v", blocks[2].Y, blocks[0].Y)
	}
	want := []string{"A1", "B1", "A2", "B2"}
	for i, w := range want {
		if blocks[i].Text != w {
			t.Errorf("block %d: got %q, want %q", i, blocks[i].Text, w)
		}
	}
}

func TestLayout_CSSDisplayTableLaysOutRowsAndCells(t *testing.T) {
	root, err := boxes.Build(`<style>body{margin:0}td{padding:0}</style><div style="display:table;width:100pt;margin:0;border-spacing:0"><div style="display:table-row"><div style="display:table-cell;width:40pt;padding:0">A</div><div style="display:table-cell;padding:0">B</div></div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 200, 100, 0
	var left, right *PlacedBox
	for _, block := range Flow(root, opts)[0].Blocks {
		switch block.Text {
		case "A":
			left = &block
		case "B":
			right = &block
		}
	}
	if left == nil || right == nil {
		t.Fatalf("table cells missing: left=%v right=%v", left != nil, right != nil)
	}
	if left.Width != 40 || right.Width != 60 || right.X != 40 {
		t.Errorf("table-cell geometry: left=(x %g,w %g), right=(x %g,w %g); want (0,40) and (40,60)", left.X, left.Width, right.X, right.Width)
	}
}

func TestLayout_SeparateTableUsesBrowserDefaultBorderSpacing(t *testing.T) {
	root, err := boxes.Build(`<style>body{margin:0}table{width:100pt;margin:0}td{padding:0;font:10pt/12pt Helvetica}</style><table><tr><td>A</td><td>B</td></tr></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 200, 100, 0
	var left, right *PlacedBox
	for _, block := range Flow(root, opts)[0].Blocks {
		switch block.Text {
		case "A":
			left = &block
		case "B":
			right = &block
		}
	}
	if left == nil || right == nil {
		t.Fatalf("table cells missing: left=%v right=%v", left != nil, right != nil)
	}
	if math.Abs(left.X-1.5) > 0.01 || math.Abs(left.Width-47.75) > 0.01 || math.Abs(right.X-50.75) > 0.01 || math.Abs(right.Width-47.75) > 0.01 {
		t.Errorf("separate table cell geometry: left=(x %.2f,w %.2f), right=(x %.2f,w %.2f); want Blink's 2px spacing at the table edges and between cells", left.X, left.Width, right.X, right.Width)
	}
}

func TestLayout_SeparateTableUsesDeclaredHorizontalAndVerticalSpacing(t *testing.T) {
	root, err := boxes.Build(`<style>body{margin:0}table{width:100pt;margin:0;border-spacing:4pt 6pt}td{padding:0;font:10pt/12pt Helvetica}</style><table><tr><td>A</td><td>B</td></tr><tr><td>C</td><td>D</td></tr></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 200, 100, 0
	blocks := Flow(root, opts)[0].Blocks
	if len(blocks) != 4 {
		t.Fatalf("table cells=%d, want 4", len(blocks))
	}
	if math.Abs(blocks[0].X-4) > 0.01 || math.Abs(blocks[0].Width-44) > 0.01 || math.Abs(blocks[1].X-52) > 0.01 {
		t.Errorf("horizontal spacing geometry: first=(%.2f,%.2f) secondX=%.2f, want (4,44) and 52", blocks[0].X, blocks[0].Width, blocks[1].X)
	}
	if math.Abs(blocks[0].Y-6) > 0.01 || math.Abs(blocks[2].Y-(blocks[0].Y+blocks[0].Height+6)) > 0.01 {
		t.Errorf("vertical spacing geometry: firstY=%.2f secondRowY=%.2f, want 6 and first row bottom plus 6pt", blocks[0].Y, blocks[2].Y)
	}
}

func TestLayout_TableFooterSectionFlowsAfterBodyRegardlessOfSourceOrder(t *testing.T) {
	root, err := boxes.Build(`<style>body{margin:0}table{width:100pt;margin:0}td{padding:0}</style><table><tfoot><tr><td>FOOTER</td></tr></tfoot><tbody><tr><td>BODY</td></tr></tbody></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	pages := Flow(root, testOptions())
	var body, footer *PlacedBox
	for _, block := range pages[0].Blocks {
		switch block.Text {
		case "BODY":
			body = &block
		case "FOOTER":
			footer = &block
		}
	}
	if body == nil || footer == nil {
		t.Fatalf("table sections missing: body=%v footer=%v", body != nil, footer != nil)
	}
	if body.Y >= footer.Y {
		t.Errorf("body row y=%g must precede footer row y=%g even when tfoot occurs first in source", body.Y, footer.Y)
	}
}

func TestLayout_TableFixedLayoutUsesColumnAndFirstRowWidths(t *testing.T) {
	root, err := boxes.Build(`<style>body{margin:0}table{width:200pt;table-layout:fixed;margin:0}td{padding:0;white-space:nowrap}</style><table><col style="width:50pt"><tr><td>FIRST</td><td style="width:60pt">SECOND</td><td>THIRD</td></tr><tr><td>THIS-LONG-CONTENT-MUST-NOT-RESIZE-COLUMN</td><td>X</td><td>Y</td></tr></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 300, 100, 0
	wants := map[string]float64{"FIRST": 50, "SECOND": 60, "THIRD": 84}
	for _, block := range Flow(root, opts)[0].Blocks {
		if want, ok := wants[block.Text]; ok {
			if math.Abs(block.Width-want) > 0.01 {
				t.Errorf("fixed table cell %q width=%gpt, want %gpt", block.Text, block.Width, want)
			}
			delete(wants, block.Text)
		}
	}
	if len(wants) != 0 {
		t.Errorf("first-row cells missing from layout: %v", wants)
	}
}

func TestLayout_TableFixedLayoutDividesColspanWidthAcrossTracks(t *testing.T) {
	root, err := boxes.Build(`<style>body{margin:0}table{width:200pt;table-layout:fixed;margin:0}td{padding:0;white-space:nowrap}</style><table><col style="width:50pt"><tr><td colspan="2" style="width:120pt">SPAN</td><td>TAIL</td></tr></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 300, 100, 0
	var span, tail *PlacedBox
	for _, block := range Flow(root, opts)[0].Blocks {
		switch block.Text {
		case "SPAN":
			span = &block
		case "TAIL":
			tail = &block
		}
	}
	if span == nil || tail == nil {
		t.Fatalf("table cells missing: span=%v tail=%v", span != nil, tail != nil)
	}
	if math.Abs(span.Width-110) > 0.01 || math.Abs(tail.Width-84) > 0.01 {
		t.Errorf("fixed colspan widths=%g/%gpt, want 110/84pt from width 50/60 across the colspan and 84 remaining after border spacing", span.Width, tail.Width)
	}
}

func TestLayout_TableFixedLayoutRepeatsColumnSpanWidthPerTrack(t *testing.T) {
	root, err := boxes.Build(`<style>body{margin:0}table{width:200pt;table-layout:fixed;margin:0}td{padding:0}</style><table><col style="width:40pt" span="2"><tr><td>A</td><td>B</td><td>C</td></tr></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 300, 100, 0
	wants := map[string]float64{"A": 40, "B": 40, "C": 114}
	for _, block := range Flow(root, opts)[0].Blocks {
		if want, ok := wants[block.Text]; ok {
			if math.Abs(block.Width-want) > 0.01 {
				t.Errorf("fixed col span cell %q width=%gpt, want %gpt", block.Text, block.Width, want)
			}
			delete(wants, block.Text)
		}
	}
	if len(wants) > 0 {
		t.Errorf("first-row cells missing: %v", wants)
	}
}

func TestLayout_TableFixedLayoutExpandsToDeclaredColumnMinimum(t *testing.T) {
	root, err := boxes.Build(`<style>body{margin:0}table{width:100pt;table-layout:fixed;margin:0}td{padding:0}</style><table><col style="width:150pt"><tr><td>A</td><td>B</td></tr></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 300, 100, 0
	head, body := tableRows(root.Children[0])
	grid, cols := tableGridFromRows(append(head, body...))
	if got := tableUsedWidth(root.Children[0], grid, cols, 300, opts); math.Abs(got-154.5) > 0.01 {
		t.Errorf("used fixed table width=%gpt, want 154.5pt from its 150pt track and default separate border spacing", got)
	}
	var left, right *PlacedBox
	for _, block := range Flow(root, opts)[0].Blocks {
		switch block.Text {
		case "A":
			left = &block
		case "B":
			right = &block
		}
	}
	if left == nil || right == nil {
		t.Fatalf("table cells missing: left=%v right=%v", left != nil, right != nil)
	}
	if math.Abs(left.Width-150) > 0.01 || math.Abs(right.Width) > 0.01 {
		t.Errorf("fixed table widths=%g/%gpt, want 150/0pt because the table expands to its column minimum", left.Width, right.Width)
	}
}

func TestLayout_AutoTableHonorsColumnElementWidth(t *testing.T) {
	root, err := boxes.Build(`<style>body{margin:0}table{width:200pt;margin:0}td{padding:0}</style><table><col style="width:150pt"><tr><td>A</td><td>B</td></tr></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 300, 100, 0
	var left, right *PlacedBox
	for _, block := range Flow(root, opts)[0].Blocks {
		switch block.Text {
		case "A":
			left = &block
		case "B":
			right = &block
		}
	}
	if left == nil || right == nil {
		t.Fatalf("cells missing: left=%v right=%v", left != nil, right != nil)
	}
	if math.Abs(left.Width-150) > 0.01 || math.Abs(right.Width-45.5) > 0.01 {
		t.Errorf("auto table widths=%g/%gpt, want 150/45.5pt from col width plus remaining auto track after border spacing", left.Width, right.Width)
	}
}

func TestLayout_AutoTableDistributesWidthFromCellIntrinsicSizesWithRowspan(t *testing.T) {
	root, err := boxes.Build(`<table style="width:100%;margin:0"><tbody><tr><td rowspan="2">ID</td><td>Item notes and detailed description for the warehouse entry</td><td>Qty</td></tr><tr><td>A second note with longer packaging instructions</td><td>8</td></tr></tbody></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 240, 842, 10
	pages := Flow(root, opts)
	if len(pages) != 1 {
		t.Fatalf("pages=%d, want 1", len(pages))
	}
	var idWidth, notesWidth, qtyWidth float64
	for _, block := range pages[0].Blocks {
		switch strings.TrimSpace(block.Text) {
		case "ID":
			idWidth = block.Width
		case "Item notes and detailed description for the warehouse entry":
			notesWidth = block.Width
		case "Qty":
			qtyWidth = block.Width
		}
	}
	if idWidth == 0 || notesWidth == 0 || qtyWidth == 0 {
		t.Fatalf("not all cells were measured: id=%g notes=%g qty=%g, blocks=%+v", idWidth, notesWidth, qtyWidth, pages[0].Blocks)
	}
	if notesWidth <= idWidth || notesWidth <= qtyWidth {
		t.Errorf("the column with the greatest intrinsic constraint should receive more width: ID=%g notes=%g qty=%g", idWidth, notesWidth, qtyWidth)
	}
}

func TestLayout_AutoTableHyphenatedIdentifierFitsAvailableWidth(t *testing.T) {
	html := `<style>body{margin:0;font:9pt Helvetica}table{width:216pt;margin:0;border-collapse:collapse}td,th{padding:5pt;border:1pt solid #64748b}.big{font-size:16pt;font-weight:bold}.fixed{white-space:nowrap;font-size:8pt}.summary{white-space:pre-wrap}</style><table><thead><tr><th>Reference</th><th>Description</th><th>State</th></tr></thead><tbody><tr><td><span class="big">INV-2026-ALPHA</span></td><td>Ordinary words should use available width without forcing equal columns.</td><td><span class="fixed">READY NOW</span></td></tr><tr><td colspan="2" class="summary">A spanning cell contributes its inline minimum across both columns.
The second preserved line contains <b>styled text</b>.</td><td>DONE</td></tr></tbody></table>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 240, 400, 12
	pages := Flow(root, opts)
	if len(pages) != 1 {
		t.Fatalf("pages=%d, want 1", len(pages))
	}
	cellCount := 0
	for _, block := range pages[0].Blocks {
		if block.source == nil || block.source.Tag != "td" && block.source.Tag != "th" {
			continue
		}
		cellCount++
		if right := block.X + block.Width; right > opts.PageWidth-opts.Margin+fragmentEpsilon {
			t.Errorf("cell %q exceeds available width: right=%gpt, limit=%gpt", block.Text, right, opts.PageWidth-opts.Margin)
		}
	}
	if cellCount != 8 {
		t.Fatalf("painted cells=%d, want 8", cellCount)
	}
}

func TestLayout_TableRepeatsHeaderAfterCompletedRowspan(t *testing.T) {
	root, err := boxes.Build(`<style>body{margin:0}table{margin:0;width:200pt}</style><table><thead><tr><th>COL-A</th><th>COL-B</th></tr></thead><tbody><tr><td rowspan="2">SPAN</td><td>FIRST</td></tr><tr><td>SECOND</td></tr><tr style="break-before:page"><td>NEXT</td><td>ROW</td></tr></tbody></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 220, 120, 10
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages=%d, want 2", len(pages))
	}
	got := map[string]bool{}
	for _, block := range pages[1].Blocks {
		got[strings.TrimSpace(block.Text)] = true
	}
	for _, marker := range []string{"COL-A", "COL-B", "NEXT", "ROW"} {
		if !got[marker] {
			t.Errorf("marker %q does not appear with the repeated header on page 2: %+v", marker, pages[1].Blocks)
		}
	}
}

func TestLayout_TableRowHonorsForcedPageBreak(t *testing.T) {
	root, err := boxes.Build(`<table style="margin:0"><tr><td>FIRST ROW</td></tr><tr style="break-before:page"><td>SECOND ROW</td></tr></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 220, 120, 0
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages=%d, want 2", len(pages))
	}
	if len(pages[0].Blocks) != 1 || pages[0].Blocks[0].Text != "FIRST ROW" {
		t.Errorf("first page=%+v, must contain the first row", pages[0].Blocks)
	}
	if len(pages[1].Blocks) != 1 || pages[1].Blocks[0].Text != "SECOND ROW" {
		t.Errorf("second page=%+v, must contain the forced row", pages[1].Blocks)
	}
}

func TestLayout_TableFragmentsOversizedRowBetweenCellLines(t *testing.T) {
	words := make([]string, 100)
	for index := range words {
		words[index] = fmt.Sprintf("ROW%02d", index+1)
	}
	html := `<style>body{margin:0} table{margin:0;width:220pt;border-collapse:collapse} td{padding:0}</style><table><tr><td>` + strings.Join(words, " ") + `</td></tr></table>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 220, 120, 0
	opts.FontSize, opts.LineHeight = 10, 1.2
	pages := Flow(root, opts)
	cell := findBoxWithTag(root, "td")
	if cell == nil {
		t.Fatal("td cell was not found in the box tree")
	}
	fragments := fragmentsForSource(pages, cell)
	if len(pages) < 2 || len(fragments) < 2 {
		var heights []float64
		for _, block := range pages[0].Blocks {
			heights = append(heights, block.Height)
		}
		t.Fatalf("pages=%d cell fragments=%d heights=%v lines=%d; the tall row must continue across pages", len(pages), len(fragments), heights, len(BreakLines(cell.Text, 220, testOptions().Measure, 10)))
	}
	assertFragmentChain(t, fragments)
	var got []string
	for _, page := range pages {
		for _, block := range page.Blocks {
			if block.source != cell {
				continue
			}
			for _, line := range block.Runs {
				for _, run := range line {
					got = append(got, strings.Fields(run.Text)...)
				}
			}
		}
	}
	if strings.Join(got, " ") != strings.Join(words, " ") {
		t.Errorf("fragmented content did not preserve order/text:\n got %q\nwant %q", strings.Join(got, " "), strings.Join(words, " "))
	}
}

func TestLayout_TableRowWithoutIntersectingRowspanCanStillFragment(t *testing.T) {
	words := make([]string, 100)
	for index := range words {
		words[index] = fmt.Sprintf("SPAN%03d", index+1)
	}
	html := `<style>body{margin:0}table{margin:0;width:220pt;border-collapse:collapse}td{padding:0}</style><table><tbody><tr><td rowspan="2">A</td><td>B</td></tr><tr><td>C</td></tr><tr><td colspan="2">` + strings.Join(words, " ") + `</td></tr></tbody></table>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 220, 120, 0
	opts.FontSize, opts.LineHeight = 10, 1.2
	pages := Flow(root, opts)
	cell := findTableCellContaining(root, "SPAN001")
	if cell == nil {
		t.Fatal("long cell was not found")
	}
	fragments := fragmentsForSource(pages, cell)
	if len(fragments) < 2 {
		t.Fatalf("long row fragments=%d; a completed rowspan must not block it, pages=%d", len(fragments), len(pages))
	}
	assertFragmentChain(t, fragments)
	var got []string
	for _, page := range pages {
		for _, block := range page.Blocks {
			if block.source != cell {
				continue
			}
			for _, line := range block.Runs {
				for _, run := range line {
					got = append(got, strings.Fields(run.Text)...)
				}
			}
		}
	}
	if strings.Join(got, " ") != strings.Join(words, " ") {
		t.Errorf("content of the row after rowspan changed: got %q, want %q", strings.Join(got, " "), strings.Join(words, " "))
	}
}

func TestLayout_TableFragmentsRowAcrossActiveRowspan(t *testing.T) {
	lines := make([]string, 17)
	for index := range lines {
		lines[index] = fmt.Sprintf("ACTIVE-%02d", index+1)
	}
	html := `<style>body{margin:0}table{margin:0;width:220pt;border-collapse:collapse}td{padding:0;border:1pt solid #111}</style><table><tbody><tr><td rowspan="2" style="background:#fee">SPAN</td><td>START</td></tr><tr><td style="background:#eef">` + strings.Join(lines, "<br>") + `</td></tr></tbody></table>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 220, 120, 0
	opts.FontSize, opts.LineHeight = 10, 1.2
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages=%d, Chromium/WeasyPrint produce 2", len(pages))
	}
	spanCell := findTableCellContaining(root, "SPAN")
	contentCell := findTableCellContaining(root, "ACTIVE-01")
	if spanCell == nil || contentCell == nil {
		t.Fatal("rowspan and content cells were not found")
	}
	spanFragments := fragmentsForSource(pages, spanCell)
	if len(spanFragments) != 2 {
		t.Fatalf("visible rowspan fragments=%d, want one segment on each page", len(spanFragments))
	}
	assertFragmentChain(t, spanFragments)
	firstSpan := pages[0].Blocks[spanFragments[0].PaintIndex]
	continuedSpan := pages[1].Blocks[spanFragments[1].PaintIndex]
	if !firstSpan.HasBg || !continuedSpan.HasBg || firstSpan.BorderBottom.Width != 0 || continuedSpan.BorderTop.Width != 0 || continuedSpan.BorderBottom.Width == 0 {
		t.Errorf("rowspan decoration does not continue across the page break: first=%+v continued=%+v", firstSpan, continuedSpan)
	}
	if firstSpan.Text != "SPAN" || continuedSpan.Text != "" || len(continuedSpan.Runs) != 0 {
		t.Errorf("content rowspan duplicado in continuation: first=%+v continued=%+v", firstSpan, continuedSpan)
	}
	if len(firstSpan.Runs) == 0 || len(firstSpan.Runs[0]) == 0 || firstSpan.Runs[0][0].Y > 2 {
		t.Errorf("rowspan with default vertical alignment must start at the top, as in Chromium, runs=%+v", firstSpan.Runs)
	}
	var gotLines []string
	pageLineCounts := make([]int, len(pages))
	for pageIndex, page := range pages {
		for _, block := range page.Blocks {
			if block.source != contentCell {
				continue
			}
			pageLineCounts[pageIndex] += len(block.Runs)
			for _, line := range block.Runs {
				for _, run := range line {
					gotLines = append(gotLines, run.Text)
				}
			}
		}
	}
	if strings.Join(gotLines, "|") != strings.Join(lines, "|") {
		t.Errorf("content below rowspan does not continue: got %q, want %q", strings.Join(gotLines, "|"), strings.Join(lines, "|"))
	}
	if pageLineCounts[0] != 8 || pageLineCounts[1] != 9 {
		t.Errorf("lines below the rowspan by page=%v, Chromium/WeasyPrint produce [8 9] with borders", pageLineCounts)
	}
	if spanFragments[0].Bounds.Height > opts.PageHeight+fragmentEpsilon || spanFragments[1].Bounds.Height > opts.PageHeight+fragmentEpsilon {
		t.Errorf("rowspan segment exceeds its fragmentainer: bounds=%+v / %+v", spanFragments[0].Bounds, spanFragments[1].Bounds)
	}
}

func TestLayout_TableFragmentsRowspanIntroducedByOversizedRow(t *testing.T) {
	lines := make([]string, 17)
	for index := range lines {
		lines[index] = fmt.Sprintf("ROWSPAN-%02d", index+1)
	}
	html := `<style>body{margin:0}table{margin:0;width:220pt;border-collapse:collapse}td{padding:0;border:1pt solid #111}</style><table><tbody><tr><td rowspan="2" style="background:#fee;width:66.6667%">` + strings.Join(lines, "<br>") + `</td><td>START</td></tr><tr><td>NEXT</td></tr></tbody></table>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 220, 120, 0
	opts.FontSize, opts.LineHeight = 10, 1.2
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages=%d, Chromium/WeasyPrint produce 2", len(pages))
	}
	spanCell := findTableCellContaining(root, "ROWSPAN-01")
	if spanCell == nil {
		t.Fatal("rowspan cell was not found")
	}
	spanFragments := fragmentsForSource(pages, spanCell)
	if len(spanFragments) != 2 {
		t.Fatalf("fragments of cell rowspan=%d, Chromium/WeasyPrint produce 2", len(spanFragments))
	}
	assertFragmentChain(t, spanFragments)
	firstSpan := pages[0].Blocks[spanFragments[0].PaintIndex]
	continuedSpan := pages[1].Blocks[spanFragments[1].PaintIndex]
	if !firstSpan.HasBg || !continuedSpan.HasBg || firstSpan.BorderBottom.Width != 0 || continuedSpan.BorderTop.Width != 0 || continuedSpan.BorderBottom.Width == 0 {
		t.Errorf("rowspan background or borders do not continue: first=%+v continued=%+v", firstSpan, continuedSpan)
	}
	var gotLines []string
	pageLineCounts := make([]int, len(pages))
	markers := map[string]int{}
	for pageIndex, page := range pages {
		for _, block := range page.Blocks {
			if block.source == spanCell {
				pageLineCounts[pageIndex] += len(block.Runs)
				for _, line := range block.Runs {
					for _, run := range line {
						gotLines = append(gotLines, run.Text)
					}
				}
			}
			for _, marker := range []string{"START", "NEXT"} {
				if strings.Contains(block.Text, marker) {
					markers[marker]++
				}
			}
		}
	}
	if strings.Join(gotLines, "|") != strings.Join(lines, "|") {
		t.Errorf("lines rowspan=%v, want %v", gotLines, lines)
	}
	if pageLineCounts[0] != 9 || pageLineCounts[1] != 8 {
		t.Errorf("lines by page=%v, Chromiumproduces [9 8]", pageLineCounts)
	}
	if markers["START"] != 1 || markers["NEXT"] != 1 {
		t.Errorf("cells vecinas duplicadas/losts: %v", markers)
	}
	for pageIndex, page := range pages {
		for _, block := range page.Blocks {
			for _, line := range block.Runs {
				for _, run := range line {
					if run.Y < -fragmentEpsilon || run.Y+run.LineH > opts.PageHeight+fragmentEpsilon {
						t.Errorf("page %d: text outside the fragmentainer: %+v", pageIndex, run)
					}
				}
			}
		}
	}
}

func TestLayout_TableCellBlocksPreserveParagraphMarginsAndAlignment(t *testing.T) {
	html := `<style>body{margin:0}table{margin:0;width:220pt;border-collapse:collapse}td{padding:2pt;border:1pt solid #111;font:10pt/12pt Helvetica}td:first-child{width:66.6667%;background:#fee}p{margin:0 0 6pt}</style><table><tbody><tr><td><div><p style="font:12pt/14pt Helvetica">ALPHA <strong>BOLD</strong></p><p>BETA second paragraph</p></div></td><td>RIGHT</td></tr></tbody></table>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 220, 150, 0
	opts.FontSize, opts.LineHeight = 10, 1.2
	pages := Flow(root, opts)
	if len(pages) != 1 {
		t.Fatalf("pages=%d, want 1", len(pages))
	}
	positions := map[string]float64{}
	boldPreserved := false
	for _, block := range pages[0].Blocks {
		for _, line := range block.Runs {
			for _, run := range line {
				if run.Text == "ALPHA" || run.Text == "BETA" || run.Text == "RIGHT" {
					positions[run.Text] = run.Y
				}
				if run.Text == "BOLD" && run.Bold {
					boldPreserved = true
				}
			}
		}
	}
	for _, text := range []string{"ALPHA", "BETA", "RIGHT"} {
		if _, ok := positions[text]; !ok {
			t.Fatalf("text not found %s in the layout: %v", text, positions)
		}
	}
	if delta := positions["BETA"] - positions["ALPHA"]; delta < 19.5 || delta > 20.5 {
		t.Errorf("line height or margin between paragraphs was lost: delta y=%.2f, Chrome/WeasyPrint produce 20pt", delta)
	}
	if delta := positions["RIGHT"] - positions["ALPHA"]; delta < 12.5 || delta > 13.5 {
		t.Errorf("neighboring cell is not centered against the height of both paragraphs: delta y=%.2f, Chrome/WeasyPrint produce 13pt", delta)
	}
	if !boldPreserved {
		t.Error("inline strong styling was lost inside the table block")
	}
}

func TestLayout_TableCellParagraphsContinueAcrossPages(t *testing.T) {
	lines := make([]string, 17)
	for index := range lines {
		lines[index] = fmt.Sprintf("FIRST-%02d", index+1)
	}
	html := `<style>body{margin:0}table{margin:0;width:220pt;border-collapse:collapse}td{padding:2pt;border:1pt solid #111;font:10pt/12pt Helvetica}td:first-child{width:66.6667%;background:#fee}p{margin:0 0 6pt}</style><table><tbody><tr><td><p>` + strings.Join(lines, "<br>") + `</p><p>end-PARAGRAPH</p></td><td>SUMMARY</td></tr></tbody></table>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 220, 150, 0
	opts.FontSize, opts.LineHeight = 10, 1.2
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages=%d, Chromium/WeasyPrint produce 2", len(pages))
	}
	cell := findTableCellContaining(root, "FIRST-01")
	if cell == nil {
		t.Fatal("paragraph cell was not found")
	}
	var got []string
	pageCounts := make([]int, len(pages))
	summaryY := -1.0
	for pageIndex, page := range pages {
		for _, block := range page.Blocks {
			for _, line := range block.Runs {
				for _, run := range line {
					if run.Text == "SUMMARY" {
						summaryY = run.Y
					}
				}
			}
			if block.source != cell {
				continue
			}
			pageCounts[pageIndex] += len(block.Runs)
			for _, line := range block.Runs {
				for _, run := range line {
					got = append(got, run.Text)
					if run.Y < -fragmentEpsilon || run.Y+run.LineH > opts.PageHeight+fragmentEpsilon {
						t.Errorf("page %d: text outside the fragmentainer: %+v", pageIndex, run)
					}
				}
			}
		}
	}
	want := append(append([]string(nil), lines...), "end-PARAGRAPH")
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("order/continuation of paragraphs alterado: got %v, want %v", got, want)
	}
	if pageCounts[0] != 12 || pageCounts[1] != 6 {
		t.Errorf("lines by page=%v, Chromiumproduces [12 6] incluyendo the second paragraph", pageCounts)
	}
	if summaryY < 0 || summaryY > 5 {
		t.Errorf("short cell must align to the top of the first fragment, as in Chromium; y=%.2f", summaryY)
	}
}

func TestLayout_TableCellBlockDecorationsFragmentWithTheirContent(t *testing.T) {
	lines := make([]string, 17)
	for index := range lines {
		lines[index] = fmt.Sprintf("DECOR-%02d", index+1)
	}
	html := `<style>body{margin:0}table{margin:0;width:220pt;border-collapse:collapse}td{padding:0;border:0;font:10pt/12pt Helvetica;vertical-align:top}td:first-child{width:66.6667%}div{background:#fff0cc;border:1pt solid #995500;padding:2pt}p{margin:0;background:#cceeff;border:2pt solid #123456;padding:3pt}</style><table><tbody><tr><td><div><p>` + strings.Join(lines, "<br>") + `</p></div></td><td>SUMMARY</td></tr></tbody></table>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 220, 150, 0
	opts.FontSize, opts.LineHeight = 10, 1.2
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages=%d, want 2 for the blockdecorated", len(pages))
	}
	paragraph := findTableCellBlock(root, "p", "DECOR-01")
	if paragraph == nil {
		t.Fatal("paragraph source was not found")
	}
	wrapper := findTableCellBlock(root, "div", "DECOR-01")
	if wrapper == nil {
		t.Fatal("wrapper block source was not found")
	}
	var fragments []PlacedBox
	var wrapperFragments []PlacedBox
	linesPerPage := make([]int, len(pages))
	fragments = placedFragmentsForSource(pages, paragraph)
	wrapperFragments = placedFragmentsForSource(pages, wrapper)
	for pageIndex, page := range pages {
		for _, cell := range page.Blocks {
			for _, line := range cell.Runs {
				for _, run := range line {
					if strings.HasPrefix(run.Text, "DECOR-") {
						linesPerPage[pageIndex]++
					}
				}
			}
		}
	}
	if len(fragments) != 2 {
		t.Fatalf("decorated paragraph fragments=%d, want 2", len(fragments))
	}
	if len(wrapperFragments) != 2 {
		t.Fatalf("decorated wrapper fragments=%d, want 2", len(wrapperFragments))
	}
	if linesPerPage[0] != 11 || linesPerPage[1] != 6 {
		t.Errorf("lines per page=%v, Chromium and WeasyPrint produce [11 6] with the decorated boxes", linesPerPage)
	}
	for index, fragment := range fragments {
		if !fragment.HasBg || fragment.Bg != paragraph.Style.Bg {
			t.Errorf("fragment %d lost the background of the paragraph: HasBg=%v Bg=%v", index, fragment.HasBg, fragment.Bg)
		}
		if (index == 0 && fragment.BorderTop.Width != 2) || (index == 1 && fragment.BorderTop.Width != 0) ||
			(index == 0 && fragment.BorderBottom.Width != 0) || (index == 1 && fragment.BorderBottom.Width != 2) {
			t.Errorf("fragment %d does not apply box-decoration-break:slice: top=%v bottom=%v", index, fragment.BorderTop.Width, fragment.BorderBottom.Width)
		}
		if fragment.Width <= 0 || fragment.Height <= 0 {
			t.Errorf("fragment %d without geometry of border-box: %+v", index, fragment)
		}
		outer := wrapperFragments[index]
		if !outer.HasBg || outer.Width <= fragment.Width || outer.Height < fragment.Height {
			t.Errorf("the nested decorated box must wrap to its paragraph: outer=%+v inner=%+v", outer, fragment)
		}
		if (index == 0 && outer.BorderTop.Width != 1) || (index == 1 && outer.BorderTop.Width != 0) ||
			(index == 0 && outer.BorderBottom.Width != 0) || (index == 1 && outer.BorderBottom.Width != 1) {
			t.Errorf("wrapper fragment %d does not preserve the border slice: top=%v bottom=%v", index, outer.BorderTop.Width, outer.BorderBottom.Width)
		}
	}
}

func findTableCellBlock(root *boxes.Box, tag, value string) *boxes.Box {
	if root.Tag == tag && strings.Contains(root.Text, value) {
		return root
	}
	for _, child := range root.Children {
		if found := findTableCellBlock(child, tag, value); found != nil {
			return found
		}
	}
	return nil
}

func TestLayout_TableCellDecorationsKeepEachBlockBoundary(t *testing.T) {
	html := `<style>body{margin:0}table{margin:0;width:220pt;border-collapse:collapse}td{padding:0;border:0;font:10pt/12pt Helvetica;vertical-align:top}td:first-child{width:66.6667%}p{margin:4pt 0;background:#cceeff;border:2pt solid #123456;padding:3pt}</style><table><tbody><tr><td><p>FIRST-BLOCK</p><p>SECOND-BLOCK</p><p>THIRD-BLOCK</p></td><td>SUMMARY</td></tr></tbody></table>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 220, 150, 0
	opts.FontSize, opts.LineHeight = 10, 1.2
	pages := Flow(root, opts)
	if len(pages) != 1 {
		t.Fatalf("pages=%d, want 1", len(pages))
	}
	var decorations []PlacedBox
	for _, pageNode := range pages[0].FragmentTree {
		var visit func([]*FragmentNode)
		visit = func(nodes []*FragmentNode) {
			for _, node := range nodes {
				if node.Kind == FragmentDecoration && node.Fragment.Source != nil && node.Fragment.Source.Tag == "p" {
					decorations = append(decorations, node.Placed)
				}
				visit(node.Children)
			}
		}
		visit([]*FragmentNode{pageNode})
	}
	if len(decorations) != 3 {
		t.Fatalf("paragraphsdecorateds=%d, want 3", len(decorations))
	}
	for index, decoration := range decorations {
		if decoration.BorderTop.Width != 2 || decoration.BorderBottom.Width != 2 {
			t.Errorf("block %d must preserve both edges: top=%v bottom=%v", index, decoration.BorderTop.Width, decoration.BorderBottom.Width)
		}
		if decoration.Height < 20 || decoration.Width <= 0 {
			t.Errorf("block %d has insufficient geometry: %+v", index, decoration)
		}
	}
	for index := 1; index < len(decorations); index++ {
		if decorations[index].Y < decorations[index-1].Y+decorations[index-1].Height {
			t.Errorf("decorated boxes of consecutive blocks overlap: prev=%+v next=%+v", decorations[index-1], decorations[index])
		}
	}
}

func TestLayout_NestedTableInCellKeepsItsOwnColumnGeometry(t *testing.T) {
	html := `<style>body{margin:0}table{margin:0;border-collapse:collapse}#outer{width:220pt}td{font:10pt/12pt Helvetica;padding:2pt;border:1pt solid #111}#inner td:first-child{width:50%;background:#cceeff}#inner td:last-child{background:#fff0cc}</style><table id="outer"><tr><td>OUTER <table id="inner"><tr><td>LEFT</td><td>RIGHT</td></tr></table> AFTER</td><td>SUMMARY</td></tr></table>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 220, 180, 0
	opts.FontSize, opts.LineHeight = 10, 1.2
	pages := Flow(root, opts)
	if len(pages) != 1 {
		t.Fatalf("pages=%d, want 1", len(pages))
	}
	leftCell := findTableCellContaining(root, "LEFT")
	rightCell := findTableCellContaining(root, "RIGHT")
	if leftCell == nil || rightCell == nil {
		t.Fatal("both cells in the nested table were not found")
	}
	var left, right *PlacedBox
	for index := range pages[0].Blocks {
		block := &pages[0].Blocks[index]
		if block.source == leftCell {
			left = block
		}
		if block.source == rightCell {
			right = block
		}
	}
	if left == nil || right == nil {
		t.Fatalf("nested cell boxes were lost: left=%v right=%v", left != nil, right != nil)
	}
	if right.X <= left.X || math.Abs(right.Y-left.Y) > fragmentEpsilon {
		t.Errorf("the nested grid must place both cells in the same row and different columns: left=(%g,%g) right=(%g,%g)", left.X, left.Y, right.X, right.Y)
	}
	if left.Width+right.Width > 110+fragmentEpsilon {
		t.Errorf("the cells exceed the outer cell width: left=%g right=%g", left.Width, right.Width)
	}
}

func TestLayout_NestedAutoTableUsesIntrinsicContentWidth(t *testing.T) {
	html := `<style>body{margin:0}table{margin:0;border-collapse:collapse}#outer{width:220pt}td{font:10pt/12pt Helvetica;padding:2pt;border:1pt solid #111}</style><table id="outer"><tr><td><table id="inner"><tr><td>LEFT</td><td>RIGHT</td></tr></table></td><td>SUMMARY</td></tr></table>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 220, 180, 0
	opts.FontSize, opts.LineHeight = 10, 1.2
	pages := Flow(root, opts)
	leftCell := findTableCellContaining(root, "LEFT")
	rightCell := findTableCellContaining(root, "RIGHT")
	var leftWidth, rightWidth float64
	for _, block := range pages[0].Blocks {
		if block.source == leftCell {
			leftWidth = block.Width
		}
		if block.source == rightCell {
			rightWidth = block.Width
		}
	}
	total := leftWidth + rightWidth
	if total <= 0 {
		t.Fatal("the inner table cells were not preserved")
	}
	if total > 85 {
		t.Errorf("auto table should shrink to its intrinsic content width, got %g pt, want <=85 pt", total)
	}
}

func TestLayout_AutoTableUsesIntrinsicContentWidth(t *testing.T) {
	root, err := boxes.Build(`<table><tr><td>ALPHA</td><td>BETA</td></tr></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("cells=%d, want 2", len(blocks))
	}
	total := blocks[0].Width + blocks[1].Width
	if total <= 0 || total >= 451 {
		t.Errorf("auto table should use intrinsic width (0,451), got %g pt", total)
	}
}

func TestLayout_NestedTableRowsContinueAcrossPages(t *testing.T) {
	html := `<style>body{margin:0}table{margin:0;border-collapse:collapse}#outer{width:220pt}td{font:10pt/12pt Helvetica;padding:1pt;border:1pt solid #111}</style><table id="outer"><tr><td><table id="inner"><tr><td>INNER-1</td></tr><tr><td>INNER-2</td></tr><tr><td>INNER-3</td></tr><tr><td>INNER-4</td></tr></table></td><td>SUMMARY</td></tr></table>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 220, 45, 0
	opts.FontSize, opts.LineHeight = 10, 1.2
	pages := Flow(root, opts)
	if len(pages) < 2 {
		t.Fatalf("nested rows occupy %d pages; they must continue on another page", len(pages))
	}
	pageFor := make(map[string]int)
	cells := make(map[*boxes.Box]string)
	for index := 1; index <= 4; index++ {
		marker := fmt.Sprintf("INNER-%d", index)
		cell := findTableCellContaining(root, marker)
		if cell == nil {
			t.Fatalf("cell %s was not found", marker)
		}
		cells[cell] = marker
	}
	for pageIndex, page := range pages {
		for _, block := range page.Blocks {
			if marker, ok := cells[block.source]; ok {
				pageFor[marker] = pageIndex + 1
			}
		}
	}
	if len(pageFor) != 4 {
		t.Fatalf("preserved inner rows=%d, want 4: %v", len(pageFor), pageFor)
	}
	if pageFor["INNER-1"] != 1 || pageFor["INNER-2"] != 1 || pageFor["INNER-3"] != 2 || pageFor["INNER-4"] != 2 {
		t.Errorf("the inner table must continue in order across pages: %v", pageFor)
	}
}

func TestLayout_NestedTableIntrinsicWidthKeepsOuterColumnNearReference(t *testing.T) {
	html := `<style>body{margin:0}table{margin:0;border-collapse:collapse}#outer{width:220pt}td{font:10pt/12pt Helvetica;padding:1pt;border:1pt solid #111}</style><table id="outer"><tr><td><table id="inner"><tr><td>INNER-1</td></tr><tr><td>INNER-2</td></tr><tr><td>INNER-3</td></tr><tr><td>INNER-4</td></tr></table></td><td>SUMMARY</td></tr></table>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatal(err)
	}
	inner := findTableCellContaining(root, "INNER-1")
	var outer *boxes.Box
	for ancestor := inner.Parent; ancestor != nil; ancestor = ancestor.Parent {
		if ancestor.Tag == "td" {
			outer = ancestor
			break
		}
	}
	if outer == nil {
		t.Fatal("cell containing the inner table was not found")
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 220, 45, 0
	opts.Measure, opts.MeasureBold = text.Measure, text.MeasureBold
	pages := Flow(root, opts)
	for _, block := range pages[0].Blocks {
		if block.source == outer {
			if block.Width < 98 || block.Width > 107 {
				t.Fatalf("column outer=%.2fpt, referencias Chromium/WeasyPrint ~102pt", block.Width)
			}
			return
		}
	}
	t.Fatal("outer cell was not emitted")
}

func findTableCellContaining(root *boxes.Box, value string) *boxes.Box {
	for _, child := range root.Children {
		if found := findTableCellContaining(child, value); found != nil {
			return found
		}
	}
	if root.Tag == "td" && strings.Contains(root.Text, value) {
		return root
	}
	return nil
}

func TestLayout_TableRowFragmentsPreserveCellsAndRepeatHeader(t *testing.T) {
	words := make([]string, 100)
	for index := range words {
		words[index] = fmt.Sprintf("DATA%03d", index+1)
	}
	html := `<style>body{margin:0} table{margin:0;width:220pt;border-collapse:collapse} th,td{padding:0}</style><table><thead><tr><th>HEADER</th><th>STATUS</th></tr></thead><tbody><tr><td>` + strings.Join(words, " ") + `</td><td>SUMMARY</td></tr></tbody></table>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 220, 120, 0
	opts.FontSize, opts.LineHeight = 10, 1.2
	pages := Flow(root, opts)
	cell := findBoxWithTag(root, "td")
	header := findBoxWithTag(root, "th")
	cellFragments := fragmentsForSource(pages, cell)
	if len(pages) < 2 || len(cellFragments) < 2 {
		t.Fatalf("pages=%d fragments of cell=%d, want row sincronizada in several pages", len(pages), len(cellFragments))
	}
	assertFragmentChain(t, cellFragments)
	var got []string
	for pageIndex, page := range pages {
		var headerSeen bool
		var headerRepeated bool
		for fragmentIndex, fragment := range page.Fragments {
			block := page.Blocks[fragment.PaintIndex]
			if block.source == header && block.Text == "HEADER" {
				headerSeen = true
				headerRepeated = fragment.Repeated
			}
			if block.source == cell {
				for _, line := range block.Runs {
					for _, run := range line {
						got = append(got, strings.Fields(run.Text)...)
					}
				}
			}
			if fragment.PaintIndex != fragmentIndex {
				t.Errorf("page %d: PaintIndex=%d in index %d", pageIndex+1, fragment.PaintIndex, fragmentIndex)
			}
		}
		if !headerSeen {
			t.Errorf("page %d without to repeated header", pageIndex+1)
		}
		if pageIndex > 0 && !headerRepeated {
			t.Errorf("header of page %d is not marked as repeated", pageIndex+1)
		}
	}
	if strings.Join(got, " ") != strings.Join(words, " ") {
		t.Errorf("text sincronizado outside order or duplicado:\n got %q\nwant %q", strings.Join(got, " "), strings.Join(words, " "))
	}
}

func TestLayout_TableFragmentsForcedLinesLikeChromeAndWeasyPrint(t *testing.T) {
	lines := make([]string, 17)
	markupLines := make([]string, 17)
	for index := range lines {
		lines[index] = fmt.Sprintf("ROW-LINE-%02d", index+1)
		markupLines[index] = lines[index]
	}
	markupLines[9] = `<strong>` + markupLines[9] + `</strong>`
	html := `<style>body{margin:0} table{margin:0;width:220pt;border-collapse:collapse} th,td{padding:0;font:10pt/12pt "Liberation Sans"}</style><table><thead><tr><th>HEADER</th><th>STATUS</th></tr></thead><tbody><tr><td>` + strings.Join(markupLines, "<br>") + `</td><td>SUMMARY</td></tr></tbody></table>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 220, 120, 0
	opts.FontSize, opts.LineHeight = 10, 1.2
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages=%d, Chromium/WeasyPrint produce 2", len(pages))
	}
	cell := findBoxWithTag(root, "td")
	header := findBoxWithTag(root, "th")
	var gotLines []string
	var styledContinuation bool
	pageLineCounts := make([]int, len(pages))
	for pageIndex, page := range pages {
		headerSeen := false
		for _, fragment := range page.Fragments {
			block := page.Blocks[fragment.PaintIndex]
			if block.source == header && block.Text == "HEADER" {
				headerSeen = true
				if pageIndex == 1 && !fragment.Repeated {
					t.Error("the header on the second page must be marked as repeated")
				}
			}
			if block.source == cell {
				pageLineCounts[pageIndex] += len(block.Runs)
				for _, line := range block.Runs {
					for _, run := range line {
						gotLines = append(gotLines, run.Text)
						if run.Text == lines[9] && run.Bold {
							styledContinuation = true
						}
					}
				}
			}
		}
		if !headerSeen {
			t.Errorf("page %d without header", pageIndex+1)
		}
	}
	if strings.Join(gotLines, "|") != strings.Join(lines, "|") {
		t.Errorf("lines of cell=%q, want %q", strings.Join(gotLines, "|"), strings.Join(lines, "|"))
	}
	if pageLineCounts[0] != 9 || pageLineCounts[1] != 8 {
		t.Errorf("lines by page=%v, Chromium/WeasyPrint produce [9 8]", pageLineCounts)
	}
	if !styledContinuation {
		t.Error("strong styling was lost from the first line of the continuation")
	}
}

func TestLayout_TableRowFragmentsEachCellIndependently(t *testing.T) {
	bigLines, smallLines := make([]string, 17), make([]string, 17)
	for index := range bigLines {
		bigLines[index] = fmt.Sprintf("BIG-%02d", index+1)
		smallLines[index] = fmt.Sprintf("SMALL-%02d", index+1)
	}
	html := `<style>body{margin:0}table{margin:0;width:220pt;border-collapse:collapse}th{padding:0;font:10pt/12pt Helvetica}td{padding:0}</style><table><thead><tr><th>BIG</th><th>SMALL</th></tr></thead><tbody><tr><td style="font:10pt/12pt Helvetica">` + strings.Join(bigLines, "<br>") + `</td><td style="font:5pt/6pt Helvetica">` + strings.Join(smallLines, "<br>") + `</td></tr></tbody></table>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 220, 120, 0
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages=%d, want 2", len(pages))
	}
	bigCell := findTableCellContaining(root, "BIG-01")
	smallCell := findTableCellContaining(root, "SMALL-01")
	if bigCell == nil || smallCell == nil {
		t.Fatal("both cells were not found")
	}
	pageCounts := func(cell *boxes.Box) []int {
		counts := make([]int, len(pages))
		for pageIndex, page := range pages {
			for _, block := range page.Blocks {
				if block.source == cell {
					counts[pageIndex] += len(block.Runs)
				}
			}
		}
		return counts
	}
	if got := pageCounts(bigCell); got[0] != 9 || got[1] != 8 {
		t.Errorf("lineslarge by page=%v, want [9 8]", got)
	}
	if got := pageCounts(smallCell); got[0] != 17 || got[1] != 0 {
		t.Errorf("small cell lines per page=%v, Chromium/WeasyPrint produces [17 0]", got)
	}
}

func findBoxWithTag(root *boxes.Box, tag string) *boxes.Box {
	if root.Tag == tag {
		return root
	}
	for _, child := range root.Children {
		if found := findBoxWithTag(child, tag); found != nil {
			return found
		}
	}
	return nil
}

func TestLayout_TransparentWrapperKeepsTableAsTable(t *testing.T) {
	root, err := boxes.Build(`<section><h2>Vital signs</h2><table><tr><th>Pressure</th><th>Heart rate</th></tr><tr><td>118/76</td><td>72 bpm</td></tr></table><p>After</p></section>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	var cells []PlacedBox
	for _, block := range blocks {
		if block.Text == "Pressure" || block.Text == "Heart rate" || block.Text == "118/76" || block.Text == "72 bpm" {
			cells = append(cells, block)
		}
	}
	if len(cells) != 4 {
		t.Fatalf("preserved cells: got %d, want 4; blocks=%#v", len(cells), blocks)
	}
	if cells[0].X == cells[1].X || cells[2].X == cells[3].X {
		t.Errorf("the cells in each row must occupy different columns: x=%v/%v and %v/%v", cells[0].X, cells[1].X, cells[2].X, cells[3].X)
	}
}

func TestLayout_TableCellPaddingOffsetsContentAndExpandsRow(t *testing.T) {
	root, err := boxes.Build(`<table><tr><td style="padding: 5px">A</td><td>B</td></tr><tr><td>C</td><td>D</td></tr></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 4 {
		t.Fatalf("blocks: got %d, want 4", len(blocks))
	}
	cell := blocks[0]
	if cell.Height != 21.9 {
		t.Errorf("cell height: got %v, want 21.9 (line + vertical padding)", cell.Height)
	}
	if len(cell.Runs) != 1 || len(cell.Runs[0]) != 1 {
		t.Fatalf("runs of cell: %#v", cell.Runs)
	}
	run := cell.Runs[0][0]
	if run.X != cell.X+3.75 || run.Y != cell.Y+3.75 {
		t.Errorf("text origin: got (%v,%v), want (%v,%v)", run.X, run.Y, cell.X+3.75, cell.Y+3.75)
	}
	if blocks[2].Y != cell.Y+cell.Height+1.5 {
		t.Errorf("following row starts in %v, want %v including the default row spacing", blocks[2].Y, cell.Y+cell.Height+1.5)
	}
}

func TestLayout_TableCellBordersContributeToRowHeightAndTextOffset(t *testing.T) {
	root, err := boxes.Build(`<style>td{padding:5pt;border:1px solid #123456}</style><table><tr><td>A</td></tr></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	cell := Flow(root, testOptions())[0].Blocks[0]
	if cell.Height != 25.9 {
		t.Errorf("outer row height: got %v, want 25.9pt (line + padding + borders)", cell.Height)
	}
	if len(cell.Runs) != 1 || len(cell.Runs[0]) != 1 {
		t.Fatalf("corridas of cell: %#v", cell.Runs)
	}
	if got := cell.Runs[0][0].Y; got != 79.25 {
		t.Errorf("text must start after table spacing, top border, and padding: y=%v, want 79.25", got)
	}
}

func TestLayout_CollapsedTableBordersShareRowGeometry(t *testing.T) {
	root, err := boxes.Build(`<style>table{border-collapse:collapse}td{padding:5pt;border:1px solid #123456}</style><table><tr><td>A</td></tr><tr><td>B</td></tr></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("cells=%d, want 2", len(blocks))
	}
	if !approx(blocks[0].Height, 25.15) || !approx(blocks[1].Height, 25.15) {
		t.Errorf("height of row with border collapsed=%v/%v, want 25.15pt", blocks[0].Height, blocks[1].Height)
	}
	if got := blocks[1].Y - blocks[0].Y; !approx(got, 25.15) {
		t.Errorf("distance between rows=%v, want 25.15pt with shared border", got)
	}
}

func TestLayout_TableCells_DefaultToVerticalMiddle(t *testing.T) {
	root, err := boxes.Build(`<table><tr><td>A</td><td>` + strings.Repeat("word ", 30) + `</td></tr></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("blocks: got %d, want 2", len(blocks))
	}
	cell, tall := blocks[0], blocks[1]
	if len(tall.Runs) < 2 {
		t.Fatalf("the second cell must contain multiple lines, got %d", len(tall.Runs))
	}
	lineH := cell.Runs[0][0].LineH
	wantY := cell.Y + (cell.Height-lineH)/2
	if cell.Runs[0][0].Y != wantY {
		t.Errorf("content td in y=%v, want middle=%v (height row %v)", cell.Runs[0][0].Y, wantY, cell.Height)
	}
}

func TestLayout_TableMarginsCollapseWithAdjacentBlocks(t *testing.T) {
	root, err := boxes.Build(`<p style="margin:0 0 8pt">A</p><table style="margin-top:12pt"><tr><td>B</td></tr></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	root.Style.BodyMargin = 0
	opts := testOptions()
	opts.Margin = 0
	blocks := Flow(root, opts)[0].Blocks
	var paragraph, cell *PlacedBox
	for i := range blocks {
		if len(blocks[i].Runs) > 0 && len(blocks[i].Runs[0]) > 0 && blocks[i].Runs[0][0].Text == "A" {
			paragraph = &blocks[i]
		}
		if blocks[i].Text == "B" {
			cell = &blocks[i]
		}
	}
	if paragraph == nil || cell == nil {
		t.Fatalf("could not locate paragraph and cell: %#v", blocks)
	}
	want := paragraph.Y + paragraph.Height + 12
	want += 1.5
	if !approx(cell.Y, want) {
		t.Errorf("cell y=%v, want %v when collapsing margins and applying default table spacing", cell.Y, want)
	}
}

func TestLayout_TableCellKeepsAncestorPaintLayers(t *testing.T) {
	html := `<table style="background:#fff2cc"><thead style="background:#163b64"><tr style="background:#008000"><th>HEAD</th></tr></thead><tbody style="background:#e6e6e6"><tr style="background:#0000ff"><td>BODY</td></tr></tbody></table>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	pages := Flow(root, testOptions())
	var head, body *PlacedBox
	for pageIndex := range pages {
		for blockIndex := range pages[pageIndex].Blocks {
			block := &pages[pageIndex].Blocks[blockIndex]
			if block.Text == "HEAD" {
				head = block
			}
			if block.Text == "BODY" {
				body = block
			}
		}
	}
	if head == nil || body == nil {
		t.Fatalf("table cell blocks not found: head=%v body=%v", head, body)
	}
	assertLayers := func(name string, block *PlacedBox, want ...string) {
		t.Helper()
		if len(block.TablePaintUnderlays) != len(want) {
			t.Fatalf("%s underlay count=%d, want %d", name, len(block.TablePaintUnderlays), len(want))
		}
		for i, tag := range want {
			if block.TablePaintUnderlays[i].Source.Tag != tag {
				t.Errorf("%s underlay %d source=%q, want %q", name, i, block.TablePaintUnderlays[i].Source.Tag, tag)
			}
		}
	}
	assertLayers("header", head, "table", "thead", "tr")
	assertLayers("body", body, "table", "tbody", "tr")
	if len(head.TablePaintOverlays) != 0 || len(body.TablePaintOverlays) != 0 {
		t.Error("layers without borders must not create border paint operations")
	}
}

func TestLayout_TableGroupOpacityWrapsItsCellContents(t *testing.T) {
	root, err := boxes.Build(`<table><thead style="opacity:.5"><tr><th>HEAD</th><th>SECOND</th></tr></thead><tbody><tr><td>BODY</td><td>ROW</td></tr></tbody></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	pages := Flow(root, testOptions())
	var header []PlacedBox
	for _, block := range pages[0].Blocks {
		if block.Text == "HEAD" || block.Text == "SECOND" {
			header = append(header, block)
		}
	}
	if len(header) != 2 {
		t.Fatalf("header cells=%d, want 2", len(header))
	}
	if len(header[0].OpacityGroups) != 1 || header[0].OpacityGroups[0].Opacity != 0.5 || header[0].OpacityGroups[0].ID != header[1].OpacityGroups[0].ID {
		t.Fatalf("header cells must share one row-group opacity: %+v", header)
	}
	for _, block := range pages[0].Blocks {
		if block.Text == "BODY" && len(block.OpacityGroups) != 0 {
			t.Errorf("body cell unexpectedly belongs to header opacity group: %+v", block.OpacityGroups)
		}
	}
}

func TestLayout_TablePaintLayersRetainStructuralBorders(t *testing.T) {
	root, err := boxes.Build(`<table style="border-collapse:separate;border:2px solid red"><thead style="border:1px solid blue"><tr style="border-bottom:1px solid green"><th>HEAD</th></tr></thead><tbody><tr><td>BODY</td></tr></tbody></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var header *PlacedBox
	for _, block := range Flow(root, testOptions())[0].Blocks {
		if block.Text == "HEAD" {
			header = &block
			break
		}
	}
	if header == nil {
		t.Fatal("header cell not found")
	}
	want := []string{"table", "thead", "tr"}
	if len(header.TablePaintOverlays) != len(want) {
		t.Fatalf("border layers=%d, want %d", len(header.TablePaintOverlays), len(want))
	}
	for i, tag := range want {
		if header.TablePaintOverlays[i].Source.Tag != tag {
			t.Errorf("border layer %d source=%q, want %q", i, header.TablePaintOverlays[i].Source.Tag, tag)
		}
	}
}

func TestLayout_CollapsedTableEdgesResolveCellBeforeStructuralTies(t *testing.T) {
	root, err := boxes.Build(`<style>table{border-collapse:collapse}.row{border-top:2pt solid red}td{border-top:2pt solid blue}</style><table><tr class="row"><td>TOTAL</td></tr></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var total *PlacedBox
	for _, block := range Flow(root, testOptions())[0].Blocks {
		if block.Text == "TOTAL" {
			total = &block
			break
		}
	}
	if total == nil {
		t.Fatal("total cell not found")
	}
	if total.BorderTop.Width != 2 {
		t.Errorf("resolved top edge width=%g, want full 2pt", total.BorderTop.Width)
	}
	if total.BorderTop.Color[2] != 1 || total.BorderTop.Color[0] != 0 {
		t.Errorf("equal-width collapsed border color=%v, want cell blue", total.BorderTop.Color)
	}
}

func TestLayout_CollapsedTableSharedCellEdgeUsesWiderCandidate(t *testing.T) {
	root, err := boxes.Build(`<style>table{border-collapse:collapse}td:first-child{border-right:2pt solid red}td:last-child{border-left:3pt solid blue}</style><table><tr><td>LEFT</td><td>RIGHT</td></tr></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var left, right *PlacedBox
	for _, block := range Flow(root, testOptions())[0].Blocks {
		switch block.Text {
		case "LEFT":
			left = &block
		case "RIGHT":
			right = &block
		}
	}
	if left == nil || right == nil {
		t.Fatalf("cells not found: left=%v right=%v", left != nil, right != nil)
	}
	for name, side := range map[string]boxes.BorderSide{"left cell": left.BorderRight, "right cell": right.BorderLeft} {
		if side.Width != 3 || side.Color[2] != 1 {
			t.Errorf("%s shared edge=%+v, want full 3pt blue winner", name, side)
		}
	}
}

func TestLayout_CollapsedTableSharedRowEdgeUsesOneWinner(t *testing.T) {
	root, err := boxes.Build(`<style>table{border-collapse:collapse}tr:first-child td{border-bottom:2pt solid red}tr:last-child td{border-top:1pt solid blue}</style><table><tr><td>UPPER</td></tr><tr><td>LOWER</td></tr></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var upper, lower *PlacedBox
	for _, block := range Flow(root, testOptions())[0].Blocks {
		switch block.Text {
		case "UPPER":
			upper = &block
		case "LOWER":
			lower = &block
		}
	}
	if upper == nil || lower == nil {
		t.Fatalf("row cells not found: upper=%v lower=%v", upper != nil, lower != nil)
	}
	for name, side := range map[string]boxes.BorderSide{"upper bottom": upper.BorderBottom, "lower top": lower.BorderTop} {
		if side.Width != 2 || side.Color[0] != 1 || side.Color[2] != 0 {
			t.Errorf("%s edge=%+v, want the shared 2pt red border winner", name, side)
		}
	}
}

func TestLayout_CollapsedTableSharedEdgeUsesBorderStylePrecedence(t *testing.T) {
	root, err := boxes.Build(`<style>body{margin:0}table{width:100pt;border-collapse:collapse}td{padding:0;font:10pt/12pt Helvetica}tr:first-child td{border-bottom:2pt dashed blue}tr:last-child td{border-top:2pt solid red}</style><table><tr><td>UPPER</td></tr><tr><td>LOWER</td></tr></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var upper, lower *PlacedBox
	for _, block := range Flow(root, testOptions())[0].Blocks {
		switch block.Text {
		case "UPPER":
			upper = &block
		case "LOWER":
			lower = &block
		}
	}
	if upper == nil || lower == nil {
		t.Fatalf("table cells missing: upper=%v lower=%v", upper != nil, lower != nil)
	}
	if upper.BorderBottom.Color != ([4]float64{1, 0, 0, 1}) || lower.BorderTop.Color != ([4]float64{1, 0, 0, 1}) {
		t.Errorf("shared edge did not use solid over dashed: upper=%+v lower=%+v", upper.BorderBottom, lower.BorderTop)
	}
}

func TestLayout_CollapsedTableHiddenBorderSuppressesSharedEdge(t *testing.T) {
	root, err := boxes.Build(`<style>body{margin:0}table{width:100pt;border-collapse:collapse}td{padding:0;font:10pt/12pt Helvetica}tr:first-child td{border-bottom:1pt solid blue}tr:last-child td{border-top:5pt hidden red}</style><table><tr><td>UPPER</td></tr><tr><td>LOWER</td></tr></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var upper, lower *PlacedBox
	for _, block := range Flow(root, testOptions())[0].Blocks {
		switch block.Text {
		case "UPPER":
			upper = &block
		case "LOWER":
			lower = &block
		}
	}
	if upper == nil || lower == nil {
		t.Fatalf("table cells missing: upper=%v lower=%v", upper != nil, lower != nil)
	}
	if upper.BorderBottom.Width != 0 || lower.BorderTop.Width != 0 {
		t.Errorf("hidden border must suppress the shared edge: upper=%+v lower=%+v", upper.BorderBottom, lower.BorderTop)
	}
}

func TestLayout_CollapsedTableUsesColumnAndColumnGroupBorders(t *testing.T) {
	root, err := boxes.Build(`<style>body{margin:0}table{width:100pt;border-collapse:collapse}td{padding:0;font:10pt/12pt Helvetica}</style><table><colgroup style="border-right:3pt solid red"><col style="border-right:2pt solid green"><col></colgroup><tr><td>LEFT</td><td>RIGHT</td></tr></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var left, right *PlacedBox
	for _, block := range Flow(root, testOptions())[0].Blocks {
		switch block.Text {
		case "LEFT":
			left = &block
		case "RIGHT":
			right = &block
		}
	}
	if left == nil || right == nil {
		t.Fatalf("table cells missing: left=%v right=%v", left != nil, right != nil)
	}
	if left.BorderRight.Color != ([4]float64{0, 0.5, 0, 1}) || right.BorderLeft.Color != left.BorderRight.Color {
		t.Errorf("column border missing at shared edge: left=%+v right=%+v", left.BorderRight, right.BorderLeft)
	}
	if right.BorderRight.Width != 3 || right.BorderRight.Color != ([4]float64{1, 0, 0, 1}) {
		t.Errorf("column-group outer border=%+v, want 3pt red", right.BorderRight)
	}
}

func TestLayout_TablePercentageColumnsDistributeUnusedWidthByDeclaredShares(t *testing.T) {
	root, err := boxes.Build(`<style>body{margin:0}table{width:200pt;margin:0;border-collapse:collapse}td:first-child{width:58%}td:last-child{width:37%}</style><table><tr><td>LEFT</td><td>RIGHT</td></tr></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 300, 100, 0
	var left, right *PlacedBox
	for _, block := range Flow(root, opts)[0].Blocks {
		switch block.Text {
		case "LEFT":
			left = &block
		case "RIGHT":
			right = &block
		}
	}
	if left == nil || right == nil {
		t.Fatalf("cells not found: left=%v right=%v", left != nil, right != nil)
	}
	wantLeft := 200.0 * 58 / 95
	if math.Abs(left.Width-wantLeft) > 0.01 {
		t.Errorf("left column width=%gpt, want %gpt after percentage columns receive leftover width", left.Width, wantLeft)
	}
	if math.Abs(right.Width-(200-wantLeft)) > 0.01 {
		t.Errorf("right column width=%gpt, want %gpt after percentage columns receive leftover width", right.Width, 200-wantLeft)
	}
}

func TestLayout_TablePercentageColspanDistributesByTrackIntrinsicWidths(t *testing.T) {
	root, err := boxes.Build(`<style>body{margin:0}table{width:200pt;margin:0;border-collapse:collapse}td{padding:0;font:10pt/12pt Helvetica;white-space:nowrap}</style><table><tr><td colspan="2" style="width:70%">SPAN</td><td>TAIL</td></tr><tr><td>WIDEVALUEWIDE</td><td>X</td><td>Z</td></tr></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 300, 100, 0
	head, body := tableRows(root.Children[0])
	rows := append(head, body...)
	grid, cols := tableGridFromRows(rows)
	tracks := tableIntrinsicColumns(grid, cols, opts)
	spanMax := tracks[0].max + tracks[1].max
	wants := map[string]float64{
		"WIDEVALUEWIDE": 140 * tracks[0].max / spanMax,
		"X":             140 * tracks[1].max / spanMax,
		"Z":             60,
	}
	for _, block := range Flow(root, opts)[0].Blocks {
		if want, ok := wants[block.Text]; ok {
			if math.Abs(block.Width-want) > 0.01 {
				t.Errorf("column for %q width=%gpt, want %.2fpt based on max-content distribution of a spanning percentage", block.Text, block.Width, want)
			}
			delete(wants, block.Text)
		}
	}
	if len(wants) > 0 {
		t.Errorf("cells missing from layout: %v", wants)
	}
}

func TestLayout_TablePercentageColumnsOver100ReduceFromLastColumn(t *testing.T) {
	root, err := boxes.Build(`<style>body{margin:0}table{width:200pt;margin:0;border-collapse:collapse}td:first-child{width:80%}td:last-child{width:80%}</style><table><tr><td>L</td><td>R</td></tr></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 300, 100, 0
	var left, right *PlacedBox
	for _, block := range Flow(root, opts)[0].Blocks {
		switch block.Text {
		case "L":
			left = &block
		case "R":
			right = &block
		}
	}
	if left == nil || right == nil {
		t.Fatalf("cells not found: left=%v right=%v", left != nil, right != nil)
	}
	if math.Abs(left.Width-160) > 0.01 || math.Abs(right.Width-40) > 0.01 {
		t.Errorf("overallocated percentage widths=%g/%gpt, want 160/40pt (trim excess from last column)", left.Width, right.Width)
	}
}

func TestLayout_TablePercentageWidthFromLaterRowConstrainsItsColumn(t *testing.T) {
	root, err := boxes.Build(`<style>body{margin:0}table{width:200pt;margin:0;border-collapse:collapse}td{padding:0;font:10pt/12pt Helvetica;white-space:nowrap}</style><table><tr><td>A</td><td>B</td></tr><tr><td style="width:70%">C</td><td>D</td></tr></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 300, 100, 0
	var firstColumn, secondColumn *PlacedBox
	for _, block := range Flow(root, opts)[0].Blocks {
		switch block.Text {
		case "A":
			firstColumn = &block
		case "B":
			secondColumn = &block
		}
	}
	if firstColumn == nil || secondColumn == nil {
		t.Fatalf("first-row cells not found: first=%v second=%v", firstColumn != nil, secondColumn != nil)
	}
	if math.Abs(firstColumn.Width-140) > 0.01 || math.Abs(secondColumn.Width-60) > 0.01 {
		t.Errorf("first-row widths=%g/%gpt, want 140/60pt from the later row's 70%% column constraint", firstColumn.Width, secondColumn.Width)
	}
}

func TestLayout_CollapsedTableOuterBottomBorderReservesHalfWidth(t *testing.T) {
	root, err := boxes.Build(`<style>table{width:100pt;border-collapse:collapse;border-bottom:1pt solid red}td{padding:0;font:10pt/12pt Helvetica}</style><table><tr><td>FIRST</td></tr><tr><td>LAST</td></tr></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var first, last *PlacedBox
	for _, block := range Flow(root, testOptions())[0].Blocks {
		switch block.Text {
		case "FIRST":
			first = &block
		case "LAST":
			last = &block
		}
	}
	if first == nil || last == nil {
		t.Fatalf("table rows not found: first=%v last=%v", first != nil, last != nil)
	}
	if math.Abs(first.Height-12) > 0.01 {
		t.Errorf("first row height=%gpt, want 12pt without the table's outer bottom border", first.Height)
	}
	if math.Abs(last.Height-12.5) > 0.01 {
		t.Errorf("last row height=%gpt, want 12.5pt including half of the 1pt collapsed outer border", last.Height)
	}
	if first.BorderBottom.Width != 0 {
		t.Errorf("intermediate row bottom border=%gpt, want no table outer border", first.BorderBottom.Width)
	}
	if last.BorderBottom.Width != 1 {
		t.Errorf("last row bottom border=%gpt, want the table's 1pt outer border", last.BorderBottom.Width)
	}
}
