package layout

import (
	"testing"

	"github.com/robtrivi/pagecraft/boxes"
)

// Position absolute content at offsets from the page content origin.
// with top/left in px uses content-box coordinates without moving the flow
// cursor. Nested containing blocks are not supported (the page is always the containing block).
func TestLayout_Absolute_PlacesAtCoordinates(t *testing.T) {
	root, err := boxes.Build(`<div style="position: absolute; top: 100px; left: 50px">Stamp</div><p style="margin: 0">Text</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pages := Flow(root, testOptions())
	blocks := pages[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("blocks: got %d, want 2", len(blocks))
	}
	// The content box starts at (72,72); the offsets place it at (122,172).
	if blocks[0].X != 109.5 || blocks[0].Y != 147 {
		t.Errorf("absolute: got (%v,%v), want (109.5,147)", blocks[0].X, blocks[0].Y)
	}
	// Absolute positioning does not advance flow.
	if blocks[1].Y != 72 {
		t.Errorf("flow must ignore absolute positioning: got y=%v, want 72", blocks[1].Y)
	}
}

func TestLayout_AbsoluteUsesRelativeAncestorOrigin(t *testing.T) {
	root, err := boxes.Build(`<div style="position: relative"><p style="margin:0">Anchor</p><div style="position:absolute; top:10px; left:20px">Stamp</div></div>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pages := Flow(root, testOptions())
	var absolute *PlacedBox
	for i := range pages[0].Blocks {
		if pages[0].Blocks[i].Text == "Stamp" {
			absolute = &pages[0].Blocks[i]
			break
		}
	}
	if absolute == nil {
		t.Fatal("absolute block not found")
	}
	// The ancestor starts at the page content origin; offsets are 20/10 px.
	if absolute.X != 87 || absolute.Y != 79.5 {
		t.Errorf("absolute: got (%v,%v), want (87,79.5)", absolute.X, absolute.Y)
	}
}

func TestLayout_AbsoluteRightAndWidthUseRelativeContainingBlock(t *testing.T) {
	root, err := boxes.Build(`<div style="position: relative; height:130pt; border:1pt solid #94a3b8"><p style="margin:0">Anchor</p><div style="position:absolute; right:10pt; top:12pt; width:130pt; padding:8pt; border:1pt solid #d97706">Stamp</div></div><p style="margin:0">After</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pages := Flow(root, testOptions())
	var absolute *PlacedBox
	for i := range pages[0].Blocks {
		if pages[0].Blocks[i].Text == "Stamp" {
			absolute = &pages[0].Blocks[i]
			break
		}
	}
	if absolute == nil {
		t.Fatal("absolute block not found")
	}
	// The ancestor's A4 content box starts at x=73 and is 449 pt wide; child box is 148 pt.
	if absolute.X != 364 || absolute.Width != 148 {
		t.Errorf("absolute: x/width got %v/%v, want 364/148", absolute.X, absolute.Width)
	}
	if pages[0].Blocks[0].Height < 130 {
		t.Errorf("ancestro: height got %v, want at least 130pt", pages[0].Blocks[0].Height)
	}
	var following *PlacedBox
	for i := range pages[0].Blocks {
		if pages[0].Blocks[i].Text == "After" {
			following = &pages[0].Blocks[i]
			break
		}
	}
	if following == nil || following.Y < 204 {
		t.Errorf("following flow: got %+v, want and >= 204pt", following)
	}
}

func TestLayout_AbsolutePercentageWidthResolvesAgainstContainingBlock(t *testing.T) {
	root, err := boxes.Build(`<div style="position:relative;width:300pt"><div style="position:absolute;left:0;width:50%;padding:8pt;border:1pt solid #000">Stamp</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	pages := Flow(root, testOptions())
	for _, block := range pages[0].Blocks {
		if block.Text == "Stamp" {
			if block.Width != 168 {
				t.Fatalf("absolute 50%% width with 18pt insets=%v, want 168pt", block.Width)
			}
			return
		}
	}
	t.Fatal("absolute positioned content not found")
}

func TestLayout_AbsoluteAutoHeightIncludesWrappedContentAndInsets(t *testing.T) {
	root, err := boxes.Build(`<div style="position:relative; width:300pt; height:130pt"><div style="position:absolute; top:12pt; right:10pt; width:130pt; padding:8pt; border:1pt solid #d97706">Position absolute en the corner upper right.</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	pages := Flow(root, testOptions())
	var absolute *PlacedBox
	for i := range pages[0].Blocks {
		if pages[0].Blocks[i].Text == "Position absolute en the corner upper right." {
			absolute = &pages[0].Blocks[i]
			break
		}
	}
	if absolute == nil {
		t.Fatal("absolute block not found")
	}
	if len(absolute.Runs) < 2 {
		t.Fatalf("the text should wrap; lines=%d", len(absolute.Runs))
	}
	contentHeight := float64(len(absolute.Runs)) * absolute.Runs[0][0].LineH
	want := contentHeight + 18 // 16pt vertical padding + 2pt vertical borders.
	if absolute.Height != want {
		t.Errorf("height automatic=%v, want content+insets=%v", absolute.Height, want)
	}
}

func TestLayout_RelativeOffsetsVisualBoxWithoutMovingFlow(t *testing.T) {
	root, err := boxes.Build(`<p style="position:relative; left:12pt; top:5pt; margin:0">Movido</p><p style="margin:0">next</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("blocks: got %d, want 2", len(blocks))
	}
	if blocks[0].X != 84 || blocks[0].Y != 77 {
		t.Errorf("box relativa: got (%v,%v), want (84,77)", blocks[0].X, blocks[0].Y)
	}
	if blocks[1].Y != 86.4 {
		t.Errorf("following flow: y=%v, want 86.4 (without relative offset)", blocks[1].Y)
	}
}
