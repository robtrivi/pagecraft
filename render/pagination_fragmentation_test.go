package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/robtrivi/pagecraft-go/boxes"
	"github.com/robtrivi/pagecraft-go/layout"
	"github.com/robtrivi/pagecraft-go/text"
)

func TestRender_BreakAfterPageWorksAcrossDocumentStructures(t *testing.T) {
	cases := []struct {
		name, html string
	}{
		{name: "block", html: `<p style="break-after:page">BLOCK CONTENT</p>`},
		{name: "decorated container", html: `<section style="break-after:page;padding:4pt;border:1pt solid #000">WRAPPER CONTENT</section>`},
		{name: "table", html: `<table style="break-after:page"><tr><td>TABLE CONTENT</td></tr></table>`},
		{name: "flex", html: `<div style="display:flex;break-after:page"><div>FLEX CONTENT</div></div>`},
		{name: "grid", html: `<div style="display:grid;break-after:page"><div>GRID CONTENT</div></div>`},
		{name: "columns", html: `<div style="column-count:2;break-after:page">COLUMN CONTENT</div>`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pdf, err := RenderHTMLToPDF(tc.html + `<p>AFTER MARK</p>`)
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			words := assertPDFHasWordCoordinates(t, pdf)
			if !hasPDFWord(words, "AFTER", "MARK") {
				t.Error("missing the coordinates of the following marker when scrolling down")
			}
			if !bytes.HasPrefix(pdf, []byte("%PDF-")) || !bytes.Contains(pdf, []byte("%%EOF")) {
				t.Fatal("output is not a complete PDF")
			}
		})
	}
}

func TestRender_BreakAfterAvoidPreservesLongFollowingParagraph(t *testing.T) {
	html := `<!doctype html><style>@page{size:260pt 260pt;margin:20pt}body{margin:0;font:10pt/12pt Helvetica}.space{height:177pt}.box{border:1pt solid #0369a1;padding:3pt;background:#e0f2fe}p{margin:0}</style>` +
		`<div class="space"></div><p class="box" style="break-after:avoid">KEEP TITLE</p><p class="box">` +
		strings.Repeat("following words ", 90) + `</p><p>END MARK</p>`
	pdf, err := RenderHTMLToPDF(html)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) || !bytes.Contains(pdf, []byte("%%EOF")) {
		t.Fatal("output is not a complete PDF")
	}
	assertPDFHasWordCoordinates(t, pdf)
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := layout.DefaultOptions()
	opts.Measure, opts.MeasureBold = text.Measure, text.MeasureBold
	opts.Faces, opts.FontVariants = root.Fonts, root.FontVariants
	if root.Page.Width > 0 && root.Page.Height > 0 {
		opts.PageWidth, opts.PageHeight = root.Page.Width, root.Page.Height
	}
	if root.Page.HasMargin {
		opts.Margin = root.Page.Margin
	}
	pages := layout.Flow(root, opts)
	followingCount := 0
	titlePage := 0
	for pageIndex, page := range pages {
		for _, block := range page.Blocks {
			if strings.Contains(block.Text, "KEEP TITLE") {
				titlePage = pageIndex + 1
			}
			for _, line := range block.Runs {
				for _, run := range line {
					followingCount += strings.Count(run.Text, "following")
				}
			}
		}
	}
	if titlePage != 2 {
		t.Errorf("title is on page %d; Chromium/WeasyPrint place it on page 2", titlePage)
	}
	if followingCount != 90 {
		t.Errorf("text extractable preserved=%d words, want 90", followingCount)
	}
}

func TestRender_ColumnBlockRunsStayInAssignedFragmentainer(t *testing.T) {
	html := `<!doctype html><style>
@page { size: Letter; margin: 42pt; }
body { font: 9.5pt/1.35 Helvetica; }
.columns { column-count: 2; column-gap: 24pt; }
.columns p { margin: 0 0 9pt; }
.float { float: left; width: 74pt; height: 52pt; margin: 2pt 8pt 6pt 0; padding: 5pt; border: 1pt solid #0f766e; }
.span { column-span: all; margin: 10pt 0; padding: 6pt; }
.keep { break-inside: avoid; border: 1pt solid #cbd5e1; padding: 5pt; }
</style><div class="columns">
<p>Text begins in two columns, with a float that takes space only beside the first lines.</p>
<div class="float">Figure A<br>74 × 52 pt</div>
<p>This paragraph wraps around the float and continues below it. Words must stay in reading order even as the available width changes within the block. The next sentence adds enough text to flow through the remaining space in this column and continue at the top of the next one. Each line must respect the float while it remains beside the text, then return to the full column width below the figure. The final sentence makes the paragraph long enough to continue through the second column before the following section begins.</p>
<p>The next section adds several sentences to balance the columns visually. Boxes must not cross the rule or jump over previous content.</p>
<div class="span">Full-width label: text continues in two columns below.</div>
<p>After the span, columns continue below the heading and retain their spacing.</p>
<div class="keep"><strong>Note unbreakable</strong><br>Two related lines that must stay together during fragmentation.</div>
<p>End content checks that flow returns to its normal sequence.</p></div>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("build boxes: %v", err)
	}
	opts := layout.DefaultOptions()
	opts.Measure, opts.MeasureBold = text.Measure, text.MeasureBold
	opts.Faces, opts.FontVariants = root.Fonts, root.FontVariants
	opts.PageWidth, opts.PageHeight, opts.Margin = root.Page.Width, root.Page.Height, root.Page.Margin
	pages := layout.Flow(root, opts)
	p2SecondColumnBottom := -1.0
	var p3 *layout.PlacedBox
	sharedFloatLineInFirstColumn := false
	for _, page := range pages {
		for _, block := range page.Blocks {
			if strings.Contains(strings.ToLower(block.Text), "must stay") && block.FragmentainerIndex == 1 {
				sharedFloatLineInFirstColumn = true
			}
			if strings.Contains(strings.ToLower(block.Text), "this paragraph") || strings.Contains(strings.ToLower(block.Text), "reading order") || strings.Contains(strings.ToLower(block.Text), "remaining space") {
				for _, line := range block.Runs {
					if len(line) > 0 && line[0].X >= 300 {
						p2SecondColumnBottom = line[0].Y + line[0].LineH
					}
				}
			}
			if !strings.Contains(strings.ToLower(block.Text), "the next section adds") {
				continue
			}
			if block.FragmentainerIndex != 2 || block.X < 300 {
				t.Fatalf("block assigned to column %d in x=%g, want second column", block.FragmentainerIndex, block.X)
			}
			if len(block.Runs) == 0 || len(block.Runs[0]) == 0 {
				t.Fatalf("block has no text runs: x=%g", block.X)
			}
			if block.Runs[0][0].X < 300 {
				t.Fatalf("first run at x=%g escaped its fragmentainer at x=%g", block.Runs[0][0].X, block.X)
			}
			p3 = &block
		}
	}
	if p3 == nil {
		t.Fatal("paragraph in the second column was not found")
	}
	if p2SecondColumnBottom < 0 {
		t.Fatal("the paragraph continuation around the float did not flow into the second column")
	}
	if !sharedFloatLineInFirstColumn {
		t.Error("the line that fits beside the float must remain in the first column, as in Chromium and WeasyPrint")
	}
	if len(p3.Runs) == 0 || len(p3.Runs[0]) == 0 || p3.Runs[0][0].Y < p2SecondColumnBottom {
		t.Errorf("following paragraph must start after the previous continuation: p2 bottom=%g, p3=%+v", p2SecondColumnBottom, p3)
	}
}
