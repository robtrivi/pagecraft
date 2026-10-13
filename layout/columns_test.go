package layout

import (
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
