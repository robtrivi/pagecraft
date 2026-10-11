package layout

import (
	"fmt"
	"strings"
	"testing"

	"github.com/robtrivi/pagecraft/boxes"
)

// Word-based line breaking with an injectable measurer. This lets render draw
// each line and keeps pagination consistent with painted output.
func TestLayout_BreakLines_SplitsLongText(t *testing.T) {
	measure := func(s string, _ float64) float64 { return float64(len([]rune(s)) * 6) }
	lines := BreakLines(strings.Repeat("ab ", 50), 451, measure, 12)
	if len(lines) != 2 {
		t.Fatalf("lines: got %d (%q), want 2", len(lines), lines)
	}
	if got := strings.Join(lines, " "); got != strings.TrimSpace(strings.Repeat("ab ", 50)) {
		t.Errorf("text recompuesto difiere: %q", got)
	}
	for _, ln := range lines {
		if measure(ln, 12) > 451 {
			t.Errorf("line exceeds available width: %q", ln)
		}
	}
}

func TestLayout_BreakLines_SingleWordOverflow(t *testing.T) {
	measure := func(s string, _ float64) float64 { return float64(len([]rune(s)) * 6) }
	lines := BreakLines(strings.Repeat("a", 100), 451, measure, 12)
	if len(lines) != 1 {
		t.Fatalf("long word was not part of the forced break: got %d lines", len(lines))
	}
}

func TestLayout_BlockLinesContinueOnNextPage(t *testing.T) {
	html := `<p style="margin:0;line-height:20pt">filler1<br>filler2<br>filler3</p>` +
		`<p style="margin:0;line-height:20pt">keep1<br>keep2<br>keep3<br>keep4</p>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth = 200
	opts.PageHeight = 140
	opts.Margin = 20
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages: got %d, want 2", len(pages))
	}
	pageText := func(page Page) string {
		var lines []string
		for _, block := range page.Blocks {
			for _, line := range block.Runs {
				var words []string
				for _, run := range line {
					words = append(words, run.Text)
				}
				lines = append(lines, strings.Join(words, ""))
			}
		}
		return strings.Join(lines, "|")
	}
	if got, want := pageText(pages[0]), "filler1|filler2|filler3|keep1|keep2"; got != want {
		t.Errorf("page 1: got %q, want %q", got, want)
	}
	if got, want := pageText(pages[1]), "keep3|keep4"; got != want {
		t.Errorf("page 2: got %q, want %q", got, want)
	}
}

func TestLayout_BodyMarginDoesNotReducePageBottomCapacity(t *testing.T) {
	lines := make([]string, 30)
	for i := range lines {
		lines[i] = fmt.Sprintf("%02d", i+1)
	}
	html := `<html><head><style>@page{size:200pt 140pt;margin:20pt}p{margin:0;font:10pt/10pt Helvetica}</style></head><body><p>` + strings.Join(lines, `<br>`) + `</p></body></html>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 200, 140, 20
	pages := Flow(root, opts)
	var firstPageLines []string
	for _, block := range pages[0].Blocks {
		for _, line := range block.Runs {
			var words []string
			for _, run := range line {
				words = append(words, run.Text)
			}
			firstPageLines = append(firstPageLines, strings.Join(words, ""))
		}
	}
	if got := strings.Join(firstPageLines, "|"); got != "01|02|03|04|05|06|07|08|09" {
		t.Errorf("lines of page 1=%q, want 9 lines up to the CSS content edge", got)
	}
}

func TestLayout_BreakInsideAvoidMovesBlockToFreshPage(t *testing.T) {
	html := `<p style="margin:0;line-height:20pt">filler1<br>filler2<br>filler3</p>` +
		`<p style="margin:0;line-height:20pt;break-inside:avoid">keep1<br>keep2<br>keep3<br>keep4</p>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth = 200
	opts.PageHeight = 140
	opts.Margin = 20
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages: got %d, want 2", len(pages))
	}
	pageText := func(page Page) string {
		var lines []string
		for _, block := range page.Blocks {
			for _, line := range block.Runs {
				var words []string
				for _, run := range line {
					words = append(words, run.Text)
				}
				lines = append(lines, strings.Join(words, ""))
			}
		}
		return strings.Join(lines, "|")
	}
	if got, want := pageText(pages[0]), "filler1|filler2|filler3"; got != want {
		t.Errorf("page 1: got %q, want %q", got, want)
	}
	if got, want := pageText(pages[1]), "keep1|keep2|keep3|keep4"; got != want {
		t.Errorf("page 2: got %q, want %q", got, want)
	}
}

func TestLayout_NestedDecoratedBlockLinesContinueOnNextPage(t *testing.T) {
	html := `<div style="opacity:0.5;background-color:#ff0000">` +
		`<p style="margin:0;line-height:20pt">filler1<br>filler2<br>filler3</p>` +
		`<p style="margin:0;line-height:20pt">keep1<br>keep2<br>keep3<br>keep4</p></div>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth = 200
	opts.PageHeight = 140
	opts.Margin = 20
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages: got %d, want 2", len(pages))
	}
	pageText := func(page Page) string {
		var lines []string
		for _, block := range page.Blocks {
			for _, line := range block.Runs {
				var words []string
				for _, run := range line {
					words = append(words, run.Text)
				}
				lines = append(lines, strings.Join(words, ""))
			}
		}
		return strings.Join(lines, "|")
	}
	wantPages := []string{"filler1|filler2|filler3|keep1|keep2", "keep3|keep4"}
	for index, page := range pages {
		if got := pageText(page); got != wantPages[index] {
			t.Errorf("page %d: got %q, want %q", index+1, got, wantPages[index])
		}
		if len(page.Blocks) == 0 || len(page.Blocks[0].OpacityGroups) != 1 {
			t.Errorf("page %d: the wrapper must preserve its opacity group", index+1)
		}
	}
}

func TestLayout_ParagraphWidowsAndOrphansKeepTwoLines(t *testing.T) {
	tests := []struct {
		name, html string
		want       []string
	}{
		{
			name: "orphans",
			html: `<p style="margin:0;line-height:20pt">f1<br>f2<br>f3<br>f4</p><p style="margin:0;line-height:20pt">p1<br>p2<br>p3</p>`,
			want: []string{"f1|f2|f3|f4", "p1|p2|p3"},
		},
		{
			name: "widows",
			html: `<p style="margin:0;line-height:20pt">f1<br>f2</p><p style="margin:0;line-height:20pt">p1<br>p2<br>p3<br>p4</p>`,
			want: []string{"f1|f2|p1|p2", "p3|p4"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root, err := boxes.Build(tc.html)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			opts := testOptions()
			opts.PageWidth, opts.PageHeight, opts.Margin = 200, 140, 20
			pages := Flow(root, opts)
			if len(pages) != len(tc.want) {
				t.Fatalf("pages: got %d, want %d", len(pages), len(tc.want))
			}
			for i, page := range pages {
				var lines []string
				for _, block := range page.Blocks {
					for _, line := range block.Runs {
						var words []string
						for _, run := range line {
							words = append(words, run.Text)
						}
						lines = append(lines, strings.Join(words, ""))
					}
				}
				if got := strings.Join(lines, "|"); got != tc.want[i] {
					t.Errorf("page %d: got %q, want %q", i+1, got, tc.want[i])
				}
			}
		})
	}
}

func TestLayout_WidowsPropertyKeepsConfiguredLines(t *testing.T) {
	root, err := boxes.Build(`<p style="margin:0;line-height:20pt;widows:3">p1<br>p2<br>p3<br>p4<br>p5<br>p6</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 200, 140, 20
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages: got %d, want 2", len(pages))
	}
	want := []string{"p1|p2|p3", "p4|p5|p6"}
	for i, page := range pages {
		var lines []string
		for _, block := range page.Blocks {
			for _, line := range block.Runs {
				var words []string
				for _, run := range line {
					words = append(words, run.Text)
				}
				lines = append(lines, strings.Join(words, ""))
			}
		}
		if got := strings.Join(lines, "|"); got != want[i] {
			t.Errorf("page %d: got %q, want %q", i+1, got, want[i])
		}
	}
}

func TestLayout_WidowsConstraintRelaxesLikeChromiumWhenNoValidCutFits(t *testing.T) {
	root, err := boxes.Build(`<p style="margin:0;line-height:20pt">f1<br>f2</p><p style="margin:0;line-height:20pt;widows:3">p1<br>p2<br>p3<br>p4</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 200, 140, 20
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages: got %d, want 2", len(pages))
	}
	want := []string{"f1|f2|p1|p2", "p3|p4"}
	for i, page := range pages {
		var lines []string
		for _, block := range page.Blocks {
			for _, line := range block.Runs {
				var words []string
				for _, run := range line {
					words = append(words, run.Text)
				}
				lines = append(lines, strings.Join(words, ""))
			}
		}
		if got := strings.Join(lines, "|"); got != want[i] {
			t.Errorf("page %d: got %q, want %q", i+1, got, want[i])
		}
	}
}
