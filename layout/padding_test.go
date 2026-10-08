package layout

import (
	"strings"
	"testing"

	"github.com/robtrivi/pagecraft-go/boxes"
)

// Padding step 2: text shifts by PadLeft, and the block uses the
// full available width so the background covers the padding.
// Normal flow only in v1 (shorthand only; not padding-left).
func TestLayout_Padding_ShiftsContent(t *testing.T) {
	root, err := boxes.Build(`<p style="padding: 0 10px 0 20px">Hello</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("blocks: got %d, want 1", len(blocks))
	}
	b := blocks[0]
	if b.PadLeft != 15 {
		t.Errorf("PadLeft: got %v, want 15", b.PadLeft)
	}
	if b.Width != 451 {
		t.Errorf("Width: got %v, want 451 (background cubre padding)", b.Width)
	}
}

func TestLayout_DirectTextBoxIncludesVerticalInsets(t *testing.T) {
	root, err := boxes.Build(`<div style="padding:12pt; border:2pt solid #123456">Text</div><div style="margin:0">After</div>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("blocks: got %d, want 2", len(blocks))
	}
	if blocks[0].Height != 42.4 {
		t.Errorf("height of box: got %v, want 42.4pt (line + padding + border)", blocks[0].Height)
	}
	if len(blocks[0].Runs) == 0 || len(blocks[0].Runs[0]) == 0 || blocks[0].Runs[0][0].X != 86 || blocks[0].Runs[0][0].Y != 86 {
		t.Errorf("text must be inside the border and padding: runs=%+v", blocks[0].Runs)
	}
	if blocks[1].Y != 114.4 {
		t.Errorf("following flow: y=%v, want 114.4", blocks[1].Y)
	}
}

func TestLayout_NestedContainerDoesNotRepeatTopPaddingOnContinuation(t *testing.T) {
	root, err := boxes.Build(`<div style="padding-top:10pt"><div style="height:70pt">ONE</div><p style="height:20pt;margin:8pt 0">TWO</p></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 200, 100, 10
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages=%d, want 2", len(pages))
	}
	var continuation *PlacedBox
	for i := range pages[1].Blocks {
		if pages[1].Blocks[i].Text == "TWO" {
			continuation = &pages[1].Blocks[i]
		}
	}
	if continuation == nil {
		t.Fatalf("continued block was not found: %+v", pages[1].Blocks)
	}
	if continuation.Y != opts.Margin {
		t.Errorf("continuation y=%v, want the start of the page area at %v; parent padding must not repeat", continuation.Y, opts.Margin)
	}
}

func TestLayout_BlockMarginAtPageTopCollapsesAfterNestedBreak(t *testing.T) {
	root, err := boxes.Build(`<div style="padding-top:17pt"><div style="height:55pt">ONE</div><div style="margin-top:9pt;padding:6pt;background:#eee">NOTICE</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 200, 100, 10
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages=%d, want 2", len(pages))
	}
	var notice *PlacedBox
	for i := range pages[1].Blocks {
		if pages[1].Blocks[i].Text == "NOTICE" {
			notice = &pages[1].Blocks[i]
		}
	}
	if notice == nil {
		t.Fatalf("not found NOTICE on page 2: %+v", pages[1].Blocks)
	}
	if notice.Y != opts.Margin {
		t.Errorf("NOTICE starts at y=%v, want %v after collapsing the margin at the start of the page", notice.Y, opts.Margin)
	}
}

func TestLayout_CodeElementStaysInlineInsideDecoratedParagraph(t *testing.T) {
	root, err := boxes.Build(`<p style="padding:10pt; border:1pt solid #0284c7; margin:0">Background with <code>background-color</code> and a uniform border.</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("blocks: got %d, want wrapper and one inline content block", len(blocks))
	}
	var runs strings.Builder
	for _, line := range blocks[1].Runs {
		for _, run := range line {
			runs.WriteString(run.Text)
			runs.WriteByte(' ')
		}
	}
	if got := strings.TrimSpace(runs.String()); got != "Background with background-color and a uniform border." {
		t.Errorf("fragmented inline text: got %q", got)
	}
}
