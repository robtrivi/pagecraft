package layout

import (
	"testing"

	"github.com/robtrivi/pagecraft-go/boxes"
)

func TestNestedContainerInsets_AnonymousInlineFragmentDoesNotNestAgain(t *testing.T) {
	box := &boxes.Box{
		Tag: "anonymous",
		Style: boxes.ComputedStyle{
			Display:  boxes.DisplayBlock,
			HasBg:    true,
			Overflow: boxes.OverflowHidden,
			Sizing:   boxes.SizeConstraints{Height: boxes.CSSDimension{Kind: boxes.DimensionLength, Value: 100}},
		},
		Parts: []boxes.Part{{Text: "fragment text"}},
	}
	if _, ok := nestedContainerInsets(box); ok {
		t.Fatal("anonymous inline fragment must be laid out as content, not nested as a container")
	}
}

func TestLayout_DirectInlineBlockKeepsBackgroundAndDeclaredHeight(t *testing.T) {
	root, err := boxes.Build(`<div style="height:75pt;background:#eee">BEFORE</div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("blocks=%d, want 1", len(blocks))
	}
	if !blocks[0].HasBg || blocks[0].Height != 75 {
		t.Fatalf("block background/height=%v/%.2f, want true/75pt", blocks[0].HasBg, blocks[0].Height)
	}
}
