package layout

import (
	"testing"

	"github.com/robtrivi/pagecraft-go/boxes"
)

// Lists flow by <li> item rather than as one block; each item is a block
// with its marker. This was found using pdftotext, whereitems had merged.
func TestLayout_ListItems_FlowSeparately(t *testing.T) {
	root, err := boxes.Build(`<ul><li>a</li><li>b</li></ul>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("blocks: got %d, want 2", len(blocks))
	}
	if blocks[0].Text != "• a" || blocks[1].Text != "• b" {
		t.Errorf("got %q %q", blocks[0].Text, blocks[1].Text)
	}
	if blocks[1].Y <= blocks[0].Y {
		t.Errorf("stacked items: got %v after %v", blocks[1].Y, blocks[0].Y)
	}
}

func TestLayout_NestedListFlowsOnIndentedLine(t *testing.T) {
	root, err := boxes.Build(`<ol><li>Preparation<ul><li>create the request</li></ul></li></ol>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("blocks: got %d (%q), want parent and nested item", len(blocks), blocks[0].Text)
	}
	if blocks[0].Text != "1. Preparation" || blocks[1].Text != "o create the request" {
		t.Errorf("texts: got %q / %q", blocks[0].Text, blocks[1].Text)
	}
	if blocks[1].Y <= blocks[0].Y || blocks[1].PadLeft <= blocks[0].PadLeft {
		t.Errorf("nested list should appear below with indented padding: parent=(%v,%v), nested=(%v,%v)", blocks[0].PadLeft, blocks[0].Y, blocks[1].PadLeft, blocks[1].Y)
	}
}
