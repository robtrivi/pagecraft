package layout

import (
	"math"
	"testing"

	"github.com/robtrivi/pagecraft/boxes"
)

// Floats follow the CSS model: they do not advance the cursor; short text
// flows beside them, longer text below (see wrap_test.go). Floats stack vertically.
func TestLayout_FloatLeft_PlacesAside(t *testing.T) {
	root, err := boxes.Build(`<div style="float: left">L</div><p style="margin: 0">Text</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("blocks: got %d, want 2", len(blocks))
	}
	// "L" = 6 pt; "Text" (30 pt) fits beside it: X=78, same y.
	if blocks[0].X != 72 || !approx(blocks[0].Width, 6) {
		t.Errorf("float: got X=%v W=%v, want X=72 W=6", blocks[0].X, blocks[0].Width)
	}
	if blocks[1].X != 72 || blocks[1].Y != 72 || len(blocks[1].Runs) == 0 || blocks[1].Runs[0][0].X != 78 {
		t.Errorf("block box preserves the flow width and its line avoids the float: box=(%v,%v) runs=%+v", blocks[1].X, blocks[1].Y, blocks[1].Runs)
	}
}

func TestLayout_FloatRight_PlacesAside(t *testing.T) {
	root, err := boxes.Build(`<div style="float: right">R</div><p style="margin: 0">Text</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("blocks: got %d, want 2", len(blocks))
	}
	// 595 - 72 - 6 = 517; text stays to the left at the same vertical position.
	if blocks[0].X != 517 {
		t.Errorf("float right X: got %v, want 517", blocks[0].X)
	}
	if blocks[1].X != 72 || blocks[1].Y != 72 {
		t.Errorf("al lado: got (%v,%v), want (72,72)", blocks[1].X, blocks[1].Y)
	}
}

func TestLayout_InlineRightFloatAlignsAtContainingBlockEdge(t *testing.T) {
	root, err := boxes.Build(`<div style="margin:0">TITLE <span style="float:right">NOTICE</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var noticeX float64
	var titleX float64
	for _, block := range Flow(root, testOptions())[0].Blocks {
		for _, line := range block.Runs {
			for _, run := range line {
				switch run.Text {
				case "NOTICE":
					noticeX = run.X
				case "TITLE":
					titleX = run.X
				}
			}
		}
	}
	if noticeX < 400 {
		t.Errorf("inline float starts at x=%g, want it aligned to the containing block's right edge", noticeX)
	}
	if titleX < 72 || titleX >= noticeX {
		t.Errorf("title x=%g should remain before the right float at x=%g", titleX, noticeX)
	}
}

func TestLayout_FloatInsideTransparentBlockKeepsItsCoordinateAndExclusion(t *testing.T) {
	root, err := boxes.Build(`<section><h2 style="margin:0">Scope and method</h2><aside style="float:right;width:120pt;padding:8pt;border:1pt solid #111;background:#fff7ed"><b>REVIEW WINDOW</b><br>03–12 September 2026<br><br><b>LEAD REVIEWER</b><br>Elena Ruiz</aside><p style="margin:0">Reviewers visited North Harbor, Mill Creek, East Yard and Ridgeway. Each visit included a walk-through, a sample of work orders, interviews with supervisors and a check of training records. Evidence was assessed against the internal safety handbook and maintenance schedule.</p></section>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	root.Style.BodyMargin = 0
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 612, 792, 34
	calloutTop, calloutBottom := math.Inf(1), math.Inf(-1)
	calloutX := math.Inf(1)
	var paragraphLines []TextRun
	calloutWords := map[string]bool{"REVIEW": true, "WINDOW": true, "03–12": true, "September": true, "2026": true, "LEAD": true, "REVIEWER": true, "Elena": true, "Ruiz": true}
	for _, page := range Flow(root, opts) {
		for _, block := range page.Blocks {
			for _, line := range block.Runs {
				for _, run := range line {
					if calloutWords[run.Text] {
						calloutX = math.Min(calloutX, run.X)
						calloutTop = math.Min(calloutTop, run.Y)
						calloutBottom = math.Max(calloutBottom, run.Y+run.LineH)
						continue
					}
					if run.Text != "" {
						paragraphLines = append(paragraphLines, run)
					}
				}
			}
		}
	}
	if calloutX < 400 || math.IsInf(calloutX, 1) {
		t.Fatalf("right float ends in x=%g, want its text to the right as in Chromium (x≈440)", calloutX)
	}
	for _, run := range paragraphLines {
		if run.Y >= calloutTop && run.Y < calloutBottom && run.X+run.Width > calloutX-10 {
			t.Errorf("paragraph line enters the float exclusion: run=%+v floatX=%g", run, calloutX)
		}
	}
}

func TestLayout_FullWidthTableMovesBelowNestedFloat(t *testing.T) {
	root, err := boxes.Build(`<section><aside style="float:right;width:120pt;height:60pt;padding:8pt;border:1pt solid #111;background:#eff6ff">float CARD</aside><table style="width:100%;border-collapse:collapse"><tr><td>ROW START</td><td>ROW END</td></tr></table></section>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	root.Style.BodyMargin = 0
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 612, 792, 34
	floatBottom, tableTop := 0.0, math.Inf(1)
	for _, page := range Flow(root, opts) {
		for _, block := range page.Blocks {
			isFloat := false
			for _, line := range block.Runs {
				for _, run := range line {
					if run.Text == "float" || run.Text == "CARD" {
						isFloat = true
					}
					if run.Text == "ROW" || run.Text == "START" {
						tableTop = math.Min(tableTop, run.Y)
					}
				}
			}
			if isFloat {
				floatBottom = math.Max(floatBottom, block.Y+block.Height)
			}
		}
	}
	if floatBottom == 0 || math.IsInf(tableTop, 1) {
		t.Fatalf("missing float or table blocks: floatBottom=%g tableTop=%g", floatBottom, tableTop)
	}
	if tableTop < floatBottom-1e-7 {
		t.Errorf("table starts at y=%g before the float ends at y=%g; Chromium moves it below", tableTop, floatBottom)
	}
}

func TestLayout_FloatContainerResolvesPercentageWidth(t *testing.T) {
	root, err := boxes.Build(`<div style="float:left;width:50%"><span>x</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	usable := opts.PageWidth - opts.left() - opts.right()
	pages := []Page{{Number: 1}}
	active := flowFloat(root.Children[0], opts.top(), &pages, opts, usable, opts.pageBottom())
	if !approx(active.w, usable/2) {
		t.Fatalf("float width=%v, want half of available %v", active.w, usable/2)
	}
}

func TestLayout_Floats_StackVertically(t *testing.T) {
	root, err := boxes.Build(`<div style="float: left">A</div><div style="float: left">B</div>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("blocks: got %d, want 2", len(blocks))
	}
	if blocks[1].Y <= blocks[0].Y {
		t.Errorf("stacked blocks: got %v after %v", blocks[1].Y, blocks[0].Y)
	}
}

func TestLayout_FloatContainerKeepsSVGChild(t *testing.T) {
	root, err := boxes.Build(`<figure style="float:right; width:190pt; padding:6pt; border:1pt solid #94a3b8; background:#f8fafc"><svg width="100" height="50" viewBox="0 0 100 50"><rect width="100" height="50" fill="#eff6ff"/></svg><figcaption>Chart</figcaption></figure><p style="margin:0">Text that must wrap around the float.</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	var svgBlock, wrapper *PlacedBox
	for i := range blocks {
		if blocks[i].SVG != nil {
			svgBlock = &blocks[i]
		}
		if blocks[i].HasBg && blocks[i].Width > 200 {
			wrapper = &blocks[i]
		}
	}
	if svgBlock == nil {
		t.Fatal("the float lost its descendant SVG")
	}
	if wrapper == nil || wrapper.Width != 204 {
		t.Errorf("box float: got %+v, want width 204pt with background", wrapper)
	}
}

func TestLayout_FloatPreservesHardBreaksInInlineContent(t *testing.T) {
	root, err := boxes.Build(`<div style="float:left;width:74pt;background:#ddd">FIGURE<br>DETAIL</div><p style="margin:0">Text that must wrap around the figure, then continue below with the rest of the content.</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	var figure *PlacedBox
	for i := range blocks {
		if blocks[i].Text != "" && blocks[i].Width == 74 {
			figure = &blocks[i]
			break
		}
	}
	if figure == nil {
		t.Fatalf("float fragment not found: %+v", blocks)
	}
	if len(figure.Runs) != 2 {
		t.Fatalf("the <br> must generate two lines of the float, got %d: %+v", len(figure.Runs), figure.Runs)
	}
	if got := figure.Runs[1][0].Y - figure.Runs[0][0].Y; !approx(got, lineHeightFor(figure.source, figure.FontSize, testOptions())) {
		t.Errorf("spacing between float lines=%g, want one line height", got)
	}
}

func TestLayout_ClearAdvancesPastMatchingFloatExclusions(t *testing.T) {
	root, err := boxes.Build(`<div style="float:left;width:100pt;height:50pt">float</div><p style="clear:left;margin:0">AFTER float</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("blocks=%d, want float and cleared paragraph", len(blocks))
	}
	if !approx(blocks[1].Y, blocks[0].Y+blocks[0].Height) {
		t.Errorf("cleared block y=%g, want after float bottom %g", blocks[1].Y, blocks[0].Y+blocks[0].Height)
	}
}

func TestLayout_FloatInlineChildrenShareDecoratedLineGeometry(t *testing.T) {
	root, err := boxes.Build(`<div style="float:left;width:52pt;padding:4pt;border:1pt solid #0f766e;background:#ccfbf1"><strong>NEXT MEETING</strong><br>June 18</div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	var callout *PlacedBox
	for index := range blocks {
		if blocks[index].HasBg {
			callout = &blocks[index]
			break
		}
	}
	if callout == nil {
		t.Fatalf("not found decorated float box: %+v", blocks)
	}
	if len(callout.Runs) != 3 {
		t.Fatalf("inline content must form three measured lines inside the float, got %d: %+v", len(callout.Runs), callout.Runs)
	}
	minimumHeight := verticalInsets(root.Children[0]) + 3*lineHeightFor(root.Children[0], resolveSize(root.Children[0], testOptions()), testOptions())
	if callout.Height+fragmentEpsilon < minimumHeight {
		t.Errorf("decorated height=%g does not contain the lines and the insets (%g)", callout.Height, minimumHeight)
	}
}
