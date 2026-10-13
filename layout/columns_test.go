package layout

import (
	"math"
	"strings"
	"testing"

	"github.com/robtrivi/pagecraft/boxes"
)

// Two columns balance four children into groups of two. Column width, gap,
// and splitting a child across columns are outside this test.
func TestLayout_Columns_DistributesChildren(t *testing.T) {
	root, err := boxes.Build(`<div style="column-count: 2"><p style="margin: 0">A</p><p style="margin: 0">B</p><p style="margin: 0">C</p><p style="margin: 0">D</p></div>`)
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
	if blocks[0].X != 72 || blocks[1].X != 72 || blocks[2].X != 297.5 || blocks[3].X != 297.5 {
		t.Errorf("columns: got %v %v %v %v, want 72 72 297.5 297.5",
			blocks[0].X, blocks[1].X, blocks[2].X, blocks[3].X)
	}
	if blocks[0].Y != 72 || blocks[1].Y <= blocks[0].Y || blocks[2].Y != 72 {
		t.Errorf("Ys: got %v %v %v, want 72 <y2 72", blocks[0].Y, blocks[1].Y, blocks[2].Y)
	}
	want := []string{"A", "B", "C", "D"}
	for i, w := range want {
		if blocks[i].Text != w {
			t.Errorf("block %d: got %q, want %q", i, blocks[i].Text, w)
		}
	}
}

func TestLayout_ColumnWidthDeterminesCountAndUsedWidth(t *testing.T) {
	root, err := boxes.Build(`<div style="column-width:200pt;column-gap:18pt"><p style="margin:0">A</p><p style="margin:0">B</p></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("blocks=%d, want 2", len(blocks))
	}
	if blocks[0].X != 72 || blocks[1].X != 306.5 {
		t.Errorf("column positions=%v/%v, want 72/306.5", blocks[0].X, blocks[1].X)
	}
	if blocks[0].Width != 216.5 || blocks[1].Width != 216.5 {
		t.Errorf("column widths=%v/%v, want 216.5/216.5", blocks[0].Width, blocks[1].Width)
	}
}

func TestLayout_ColumnTextBreakBeforePageStartsNewPage(t *testing.T) {
	root, err := boxes.Build(`<div style="column-count:2;column-gap:0"><p style="margin:0">FIRST COLUMN TEXT</p><p style="margin:0;break-before:page">NEXT PAGE TEXT</p></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 220, 160, 10
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages=%d, want 2", len(pages))
	}
	if pageContainsText(pages[0], "NEXT PAGE TEXT") {
		t.Fatal("break-before:page on column text must not place it in another column on the same page")
	}
	if !pageContainsText(pages[1], "NEXT PAGE TEXT") {
		t.Fatal("the column text after break-before:page must start on page 2")
	}
}

func TestLayout_ColumnTextBreakAfterPageStartsFollowingBlockOnNewPage(t *testing.T) {
	root, err := boxes.Build(`<div style="column-count:2;column-gap:0"><p style="margin:0;break-after:page">FIRST PAGE TEXT</p><p style="margin:0">NEXT PAGE TEXT</p></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 220, 160, 10
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages=%d, want 2", len(pages))
	}
	if !pageContainsText(pages[0], "FIRST PAGE TEXT") {
		t.Fatal("the block before break-after:page must remain on page 1")
	}
	if pageContainsText(pages[0], "NEXT PAGE TEXT") || !pageContainsText(pages[1], "NEXT PAGE TEXT") {
		t.Fatal("the following column text must start on page 2")
	}
}

func TestLayout_ColumnTextBreakAfterPageStartsDecoratedBlockOnNewPage(t *testing.T) {
	root, err := boxes.Build(`<div style="column-count:2;column-gap:0"><p style="margin:0;break-after:page">FIRST PAGE TEXT</p><div style="padding:1pt;background:#eee">NEXT PAGE BLOCK</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 220, 160, 10
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages=%d, want 2", len(pages))
	}
	if pageContainsText(pages[0], "NEXT PAGE BLOCK") || !pageContainsText(pages[1], "NEXT PAGE BLOCK") {
		t.Fatal("break-after:page must apply when the following column fragment is a decorated block")
	}
}

func TestLayout_ColumnFillAutoFillsEachColumnSequentially(t *testing.T) {
	root, err := boxes.Build(`<div style="column-count:3;column-gap:0;column-fill:auto"><p style="margin:0">FIRST</p><p style="margin:0">SECOND</p><p style="margin:0">THIRD</p></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 3 {
		t.Fatalf("blocks=%d, want 3", len(blocks))
	}
	for index, block := range blocks {
		if block.X != 72 {
			t.Errorf("block %q starts at x=%g, want sequential flow in first column at x=72", block.Text, block.X)
		}
		if index > 0 && block.Y <= blocks[index-1].Y {
			t.Errorf("block %q y=%g must follow the previous block at y=%g", block.Text, block.Y, blocks[index-1].Y)
		}
	}
}

func TestLayout_ColumnFillAutoKeepsDecoratedBlocksInSourceColumn(t *testing.T) {
	root, err := boxes.Build(`<div style="column-count:3;column-gap:0;column-fill:auto"><div style="padding:1pt">FIRST</div><div style="padding:1pt">SECOND</div><div style="padding:1pt">THIRD</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 3 {
		t.Fatalf("blocks=%d, want 3", len(blocks))
	}
	for _, block := range blocks {
		if block.X != 72 {
			t.Errorf("decorated block %q starts at x=%g, want sequential flow in first column at x=72", block.Text, block.X)
		}
	}
}

func TestLayout_ColumnFillAutoBalancesContentBeforeSpanner(t *testing.T) {
	root, err := boxes.Build(`<div style="column-count:3;column-gap:0;column-fill:auto"><p style="margin:0">BEFORE A</p><p style="margin:0">BEFORE B</p><p style="margin:0">BEFORE C</p><h2 style="column-span:all;margin:0">SPANNER</h2><p style="margin:0">AFTER A</p></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 5 {
		t.Fatalf("blocks=%d, want 5", len(blocks))
	}
	if blocks[0].X == blocks[1].X || blocks[1].X == blocks[2].X {
		t.Errorf("content before the spanner must balance across columns: x=%g/%g/%g", blocks[0].X, blocks[1].X, blocks[2].X)
	}
}

func TestLayout_ColumnsBalanceUnequalLineHeightsAcrossColumns(t *testing.T) {
	root, err := boxes.Build(`<div style="column-count:2;column-gap:0"><p style="margin:0;font:10pt/20pt Helvetica">A</p><p style="margin:0;font:10pt/20pt Helvetica">B</p><p style="margin:0;font:10pt/20pt Helvetica">C</p><p style="margin:0;font:10pt/1pt Helvetica">D</p></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 220, 200, 10
	blocks := Flow(root, opts)[0].Blocks
	if len(blocks) != 4 {
		t.Fatalf("blocks=%d, want 4", len(blocks))
	}
	if blocks[0].X != 10 || blocks[1].X != 10 || blocks[2].X != 110 || blocks[3].X != 110 {
		t.Errorf("column positions=%g/%g/%g/%g, want 10/10/110/110", blocks[0].X, blocks[1].X, blocks[2].X, blocks[3].X)
	}
	if blocks[1].Y != 30 || blocks[2].Y != 10 || blocks[3].Y != 30 {
		t.Errorf("block y positions=%g/%g/%g, want 30/10/30", blocks[1].Y, blocks[2].Y, blocks[3].Y)
	}
}

func TestLayout_ColumnsBalanceIndivisibleBlocksAcrossAllColumns(t *testing.T) {
	root, err := boxes.Build(`<div style="column-count:3;column-gap:0"><div style="margin:0;padding:1pt 0;font:1pt/5pt Helvetica">A</div><div style="margin:0;padding:1pt 0;font:1pt/5pt Helvetica">B</div><div style="margin:0;padding:1pt 0;font:1pt/4pt Helvetica">C</div><div style="margin:0;padding:1pt 0;font:1pt/4pt Helvetica">D</div><div style="margin:0;padding:1pt 0;font:1pt/4pt Helvetica">E</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 220, 200, 10
	blocks := Flow(root, opts)[0].Blocks
	if len(blocks) != 5 {
		t.Fatalf("blocks=%d, want 5", len(blocks))
	}
	if blocks[0].X != 10 || blocks[0].X == blocks[1].X || blocks[1].X != blocks[2].X || blocks[2].X == blocks[3].X || blocks[3].X != blocks[4].X {
		t.Errorf("column positions=%g/%g/%g/%g/%g, want groups 1/2/2", blocks[0].X, blocks[1].X, blocks[2].X, blocks[3].X, blocks[4].X)
	}
}

func TestLayout_ColumnForcedBreakStartsNewContentRunInFirstColumn(t *testing.T) {
	root, err := boxes.Build(`<div style="column-count:2;column-gap:0"><p style="margin:0;font:10pt/20pt Helvetica">A</p><p style="margin:0;font:10pt/20pt Helvetica">B</p><p style="margin:0;font:10pt/20pt Helvetica">C</p><p style="margin:0;font:10pt/20pt Helvetica;break-before:page">D</p><p style="margin:0;font:10pt/20pt Helvetica">E</p></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 220, 200, 10
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages=%d, want 2", len(pages))
	}
	var forced *PlacedBox
	for index := range pages[1].Blocks {
		if pages[1].Blocks[index].Text == "D" {
			forced = &pages[1].Blocks[index]
			break
		}
	}
	if forced == nil {
		t.Fatal("the content after break-before:page was not placed on page 2")
	}
	if forced.X != 10 || forced.FragmentainerIndex != 1 {
		t.Errorf("forced content starts at x=%g in column %d, want first column at x=10", forced.X, forced.FragmentainerIndex)
	}
}

func TestLayout_ColumnBalancingAssignsImplicitBreaksToTallestContentRun(t *testing.T) {
	runs := []columnContentRun{{contentHeight: 90}, {contentHeight: 60}, {contentHeight: 30}}
	target := distributeImplicitColumnBreaks(runs, 4)
	if runs[0].implicitBreaks != 1 || runs[1].implicitBreaks != 0 || runs[2].implicitBreaks != 0 {
		t.Errorf("implicit breaks=%d/%d/%d, want 1/0/0", runs[0].implicitBreaks, runs[1].implicitBreaks, runs[2].implicitBreaks)
	}
	if target != 60 {
		t.Errorf("balanced target=%g, want tallest run column height 60", target)
	}
}

func TestLayout_ColumnBalancingRaisesTargetForUnbreakableUnits(t *testing.T) {
	runs := []columnContentRun{{
		units:          []fragmentUnit{{height: 30}, {height: 10}},
		implicitBreaks: 1,
	}}
	target := raiseTargetForLegalBreaks(runs, 15)
	if target != 30 {
		t.Errorf("balanced target=%g, want tallest unbreakable unit 30", target)
	}
}

func TestLayout_ColumnBalancingRetriesUntilMeasuredShortageIsZero(t *testing.T) {
	attempts := 0
	var committed []float64
	target := retryColumnBalance(40, 100, func(height float64) (float64, func()) {
		attempts++
		commit := func() { committed = append(committed, height) }
		if height < 55 {
			return 15, commit
		}
		return 0, commit
	})
	if target != 55 {
		t.Errorf("balanced target=%g, want 55 from the measured 15pt shortage", target)
	}
	if attempts != 2 {
		t.Errorf("layout attempts=%d, want 2", attempts)
	}
	if len(committed) != 1 || committed[0] != 55 {
		t.Errorf("committed layout attempts=%v, want only the accepted 55pt pass", committed)
	}
}

func TestLayout_ColumnBalancingRetriesActualOverflowBeforeCommittingFragments(t *testing.T) {
	root, err := boxes.Build(`<div style="column-count:2;column-gap:0"><div style="height:20pt;margin:0">A</div><div style="height:20pt;margin:0">B</div><div style="height:20pt;margin:0">C</div><div style="height:20pt;margin:0">D</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	container := root.Children[0]
	kids := nestedContainerItems(container)
	opts := testOptions()
	usable, bottom := opts.PageWidth-opts.left()-opts.right(), opts.pageBottom()
	gap := 0.0
	columnWidth := usable / 2
	layouts := make([]columnItemLayout, len(kids))
	for index, item := range kids {
		contentWidth := math.Max(0, columnWidth-horizontalInsets(item))
		lines := breakInlineStream(buildInlineStream(item, opts), contentWidth, wraps(item))
		layouts[index] = columnItemLayout{lines: lines, height: usedHeight(item, inlineHeight(lines, item, opts), opts)}
	}
	runs := []columnContentRun{{kids: kids, layouts: layouts}}
	pages := []Page{{Number: 1}}
	pages[0].emitBlock(PlacedBox{Text: "PREEXISTING"})
	attempts := 0
	var acceptedEnd float64
	initialTarget := layouts[0].height
	target := retryColumnBalance(initialTarget, initialTarget*4, func(height float64) (float64, func()) {
		attempts++
		staged, existingCounts := stageColumnAttemptPages(pages)
		end, shortage := flowColumnRuns(container, runs, opts.top(), &staged, opts, usable, bottom, 2, gap, columnWidth, height, true)
		return shortage, func() {
			mergeColumnAttemptPages(&pages, staged, existingCounts)
			acceptedEnd = end
		}
	})
	if attempts != 2 || target <= initialTarget {
		t.Fatalf("actual layout retry: attempts=%d target=%g, want one retry above initial %gpt", attempts, target, initialTarget)
	}
	if math.Abs(acceptedEnd-opts.top()-target) > fragmentEpsilon {
		t.Errorf("accepted content height=%g, want measured target %gpt", acceptedEnd-opts.top(), target)
	}
	if got := pages[0].blockCount(); got != 5 {
		t.Fatalf("committed blocks=%d, want the existing block and 4 accepted blocks", got)
	}
	if pages[0].block(0).Text != "PREEXISTING" {
		t.Errorf("staged retry changed the existing page prefix: %q", pages[0].block(0).Text)
	}
	for index := 1; index < pages[0].blockCount(); index++ {
		block := pages[0].block(index)
		if index < 3 && block.X != opts.left() {
			t.Errorf("block %q x=%g, want first column x=%g", block.Text, block.X, opts.left())
		}
		if index >= 3 && block.X == opts.left() {
			t.Errorf("block %q remained in first column after retry", block.Text)
		}
	}
}

func TestLayout_ColumnBalancingKeepsAvoidBreakGroupTogether(t *testing.T) {
	root, err := boxes.Build(`<div style="column-count:2;column-gap:0"><div style="margin:0;padding:1pt 0;font:1pt/23pt Helvetica">A</div><div style="margin:0;padding:1pt 0;font:1pt/18pt Helvetica;break-after:avoid">B</div><div style="margin:0;padding:1pt 0;font:1pt/18pt Helvetica">C</div><div style="margin:0;padding:1pt 0;font:1pt/23pt Helvetica">D</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 220, 200, 10
	blocks := Flow(root, opts)[0].Blocks
	if len(blocks) != 4 {
		t.Fatalf("blocks=%d, want 4", len(blocks))
	}
	if blocks[1].X != blocks[2].X {
		t.Errorf("avoid group split across columns: B x=%g, C x=%g", blocks[1].X, blocks[2].X)
	}
}

func TestLayout_ColumnBalancingKeepsBreakBeforeAvoidGroupTogether(t *testing.T) {
	root, err := boxes.Build(`<div style="column-count:2;column-gap:0"><div style="margin:0;padding:1pt 0;font:1pt/23pt Helvetica">A</div><div style="margin:0;padding:1pt 0;font:1pt/18pt Helvetica">B</div><div style="margin:0;padding:1pt 0;font:1pt/18pt Helvetica;break-before:avoid">C</div><div style="margin:0;padding:1pt 0;font:1pt/23pt Helvetica">D</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 220, 200, 10
	blocks := Flow(root, opts)[0].Blocks
	if len(blocks) != 4 {
		t.Fatalf("blocks=%d, want 4", len(blocks))
	}
	if blocks[1].X != blocks[2].X {
		t.Errorf("avoid group split across columns: B x=%g, C x=%g", blocks[1].X, blocks[2].X)
	}
}

func TestLayout_ColumnsPreserveDirectInlineContentAcrossForcedPageBreak(t *testing.T) {
	root, err := boxes.Build(`<div style="column-count:2;break-after:page">COLUMN CONTENT</div><p>AFTER MARK</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	pages := Flow(root, testOptions())
	if len(pages) != 2 {
		t.Fatalf("pages=%d, want 2", len(pages))
	}
	if !pageContainsText(pages[0], "COLUMN CONTENT") {
		t.Fatal("the direct inline content of multicolumn layout disappeared")
	}
	if !pageContainsText(pages[1], "AFTER MARK") {
		t.Fatal("break-after:page must move the following content")
	}
}

func TestLayout_ColumnFragmentsIdentifyTheirFragmentainer(t *testing.T) {
	root, err := boxes.Build(`<div style="column-count:2"><p style="margin:0">FIRST COLUMN</p><p style="margin:0">SECOND COLUMN</p></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("blocks=%d, want 2", len(blocks))
	}
	if blocks[0].FragmentainerIndex != 1 || blocks[1].FragmentainerIndex != 2 {
		t.Errorf("fragmentainers=%d/%d, want columns 1/2", blocks[0].FragmentainerIndex, blocks[1].FragmentainerIndex)
	}
}

func TestLayout_ColumnsForcedPageBreakEscapesCurrentPage(t *testing.T) {
	root, err := boxes.Build(`<div style="column-count:2"><div style="padding:1pt">FIRST</div><div style="padding:1pt;break-before:page">SECOND</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 220, 120, 0
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages=%d, want 2", len(pages))
	}
	if len(pages[0].Blocks) == 0 || pages[0].Blocks[0].Text != "FIRST" {
		t.Errorf("first page=%+v, want FIRST", pages[0].Blocks)
	}
	if len(pages[1].Blocks) == 0 || pages[1].Blocks[0].Text != "SECOND" {
		t.Errorf("second page=%+v, want SECOND", pages[1].Blocks)
	}
}

func TestLayout_Columns_SpannerSeparatesSegments(t *testing.T) {
	root, err := boxes.Build(`<div style="column-count:3; column-gap:18pt"><p style="margin:0">before A</p><p style="margin:0">before B</p><p style="margin:0">before C</p><h2 style="column-span:all; margin:8pt 0; padding:8pt; background:#dbeafe">Summary a todo it width</h2><p style="margin:0">After A</p><p style="margin:0">After B</p><p style="margin:0">After C</p></div>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 7 {
		t.Fatalf("blocks: got %d, want 7", len(blocks))
	}
	span := blocks[3]
	if span.Text != "Summary a todo it width" || span.X != 72 || span.Width != 451 {
		t.Errorf("spanner: text=%q X=%v width=%v, want full container 72/451", span.Text, span.X, span.Width)
	}
	if span.Y <= blocks[0].Y || blocks[4].Y < span.Y+span.Height {
		t.Errorf("spanner must separate both segments: beforeY=%v span=(%v,%v) afterY=%v", blocks[0].Y, span.Y, span.Height, blocks[4].Y)
	}
	if blocks[0].X == blocks[1].X || blocks[1].X == blocks[2].X {
		t.Errorf("the first segment was not distributed across columns: X=%v,%v,%v", blocks[0].X, blocks[1].X, blocks[2].X)
	}
}

func TestLayout_ColumnSpanBreakAfterPageStartsFollowingSegmentOnNewPage(t *testing.T) {
	root, err := boxes.Build(`<div style="column-count:2;column-gap:0"><p style="margin:0">BEFORE SPANNER</p><h2 style="column-span:all;margin:0;break-after:page">SPANNER</h2><p style="margin:0">AFTER SPANNER</p></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 220, 120, 0
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages=%d, want 2", len(pages))
	}
	if !pageContainsText(pages[0], "BEFORE SPANNER") || !pageContainsText(pages[0], "SPANNER") {
		t.Fatal("content before the forced break must remain on page 1")
	}
	if pageContainsText(pages[0], "AFTER SPANNER") || !pageContainsText(pages[1], "AFTER SPANNER") {
		t.Fatal("the segment after a spanner with break-after:page must start on page 2")
	}
}

func TestLayout_Columns_DrawsRulesInGaps(t *testing.T) {
	root, err := boxes.Build(`<div style="column-count:3;column-gap:18pt;column-rule:1pt solid #cbd5e1"><p style="margin:0">A</p><p style="margin:0">B</p><p style="margin:0">C</p></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	var rules []PlacedBox
	for _, block := range blocks {
		if block.BorderLeft.Width == 1 && block.BorderLeft.Color == [4]float64{203.0 / 255, 213.0 / 255, 225.0 / 255, 1} {
			rules = append(rules, block)
		}
	}
	if len(rules) != 2 {
		t.Fatalf("column rules: got %d, want 2", len(rules))
	}
	if !approx(rules[0].X, 218.83333333333334) || !approx(rules[1].X, 375.1666666666667) {
		t.Errorf("X of rules: %v, %v", rules[0].X, rules[1].X)
	}
	if rules[0].Y != 72 || !approx(rules[0].Height, 14.4) || rules[1].Y != 72 || !approx(rules[1].Height, 14.4) {
		t.Errorf("height/rule: %+v / %+v", rules[0], rules[1])
	}
}

func TestLayout_Columns_DrawsRulesForEachPageFragment(t *testing.T) {
	root, err := boxes.Build(`<div style="column-count:2;column-gap:18pt;column-rule:1pt solid #cbd5e1"><p style="margin:0;padding:1pt 0;font:1pt/20pt Helvetica">A</p><p style="margin:0;padding:1pt 0;font:1pt/20pt Helvetica">B</p><p style="margin:0;padding:1pt 0;font:1pt/20pt Helvetica">C</p><p style="margin:0;padding:1pt 0;font:1pt/20pt Helvetica">D</p><p style="margin:0;padding:1pt 0;font:1pt/20pt Helvetica">E</p><p style="margin:0;padding:1pt 0;font:1pt/20pt Helvetica">F</p></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 220, 60, 0
	pages := Flow(root, opts)
	if len(pages) < 2 {
		t.Fatalf("pages=%d, want column content to continue on another page", len(pages))
	}
	for pageIndex, page := range pages {
		rules := 0
		for _, block := range page.Blocks {
			if block.BorderLeft.Width == 1 && block.BorderLeft.Color == [4]float64{203.0 / 255, 213.0 / 255, 225.0 / 255, 1} {
				rules++
				if block.Height > 60 {
					t.Errorf("page %d rule height=%g exceeds its 60pt page", pageIndex+1, block.Height)
				}
			}
		}
		if rules != 1 {
			t.Errorf("page %d rules=%d, want 1", pageIndex+1, rules)
		}
	}
}

func TestLayout_ColumnsBalancesItemsByHeight(t *testing.T) {
	root, err := boxes.Build(`<div style="column-count:2;column-gap:0"><p style="margin:0;font-size:30pt">Tall</p><p style="margin:0;font-size:10pt">Short A</p><p style="margin:0;font-size:10pt">Short B</p></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 3 {
		t.Fatalf("blocks=%d, want 3", len(blocks))
	}
	if blocks[0].X == blocks[1].X || blocks[1].X != blocks[2].X {
		t.Errorf("items by column: Tall x=%v, Short A x=%v, Short B x=%v; want one tall block and two short blocks in another column", blocks[0].X, blocks[1].X, blocks[2].X)
	}
}

func TestLayout_ColumnsContinuesLongTextInNextColumn(t *testing.T) {
	text := "one two three four five six seven eight nine ten eleven twelve thirteen fourteen fifteen sixteen seventeen eighteen nineteen twenty twenty-one twenty-two twenty-three twenty-four twenty-five twenty-six twenty-seven twenty-eight twenty-nine thirty"
	root, err := boxes.Build(`<div style="column-count:2;column-gap:0"><p style="margin:0">` + text + `</p></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("fragments of text=%d, want 2 columns", len(blocks))
	}
	if blocks[0].X == blocks[1].X || len(blocks[0].Runs) == 0 || len(blocks[1].Runs) == 0 {
		t.Errorf("text must continue in another column: %+v", blocks)
	}
	if blocks[0].Text == "" || blocks[1].Text == "" || blocks[0].Text[len(blocks[0].Text)-1:] == " " {
		t.Errorf("empty or malformed column fragments: %q / %q", blocks[0].Text, blocks[1].Text)
	}
	joined := strings.Join(strings.Fields(blocks[0].Text+" "+blocks[1].Text), " ")
	if joined != text {
		t.Errorf("text order must be preserved across columns: got %q, want %q", joined, text)
	}
}

func TestLayout_ColumnsContinuesTextBeforeFollowingBlock(t *testing.T) {
	text := "one two three four five six seven eight nine ten eleven twelve thirteen fourteen fifteen sixteen seventeen eighteen nineteen twenty twenty-one twenty-two twenty-three twenty-four twenty-five twenty-six twenty-seven twenty-eight twenty-nine thirty"
	root, err := boxes.Build(`<div style="column-count:2;column-gap:0"><p style="margin:0">` + text + `</p><p style="margin:0">end</p></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	var fragmentXs []float64
	last := -1
	for i, block := range blocks {
		if strings.Contains(block.Text, "one") || strings.Contains(block.Text, "thirty") {
			fragmentXs = append(fragmentXs, block.X)
		}
		if block.Text == "end" {
			last = i
		}
	}
	if len(fragmentXs) != 2 || fragmentXs[0] == fragmentXs[1] {
		t.Fatalf("the long paragraph must fragment across columns before the following block: X=%v, blocks=%+v", fragmentXs, blocks)
	}
	if last < 0 || blocks[last].X != fragmentXs[1] {
		t.Errorf("the following block must come after the text in the second column: blocks=%+v", blocks)
	}
}

func TestLayout_ColumnsFloatExcludesFollowingTextInSameFragmentainer(t *testing.T) {
	root, err := boxes.Build(`<div style="column-count:2;column-gap:24pt">
		<p style="margin:0">PREVIOUS</p>
		<div style="float:left;width:74pt;height:52pt;margin:2pt 8pt 6pt 0;background:#ddd">FIGURE<br>DETAIL</div>
		<p style="margin:0">Text that must flow beside the figure before continuing below, then follow in order through the second column with enough words to occupy several lines and continue in the next fragmentainer.</p>
		<p style="margin:0">DEFERRED SECTION must start inside its assigned second column after the preceding paragraph.</p>
	</div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	var figure *PlacedBox
	var following []*PlacedBox
	var followingStarted bool
	var deferred *PlacedBox
	for i := range blocks {
		if strings.Contains(blocks[i].Text, "FIGURE") {
			figure = &blocks[i]
		}
		if strings.Contains(blocks[i].Text, "Text that must") {
			following = append(following, &blocks[i])
			followingStarted = true
		} else if followingStarted && len(following) > 0 && blocks[i].Text != "" && !strings.Contains(blocks[i].Text, "FIGURE") && !strings.Contains(blocks[i].Text, "DEFERRED SECTION") {
			following = append(following, &blocks[i])
		}
		if strings.Contains(blocks[i].Text, "DEFERRED SECTION") {
			deferred = &blocks[i]
		}
	}
	if figure == nil || len(following) == 0 {
		t.Fatalf("figure or text was not found")
	}
	firstColumnRight := 72.0 + (451.0-24)/2
	if figure.X != 72 || figure.Width < 74 || figure.Width >= firstColumnRight-72 {
		t.Errorf("the float must preserve its constrained box and start in the first column: x=%g width=%g", figure.X, figure.Width)
	}
	var text strings.Builder
	besideFloat, firstLineBesideFloat, firstLineChecked, nextColumn := false, false, false, false
	for _, block := range following {
		text.WriteString(block.Text)
		text.WriteByte(' ')
		for _, line := range block.Runs {
			if len(line) == 0 {
				continue
			}
			y := line[0].Y
			if y < figure.Y+figure.Height && y+line[0].LineH > figure.Y && line[0].X >= figure.X+figure.Width && line[0].X < firstColumnRight {
				besideFloat = true
			}
			if strings.HasPrefix(block.Text, "Text that must") && !firstLineChecked {
				firstLineChecked = true
				firstLineBesideFloat = y < figure.Y+figure.Height && y+line[0].LineH > figure.Y && line[0].X >= figure.X+figure.Width
			}
			if line[0].X >= firstColumnRight+24 {
				nextColumn = true
			}
		}
	}
	if !besideFloat {
		t.Errorf("not following text line occupied the space beside the float: float=(x=%g y=%g w=%g h=%g)", figure.X, figure.Y, figure.Width, figure.Height)
	}
	if !firstLineBesideFloat {
		t.Errorf("the first line that intersects the float must use the reduced width")
	}
	if !nextColumn {
		t.Errorf("remaining text did not continue in the second column")
	}
	if deferred == nil || deferred.X < firstColumnRight || len(deferred.Runs) == 0 || len(deferred.Runs[0]) == 0 || deferred.Runs[0][0].X < firstColumnRight {
		t.Errorf("the deferred block must use the coordinates of the second column: %+v", deferred)
	}
	wantText := "Text that must flow beside the figure before continuing below, then follow in order through the second column with enough words to occupy several lines and continue in the next fragmentainer."
	if got := strings.Join(strings.Fields(text.String()), " "); got != wantText {
		t.Errorf("fragmented text: got %q, want %q", got, wantText)
	}
}

func TestLayout_ColumnsFloatRetainsDeclaredSizeWithoutSplittableText(t *testing.T) {
	root, err := boxes.Build(`<div style="column-count:2"><div style="float:left;width:74pt;height:52pt">FIGURE</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("blocks=%d, want 1", len(blocks))
	}
	if blocks[0].Width != 74 || blocks[0].Height != 52 || blocks[0].FragmentainerIndex != 1 {
		t.Errorf("float=(%gx%g) in fragmentainer %d; want 74x52 in 1", blocks[0].Width, blocks[0].Height, blocks[0].FragmentainerIndex)
	}
}

func TestLayout_ColumnsCollapseMarginsAfterFragmentedParagraph(t *testing.T) {
	text := "one two three four five six seven eight nine ten eleven twelve thirteen fourteen fifteen sixteen seventeen eighteen nineteen twenty twenty-one twenty-two twenty-three twenty-four twenty-five twenty-six twenty-seven twenty-eight twenty-nine thirty thirty and one thirty and two thirty and three thirty and four thirty and five thirty and six thirty and seven thirty and eight thirty and nine forty"
	root, err := boxes.Build(`<div style="column-count:2;column-gap:0"><p style="margin:0 0 8pt">` + text + `</p><div style="margin:8pt 0;padding:4pt;background:#eee">BOX</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	lastTextBottom := -1.0
	var following *PlacedBox
	for i := range blocks {
		block := &blocks[i]
		if block.Text == "BOX" {
			following = block
		}
		if strings.Contains(block.Text, "forty") && len(block.Runs) > 0 && len(block.Runs[len(block.Runs)-1]) > 0 {
			line := block.Runs[len(block.Runs)-1]
			lastTextBottom = line[0].Y + line[0].LineH
		}
	}
	if lastTextBottom < 0 || following == nil {
		t.Fatalf("end fragment and following block were not found: %#v", blocks)
	}
	if got := following.Y - lastTextBottom; !approx(got, 8) {
		t.Errorf("spacing after collapsing margins=%gpt, want 8pt", got)
	}
}
