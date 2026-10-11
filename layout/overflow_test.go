package layout

import (
	"strings"
	"testing"

	"github.com/robtrivi/pagecraft/boxes"
)

func TestLayout_OverflowHiddenClipsDescendantsToPaddingBox(t *testing.T) {
	root, err := boxes.Build(`<div style="width:100pt;height:40pt;padding:5pt;border:2pt solid black;overflow:hidden"><p>content overflowing</p><p>second block that also exceeds the fixed height</p><p>third content block</p></div><p>following</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	pages := Flow(root, testOptions())
	if len(pages) != 1 {
		t.Fatalf("pages=%d, want 1", len(pages))
	}
	var inside, outside, wrapper *PlacedBox
	for i := range pages[0].Blocks {
		b := &pages[0].Blocks[i]
		if b.Text == "content overflowing" {
			inside = b
		}
		if b.Text == "following" {
			outside = b
		}
		if b.Text == "" && b.Width == 114 {
			wrapper = b
		}
	}
	if inside == nil || outside == nil || wrapper == nil {
		t.Fatalf("could not find the wrapper and both text blocks: %+v", pages[0].Blocks)
	}
	if wrapper.Height != 54 {
		t.Errorf("height of the wrapper=%g, want 54pt (height CSS + padding + border)", wrapper.Height)
	}
	if outside.Y > wrapper.Y+wrapper.Height+20 {
		t.Errorf("following block starts at y=%g, too far below wrapper y=%g h=%g", outside.Y, wrapper.Y, wrapper.Height)
	}
	if len(inside.ClipRects) != 1 {
		t.Fatalf("descendant clips=%v, want 1 padding-box clip", inside.ClipRects)
	}
	if len(outside.ClipRects) != 0 {
		t.Fatalf("clipping escaped the container: %v", outside.ClipRects)
	}
	clip := inside.ClipRects[0]
	if clip.Width != 110 || clip.Height != 50 {
		t.Errorf("clip=%+v; must measure the padding box 110x50pt", clip)
	}
}

func TestLayout_OverflowHiddenKeepsNestedClipRegions(t *testing.T) {
	root, err := boxes.Build(`<div style="width:120pt;height:100pt;overflow:hidden"><div style="width:80pt;height:30pt;overflow:hidden"><p>content</p></div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	pages := Flow(root, testOptions())
	found := false
	for _, page := range pages {
		for _, block := range page.Blocks {
			if block.Text == "content" {
				found = true
				if len(block.ClipRects) != 2 {
					t.Fatalf("nested clips=%v, want 2", block.ClipRects)
				}
			}
		}
	}
	if !found {
		t.Fatal("inner content was not found")
	}
}

func TestLayout_OverflowHiddenFixedHeightDoesNotExposeDescendantsOnLaterPages(t *testing.T) {
	root, err := boxes.Build(`<div style="height:105pt"></div><div style="height:35pt;overflow:hidden;background:red;border:2pt solid blue"><p>ONE visible first line</p><p>TWO should be clipped</p><p>THREE should be clipped</p><p>FOUR should be clipped</p><p>FIVE should be clipped</p><p>SIX should be clipped</p><p>SEVEN should be clipped</p></div><p>AFTER visible</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 300, 160, 10
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages=%d, want 2; hidden overflow must not add pages", len(pages))
	}
	var afterVisible bool
	for _, block := range pages[1].Blocks {
		if block.Text == "AFTER visible" {
			afterVisible = true
			continue
		}
		if strings.TrimSpace(block.Text) != "" && (len(block.ClipRects) != 1 || block.ClipRects[0].Height != 2) {
			t.Errorf("descendant continuation must be limited to 2 visible pt: %q clips=%+v", block.Text, block.ClipRects)
		}
	}
	if !afterVisible {
		t.Errorf("following flow must remain on page 2: %+v", pages[1].Blocks)
	}
}

func TestLayout_OverflowHiddenFixedHeightDoesNotPaginateHiddenDescendants(t *testing.T) {
	content := strings.Repeat("HIDDEN CONTENT THAT MUST STAY INSIDE THE FIXED CLIP. ", 40)
	cases := []struct {
		name  string
		start string
		end   string
	}{
		{name: "block", start: `<div style="width:100pt;height:35pt;overflow:hidden"><p style="margin:0">`, end: `</p></div>`},
		{name: "flex", start: `<div style="display:flex;width:100pt;height:35pt;overflow:hidden"><div style="width:100pt"><p style="margin:0">`, end: `</p></div></div>`},
		{name: "grid", start: `<div style="display:grid;width:100pt;height:35pt;grid-template-columns:100pt;overflow:hidden"><div><p style="margin:0">`, end: `</p></div></div>`},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			root, err := boxes.Build(test.start + content + test.end + `<p style="margin:0">AFTER CLIP</p>`)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			opts := testOptions()
			opts.PageWidth, opts.PageHeight, opts.Margin = 225, 135, 9
			pages := Flow(root, opts)
			if len(pages) != 1 {
				t.Fatalf("pages=%d, want 1: clipped content must not create fragmentainers", len(pages))
			}
			foundAfter, foundClippedContent := false, false
			for _, block := range pages[0].Blocks {
				if block.Text == "AFTER CLIP" {
					foundAfter = true
				}
				if strings.Contains(block.Text, "HIDDEN CONTENT") {
					foundClippedContent = true
					if len(block.ClipRects) != 1 || block.ClipRects[0].Height != 35 {
						t.Errorf("descendant clip=%+v, want a 35pt region", block.ClipRects)
					}
				}
			}
			if !foundAfter {
				t.Fatal("flow must continue after the container’s declared height")
			}
			if !foundClippedContent {
				t.Fatal("descendant content must be painted below the container clip")
			}
		})
	}
}

func TestLayout_OverflowHiddenAllowsDescendantThroughPaddingBoxAtPageBottom(t *testing.T) {
	root, err := boxes.Build(`<div style="height:71.25pt"></div><div style="width:180pt;height:26.25pt;overflow:hidden;border:1.5pt solid blue"><p style="margin:0;font-size:9pt;line-height:10.5pt">ONE visible</p><p style="margin:0;font-size:9pt;line-height:10.5pt">TWO visible</p><p style="margin:0;font-size:9pt;line-height:10.5pt">THREE clipped partially</p></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 225, 120, 7.5
	pages := Flow(root, opts)
	for _, block := range pages[0].Blocks {
		if strings.HasPrefix(block.Text, "THREE clipped partially") {
			if len(block.ClipRects) != 1 {
				t.Fatalf("descendant clip=%+v, want a region", block.ClipRects)
			}
			return
		}
	}
	t.Fatalf("the third descendant must remain on the first page so that CSS clipping shows the part inside the padding box; pages: %+v", pages)
}

func TestLayout_OverflowHiddenFixedHeightContinuesVisibleClipAcrossPages(t *testing.T) {
	root, err := boxes.Build(`<div style="height:79.5pt"></div><div style="width:127.5pt;height:52.5pt;overflow:hidden;border:1.5pt solid blue"><p style="margin:0;height:10.5pt;font-size:9pt;line-height:10.5pt">LINE ONE</p><p style="margin:0;height:10.5pt;font-size:9pt;line-height:10.5pt">LINE TWO</p><p style="margin:0;height:10.5pt;font-size:9pt;line-height:10.5pt">LINE THREE</p><p style="margin:0;height:10.5pt;font-size:9pt;line-height:10.5pt">LINE FOUR</p><p style="margin:0;height:10.5pt;font-size:9pt;line-height:10.5pt">LINE FIVE</p><p style="margin:0;height:10.5pt;font-size:9pt;line-height:10.5pt">LINE SIX</p></div><p style="margin:0">AFTER</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 225, 135, 9
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages=%d, want 2; the visible container fragment must continue on page 2: %+v", len(pages), pages)
	}
	var lineOnePage, lineFourPage, afterPage int
	var fragments []PlacedBox
	for pageIndex, page := range pages {
		for _, block := range page.Blocks {
			switch block.Text {
			case "LINE ONE":
				lineOnePage = pageIndex + 1
			case "LINE FOUR":
				lineFourPage = pageIndex + 1
			case "AFTER":
				afterPage = pageIndex + 1
			}
			if block.Text == "" && block.Width == 130.5 && block.BorderLeft.Width == 1.5 {
				fragments = append(fragments, block)
			}
			if pageIndex == 1 && strings.HasPrefix(block.Text, "LINE ") {
				if len(block.ClipRects) != 1 || block.ClipRects[0].Y != 10.5 || block.ClipRects[0].Height != 16.5 {
					t.Errorf("clip of continuation=%+v, want y=10.5 h=16.5pt", block.ClipRects)
				}
			}
		}
	}
	if lineOnePage != 1 || lineFourPage != 2 || afterPage != 2 {
		t.Errorf("pages of LINE ONE/LINE FOUR/AFTER = %d/%d/%d, want 1/2/2", lineOnePage, lineFourPage, afterPage)
	}
	if len(fragments) != 2 || fragments[0].Height != 37.5 || fragments[1].Height != 18 {
		t.Errorf("container fragments=%+v; want heights 37.5pt and 18pt", fragments)
	}
	if len(fragments) == 2 && (fragments[0].BorderBottom.Width != 0 || fragments[1].BorderTop.Width != 0 || fragments[0].BorderWidth != 0 || fragments[1].BorderWidth != 0) {
		t.Errorf("bottom borders must appear only at the fragment edge: %+v", fragments)
	}
}

func TestLayout_FlexOverflowHiddenClipsItemsToContainerWidth(t *testing.T) {
	root, err := boxes.Build(`<div style="display:flex;width:100px;overflow-x:hidden;overflow-y:visible"><div style="white-space:nowrap">FLEX OVERFLOW CONTENT MUST STOP AT THE CONTAINER EDGE repeat repeat repeat</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 225, 135, 9
	pages := Flow(root, opts)
	for _, page := range pages {
		for _, block := range page.Blocks {
			if !strings.HasPrefix(block.Text, "FLEX OVERFLOW") {
				continue
			}
			if len(block.ClipRects) != 1 {
				t.Fatalf("clip of the item flex=%+v, want the container clip", block.ClipRects)
			}
			clip := block.ClipRects[0]
			if clip.X != 9 || clip.Width != 75 {
				t.Errorf("clip=%+v, want content box x=9 width=75pt", clip)
			}
			return
		}
	}
	t.Fatal("flex item was not found")
}

func TestLayout_FlexOverflowHiddenFixedHeightClipsAndKeepsFollowingFlow(t *testing.T) {
	root, err := boxes.Build(`<div style="display:flex;height:20pt;overflow:hidden"><div>OVERFLOW ` + strings.Repeat("content largo ", 18) + `</div></div><p>AFTER FLEX</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 225, 200, 9
	pages := Flow(root, opts)
	var overflowing, after *PlacedBox
	for i := range pages[0].Blocks {
		block := &pages[0].Blocks[i]
		if strings.HasPrefix(block.Text, "OVERFLOW") {
			overflowing = block
		}
		if block.Text == "AFTER FLEX" {
			after = block
		}
	}
	if overflowing == nil || after == nil {
		t.Fatalf("items flex yflowfollowing not ubicatwo: %+v", pages[0].Blocks)
	}
	if len(overflowing.ClipRects) != 1 || overflowing.ClipRects[0].Height != 20 {
		t.Errorf("clip of the item=%+v, want fixed height of 20pt", overflowing.ClipRects)
	}
	if after.Y > overflowing.Y+20+20 {
		t.Errorf("following flow y=%g must follow the fixed height of 20pt, item starts in %g", after.Y, overflowing.Y)
	}
}

func TestLayout_GridOverflowHiddenClipsItemsToContainerWidth(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;width:100px;grid-template-columns:1fr;overflow-x:hidden;overflow-y:visible"><div style="white-space:nowrap">GRID OVERFLOW CONTENT MUST STOP AT THE CONTAINER EDGE repeat repeat repeat</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 225, 135, 9
	pages := Flow(root, opts)
	for _, page := range pages {
		for _, block := range page.Blocks {
			if !strings.HasPrefix(block.Text, "GRID OVERFLOW") {
				continue
			}
			if len(block.ClipRects) != 1 {
				t.Fatalf("clip of the item grid=%+v, want the container clip", block.ClipRects)
			}
			clip := block.ClipRects[0]
			if clip.X != 9 || clip.Width != 75 {
				t.Errorf("clip=%+v, want content box x=9 width=75pt", clip)
			}
			return
		}
	}
	t.Fatal("grid item was not found")
}

func TestLayout_OverflowXHiddenComputesVisibleYToAuto(t *testing.T) {
	root, err := boxes.Build(`<div style="width:100px;overflow-x:hidden;overflow-y:visible"><p style="white-space:nowrap">AXIS OVERFLOW CONTENT MUST STOP AT ONE HUNDRED PIXELS repeat repeat repeat</p></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 225, 135, 9
	pages := Flow(root, opts)
	for _, block := range pages[0].Blocks {
		if !strings.HasPrefix(block.Text, "AXIS OVERFLOW") {
			continue
		}
		if len(block.ClipRects) != 1 {
			t.Fatalf("clips=%+v, want horizontal clipping", block.ClipRects)
		}
		clip := block.ClipRects[0]
		if clip.X != 9 || clip.Width != 75 || clip.Y <= 0 || clip.Height >= opts.PageHeight {
			t.Errorf("clip=%+v, want horizontal clipping plus the computed vertical auto scrollport", clip)
		}
		return
	}
	t.Fatal("block descendant not found")
}

func TestLayout_VisibleOverflowXComputesToAutoWhenYIsHidden(t *testing.T) {
	root, err := boxes.Build(`<div style="width:100px;height:20pt;overflow:hidden;overflow-x:visible"><p>CONTENT</p></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 225, 135, 9
	pages := Flow(root, opts)
	for _, block := range pages[0].Blocks {
		if block.Text != "CONTENT" {
			continue
		}
		if len(block.ClipRects) != 1 {
			t.Fatalf("clips=%+v, want the vertical axis only", block.ClipRects)
		}
		clip := block.ClipRects[0]
		if clip.X != 9 || clip.Width != 75 || clip.Y == 0 || clip.Height != 20 {
			t.Errorf("clip=%+v; overflow-x:visible must compute to auto when overflow-y is hidden", clip)
		}
		return
	}
	t.Fatal("content was not found")
}

func TestLayout_GridOverflowHiddenFixedHeightClipsWithoutPagination(t *testing.T) {
	content := strings.Repeat("GRID CONTENT OVERFLOWS THE FIXED TRACK ", 16)
	root, err := boxes.Build(`<div style="height:63.75pt">BEFORE</div><div style="display:grid;grid-template-columns:100px;height:30pt;overflow:hidden"><div>` + content + `</div></div><p style="margin:0">AFTER GRID</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 225, 135, 9
	pages := Flow(root, opts)
	if len(pages) != 1 {
		t.Fatalf("pages=%d, want 1: the hidden grid overflow must not create fragments", len(pages))
	}
	var overflowing, after *PlacedBox
	for i := range pages[0].Blocks {
		block := &pages[0].Blocks[i]
		if strings.HasPrefix(block.Text, "GRID CONTENT") {
			overflowing = block
		}
		if block.Text == "AFTER GRID" {
			after = block
		}
	}
	if overflowing == nil || after == nil {
		t.Fatalf("Grid and following flow were not positioned: %+v", pages[0].Blocks)
	}
	if len(overflowing.ClipRects) != 1 || overflowing.ClipRects[0].Height != 30 {
		t.Errorf("clip of the item=%+v, want declared height of 30pt", overflowing.ClipRects)
	}
	if after.Y > overflowing.ClipRects[0].Y+30+20 {
		t.Errorf("following flow y=%g must follow the fixed box, clip=%+v", after.Y, overflowing.ClipRects[0])
	}
}

func TestLayout_FlexOverflowHiddenFixedHeightDoesNotPaginateDescendants(t *testing.T) {
	content := strings.Repeat("FLEX CONTENT OVERFLOWS THE FIXED TRACK ", 16)
	root, err := boxes.Build(`<div style="height:63.75pt">BEFORE</div><div style="display:flex;height:30pt;overflow:hidden"><div>` + content + `</div></div><p style="margin:0">AFTER FLEX</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 225, 135, 9
	pages := Flow(root, opts)
	if len(pages) != 1 {
		t.Fatalf("pages=%d, want 1: the clipped flex descendants must not create fragments", len(pages))
	}
	var overflowing, after *PlacedBox
	for i := range pages[0].Blocks {
		block := &pages[0].Blocks[i]
		if strings.HasPrefix(block.Text, "FLEX CONTENT") {
			overflowing = block
		}
		if block.Text == "AFTER FLEX" {
			after = block
		}
	}
	if overflowing == nil || after == nil {
		t.Fatalf("Flex and following flow were not positioned: %+v", pages[0].Blocks)
	}
	if len(overflowing.ClipRects) != 1 || overflowing.ClipRects[0].Height != 30 {
		t.Errorf("clip of the item=%+v, want declared height of 30pt", overflowing.ClipRects)
	}
	if after.Y > overflowing.ClipRects[0].Y+30+20 {
		t.Errorf("following flow y=%g must follow the fixed box, clip=%+v", after.Y, overflowing.ClipRects[0])
	}
}
