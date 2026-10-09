package layout

import (
	"strings"
	"testing"

	"github.com/robtrivi/pagecraft-go/boxes"
)

func TestLayout_BreakAfterAvoidKeepsFollowingFragmentTogether(t *testing.T) {
	for _, property := range []string{"break-after: avoid", "page-break-after: avoid"} {
		t.Run(property, func(t *testing.T) {
			html := strings.Repeat(`<p style="margin: 0">x</p>`, 40) +
				`<p style="margin: 0; ` + property + `">Title</p>` +
				`<p style="margin: 0">` + strings.Repeat("ab ", 200) + `</p>`
			root, err := boxes.Build(html)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			pages := Flow(root, testOptions())
			if len(pages) != 2 {
				t.Fatalf("pages=%d, want 2", len(pages))
			}
			if !pageContainsText(pages[0], "Title") {
				t.Fatal("Chromium/WeasyPrint keep the title with the start of the paragraph on page 1")
			}
			if !pageContainsText(pages[0], "ab ") || !pageContainsText(pages[1], "ab ") {
				t.Fatal("the paragraph must start on page 1 and continue on page 2")
			}
		})
	}
}

func pageContainsText(page Page, text string) bool {
	for _, block := range page.Blocks {
		if strings.Contains(block.Text, text) {
			return true
		}
	}
	return false
}

func TestLayout_BreakBeforePageStartsBlockOnNewPage(t *testing.T) {
	root, err := boxes.Build(`<p>A</p><p style="break-before: page">B</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	pages := Flow(root, testOptions())
	if len(pages) != 2 {
		t.Fatalf("pages=%d, want 2", len(pages))
	}
	if len(pages[1].Blocks) == 0 || pages[1].Blocks[0].Text != "B" {
		t.Fatalf("page 2 must start with B; blocks=%+v", pages[1].Blocks)
	}
}

func TestLayout_BreakAfterPageStartsFollowingBlockOnNewPage(t *testing.T) {
	for _, property := range []string{"break-after: page", "page-break-after: always"} {
		t.Run(property, func(t *testing.T) {
			root, err := boxes.Build(`<p>A</p><p style="` + property + `">B</p><p>C</p>`)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			pages := Flow(root, testOptions())
			if len(pages) != 2 {
				t.Fatalf("pages=%d, want 2", len(pages))
			}
			if len(pages[0].Blocks) != 2 || pages[0].Blocks[1].Text != "B" {
				t.Fatalf("page 1 must end with B; blocks=%+v", pages[0].Blocks)
			}
			if len(pages[1].Blocks) == 0 || pages[1].Blocks[0].Text != "C" {
				t.Fatalf("page 2 must start with C; blocks=%+v", pages[1].Blocks)
			}
		})
	}
}

func TestLayout_BreakAfterPageDoesNotCreateBlankPageBeforeBreakBefore(t *testing.T) {
	root, err := boxes.Build(`<p style="break-after: page">A</p><p style="break-before: page">B</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	pages := Flow(root, testOptions())
	if len(pages) != 2 {
		t.Fatalf("pages=%d, want 2 without a blank page", len(pages))
	}
	if len(pages[1].Blocks) == 0 || pages[1].Blocks[0].Text != "B" {
		t.Fatalf("page 2 must start with B; blocks=%+v", pages[1].Blocks)
	}
}

func TestLayout_BreakAfterPageOnFinalBlockDoesNotCreateBlankPage(t *testing.T) {
	root, err := boxes.Build(`<p style="break-after: page">A</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if pages := Flow(root, testOptions()); len(pages) != 1 {
		t.Fatalf("pages=%d, want 1", len(pages))
	}
}

func TestLayout_BreakAfterPageWorksAcrossDocumentStructures(t *testing.T) {
	cases := []struct {
		name, content string
	}{
		{name: "block", content: `<p style="margin:0;break-after:page">CUT BLOCK</p>`},
		{name: "decorated container", content: `<section style="break-after:page;padding:1pt;border:1pt solid #000"><p>CUT WRAPPER</p></section>`},
		{name: "nested block", content: `<section><p style="break-after:page">CUT INNER</p><p>INNER AFTER</p></section>`},
		{name: "table", content: `<table style="break-after:page"><tr><td>CUT TABLE</td></tr></table>`},
		{name: "flex", content: `<div style="display:flex;break-after:page"><div>CUT FLEX</div></div>`},
		{name: "grid", content: `<div style="display:grid;break-after:page"><div>CUT GRID</div></div>`},
		{name: "columns", content: `<div style="column-count:2;break-after:page"><p>CUT COLUMNS</p></div>`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			html := `<style>body{margin:0}</style><div style="height:10pt"></div>` + tc.content + `<p>AFTER MARK</p>`
			root, err := boxes.Build(html)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			pages := Flow(root, testOptions())
			if len(pages) != 2 {
				t.Fatalf("pages=%d, want 2", len(pages))
			}
			if !pageContainsText(pages[0], "CUT ") {
				t.Fatalf("the structure must remain above the fold: %+v", pages[0].Blocks)
			}
			if !pageContainsText(pages[1], "AFTER MARK") {
				t.Fatalf("following flow must start on page 2: %+v", pages[1].Blocks)
			}
		})
	}
}

func TestLayout_BreakAfterAvoidBeforeFragmentingFlexKeepsTitleWithFirstItem(t *testing.T) {
	html := `<style>body{margin:0}.space{height:175pt}.box{border:1pt solid #0369a1;padding:3pt}p{margin:0}.flex{display:flex;gap:4pt}.item{padding:3pt}</style>` +
		`<div class="space"></div><p class="box" style="break-after:avoid">KEEP TITLE</p>` +
		`<div class="flex box" style="height:120pt"><div class="item">FIRST FLEX ITEM</div><div class="item">SECOND FLEX ITEM</div></div><p>END MARK</p>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 260, 260, 20
	opts.FontSize, opts.LineHeight = 10, 1.2
	opts.Measure = func(text string, _ float64) float64 { return float64(len([]rune(text)) * 5) }
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages=%d, want 2", len(pages))
	}
	if !pageContainsText(pages[0], "KEEP TITLE") || !pageContainsText(pages[0], "FIRST FLEX ITEM") {
		t.Fatal("Chromium keeps the title and first Flex item together when fragmentation starts")
	}
	if !pageContainsText(pages[1], "END MARK") {
		t.Fatal("flow after Flex must continue on page 2")
	}
}

func TestLayout_DecoratedBreakAfterAvoidKeepsParagraphOrphansTogether(t *testing.T) {
	html := `<style>body{margin:0}.space{height:177pt}.box{border:1pt solid #0369a1;padding:3pt}p{margin:0}</style>` +
		`<div class="space"></div><p class="box" style="break-after:avoid">KEEP TITLE</p>` +
		`<p>` + strings.Repeat("following words ", 90) + `</p><p>END MARK</p>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 260, 260, 20
	opts.FontSize, opts.LineHeight = 10, 1.2
	opts.Measure = func(text string, _ float64) float64 { return float64(len([]rune(text))) * 4.5 }
	pages := Flow(root, opts)
	if len(pages) != 3 {
		t.Fatalf("pages=%d, want 3", len(pages))
	}
	if pageContainsText(pages[0], "KEEP TITLE") || !pageContainsText(pages[1], "KEEP TITLE") {
		t.Fatal("Chromium and WeasyPrint move the title to page 2 to preserve the orphan lines")
	}
	followingCount := 0
	for _, page := range pages {
		for _, block := range page.Blocks {
			followingCount += strings.Count(block.Text, "following")
		}
	}
	if followingCount != 90 {
		t.Fatalf("continuation text=%d words, want 90 without loss", followingCount)
	}
}
