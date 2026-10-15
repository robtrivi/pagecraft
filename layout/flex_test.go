package layout

import (
	"strings"
	"testing"

	"github.com/robtrivi/pagecraft/boxes"
)

func TestLayout_FlexRowUsesIntrinsicBasisAndDeclaredGrowth(t *testing.T) {
	tests := []struct {
		name  string
		html  string
		texts []string
		want  []float64
	}{
		{
			name:  "equal growth fills a row across three items",
			html:  `<div style="display:flex"><span style="flex:1">A</span><span style="flex:1">B</span><span style="flex:1">C</span></div>`,
			texts: []string{"A", "B", "C"},
			want:  []float64{451.0 / 3, 451.0 / 3, 451.0 / 3},
		},
		{
			name:  "automatic basis preserves distinct content widths",
			html:  `<div style="display:flex"><span>AAAA</span><span>B</span></div>`,
			texts: []string{"AAAA", "B"},
			want:  []float64{24, 6},
		},
		{
			name:  "flex one distributes positive free space by grow factor",
			html:  `<div style="display:flex"><span style="flex:1">A</span><span style="flex:2">B</span></div>`,
			texts: []string{"A", "B"},
			want:  []float64{451.0 / 3, 902.0 / 3},
		},
		{
			name:  "flex one zero basis overrides declared width",
			html:  `<div style="display:flex"><span style="flex:1;width:100pt">A</span><span style="flex:1">B</span></div>`,
			texts: []string{"A", "B"},
			want:  []float64{225.5, 225.5},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, err := boxes.Build(tt.html)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			blocks := Flow(root, testOptions())[0].Blocks
			if len(blocks) != len(tt.want) {
				t.Fatalf("items=%d, want %d", len(blocks), len(tt.want))
			}
			x := 72.0
			for i, block := range blocks {
				if !approx(block.X, x) || !approx(block.Width, tt.want[i]) {
					t.Errorf("item %d: x=%v width=%v, want x=%v width=%v", i, block.X, block.Width, x, tt.want[i])
				}
				if block.Y != 72 {
					t.Errorf("item %d y=%v, want all row items at y=72", i, block.Y)
				}
				if block.Text != tt.texts[i] {
					t.Errorf("item %d text=%q, want %q", i, block.Text, tt.texts[i])
				}
				x += tt.want[i]
			}
		})
	}
}

func TestLayout_FlexBasisResolvesLengthAndPercentageAgainstContentBox(t *testing.T) {
	root, err := boxes.Build(`<div style="display:flex;width:200pt;padding:10pt"><div style="flex:0 0 64pt;min-width:0">A</div><div style="flex:0 0 25%;min-width:0">B</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("items=%d, want 2", len(blocks))
	}
	for i, want := range []float64{64, 50} {
		if !approx(blocks[i].Width, want) {
			t.Errorf("item %d width=%v, want %v", i, blocks[i].Width, want)
		}
	}
	if !approx(blocks[1].X-blocks[0].X, 64) {
		t.Errorf("second item starts at %v; want 64pt after first", blocks[1].X-blocks[0].X)
	}
}

func TestLayout_FlexBasisZeroGrowthKeepsItemsInsideAvailableRow(t *testing.T) {
	root, err := boxes.Build(`<div style="display:flex;width:200pt"><div style="flex:1 1 0%;min-width:0">PRIMARY</div><div style="flex:0 1 64pt;min-width:0">DETAIL</div><div style="flex:1 1 0%;min-width:0">NOTE</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 3 {
		t.Fatalf("items=%d, want 3", len(blocks))
	}
	if blocks[0].Text != "PRIMARY" || blocks[1].Text != "DETAIL" || blocks[2].Text != "NOTE" {
		t.Fatalf("item text = %q, %q, %q", blocks[0].Text, blocks[1].Text, blocks[2].Text)
	}
	if !approx(blocks[1].Width, 64) {
		t.Errorf("fixed basis width=%v, want 64", blocks[1].Width)
	}
	end := blocks[2].X + blocks[2].Width
	if end > 72+200+1e-7 {
		t.Errorf("last item ends at %v, past container edge %v", end, 72+200)
	}
}

func TestLayout_FlexGrowthHonorsAutomaticMinimumSize(t *testing.T) {
	html := `<div style="display:flex;width:280pt;gap:6pt"><div style="flex:1 1 0%;padding:6pt;border:1pt solid;font-size:17pt;font-weight:bold">UNBREAKABLE-FLEX-CONTENT</div><div style="flex:1 1 0%;padding:6pt;border:1pt solid">A neighboring paragraph has several ordinary soft wrap opportunities.</div><div style="flex:0 1 64pt;padding:6pt;border:1pt solid">NOTE</div></div>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	blocks := Flow(root, opts)[0].Blocks
	if len(blocks) != 3 {
		t.Fatalf("items=%d, want 3", len(blocks))
	}
	minimum := flexItemMinContentWidth(root.Children[0].Children[0], opts, 280)
	if blocks[0].Width+1e-7 < minimum {
		t.Errorf("first flex item width=%v, below automatic minimum %v", blocks[0].Width, minimum)
	}
	if blocks[0].X+blocks[0].Width > blocks[1].X+1e-7 {
		t.Errorf("first item content box overlaps its sibling: end=%v next=%v", blocks[0].X+blocks[0].Width, blocks[1].X)
	}
	if blocks[2].Text != "NOTE" {
		t.Errorf("last flex item text=%q, want NOTE", blocks[2].Text)
	}
	if right := blocks[2].X + blocks[2].Width; right > 72+280+1e-7 {
		t.Errorf("last flex item ends at %v, past container edge %v", right, 72+280)
	}
}

func TestLayout_FlexShrinkUsesUnclampedBasesBeforeMaxSizing(t *testing.T) {
	root, err := boxes.Build(`<div style="display:flex;width:250pt"><div style="flex:1 1 300pt;max-width:200pt;min-width:0">A</div><div style="flex:1 1 100pt;min-width:0">B</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("items=%d, want 2", len(blocks))
	}
	for i, want := range []float64{187.5, 62.5} {
		if !approx(blocks[i].Width, want) {
			t.Errorf("item %d width=%v, want %v (shrink ratio uses flex bases 300:100)", i, blocks[i].Width, want)
		}
	}
}

func TestLayout_FlexBorderBoxFloorsContentSizeBeforeApplyingMaxWidth(t *testing.T) {
	root, err := boxes.Build(`<div style="display:flex;width:100pt"><div style="flex:0 0 0pt;box-sizing:border-box;padding:10pt;min-width:0;max-width:0">A</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("items=%d, want 1", len(blocks))
	}
	if !approx(blocks[0].Width, 20) {
		t.Errorf("border-box width=%v, want 20pt padding with zero content width", blocks[0].Width)
	}
}

func TestLayout_FlexGrowRedistributesSpaceAfterMaxWidth(t *testing.T) {
	tests := []struct {
		name string
		html string
		want []float64
	}{
		{
			name: "fixed maximum redistributes remaining free space",
			html: `<div style="display:flex;width:200pt"><div style="flex:1;max-width:60pt">LIMITED</div><div style="flex:1">OTHER</div></div>`,
			want: []float64{60, 140},
		},
		{
			name: "multiple maximums freeze and redistribute iteratively",
			html: `<div style="display:flex;width:300pt"><div style="flex:1;max-width:60pt">A</div><div style="flex:1;max-width:80pt">B</div><div style="flex:1">C</div></div>`,
			want: []float64{60, 80, 160},
		},
		{
			name: "percentage maximum uses the flex container content width",
			html: `<div style="display:flex;width:200pt"><div style="flex:1;max-width:25%">LIMITED</div><div style="flex:1">OTHER</div></div>`,
			want: []float64{50, 150},
		},
		{
			name: "maximum clamps intrinsic base without flex growth",
			html: `<div style="display:flex;width:200pt"><div style="max-width:30pt;font:10pt/12pt Helvetica;white-space:nowrap">XXXXXXXXXX</div><div>OTHER</div></div>`,
			want: []float64{30, 30},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, err := boxes.Build(tt.html)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			blocks := Flow(root, testOptions())[0].Blocks
			if len(blocks) != len(tt.want) {
				t.Fatalf("items=%d, want %d", len(blocks), len(tt.want))
			}
			for index, width := range tt.want {
				if !approx(blocks[index].Width, width) {
					t.Errorf("item %d width=%v, want %v", index, blocks[index].Width, width)
				}
			}
		})
	}
}

func TestLayout_FlexColumnStretchRespectsItemMaxWidth(t *testing.T) {
	root, err := boxes.Build(`<div style="display:flex;flex-direction:column;align-items:stretch;width:200pt"><div style="max-width:60pt">LIMITED</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	for _, block := range blocks {
		if block.Text == "LIMITED" {
			if !approx(block.Width, 60) {
				t.Fatalf("stretched item width=%v, want max-width 60", block.Width)
			}
			return
		}
	}
	t.Fatal("stretched flex item was not laid out")
}

func TestLayout_FlexRowContainerAppliesHeightConstraints(t *testing.T) {
	tests := []struct {
		name, style, itemHeight string
		want                    float64
	}{
		{name: "minimum height", style: "min-height:100pt", itemHeight: "20pt", want: 112},
		{name: "maximum height caps declared height", style: "height:100pt;max-height:60pt", itemHeight: "20pt", want: 72},
		{name: "maximum height caps content overflow box", style: "max-height:60pt", itemHeight: "80pt", want: 72},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := `<div style="display:flex;width:100pt;` + tt.style + `;padding:5pt;border:1pt solid #000;background:#eee"><div style="width:20pt;height:` + tt.itemHeight + `"></div></div><p style="margin:0">AFTER</p>`
			root, err := boxes.Build(html)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			var outer, after *PlacedBox
			for _, block := range Flow(root, testOptions())[0].Blocks {
				if block.HasBg {
					outer = &block
					if !approx(block.Height, tt.want) {
						t.Fatalf("flex container height=%v, want %v", block.Height, tt.want)
					}
				}
				if block.Text == "AFTER" {
					after = &block
				}
			}
			if outer == nil {
				t.Fatal("decorated flex container was not placed")
			}
			if after == nil {
				t.Fatal("following block was not placed")
			}
			if after.Y < outer.Y+outer.Height {
				t.Fatalf("following block y=%v, want at or after flex container bottom %v", after.Y, outer.Y+outer.Height)
			}
		})
	}
}

func TestLayout_FlexColumnContainerAppliesHeightConstraints(t *testing.T) {
	tests := []struct {
		name, style, itemHeight string
		want                    float64
	}{
		{name: "minimum height", style: "min-height:100pt", itemHeight: "20pt", want: 112},
		{name: "maximum height caps overflowing content", style: "max-height:60pt", itemHeight: "80pt", want: 72},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := `<div style="display:flex;flex-direction:column;width:100pt;` + tt.style + `;padding:5pt;border:1pt solid #000;background:#eee"><div style="height:` + tt.itemHeight + `"></div></div>`
			root, err := boxes.Build(html)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			for _, block := range Flow(root, testOptions())[0].Blocks {
				if block.HasBg {
					if !approx(block.Height, tt.want) {
						t.Fatalf("flex container height=%v, want %v", block.Height, tt.want)
					}
					return
				}
			}
			t.Fatal("decorated flex container was not placed")
		})
	}
}

func TestLayout_FlexRowShrinksItemsByDefaultUnlessDisabled(t *testing.T) {
	tests := []struct {
		name string
		html string
		want []float64
	}{
		{
			name: "default shrink fits percentage widths with padding",
			html: `<div style="display:flex"><div style="width:50%;padding:10pt">A</div><div style="width:50%;padding:10pt">B</div></div>`,
			want: []float64{225.5, 225.5},
		},
		{
			name: "shrink zero preserves the first track and shrinks its sibling",
			html: `<div style="display:flex"><div style="width:50%;padding:10pt;flex-shrink:0">A</div><div style="width:50%;padding:10pt">B</div></div>`,
			want: []float64{245.5, 205.5},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, err := boxes.Build(tt.html)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			blocks := Flow(root, testOptions())[0].Blocks
			if len(blocks) != len(tt.want) {
				t.Fatalf("items=%d, want %d", len(blocks), len(tt.want))
			}
			x := 72.0
			for i, block := range blocks {
				if !approx(block.X, x) || !approx(block.Width, tt.want[i]) {
					t.Errorf("item %d: x=%v width=%v, want x=%v width=%v", i, block.X, block.Width, x, tt.want[i])
				}
				x += tt.want[i]
			}
		})
	}
}

func TestLayout_FlexShrinkPreservesAutomaticMinContentWidth(t *testing.T) {
	root, err := boxes.Build(`<div style="display:flex;width:200pt"><div style="width:150pt">UNBREAKABLEIDENTIFIER</div><div style="width:150pt">B</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("items=%d, want 2", len(blocks))
	}
	if blocks[0].Width < 110 {
		t.Errorf("unbreakable item shrank below its min-content width: got %vpt, want at least 110pt", blocks[0].Width)
	}
	if blocks[1].Width >= 90 {
		t.Errorf("sibling should absorb remaining shrink: got %vpt, want below 90pt", blocks[1].Width)
	}
	if !approx(blocks[0].Width+blocks[1].Width, 200) {
		t.Errorf("total shrunk item width=%vpt, want container width 200pt", blocks[0].Width+blocks[1].Width)
	}
}

func TestLayout_FlexShrinkOverflowHiddenUsesZeroAutomaticMinimum(t *testing.T) {
	for _, scenario := range []struct {
		name, overflow string
		want           []float64
	}{
		{name: "shorthand hidden", overflow: "overflow:hidden", want: []float64{100, 100}},
		{name: "horizontal hidden", overflow: "overflow-x:hidden", want: []float64{100, 100}},
		{name: "vertical hidden computes horizontal auto", overflow: "overflow-x:visible;overflow-y:hidden", want: []float64{100, 100}},
		{name: "visible keeps content minimum", overflow: "overflow:visible", want: []float64{126, 74}},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			html := `<div style="display:flex;width:200pt"><div style="width:150pt;` + scenario.overflow + `">UNBREAKABLEIDENTIFIER</div><div style="width:150pt">B</div></div>`
			root, err := boxes.Build(html)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			blocks := Flow(root, testOptions())[0].Blocks
			if len(blocks) != len(scenario.want) {
				t.Fatalf("items=%d, want %d", len(blocks), len(scenario.want))
			}
			for index, block := range blocks {
				if !approx(block.Width, scenario.want[index]) {
					t.Errorf("item %d width=%vpt, want %vpt", index, block.Width, scenario.want[index])
				}
			}
		})
	}
}

func TestLayout_FlexOverflowHiddenClipsShrunkItemContent(t *testing.T) {
	root, err := boxes.Build(`<div style="display:flex;width:200pt"><div style="width:150pt;min-width:0;overflow:hidden">UNBREAKABLEIDENTIFIER</div><div style="width:150pt">B</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("items=%d, want 2", len(blocks))
	}
	if !approx(blocks[0].Width, 100) {
		t.Fatalf("overflow item width=%vpt, want 100pt", blocks[0].Width)
	}
	if len(blocks[0].ClipRects) != 1 {
		t.Fatalf("overflow item clips=%+v, want one clip rectangle", blocks[0].ClipRects)
	}
	clip := blocks[0].ClipRects[0]
	if !approx(clip.X, blocks[0].X) || !approx(clip.Width, blocks[0].Width) {
		t.Errorf("clip=%+v, want item padding box x=%v width=100pt", clip, blocks[0].X)
	}
}

func TestLayout_FlexOverflowVisibleAxisComputesToAutoAndClips(t *testing.T) {
	root, err := boxes.Build(`<div style="display:flex;width:200pt"><div style="width:150pt;min-width:0;overflow-x:visible;overflow-y:hidden">UNBREAKABLEIDENTIFIER</div><div style="width:150pt">B</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("items=%d, want 2", len(blocks))
	}
	if len(blocks[0].ClipRects) != 1 {
		t.Fatalf("overflow item clips=%+v, want a horizontal clip from computed overflow-x:auto", blocks[0].ClipRects)
	}
	clip := blocks[0].ClipRects[0]
	if !approx(clip.X, blocks[0].X) || !approx(clip.Width, blocks[0].Width) {
		t.Errorf("clip=%+v, want item padding box x=%v width=%vpt", clip, blocks[0].X, blocks[0].Width)
	}
}

func TestLayout_FlexShrinkNestedMinContentCountsInsetsOnce(t *testing.T) {
	root, err := boxes.Build(`<div style="display:flex;width:200pt"><section style="width:150pt;padding:5pt"><p style="padding:5pt;margin:0">UNBREAKABLEIDENTIFIER</p></section><div style="width:150pt">B</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 3 {
		t.Fatalf("blocks=%d, want flex item, nested block and sibling", len(blocks))
	}
	item, child := root.Children[0].Children[0], root.Children[0].Children[0].Children[0]
	words := buildInlineStream(child, testOptions())
	wantMinimum := words[0].w + horizontalInsets(child) + horizontalInsets(item)
	if !approx(blocks[0].Width, wantMinimum) {
		t.Errorf("nested flex item width=%vpt, want min-content %vpt without duplicate insets", blocks[0].Width, wantMinimum)
	}
	if !approx(blocks[0].Width+blocks[2].Width, 200) {
		t.Errorf("shrunk flex items total=%vpt, want container width 200pt", blocks[0].Width+blocks[2].Width)
	}
}

func TestLayout_FlexMinWidthZeroOverridesAutomaticMinContent(t *testing.T) {
	root, err := boxes.Build(`<div style="display:flex;width:200pt"><div style="width:150pt;min-width:0">UNBREAKABLEIDENTIFIER</div><div style="width:150pt">B</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("items=%d, want 2", len(blocks))
	}
	if !approx(blocks[0].Width, 100) || !approx(blocks[1].Width, 100) {
		t.Errorf("min-width:0 tracks=%v/%vpt, want 100/100pt", blocks[0].Width, blocks[1].Width)
	}
}

func TestLayout_FlexMinWidthPercentageUsesContainerWidth(t *testing.T) {
	root, err := boxes.Build(`<div style="display:flex;width:200pt"><div style="width:150pt;min-width:75%">A</div><div style="width:150pt">B</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("items=%d, want 2", len(blocks))
	}
	if !approx(blocks[0].Width, 150) || !approx(blocks[1].Width, 50) {
		t.Errorf("percentage min-width tracks=%v/%vpt, want 150/50pt", blocks[0].Width, blocks[1].Width)
	}
}

func TestLayout_FlexMinWidthRespectsBoxSizing(t *testing.T) {
	tests := []struct {
		name       string
		boxSizing  string
		firstWidth float64
		second     float64
	}{
		{name: "content-box includes padding", boxSizing: "content-box", firstWidth: 120, second: 80},
		{name: "border-box scales shrink by inner flex base", boxSizing: "border-box", firstWidth: 103.57142857142857, second: 96.42857142857143},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, err := boxes.Build(`<div style="display:flex;width:200pt"><div style="box-sizing:` + tt.boxSizing + `;width:150pt;padding:10pt;min-width:100pt">A</div><div style="width:150pt">B</div></div>`)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			blocks := Flow(root, testOptions())[0].Blocks
			if len(blocks) != 2 {
				t.Fatalf("items=%d, want 2", len(blocks))
			}
			if !approx(blocks[0].Width, tt.firstWidth) || !approx(blocks[1].Width, tt.second) {
				t.Errorf("tracks=%v/%vpt, want %v/%vpt", blocks[0].Width, blocks[1].Width, tt.firstWidth, tt.second)
			}
		})
	}
}

func TestLayout_FlexContainerInsetsPaintAndAdvanceFollowingBlock(t *testing.T) {
	root, err := boxes.Build(`<div style="display:flex;background:#ffeecc;border:2pt solid #123456;padding:8pt 10pt"><span>A</span></div><p style="margin:0">B</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 3 {
		t.Fatalf("blocks=%d, want box flex, item and following paragraph", len(blocks))
	}
	if !blocks[0].HasBg || blocks[0].X != 72 || blocks[0].Y != 72 || !approx(blocks[0].Height, 34.4) {
		t.Errorf("flex wrapper does not preserve outer background/height: %+v", blocks[0])
	}
	if len(blocks[1].Runs) != 1 || len(blocks[1].Runs[0]) != 1 {
		t.Fatalf("flex item runs: %#v", blocks[1].Runs)
	}
	if run := blocks[1].Runs[0][0]; run.Text != "A" || run.X != 84 || run.Y != 82 {
		t.Errorf("item flex must respect the container border and padding: %+v", run)
	}
	if len(blocks[2].Runs) != 1 || len(blocks[2].Runs[0]) != 1 || !approx(blocks[2].Runs[0][0].Y, 106.4) {
		t.Errorf("following block must advance past the outer height: %#v", blocks[2].Runs)
	}
}

func TestLayout_FlexColumnStacksItemsVertically(t *testing.T) {
	root, err := boxes.Build(`<div style="display:flex;flex-direction:column;row-gap:6pt"><span>A</span><span>B</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 || len(blocks[0].Runs) != 1 || len(blocks[1].Runs) != 1 {
		t.Fatalf("items flex column: %#v", blocks)
	}
	firstY := blocks[0].Runs[0][0].Y
	secondY := blocks[1].Runs[0][0].Y
	if !approx(secondY-firstY, 20.4) {
		t.Errorf("vertical spacing=%v, want line-height 14.4 + row-gap 6", secondY-firstY)
	}
}

func TestLayout_FlexColumnFollowingBlockStartsAfterLastItemAndGap(t *testing.T) {
	root, err := boxes.Build(`<div style="display:flex;flex-direction:column;row-gap:6pt"><div>FIRST</div><div>SECOND</div></div><p style="margin:0">AFTER</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	var lastItemBottom, followingY float64
	for _, block := range blocks {
		if block.Text == "SECOND" && len(block.Runs) > 0 {
			lastItemBottom = block.Runs[len(block.Runs)-1][0].Y + block.Runs[len(block.Runs)-1][0].LineH
		}
		if block.Text == "AFTER" && len(block.Runs) > 0 {
			followingY = block.Runs[0][0].Y
		}
	}
	if lastItemBottom == 0 || followingY == 0 {
		t.Fatalf("last item and following block were not found: %#v", blocks)
	}
	if !approx(followingY, lastItemBottom) {
		t.Errorf("following block starts at y=%v; last item ends at y=%v", followingY, lastItemBottom)
	}
}

func TestLayout_FlexColumnSpaceBetweenUsesDeclaredHeight(t *testing.T) {
	root, err := boxes.Build(`<div style="display:flex;flex-direction:column;justify-content:space-between;height:100pt"><span>A</span><span>B</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 || len(blocks[0].Runs) != 1 || len(blocks[1].Runs) != 1 {
		t.Fatalf("items flex column: %#v", blocks)
	}
	if got := blocks[1].Runs[0][0].Y - blocks[0].Runs[0][0].Y; !approx(got, 85.6) {
		t.Errorf("spacing between items=%v, want 85.6 in fixed height with space-between", got)
	}
}

func TestLayout_FlexWrapsTallItemAcrossPages(t *testing.T) {
	content := "A1 alpha A2 bravo A3 charlie A4 delta A5 echo A6 foxtrot A7 golf A8 hotel A9 india A10 juliet A11 kilo A12 lima A13 mike A14 november A15 oscar A16 papa A17 quebec A18 romeo A19 sierra A20 tango A21 uniform A22 victor A23 whiskey A24 xray A25 yankee A26 zulu"
	root, err := boxes.Build(`<div style="height:75pt">BEFORE</div><div style="display:flex"><div style="width:150pt;font-size:9pt;line-height:10.5pt">` + content + `</div></div><p style="margin:0">AFTER FLEX</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 225, 135, 9
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Errorf("pages=%d, want 2 as in Chromium", len(pages))
	}
	itemPages := make(map[int]bool)
	var rendered strings.Builder
	afterPage := 0
	for pageIndex, page := range pages {
		for _, block := range page.Blocks {
			if block.Width == 150 && block.FontSize == 9 && len(block.Runs) > 0 {
				itemPages[pageIndex+1] = true
				for _, line := range block.Runs {
					for _, run := range line {
						rendered.WriteString(run.Text)
						rendered.WriteByte(' ')
					}
				}
			}
			if block.Text == "AFTER FLEX" {
				afterPage = pageIndex + 1
			}
		}
	}
	if !itemPages[1] || !itemPages[2] {
		t.Errorf("the tall item must start on page 1 and continue on page 2; pages=%v, layout=%+v", itemPages, pages)
	}
	renderedTokens := strings.Fields(rendered.String())
	if got, want := strings.Join(renderedTokens, " "), strings.Join(strings.Fields(content), " "); got != want {
		t.Errorf("fragmented runs lost or repeated text:\n got %q\nwant %q", got, want)
	}
	if afterPage != 2 {
		t.Errorf("AFTER FLEX appears on page %d; must follow the fragmented content on page 2", afterPage)
	}
}

func TestLayout_FlexColumnContinuesTallItemAcrossPages(t *testing.T) {
	content := "A1 alpha A2 bravo A3 charlie A4 delta A5 echo A6 foxtrot A7 golf A8 hotel A9 india A10 juliet A11 kilo A12 lima A13 mike A14 november A15 oscar A16 papa A17 quebec A18 romeo A19 sierra A20 tango A21 uniform A22 victor A23 whiskey A24 xray A25 yankee A26 zulu"
	root, err := boxes.Build(`<div style="height:65pt">BEFORE</div><div style="display:flex;flex-direction:column"><div style="width:180pt;font-size:9pt;line-height:10.5pt">` + content + `</div></div><p style="margin:0">AFTER FLEX</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 225, 135, 9
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Errorf("pages=%d, want 2 as in Chromium", len(pages))
	}
	itemPages := make(map[int]bool)
	var rendered strings.Builder
	afterPage := 0
	for pageIndex, page := range pages {
		for _, block := range page.Blocks {
			if block.FontSize == 9 && len(block.Runs) > 0 {
				itemPages[pageIndex+1] = true
				for _, line := range block.Runs {
					for _, run := range line {
						rendered.WriteString(run.Text)
						rendered.WriteByte(' ')
					}
				}
			}
			if block.Text == "AFTER FLEX" {
				afterPage = pageIndex + 1
			}
		}
	}
	if !itemPages[1] || !itemPages[2] {
		t.Errorf("the tall item must start on page 1 and continue on page 2; pages=%v, layout=%+v", itemPages, pages)
	}
	if got, want := strings.Join(strings.Fields(rendered.String()), " "), strings.Join(strings.Fields(content), " "); got != want {
		t.Errorf("fragmented runs lost or repeated text:\n got %q\nwant %q", got, want)
	}
	if afterPage != 2 {
		t.Errorf("AFTER FLEX appears on page %d; must follow the fragmented content on page 2", afterPage)
	}
}

func TestLayout_FlexColumnContinuesBlockifiedInlineItemAcrossPages(t *testing.T) {
	content := "I1 alpha I2 bravo I3 charlie I4 delta I5 echo I6 foxtrot I7 golf I8 hotel I9 india I10 juliet I11 kilo I12 lima I13 mike I14 november I15 oscar I16 papa I17 quebec I18 romeo I19 sierra I20 tango I21 uniform I22 victor I23 whiskey I24 xray"
	root, err := boxes.Build(`<div style="height:65pt">BEFORE</div><div style="display:flex;flex-direction:column"><span style="width:180pt;font-size:9pt;line-height:10.5pt">` + content + `</span></div><p style="margin:0">AFTER FLEX</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 225, 135, 9
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Errorf("pages=%d, want 2", len(pages))
	}
	itemPages := make(map[int]bool)
	var rendered strings.Builder
	afterPage := 0
	for pageIndex, page := range pages {
		for _, block := range page.Blocks {
			if block.FontSize == 9 {
				itemPages[pageIndex+1] = true
				for _, line := range block.Runs {
					for _, run := range line {
						rendered.WriteString(run.Text)
						rendered.WriteByte(' ')
					}
				}
			}
			if block.Text == "AFTER FLEX" {
				afterPage = pageIndex + 1
			}
		}
	}
	if !itemPages[1] || !itemPages[2] {
		t.Errorf("the span converted to a flex item must continue across both pages; pages=%v", itemPages)
	}
	if got, want := strings.Join(strings.Fields(rendered.String()), " "), strings.Join(strings.Fields(content), " "); got != want {
		t.Errorf("the flex item inline was lost or duplicated text:\n got %q\nwant %q", got, want)
	}
	if afterPage != 2 {
		t.Errorf("AFTER FLEX appears on page %d; want 2", afterPage)
	}
}

func TestLayout_FlexColumnDecorationContinuesWithItems(t *testing.T) {
	content := strings.Repeat("line ", 120)
	root, err := boxes.Build(`<div style="display:flex;flex-direction:column;background:#eee;border:1pt solid #111"><div style="font-size:9pt;line-height:10.5pt">` + content + `</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 225, 135, 9
	pages := Flow(root, opts)
	if len(pages) < 2 {
		t.Fatalf("pages=%d, want at least 2", len(pages))
	}
	for pageIndex, page := range pages {
		decorated := false
		for _, block := range page.Blocks {
			if block.HasBg {
				decorated = true
				break
			}
		}
		if !decorated {
			t.Errorf("Flex column decoration is missing on page %d", pageIndex+1)
		}
	}
}

func TestLayout_FlexColumnContinuesNestedBlockFlowAcrossPages(t *testing.T) {
	content := `P1 alpha P2 bravo P3 charlie P4 delta P5 echo P6 foxtrot P7 golf P8 hotel P9 india P10 juliet P11 kilo P12 lima P13 mike P14 november P15 oscar P16 papa P17 quebec P18 romeo P19 sierra P20 tango P21 uniform P22 victor P23 whiskey P24 xray`
	root, err := boxes.Build(`<div style="height:65pt">BEFORE</div><div style="display:flex;flex-direction:column"><section style="width:180pt;font-size:9pt;line-height:10.5pt"><p style="margin:0">` + content + `</p></section></div><p style="margin:0">AFTER FLEX</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 225, 135, 9
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Errorf("pages=%d, want 2", len(pages))
	}
	itemPages := make(map[int]bool)
	var rendered strings.Builder
	afterPage := 0
	for pageIndex, page := range pages {
		for _, block := range page.Blocks {
			if block.FontSize != 9 {
				if block.Text == "AFTER FLEX" {
					afterPage = pageIndex + 1
				}
				continue
			}
			for _, line := range block.Runs {
				for _, run := range line {
					if strings.HasPrefix(run.Text, "P") {
						itemPages[pageIndex+1] = true
					}
					rendered.WriteString(run.Text)
					rendered.WriteByte(' ')
				}
			}
		}
	}
	if !itemPages[1] || !itemPages[2] {
		t.Errorf("the nested block must flow across both pages; pages=%v", itemPages)
	}
	if got, want := strings.Join(strings.Fields(rendered.String()), " "), strings.Join(strings.Fields(content), " "); got != want {
		t.Errorf("nested flow lost or duplicated text:\n got %q\nwant %q", got, want)
	}
	if afterPage != 2 {
		t.Errorf("AFTER FLEX appears on page %d; want 2", afterPage)
	}
}

func TestLayout_FlexItemWithBlockChildrenContinuesAtContentBreaks(t *testing.T) {
	content := `<div style="height:65pt">BEFORE</div><div style="display:flex;background:#eee;border:1pt solid #222"><section style="width:180pt;padding:4pt;border:1pt solid #123456;background:#eaf3ff;font-size:9pt;line-height:10.5pt"><p style="margin:0 0 3pt">P1: A1 alpha A2 bravo A3 charlie A4 delta A5 echo A6 foxtrot A7 golf A8 hotel A9 india A10 juliet A11 kilo A12 lima A13 mike</p><p style="margin:0">P2: A14 november A15 oscar A16 papa A17 quebec A18 romeo A19 sierra A20 tango A21 uniform A22 victor A23 whiskey A24 xray A25 yankee A26 zulu</p></section><div style="width:24pt;font-size:9pt;line-height:10.5pt">SIDE</div></div><p style="margin:0">AFTER FLEX</p>`
	root, err := boxes.Build(content)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 225, 135, 9
	pages := Flow(root, opts)
	pageText := make([]string, len(pages))
	for pageIndex, page := range pages {
		var text strings.Builder
		for _, block := range page.Blocks {
			if len(block.Runs) == 0 {
				text.WriteString(block.Text)
			}
			for _, line := range block.Runs {
				for _, run := range line {
					text.WriteString(run.Text)
					text.WriteByte(' ')
				}
			}
		}
		pageText[pageIndex] = text.String()
	}
	if len(pages) != 2 {
		t.Fatalf("pages=%d, want 2 as in Chromium; text by page=%q", len(pages), pageText)
	}
	if !strings.Contains(pageText[0], "P1:") || !strings.Contains(pageText[0], "SIDE") {
		t.Error("content in a Flex item with block children must start in the remaining space on page 1")
	}
	if !strings.Contains(pageText[1], "P2:") || !strings.Contains(pageText[1], "AFTER FLEX") {
		t.Errorf("item continuation and following block must stay ordered on page 2: %q", pageText[1])
	}
	if strings.Contains(pageText[1], "SIDE") {
		t.Errorf("short Flex item must be preserved only in the fragment where it fits: %q", pageText[1])
	}
	sectionFragments := make([]PlacedBox, 0, 2)
	for _, page := range pages {
		for _, block := range page.Blocks {
			if block.source != nil && block.source.Tag == "section" {
				sectionFragments = append(sectionFragments, block)
			}
		}
	}
	if len(sectionFragments) != 2 {
		t.Fatalf("decorated box fragments=%d, want 2", len(sectionFragments))
	}
	if !sectionFragments[0].HasBg || sectionFragments[0].BorderTop.Width != 1 || sectionFragments[0].BorderBottom.Width != 0 ||
		!sectionFragments[1].HasBg || sectionFragments[1].BorderTop.Width != 0 || sectionFragments[1].BorderBottom.Width != 1 {
		t.Errorf("slice decoration must preserve background and borders only at the edges: first=%+v last=%+v", sectionFragments[0], sectionFragments[1])
	}
	containerFragments := make([]PlacedBox, 0, 2)
	for _, page := range pages {
		for _, block := range page.Blocks {
			if block.source != nil && block.source.Tag == "div" && block.HasBg {
				containerFragments = append(containerFragments, block)
			}
		}
	}
	if len(containerFragments) != 2 || containerFragments[0].BorderTop.Width != 1 || containerFragments[0].BorderBottom.Width != 0 ||
		containerFragments[1].BorderTop.Width != 0 || containerFragments[1].BorderBottom.Width != 1 {
		t.Errorf("the flex container must also fragment its own decoration: %+v", containerFragments)
	}
	var side PlacedBox
	for _, block := range pages[0].Blocks {
		if block.Text == "SIDE" {
			side = block
		}
	}
	if !approx(side.Height, sectionFragments[0].Height) {
		t.Errorf("stretch item should fill the Flex row fragment height: SIDE=%.2f, section=%.2f", side.Height, sectionFragments[0].Height)
	}
	joined := strings.Join(pageText, " ")
	gotTokens := make(map[string]int)
	for _, token := range strings.Fields(joined) {
		gotTokens[strings.Trim(token, ":,.;")]++
	}
	for _, token := range strings.Fields("A1 A2 A3 A4 A5 A6 A7 A8 A9 A10 A11 A12 A13 A14 A15 A16 A17 A18 A19 A20 A21 A22 A23 A24 A25 A26") {
		if got := gotTokens[token]; got != 1 {
			t.Errorf("the token %q appears %d times across the fragmented rows, want 1", token, got)
		}
	}
}

func TestLayout_FlexSubtreeAlignsEachFragmentainer(t *testing.T) {
	for _, test := range []struct {
		align string
		want  func(section, item PlacedBox) float64
	}{
		{align: "center", want: func(section, item PlacedBox) float64 { return section.Y + (section.Height-item.Height)/2 }},
		{align: "flex-end", want: func(section, item PlacedBox) float64 { return section.Y + section.Height - item.Height }},
	} {
		t.Run(test.align, func(t *testing.T) {
			html := `<div style="height:65pt">BEFORE</div><div style="display:flex;align-items:` + test.align + `"><section style="width:180pt;padding:4pt;border:1pt solid #123456;font-size:9pt;line-height:10.5pt"><p style="margin:0">A1 alpha A2 bravo A3 charlie A4 delta A5 echo A6 foxtrot A7 golf A8 hotel A9 india A10 juliet A11 kilo A12 lima A13 mike A14 november A15 oscar A16 papa A17 quebec A18 romeo A19 sierra A20 tango A21 uniform A22 victor A23 whiskey A24 xray A25 yankee A26 zulu</p></section><div style="width:24pt;font-size:9pt;line-height:10.5pt">SIDE</div></div>`
			root, err := boxes.Build(html)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			opts := testOptions()
			opts.PageWidth, opts.PageHeight, opts.Margin = 225, 135, 9
			pages := Flow(root, opts)
			if len(pages) != 2 {
				t.Fatalf("pages=%d, want 2 with subtree continuation", len(pages))
			}
			var section, side PlacedBox
			for _, block := range pages[0].Blocks {
				if block.source != nil && block.source.Tag == "section" {
					section = block
				}
				if block.Text == "SIDE" {
					side = block
				}
			}
			if section.source == nil || side.source == nil {
				t.Fatalf("the flex item must start in the remaining space on page 1: section=%+v SIDE=%+v", section.source, side.source)
			}
			if !approx(side.Y, test.want(section, side)) {
				t.Errorf("short item does not honor align-items:%s inside the fragment: section y=%.2f h=%.2f; SIDE y=%.2f h=%.2f", test.align, section.Y, section.Height, side.Y, side.Height)
			}
		})
	}
}

func TestLayout_FlexWrapsTallItemAcrossPagesWithAlignment(t *testing.T) {
	content := "A1 alpha A2 bravo A3 charlie A4 delta A5 echo A6 foxtrot A7 golf A8 hotel A9 india A10 juliet A11 kilo A12 lima A13 mike A14 november A15 oscar A16 papa A17 quebec A18 romeo A19 sierra A20 tango A21 uniform A22 victor A23 whiskey A24 xray A25 yankee A26 zulu"
	for _, align := range []string{"center", "flex-end", "baseline"} {
		t.Run(align, func(t *testing.T) {
			root, err := boxes.Build(`<div style="height:75pt">BEFORE</div><div style="display:flex;align-items:` + align + `"><div style="width:150pt;font-size:9pt;line-height:10.5pt">` + content + `</div></div><p style="margin:0">AFTER FLEX</p>`)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			opts := testOptions()
			opts.PageWidth, opts.PageHeight, opts.Margin = 225, 135, 9
			pages := Flow(root, opts)
			if len(pages) != 2 {
				t.Fatalf("pages=%d, want 2", len(pages))
			}
			itemPages := make(map[int]bool)
			var rendered strings.Builder
			afterPage := 0
			for pageIndex, page := range pages {
				for _, block := range page.Blocks {
					if block.Width == 150 && block.FontSize == 9 && len(block.Runs) > 0 {
						itemPages[pageIndex+1] = true
						for _, line := range block.Runs {
							for _, run := range line {
								rendered.WriteString(run.Text)
								rendered.WriteByte(' ')
							}
						}
					}
					if block.Text == "AFTER FLEX" {
						afterPage = pageIndex + 1
					}
				}
			}
			if !itemPages[1] || !itemPages[2] {
				t.Errorf("the item must start on page 1 and continue on page 2; pages=%v", itemPages)
			}
			if got, want := strings.Join(strings.Fields(rendered.String()), " "), strings.Join(strings.Fields(content), " "); got != want {
				t.Errorf("fragmented runs lost or repeated text: got %q want %q", got, want)
			}
			if afterPage != 2 {
				t.Errorf("AFTER FLEX appears on page %d; must follow the fragmented content on page 2", afterPage)
			}
		})
	}
}

func TestLayout_FlexWrapsDecoratedTallItemAcrossPages(t *testing.T) {
	content := "A1 alpha A2 bravo A3 charlie A4 delta A5 echo A6 foxtrot A7 golf A8 hotel A9 india A10 juliet A11 kilo A12 lima A13 mike A14 november A15 oscar A16 papa A17 quebec A18 romeo A19 sierra A20 tango A21 uniform A22 victor A23 whiskey A24 xray A25 yankee A26 zulu"
	root, err := boxes.Build(`<div style="height:75pt">BEFORE</div><div style="display:flex"><div style="width:150pt;padding:4pt;border:1pt solid #123456;background:#eaf3ff;font-size:9pt;line-height:10.5pt">` + content + `</div></div><p style="margin:0">AFTER FLEX</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 225, 135, 9
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages=%d, want 2", len(pages))
	}
	itemPages := make(map[int]bool)
	fragments := make(map[int]PlacedBox)
	var rendered strings.Builder
	afterPage := 0
	for pageIndex, page := range pages {
		for _, block := range page.Blocks {
			if block.FontSize == 9 && len(block.Runs) > 0 {
				itemPages[pageIndex+1] = true
				fragments[pageIndex+1] = block
				for _, line := range block.Runs {
					for _, run := range line {
						rendered.WriteString(run.Text)
						rendered.WriteByte(' ')
					}
				}
			}
			if block.Text == "AFTER FLEX" {
				afterPage = pageIndex + 1
			}
		}
	}
	if !itemPages[1] || !itemPages[2] {
		t.Errorf("decorated item must start on page 1 and continue on page 2; pages=%v", itemPages)
	}
	if len(fragments) != 2 {
		t.Fatalf("decorated fragments=%d, want one on each page", len(fragments))
	}
	first, continuation := fragments[1], fragments[2]
	if !first.HasBg || !continuation.HasBg || first.BorderTop.Width != 1 || first.BorderBottom.Width != 0 || continuation.BorderTop.Width != 0 || continuation.BorderBottom.Width != 1 {
		t.Errorf("background and borders at the break are incorrect: first=%+v continuation=%+v", first, continuation)
	}
	if first.BorderLeft.Width != 1 || first.BorderRight.Width != 1 || continuation.BorderLeft.Width != 1 || continuation.BorderRight.Width != 1 {
		t.Errorf("side borders must continue across both fragments: first=%+v continuation=%+v", first, continuation)
	}
	if got, want := strings.Join(strings.Fields(rendered.String()), " "), strings.Join(strings.Fields(content), " "); got != want {
		t.Errorf("fragmented runs lost or repeated text: got %q want %q", got, want)
	}
	if afterPage != 2 {
		t.Errorf("AFTER FLEX appears on page %d; must follow the content on page 2", afterPage)
	}
}

func TestLayout_FlexWrapsTextBesideImageAcrossPages(t *testing.T) {
	content := "A1 alpha A2 bravo A3 charlie A4 delta A5 echo A6 foxtrot A7 golf A8 hotel A9 india A10 juliet A11 kilo A12 lima A13 mike A14 november A15 oscar A16 papa A17 quebec A18 romeo A19 sierra A20 tango A21 uniform A22 victor A23 whiskey A24 xray A25 yankee A26 zulu"
	root, err := boxes.Build(`<div style="height:75pt">BEFORE</div><div style="display:flex;align-items:flex-start"><img style="width:24pt;height:24pt" src="` + imgURI(t, 32, 32) + `"><div style="width:150pt;font-size:9pt;line-height:10.5pt">` + content + `</div></div><p style="margin:0">AFTER FLEX</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 225, 135, 9
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages=%d, want 2", len(pages))
	}
	imagePages := 0
	itemPages := make(map[int]bool)
	textX := make(map[int]float64)
	imageX, imageWidth := 0.0, 0.0
	var rendered strings.Builder
	afterPage := 0
	for pageIndex, page := range pages {
		for _, block := range page.Blocks {
			if block.Image != nil {
				imagePages++
				imageX, imageWidth = block.X, block.Width
				if pageIndex != 0 {
					t.Errorf("image repeated on page %d", pageIndex+1)
				}
			}
			if block.FontSize == 9 && len(block.Runs) > 0 {
				itemPages[pageIndex+1] = true
				textX[pageIndex+1] = block.X
				for _, line := range block.Runs {
					for _, run := range line {
						rendered.WriteString(run.Text)
						rendered.WriteByte(' ')
					}
				}
			}
			if block.Text == "AFTER FLEX" {
				afterPage = pageIndex + 1
			}
		}
	}
	if imagePages != 1 {
		t.Errorf("image must appear once in the first fragment; occurrences=%d", imagePages)
	}
	if !approx(textX[1], imageX+imageWidth) || !approx(textX[2], textX[1]) {
		t.Errorf("text must remain in the column next to the image on both pages; x image=%v width=%v text=%v/%v", imageX, imageWidth, textX[1], textX[2])
	}
	if !itemPages[1] || !itemPages[2] {
		t.Errorf("Flex text must fragment across both pages; pages=%v", itemPages)
	}
	if got, want := strings.Join(strings.Fields(rendered.String()), " "), strings.Join(strings.Fields(content), " "); got != want {
		t.Errorf("fragmented runs lost or repeated text: got %q want %q", got, want)
	}
	if afterPage != 2 {
		t.Errorf("AFTER FLEX appears on page %d; must follow the content on page 2", afterPage)
	}
}

func TestLayout_FlexMovesOversizedImageItemIntactToNextPage(t *testing.T) {
	root, err := boxes.Build(`<div style="height:75pt">BEFORE</div><div style="display:flex"><img src="` + imgURI(t, 200, 100) + `"></div><p style="margin:0">AFTER FLEX</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 225, 135, 9
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages=%d, want 2", len(pages))
	}
	imagePages, afterPage := 0, 0
	for pageIndex, page := range pages {
		for _, block := range page.Blocks {
			if block.Image != nil {
				imagePages++
				if pageIndex != 1 || !approx(block.Height, 75) {
					t.Errorf("image must move intact to page 2; page=%d height=%v", pageIndex+1, block.Height)
				}
			}
			if block.Text == "AFTER FLEX" {
				afterPage = pageIndex + 1
			}
		}
	}
	if imagePages != 1 || afterPage != 2 {
		t.Errorf("image appears %d times; AFTER FLEX is on page %d. Want the image and following content on page 2", imagePages, afterPage)
	}
}

func TestLayout_FlexBreakInsideAvoidMovesItemThatFitsFreshPage(t *testing.T) {
	content := "A1 alpha A2 bravo A3 charlie A4 delta A5 echo A6 foxtrot A7 golf A8 hotel A9 india A10 juliet A11 kilo A12"
	root, err := boxes.Build(`<div style="height:90pt">BEFORE</div><div style="display:flex"><div style="width:150pt;font:9pt/10.5pt Helvetica;break-inside:avoid">` + content + `</div><div style="width:50pt">SIDE</div></div><p>AFTER FLEX</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 225, 135, 9
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages=%d, want 2", len(pages))
	}
	var itemPages []int
	afterPage := 0
	sidePage := 0
	var itemText strings.Builder
	for pageIndex, page := range pages {
		for _, block := range page.Blocks {
			if strings.Contains(block.Text, "A1") {
				itemPages = append(itemPages, pageIndex+1)
				itemText.WriteString(block.Text)
			}
			if strings.Contains(block.Text, "AFTER FLEX") {
				afterPage = pageIndex + 1
			}
			if strings.Contains(block.Text, "SIDE") {
				sidePage = pageIndex + 1
			}
		}
	}
	if len(itemPages) == 0 || itemPages[0] != 2 {
		t.Errorf("item with break-inside:avoid starts on page %v; it must move intact to page 2", itemPages)
	}
	if got, want := strings.Join(strings.Fields(itemText.String()), " "), content; got != want {
		t.Errorf("item with avoid was lost or duplicated text: got %q want %q", got, want)
	}
	if afterPage != 2 {
		t.Errorf("AFTER FLEX appears on page %d; must follow the completed item on page 2", afterPage)
	}
	if sidePage != 2 {
		t.Errorf("the SIDE item in the Flex row appears on page %d; must remain with the row on page 2", sidePage)
	}
}

func TestLayout_FlexItemPreservesBlockChildStyle(t *testing.T) {
	root, err := boxes.Build(`<div style="display:flex"><div><strong style="display:block;font-size:18pt;color:#0369a1">Value</strong>label</div></div>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) < 3 {
		t.Fatalf("Flex item and descendants: got %d boxes, want at least 3", len(blocks))
	}
	var strong, label TextRun
	for _, block := range blocks {
		for _, line := range block.Runs {
			for _, run := range line {
				switch run.Text {
				case "Value":
					strong = run
				case "label":
					label = run
				}
			}
		}
	}
	if strong.Text != "Value" || !strong.Bold || strong.FontSize != 18 || !strong.HasColor || strong.Color != [4]float64{0.011764705882352941, 0.4117647058823529, 0.6313725490196078, 1} {
		t.Errorf("strong lost its style: %+v", strong)
	}
	if label.Text != "label" || label.Bold || label.FontSize >= strong.FontSize {
		t.Errorf("label incorrectly inherited block styling: %+v", label)
	}
}

func TestLayout_FlexItemPreservesNestedBlockComposition(t *testing.T) {
	tests := []struct {
		name, html  string
		backgrounds int
		text        string
	}{
		{
			name:        "decorated heading and paragraph",
			html:        `<div style="display:flex"><section style="width:180pt;padding:4pt;background-color:#eeeeee"><h2 style="background-color:#ddeeff;font-size:14pt">Heading</h2><p style="background-color:#fef3c7;margin:0">Body</p></section></div>`,
			backgrounds: 3, text: "Heading Body",
		},
		{
			name:        "empty progress track and fill",
			html:        `<div style="display:flex"><section style="width:180pt"><div style="height:8pt;background-color:#ddeeff"><i style="display:block;width:50%;height:8pt;background-color:#2563eb"></i></div></section></div>`,
			backgrounds: 2,
		},
		{
			name:        "nested grid cards",
			html:        `<div style="display:flex"><section style="width:180pt"><div style="display:grid;grid-template-columns:1fr 1fr"><div style="background-color:#fee2e2">A</div><div style="background-color:#dbeafe">B</div></div></section></div>`,
			backgrounds: 2, text: "A B",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, err := boxes.Build(tt.html)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			blocks := Flow(root, testOptions())[0].Blocks
			backgrounds := 0
			var text strings.Builder
			for _, block := range blocks {
				if block.HasBg {
					backgrounds++
				}
				text.WriteString(block.Text)
				for _, line := range block.Runs {
					for _, run := range line {
						text.WriteString(run.Text)
						text.WriteByte(' ')
					}
				}
			}
			if backgrounds < tt.backgrounds {
				t.Errorf("boxes with background=%d, want at least %d; blocks=%#v", backgrounds, tt.backgrounds, blocks)
			}
			if tt.text != "" && !strings.Contains(strings.Join(strings.Fields(text.String()), " "), tt.text) {
				t.Errorf("subtree text was lost: got %q, want %q", text.String(), tt.text)
			}
		})
	}
}

func TestLayout_FlexItemPreservesNestedGridContext(t *testing.T) {
	root, err := boxes.Build(`<div style="display:flex"><div style="display:grid;width:120pt;grid-template-columns:50pt 70pt"><span>ALPHA</span><span>BETA</span></div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	runs := map[string]TextRun{}
	for _, block := range blocks {
		for _, line := range block.Runs {
			for _, run := range line {
				runs[run.Text] = run
			}
		}
	}
	alpha, hasAlpha := runs["ALPHA"]
	beta, hasBeta := runs["BETA"]
	if !hasAlpha || !hasBeta {
		t.Fatalf("grid item content missing: ALPHA=%t BETA=%t", hasAlpha, hasBeta)
	}
	if got := beta.X - alpha.X; !approx(got, 50) {
		t.Errorf("grid column offset=%gpt, want 50pt", got)
	}
}

func TestLayout_FlexItemPreservesNestedMulticolumnContext(t *testing.T) {
	root, err := boxes.Build(`<div style="display:flex"><div style="width:120pt;column-count:2;column-gap:0"><p style="margin:0">A</p><p style="margin:0">B</p><p style="margin:0">C</p><p style="margin:0">D</p></div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	positions := map[string]TextRun{}
	for _, block := range blocks {
		for _, line := range block.Runs {
			for _, run := range line {
				positions[run.Text] = run
			}
		}
	}
	for _, word := range []string{"A", "B", "C", "D"} {
		if _, ok := positions[word]; !ok {
			t.Fatalf("multicolumn content %q missing; got %v", word, positions)
		}
	}
	if positions["A"].X != positions["B"].X || positions["C"].X != positions["D"].X || positions["C"].X <= positions["A"].X {
		t.Errorf("nested columns not preserved: A=%+v B=%+v C=%+v D=%+v", positions["A"], positions["B"], positions["C"], positions["D"])
	}
}

func TestLayout_FlexWrapPlacesPercentageItemsOnMultipleRows(t *testing.T) {
	root, err := boxes.Build(`<div style="display:flex; flex-wrap:wrap; gap:8pt"><div style="width:28%;padding:8pt;border:1pt solid #94a3b8">One</div><div style="width:28%;padding:8pt;border:1pt solid #94a3b8">Two</div><div style="width:28%;padding:8pt;border:1pt solid #94a3b8">Three</div><div style="width:28%;padding:8pt;border:1pt solid #94a3b8">Four</div><div style="width:28%;padding:8pt;border:1pt solid #94a3b8">Five</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 5 {
		t.Fatalf("items: got %d, want 5", len(blocks))
	}
	if blocks[0].Y != blocks[1].Y || blocks[1].Y != blocks[2].Y || blocks[3].Y <= blocks[2].Y || blocks[3].Y != blocks[4].Y {
		t.Errorf("wrap: ys=%v,%v,%v,%v,%v; want rows 3+2", blocks[0].Y, blocks[1].Y, blocks[2].Y, blocks[3].Y, blocks[4].Y)
	}
	for i, block := range blocks {
		if !approx(block.Width, 144.28) {
			t.Errorf("item %d width=%v, want 144.28", i, block.Width)
		}
	}
}

func TestLayout_FlexWrapResolvesFlexibleSpacePerLine(t *testing.T) {
	root, err := boxes.Build(`<div style="display:flex;flex-wrap:wrap;width:200pt"><div style="flex:1 1 120pt;min-width:0">A</div><div style="flex:1 1 120pt;min-width:0">B</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("items=%d, want 2", len(blocks))
	}
	if blocks[0].Y == blocks[1].Y {
		t.Fatalf("both items stayed on one line at y=%v", blocks[0].Y)
	}
	for i, block := range blocks {
		if !approx(block.Width, 200) {
			t.Errorf("item %d width=%v, want its row's 200pt available width", i, block.Width)
		}
	}
}

func TestLayout_FlexExplicitHeightsControlCrossAxisAlignment(t *testing.T) {
	root, err := boxes.Build(`<div style="display:flex;align-items:center;height:80pt"><div style="height:60pt">tall</div><div style="height:20pt">low</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("items: got %d, want 2", len(blocks))
	}
	if blocks[0].Height != 60 || blocks[1].Height != 20 || blocks[0].Y != 82 || blocks[1].Y != 102 {
		t.Errorf("geometry flex: tall=(y=%v,h%v), short=(y=%v,h%v), want tall=(82,60) short=(102,20)", blocks[0].Y, blocks[0].Height, blocks[1].Y, blocks[1].Height)
	}
}

func TestLayout_FlexAlignItemsPositionsShortItemInTallRow(t *testing.T) {
	cases := []struct {
		name, alignItems string
		shortY, shortH   float64
	}{
		{"center", "center", 79.2, 14.4},
		{"flex_end", "flex-end", 86.4, 14.4},
		{"default_stretch", "", 72, 28.8},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			html := `<div style="display: flex; align-items: ` + tc.alignItems + `"><div style="font-size: 12pt">small</div><div style="font-size: 24pt">big</div></div>`
			root, err := boxes.Build(html)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			blocks := Flow(root, testOptions())[0].Blocks
			if len(blocks) != 2 {
				t.Fatalf("boxes flex: got %d, want 2", len(blocks))
			}
			short, tall := blocks[0], blocks[1]
			if !approx(short.Y, tc.shortY) || !approx(short.Height, tc.shortH) {
				t.Errorf("lower child: y=%v height=%v, want y=%v height=%v", short.Y, short.Height, tc.shortY, tc.shortH)
			}
			if !approx(tall.Y, 72) || !approx(tall.Height, 28.8) {
				t.Errorf("child height: y=%v height=%v, want y=72 height=28.8", tall.Y, tall.Height)
			}
			if len(short.Runs) == 0 || len(short.Runs[0]) == 0 || !approx(short.Runs[0][0].Y, tc.shortY) {
				t.Errorf("text in the lower child does not follow alignment: %#v", short.Runs)
			}
		})
	}
}

func TestLayout_FlexAlignItemsFlexStartKeepsShortItemAtRowStart(t *testing.T) {
	root, err := boxes.Build(`<div style="display: flex; align-items: flex-start"><div style="font-size: 12pt">small</div><div style="font-size: 24pt">big</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("boxes flex: got %d, want 2", len(blocks))
	}
	short, tall := blocks[0], blocks[1]
	if !approx(short.Y, 72) || !approx(short.Height, 14.4) {
		t.Errorf("lower child: y=%v height=%v, want y=72 height=14.4", short.Y, short.Height)
	}
	if !approx(tall.Y, 72) || !approx(tall.Height, 28.8) {
		t.Errorf("child height: y=%v height=%v, want y=72 height=28.8", tall.Y, tall.Height)
	}
	if len(short.Runs) == 0 || len(short.Runs[0]) == 0 || !approx(short.Runs[0][0].Y, 72) {
		t.Errorf("text of the lower child does not start with the row: %#v", short.Runs)
	}
}

func TestLayout_FlexAlignItemsBaselineAlignsTextRuns(t *testing.T) {
	root, err := boxes.Build(`<div style="display: flex; align-items: baseline"><div style="font-size: 12pt">small</div><div style="font-size: 24pt">big</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("boxes flex: got %d, want 2", len(blocks))
	}
	short, tall := blocks[0], blocks[1]
	if !approx(short.Y, 84) || !approx(short.Height, 14.4) {
		t.Errorf("short box: y=%v height=%v, want y=84 height=14.4", short.Y, short.Height)
	}
	if !approx(tall.Y, 72) || !approx(tall.Height, 28.8) {
		t.Errorf("tall box: y=%v height=%v, want y=72 height=28.8", tall.Y, tall.Height)
	}
	if len(short.Runs) == 0 || len(short.Runs[0]) == 0 || len(tall.Runs) == 0 || len(tall.Runs[0]) == 0 {
		t.Fatal(" expectedn corridas of text enbothchildren")
	}
	smallBaseline := short.Runs[0][0].Y + short.Runs[0][0].BaselineOffset
	bigBaseline := tall.Runs[0][0].Y + tall.Runs[0][0].BaselineOffset
	if !approx(smallBaseline, bigBaseline) {
		t.Errorf("lines base: got %v/%v, want equal baselines", smallBaseline, bigBaseline)
	}
}

func TestLayout_FlexAlignSelfOverridesContainerAlignment(t *testing.T) {
	root, err := boxes.Build(`<div style="display: flex; align-items: flex-start"><div style="font-size:12pt; align-self:flex-end">small</div><div style="font-size:18pt; align-self:center">medium</div><div style="font-size:24pt">big</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 3 {
		t.Fatalf("boxes flex: got %d, want 3", len(blocks))
	}
	wantY := []float64{86.4, 75.6, 72}
	wantH := []float64{14.4, 21.6, 28.8}
	for i, block := range blocks {
		if !approx(block.Y, wantY[i]) || !approx(block.Height, wantH[i]) {
			t.Errorf("child %d: y=%v height=%v, want y=%v height=%v", i, block.Y, block.Height, wantY[i], wantH[i])
		}
	}
}

func TestLayout_FlexAlignSelfBaselineAlignsTextRuns(t *testing.T) {
	root, err := boxes.Build(`<div style="display: flex; align-items: flex-start"><div style="font-size:12pt; align-self:baseline">small</div><div style="font-size:24pt; align-self:baseline">big</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("boxes flex: got %d, want 2", len(blocks))
	}
	smallRun, bigRun := blocks[0].Runs[0][0], blocks[1].Runs[0][0]
	if !approx(smallRun.Y+smallRun.BaselineOffset, bigRun.Y+bigRun.BaselineOffset) {
		t.Errorf("lines base: got %v/%v, want equal baselines", smallRun.Y+smallRun.BaselineOffset, bigRun.Y+bigRun.BaselineOffset)
	}
}

func TestLayout_FlexColumnGapSeparatesIntrinsicItems(t *testing.T) {
	cases := []struct {
		name, gap string
		wantX     float64
		wantW     float64
	}{
		{"gap_px", "gap: 10px", 85.5, 6},
		{"column_gap_pt", "column-gap: 6pt", 84, 6},
		{"two_value_gap_uses_column_component", "gap: 4px 8pt", 86, 6},
		{"column_gap_percent", "column-gap: 10%", 123.1, 6},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, err := boxes.Build(`<div style="display:flex; ` + tc.gap + `"><span>A</span><span>B</span></div>`)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			blocks := Flow(root, testOptions())[0].Blocks
			if len(blocks) != 2 {
				t.Fatalf("boxes flex: got %d, want 2", len(blocks))
			}
			if !approx(blocks[0].X, 72) || !approx(blocks[0].Width, tc.wantW) {
				t.Errorf("first item: x=%v width=%v, want x=72 width=%v", blocks[0].X, blocks[0].Width, tc.wantW)
			}
			if !approx(blocks[1].X, tc.wantX) || !approx(blocks[1].Width, tc.wantW) {
				t.Errorf("second item: x=%v width=%v, want x=%v width=%v", blocks[1].X, blocks[1].Width, tc.wantX, tc.wantW)
			}
			if len(blocks[1].Runs) == 0 || len(blocks[1].Runs[0]) == 0 || !approx(blocks[1].Runs[0][0].X, tc.wantX) {
				t.Errorf("second item run does not follow the gap: %#v", blocks[1].Runs)
			}
		})
	}
}

func approx(a, b float64) bool {
	d := a - b
	return d < 1e-9 && d > -1e-9
}

func TestLayout_FlexPercentBorderBoxItemsFitThreePerRow(t *testing.T) {
	root, err := boxes.Build(`<div style="display:flex;flex-wrap:wrap;gap:9pt"><div style="width:30%;box-sizing:border-box;padding:9pt;border:1pt solid black">one</div><div style="width:30%;box-sizing:border-box;padding:9pt;border:1pt solid black">two</div><div style="width:30%;box-sizing:border-box;padding:9pt;border:1pt solid black">three</div><div style="width:30%;box-sizing:border-box;padding:9pt;border:1pt solid black">four</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	pages := Flow(root, testOptions())
	var rows []float64
	for _, block := range pages[0].Blocks {
		if block.Text == "one" || block.Text == "two" || block.Text == "three" || block.Text == "four" {
			rows = append(rows, block.Y)
			if block.Width != 0 && block.Width > 451*0.3+0.001 {
				t.Errorf("width border-box=%g, exceeds 30%% of 451pt", block.Width)
			}
		}
	}
	if len(rows) != 4 {
		t.Fatalf("placed items=%d, want 4", len(rows))
	}
	if rows[0] != rows[1] || rows[1] != rows[2] {
		t.Errorf("the first three items must share a row: y=%v", rows)
	}
	if rows[3] <= rows[2] {
		t.Errorf("the fourth item must move to the second row: y=%v", rows)
	}
}

func TestLayout_FlexMarginsCollapseWithFollowingBlock(t *testing.T) {
	root, err := boxes.Build(`<div style="display:flex;height:20pt;margin-bottom:12pt"><span>A</span></div><p style="margin:8pt 0 0">B</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	root.Style.BodyMargin = 0
	opts := testOptions()
	opts.Margin = 0
	blocks := Flow(root, opts)[0].Blocks
	if len(blocks) != 2 || len(blocks[0].Runs) != 1 || len(blocks[1].Runs) != 1 {
		t.Fatalf("blocks of flex and paragraph: %#v", blocks)
	}
	if got := blocks[1].Runs[0][0].Y; !approx(got, 32) {
		t.Errorf("paragraph y=%v, want 32pt after collapsing margin 12pt with 8pt", got)
	}
}

func TestLayout_FlexNestedBlockTextUsesItsFontBaseline(t *testing.T) {
	root, err := boxes.Build(`<div style="display:flex"><div><small>Label</small><h1 style="font-size:18pt;line-height:1.25;margin:0">Title</h1></div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	var title *TextRun
	for _, block := range blocks {
		for _, line := range block.Runs {
			for i := range line {
				if line[i].Text == "Title" {
					title = &line[i]
				}
			}
		}
	}
	if title == nil {
		t.Fatalf("title not found among Flex descendants: %#v", blocks)
	}
	if got := title.BaselineOffset; !approx(got, 18) {
		t.Errorf("title baseline=%v, want 18pt from the title block", got)
	}
}

func TestLayout_FlexContainerDirectTextCreatesAnonymousItem(t *testing.T) {
	root, err := boxes.Build(`<div style="display:flex;width:84pt;height:84pt;align-items:center;justify-content:center;border:2pt solid #a16207;border-radius:50%;font-size:20pt">X</div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	var item, wrapper *PlacedBox
	for i := range blocks {
		if blocks[i].Text == "X" {
			item = &blocks[i]
		}
		if blocks[i].BorderWidth == 2 {
			wrapper = &blocks[i]
		}
	}
	if item == nil {
		t.Fatalf("direct text must produce an anonymous flex item: %#v", blocks)
	}
	if len(item.Runs) == 0 || len(item.Runs[0]) == 0 {
		t.Fatal("the anonymous item must preserve the text run")
	}
	if wrapper == nil || wrapper.Height != 88 {
		t.Errorf("the decorated flex box must preserve 84pt of content and a 2pt border: %#v", blocks)
	}
	if item.X <= 72 || item.Y <= 72 {
		t.Errorf("justify/align center must center the item in its box: x=%v y=%v", item.X, item.Y)
	}
}

func TestLayout_FlexBlockResolvesAutoHorizontalMargins(t *testing.T) {
	root, err := boxes.Build(`<div style="display:flex;width:84pt;margin:0 auto"><span>x</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("flex items: %#v", blocks)
	}
	if !approx(blocks[0].X, 255.5) {
		t.Errorf("flex block x=%v, want centered at 255.5pt in 451pt content area", blocks[0].X)
	}
}
