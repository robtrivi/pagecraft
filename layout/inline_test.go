package layout

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/robtrivi/pagecraft/boxes"
)

func TestLayout_InlineBoxDecorationFollowsTextRunGeometry(t *testing.T) {
	root, err := boxes.Build(`<p style="margin:0;font:10pt/14pt Helvetica">before <span style="background:#ffee99;border:1pt solid #123456;padding:2pt 3pt">HIGHLIGHTED LABEL</span> after</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var found bool
	var beforeEnd, highlightedStart, labelEnd, afterStart float64
	for _, page := range Flow(root, testOptions()) {
		for _, block := range page.Blocks {
			for _, line := range block.Runs {
				for _, run := range line {
					if run.Text != "HIGHLIGHTED" {
						if run.Text == "before" {
							beforeEnd = run.X + run.Width
						}
						if run.Text == "LABEL" {
							labelEnd = run.X + run.Width
						}
						if run.Text == "after" {
							afterStart = run.X
						}
						continue
					}
					found = true
					highlightedStart = run.X
					if len(run.InlineDecorations) != 1 {
						t.Fatalf("the first span run has %d inline decorations; want 1", len(run.InlineDecorations))
					}
					decoration := run.InlineDecorations[0]
					box := decoration.Box
					if !box.HasBg || box.Width < run.Width+8 || box.Height < run.LineH+4 {
						t.Errorf("inline fragment does not cover text, padding, and border: run=%+v decoration=%+v", run, decoration)
					}
					if box.BorderLeft.Width != 1 || box.BorderRight.Width != 1 || math.Abs(decoration.OffsetX+4) > 0.01 {
						t.Errorf("span borders or horizontal offset are incorrect: %+v", decoration)
					}
				}
			}
		}
	}
	if !found {
		t.Fatal("was not generated the run HIGHLIGHTED")
	}
	if highlightedStart-beforeEnd < 9.5 || afterStart-labelEnd < 9.5 {
		t.Errorf("inline padding does not participate in horizontal flow: left/right gaps=%.2f/%.2f pt; want space plus 4pt of padding and border on each side", highlightedStart-beforeEnd, afterStart-labelEnd)
	}
}

func TestLayout_CollectInlineDraftsAppendsNestedRunsInPlace(t *testing.T) {
	root, err := boxes.Build(`<p>prefix <span>nested <b>deep</b></span> tail</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	paragraph := root
	for paragraph.Tag != "p" {
		if len(paragraph.Children) == 0 {
			t.Fatalf("paragraph not found in box tree")
		}
		paragraph = paragraph.Children[0]
	}
	prefix := inlineDraft{inlineItem: inlineItem{text: "existing"}}
	got := appendFlattenInlineDraftsInPlace([]inlineDraft{prefix}, paragraph, 0, testOptions(), &inlineScope{box: paragraph})
	if len(got) < 2 || got[0].text != "existing" {
		t.Fatalf("existing draft was not preserved: %#v", got)
	}
	var words []string
	for _, draft := range got[1:] {
		words = append(words, draft.text)
	}
	if want := []string{"prefix", "nested", "deep", "tail"}; !reflect.DeepEqual(words, want) {
		t.Fatalf("nested inline text=%v, want %v", words, want)
	}
}

func TestLayout_InlineBoxDecorationSlicesBordersAcrossWrappedLines(t *testing.T) {
	root, err := boxes.Build(`<p style="width:82pt;margin:0;font:10pt/14pt Helvetica"><span style="background:#ffee99;border:1pt solid #123456;padding:2pt 3pt">ALPHA BRAVO CHARLIE DELTA ECHO FOXTROT</span></p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	var fragments []PlacedBox
	for _, page := range Flow(root, testOptions()) {
		for _, block := range page.Blocks {
			for _, line := range block.Runs {
				for _, run := range line {
					if len(run.InlineDecorations) > 0 {
						fragments = append(fragments, run.InlineDecorations[0].Box)
					}
				}
			}
		}
	}
	if len(fragments) < 2 {
		t.Fatalf("span should produces multiple line fragments, got %d", len(fragments))
	}
	if fragments[0].BorderLeft.Width != 1 || fragments[0].BorderRight.Width != 0 {
		t.Errorf("first fragment must preserve the starting border and continue with the right border: left/right=%g/%g", fragments[0].BorderLeft.Width, fragments[0].BorderRight.Width)
	}
	for index := 1; index < len(fragments)-1; index++ {
		if fragments[index].BorderLeft.Width != 0 || fragments[index].BorderRight.Width != 0 {
			t.Errorf("middle fragment %d has side borders left/right=%g/%g", index, fragments[index].BorderLeft.Width, fragments[index].BorderRight.Width)
		}
	}
	last := fragments[len(fragments)-1]
	if last.BorderLeft.Width != 0 || last.BorderRight.Width != 1 {
		t.Errorf("last fragment must preserve only the ending border: left/right=%g/%g", last.BorderLeft.Width, last.BorderRight.Width)
	}
}

func TestLayout_InlineNoWrapGroupBreaksOnlyAtItsOuterBoundaries(t *testing.T) {
	cases := []struct {
		name, html   string
		keepTogether bool
	}{
		{name: "same-style-runs", html: `<div style="width:100pt;font:10pt Helvetica">AAAA <span style="white-space:nowrap;font-size:8pt">KEEP TOGETHER</span></div>`, keepTogether: true},
		{name: "nested-style-runs", html: `<div style="width:100pt;font:10pt Helvetica">AAAA <span style="white-space:nowrap;font-size:8pt">KEEP <b>TOGETHER</b></span></div>`, keepTogether: true},
		{name: "group-wider-than-line", html: `<div style="width:60pt;font:10pt Helvetica">AAAA <span style="white-space:nowrap;font-size:8pt">KEEP TOGETHER</span></div>`, keepTogether: true},
		{name: "nested-normal-resets-nowrap", html: `<div style="width:60pt;font:10pt Helvetica">AAAA <span style="white-space:nowrap;font-size:8pt">KEEP <b style="white-space:normal">TOGETHER</b></span></div>`, keepTogether: true},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			root, err := boxes.Build(test.html)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			opts := testOptions()
			opts.Measure = func(value string, _ float64) float64 {
				switch value {
				case "AAAA":
					return 40
				case "KEEP":
					return 24
				case "TOGETHER":
					return 48
				case " ":
					return 4
				}
				return float64(len([]rune(value))) * 5
			}
			var keepY, togetherY, prefixY float64
			for _, page := range Flow(root, opts) {
				for _, block := range page.Blocks {
					for _, line := range block.Runs {
						for _, run := range line {
							switch run.Text {
							case "AAAA":
								prefixY = run.LineY
							case "KEEP":
								keepY = run.LineY
							case "TOGETHER":
								togetherY = run.LineY
							}
						}
					}
				}
			}
			if keepY == 0 || togetherY == 0 || prefixY == 0 {
				t.Fatalf("missing coordinates: prefix=%.2f KEEP=%.2f TOGETHER=%.2f", prefixY, keepY, togetherY)
			}
			if got := keepY == togetherY; got != test.keepTogether {
				t.Errorf("KEEP/TOGETHER in same line=%v, want %v (y=%.2f/%.2f)", got, test.keepTogether, keepY, togetherY)
			}
			if keepY == prefixY {
				t.Errorf("nowrap group did not move as a whole: prefix y=%.2f, group y=%.2f", prefixY, keepY)
			}
		})
	}
}

func TestLayout_InlineWrapsBetweenCJKIdeographs(t *testing.T) {
	root, err := boxes.Build(`<div style="width:20pt;margin:0;font:10pt Helvetica">中文排版测试</div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.Measure = func(value string, _ float64) float64 { return float64(len([]rune(value))) * 10 }
	pages := Flow(root, opts)
	if len(pages) != 1 || len(pages[0].Blocks) != 1 {
		t.Fatalf("layout pages/blocks: %d", len(pages))
	}
	block := pages[0].Blocks[0]
	var got strings.Builder
	if len(block.Runs) < 2 {
		t.Fatalf("CJK text should wrap into multiple lines within 20pt, got %d", len(block.Runs))
	}
	for lineIndex, line := range block.Runs {
		lineWidth := 0.0
		for _, run := range line {
			got.WriteString(run.Text)
			lineWidth += run.Width
		}
		if lineWidth > 20+fragmentEpsilon {
			t.Errorf("line %d overflows: width=%.2fpt", lineIndex, lineWidth)
		}
	}
	if got.String() != "中文排版测试" {
		t.Errorf("text alterado when split: %q", got.String())
	}
}

func TestLayout_InlineWrapsBetweenCJKRuns(t *testing.T) {
	root, err := boxes.Build(`<div style="width:30pt;margin:0;font:10pt Helvetica">中<span>文</span>字</div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.Measure = func(value string, _ float64) float64 { return float64(len([]rune(value))) * 10 }
	block := Flow(root, opts)[0].Blocks[0]
	if len(block.Runs) != 1 || len(block.Runs[0]) != 3 {
		t.Fatalf("three ideographs must fit together without synthetic spacing: lines=%d runs=%v", len(block.Runs), block.Runs)
	}
	for index := 1; index < 3; index++ {
		previous, current := block.Runs[0][index-1], block.Runs[0][index]
		if current.X-previous.X-previous.Width != 0 {
			t.Errorf("run CJK %q introdujo gapsynthetic %.2fpt", current.Text, current.X-previous.X-previous.Width)
		}
	}
}

func TestLayout_InlineBreaksCJKAtStyledRunBoundary(t *testing.T) {
	root, err := boxes.Build(`<div style="width:20pt;margin:0;font:10pt Helvetica">中<span>文</span>字</div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.Measure = func(value string, _ float64) float64 { return float64(len([]rune(value))) * 10 }
	block := Flow(root, opts)[0].Blocks[0]
	if len(block.Runs) < 2 {
		t.Fatalf("CJK must be splittable at inline run boundaries, got %d lines", len(block.Runs))
	}
}

func TestLayout_InlineNoWrapSuppressesCJKRunBreaks(t *testing.T) {
	root, err := boxes.Build(`<div style="width:20pt;margin:0;font:10pt Helvetica"><span style="white-space:nowrap">中<span>文</span>字</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.Measure = func(value string, _ float64) float64 { return float64(len([]rune(value))) * 10 }
	block := Flow(root, opts)[0].Blocks[0]
	if len(block.Runs) != 1 {
		t.Fatalf("nowrap must not split ideographs in child runs, got %d lines", len(block.Runs))
	}
}

func TestLayout_InlinePreservesExplicitCJKWhitespace(t *testing.T) {
	root, err := boxes.Build(`<div style="width:100pt;margin:0;font:10pt Helvetica">中　文</div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.Measure = func(value string, _ float64) float64 { return float64(len([]rune(value))) * 10 }
	line := Flow(root, opts)[0].Blocks[0].Runs[0]
	if len(line) != 2 {
		t.Fatalf("runs CJK: got %d, want 2", len(line))
	}
	gap := line[1].X - line[0].X - line[0].Width
	if gap <= 0 {
		t.Errorf("explicit ideographic space was removed: gap=%.2fpt", gap)
	}
}

func TestLayout_InlineStreamSeparatesBreakOpportunityFromAdvance(t *testing.T) {
	root, err := boxes.Build(`<div style="font:10pt Helvetica">中<span>文</span> 甲<br>乙</div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	stream := buildInlineStream(root.Children[0], testOptions())
	if len(stream) != 5 {
		t.Fatalf("inline items=%d, want 5 (four text items and one hard break)", len(stream))
	}
	if got := stream[1].before; got.kind != inlineBreakOpportunity || got.advance != 0 {
		t.Errorf("CJK boundary=%+v, want break opportunity with zero advance", got)
	}
	if got := stream[2].before; got.kind != inlineBreakOpportunity || got.advance <= 0 {
		t.Errorf("whitespace boundary=%+v, want break opportunity with positive advance", got)
	}
	if got := stream[3].kind; got != inlineHardBreak {
		t.Errorf("br item kind=%v, want hard break marker", got)
	}
}

func TestLayout_MinContentExcludesLeadingSpaceAtBreakBoundary(t *testing.T) {
	root, err := boxes.Build(`<span>AAAA BBBB</span>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.Measure = func(value string, _ float64) float64 {
		switch value {
		case "AAAA", "BBBB":
			return 20
		case " ":
			return 5
		default:
			return float64(len(value))
		}
	}
	got := inlineContentWidths(root.Children[0], opts).minContent
	if got != 20 {
		t.Errorf("min-content=%gpt, want 20pt (excluding the space that remains at the start of the line)", got)
	}
}

// Inline step 2a: the block mixes word styles (bold on "world") with
// increasing X positions on one line, represented by styled runs.
func TestLayout_InlineRuns_CarryStyles(t *testing.T) {
	root, err := boxes.Build(`<p>Hello <b>world</b> end</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("blocks: got %d, want 1", len(blocks))
	}
	b := blocks[0]
	if lines := runLines(b); len(lines) != 1 || lines[0] != "Hello world end" {
		t.Fatalf("lines: got %q", runLines(b))
	}
	if len(b.Runs) != 1 || len(b.Runs[0]) != 3 {
		t.Fatalf("runs: got %+v", b.Runs)
	}
	r := b.Runs[0]
	if r[0].Text != "Hello" || r[1].Text != "world" || r[2].Text != "end" {
		t.Errorf("texts: got %q %q %q", r[0].Text, r[1].Text, r[2].Text)
	}
	if r[0].Bold || !r[1].Bold || r[2].Bold {
		t.Errorf("bolds: got %v %v %v", r[0].Bold, r[1].Bold, r[2].Bold)
	}
	if !(r[0].X < r[1].X && r[1].X < r[2].X) {
		t.Errorf("X positions are not increasing: %v %v %v", r[0].X, r[1].X, r[2].X)
	}
	if r[0].X != 72 {
		t.Errorf("initial X: got %v, want 72", r[0].X)
	}
}

func TestLayout_InlineHyphenBreakContributesToMinContent(t *testing.T) {
	root, err := boxes.Build(`<div>ABCD-1234</div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	box := root.Children[0]
	words := buildInlineStream(box, opts)
	widths := inlineContentWidths(box, opts)
	if len(words) != 1 {
		t.Fatalf("inline segments=%q, want a single intact token", wordTexts(words))
	}
	if words[0].text != "ABCD-1234" || !equalInts(words[0].breakOffsets, []int{5}) {
		t.Fatalf("inline opportunities=%v in %q, want [5] in one intact token", words[0].breakOffsets, wordTexts(words))
	}
	if widths.minContent != 30 || widths.maxContent != 54 {
		t.Errorf("min/max-content=%g/%gpt, want 30/54pt", widths.minContent, widths.maxContent)
	}
}

func TestLayout_InlineHyphenBreakWrapsAtOpportunity(t *testing.T) {
	root, err := boxes.Build(`<div>ABCD-1234</div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	words := buildInlineStream(root.Children[0], testOptions())
	lines := breakInlineStream(words, 30, true)
	if len(lines) != 2 {
		t.Fatalf("lines=%d, want 2: %q", len(lines), lineWordTexts(lines))
	}
	if got := lineWordTexts(lines); got[0] != "ABCD-" || got[1] != "1234" {
		t.Errorf("corte=%q, want [ABCD- 1234]", got)
	}
}

func TestLayout_InlineHyphenBreakPreservesUnwrappedRun(t *testing.T) {
	root, err := boxes.Build(`<div>ABCD-1234</div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	lines := breakInlineStream(buildInlineStream(root.Children[0], testOptions()), 60, true)
	if len(lines) != 1 || len(lines[0]) != 1 || lines[0][0].text != "ABCD-1234" {
		t.Fatalf("unwrapped token=%q, want one intact run", lineWordTexts(lines))
	}
}

func TestLayout_InlineHyphenBreakRespectsWhiteSpace(t *testing.T) {
	cases := []struct {
		name       string
		whiteSpace string
		wantBreaks bool
	}{
		{name: "normal", whiteSpace: "normal", wantBreaks: true},
		{name: "pre-wrap", whiteSpace: "pre-wrap", wantBreaks: true},
		{name: "nowrap", whiteSpace: "nowrap"},
		{name: "pre", whiteSpace: "pre"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			root, err := boxes.Build(`<div style="white-space:` + test.whiteSpace + `">ABCD-1234</div>`)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			words := buildInlineStream(root.Children[0], testOptions())
			got := len(words) == 1 && len(words[0].breakOffsets) > 0
			if got != test.wantBreaks {
				t.Errorf("oportunidades habilitadas=%v, want %v", got, test.wantBreaks)
			}
		})
	}
}

func TestLayout_InlineHyphenBreakDoesNotSplitNegativeNumber(t *testing.T) {
	cases := []struct {
		text       string
		wantBreaks bool
	}{
		{text: "-13"},
		{text: "ABCD-1234", wantBreaks: true},
		{text: "1234-5678", wantBreaks: true},
		{text: "ABCD‐1234", wantBreaks: true},
	}
	for _, test := range cases {
		t.Run(test.text, func(t *testing.T) {
			root, err := boxes.Build(`<div>` + test.text + `</div>`)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			words := buildInlineStream(root.Children[0], testOptions())
			got := len(words) == 1 && len(words[0].breakOffsets) > 0
			if got != test.wantBreaks {
				t.Errorf("oportunidades habilitadas=%v, want %v", got, test.wantBreaks)
			}
		})
	}
}

func TestLayout_InlineBreaksPreserveGraphemeAndCJKPunctuationRules(t *testing.T) {
	cases := []struct {
		name string
		text string
		want []int
	}{
		{name: "combining-mark-cluster", text: "か\u3099漢", want: []int{2}},
		{name: "japanese-brackets-y-closing-punctuation", text: "漢「字」語", want: []int{4}},
		{name: "emoji-zwj-sequence", text: "漢👨‍👩‍👧‍👦漢", want: []int{1, 8}},
		{name: "regional-indicator-pair", text: "漢🇺🇸字", want: []int{1, 3}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := inlineBreakOffsets(test.text); !equalInts(got, test.want) {
				t.Errorf("offsets=%v in %q, want %v (rune indices; valid grapheme boundaries only)", got, test.text, test.want)
			}
		})
	}
}

func wordTexts(words inlineStream) []string {
	texts := make([]string, len(words))
	for index, word := range words {
		texts[index] = word.text
	}
	return texts
}

func lineWordTexts(lines []inlineStream) []string {
	texts := make([]string, len(lines))
	for index, line := range lines {
		for _, word := range line {
			texts[index] += word.text
		}
	}
	return texts
}

func equalInts(left, right []int) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func TestLayout_InlineWrapsAtUnicodeEmDashOpportunity(t *testing.T) {
	root, err := boxes.Build(`<div style="width:60pt;margin:0;font:10pt Helvetica">ALPHA—BETA</div>`)
	if err != nil {
		t.Fatal(err)
	}
	opts := testOptions()
	opts.Measure = func(value string, _ float64) float64 { return float64(len([]rune(value))) * 10 }
	var alphaY, betaY float64
	for _, page := range Flow(root, opts) {
		for _, block := range page.Blocks {
			for _, line := range block.Runs {
				for _, run := range line {
					if strings.Contains(run.Text, "ALPHA") {
						alphaY = run.LineY
					}
					if strings.Contains(run.Text, "BETA") {
						betaY = run.LineY
					}
				}
			}
		}
	}
	if alphaY == 0 || betaY <= alphaY {
		t.Fatalf("UAX #14 should place BETA on the second line; ALPHA— and BETA y=%.2f/%.2f", alphaY, betaY)
	}
}

func TestLayout_InlineBreakOffsetsRespectUnicodeGlueAndPunctuation(t *testing.T) {
	cases := []struct {
		name string
		text string
		want []int
	}{
		{name: "em-dash", text: "ONE—TWO", want: []int{3, 4}},
		{name: "nonbreaking-space", text: "A\u00a0B"},
		{name: "word-joiner", text: "A\u2060B"},
		{name: "japanese-closing-punctuation", text: "漢、字", want: []int{2}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := inlineBreakOffsets(test.text); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("break offsets for %q=%v, want %v", test.text, got, test.want)
			}
		})
	}
}

func TestLayout_InlineBreakOpportunityCrossesStyledUnicodeBoundary(t *testing.T) {
	root, err := boxes.Build(`<div style="width:60pt;margin:0;font:10pt Helvetica">ALPHA<span style="color:#c00">—</span>BETA</div>`)
	if err != nil {
		t.Fatal(err)
	}
	opts := testOptions()
	opts.Measure = func(value string, _ float64) float64 { return float64(len([]rune(value))) * 10 }
	var alphaY, betaY float64
	for _, page := range Flow(root, opts) {
		for _, block := range page.Blocks {
			for _, line := range block.Runs {
				for _, run := range line {
					if strings.Contains(run.Text, "ALPHA") {
						alphaY = run.LineY
					}
					if strings.Contains(run.Text, "BETA") {
						betaY = run.LineY
					}
				}
			}
		}
	}
	if alphaY == 0 || betaY <= alphaY {
		t.Fatalf("UAX #14 opportunity must cross the styled span: ALPHA— and BETA y=%.2f/%.2f", alphaY, betaY)
	}
}
