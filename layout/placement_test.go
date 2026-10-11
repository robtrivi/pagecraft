package layout

import (
	"testing"

	"github.com/robtrivi/pagecraft/boxes"
)

func TestLayout_ZeroPercentWidthResolvesToZero(t *testing.T) {
	root, err := boxes.Build(`<div style="width:0%"></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if got := usedWidth(root.Children[0], 240); got != 0 {
		t.Fatalf("used width for 0%% = %v, want 0", got)
	}
}

func TestLayout_DimensionResolutionUsesOptionalContainingSize(t *testing.T) {
	cases := []struct {
		name          string
		dimension     boxes.CSSDimension
		containing    float64
		contextExists bool
		want          float64
		wantResolved  bool
	}{
		{name: "length", dimension: boxes.CSSDimension{Kind: boxes.DimensionLength, Value: 12}, want: 12, wantResolved: true},
		{name: "percentage", dimension: boxes.CSSDimension{Kind: boxes.DimensionPercentage, Value: 25}, containing: 200, contextExists: true, want: 50, wantResolved: true},
		{name: "zero_percentage", dimension: boxes.CSSDimension{Kind: boxes.DimensionPercentage}, containing: 200, contextExists: true, wantResolved: true},
		{name: "percentage_without_context", dimension: boxes.CSSDimension{Kind: boxes.DimensionPercentage, Value: 25}},
		{name: "auto", dimension: boxes.CSSDimension{Kind: boxes.DimensionAuto}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, resolved := resolveDimension(tc.dimension, sizeBasis{Value: tc.containing, Definite: tc.contextExists})
			if got != tc.want || resolved != tc.wantResolved {
				t.Fatalf("resolveDimension()=(%v,%v), want (%v,%v)", got, resolved, tc.want, tc.wantResolved)
			}
		})
	}
}

func TestLayout_DimensionResolutionUsesExplicitAxisDefiniteness(t *testing.T) {
	context := sizeContext{
		Inline: sizeBasis{Value: 240, Definite: true},
		Block:  sizeBasis{Value: 120, Definite: false},
	}
	percentage := boxes.CSSDimension{Kind: boxes.DimensionPercentage, Value: 25}
	if got, resolved := resolveDimension(percentage, context.Inline); got != 60 || !resolved {
		t.Errorf("percentage inline=(%g,%v), want (60,true)", got, resolved)
	}
	if got, resolved := resolveDimension(percentage, context.Block); got != 0 || resolved {
		t.Errorf("percentage block without size definite=(%g,%v), want (0,false)", got, resolved)
	}
	context.Block.Definite = true
	if got, resolved := resolveDimension(percentage, context.Block); got != 30 || !resolved {
		t.Errorf("percentage block definite=(%g,%v), want (30,true)", got, resolved)
	}
}

func TestLayout_ResolveBorderBoxDimensionCombinesContextAndInsets(t *testing.T) {
	cases := []struct {
		name          string
		dimension     boxes.CSSDimension
		containing    float64
		contextExists bool
		insets        float64
		boxSizing     boxes.BoxSizing
		want          float64
		wantResolved  bool
	}{
		{name: "content box length", dimension: boxes.CSSDimension{Kind: boxes.DimensionLength, Value: 20}, insets: 8, want: 28, wantResolved: true},
		{name: "border box percentage", dimension: boxes.CSSDimension{Kind: boxes.DimensionPercentage, Value: 25}, containing: 200, contextExists: true, insets: 8, boxSizing: boxes.BoxSizingBorderBox, want: 50, wantResolved: true},
		{name: "zero content size retains insets", dimension: boxes.CSSDimension{Kind: boxes.DimensionLength}, insets: 8, want: 8, wantResolved: true},
		{name: "percentage without context", dimension: boxes.CSSDimension{Kind: boxes.DimensionPercentage, Value: 25}, insets: 8},
		{name: "auto", dimension: boxes.CSSDimension{Kind: boxes.DimensionAuto}, insets: 8},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, resolved := resolveBorderBoxDimension(tc.dimension, sizeBasis{Value: tc.containing, Definite: tc.contextExists}, tc.insets, tc.boxSizing)
			if got != tc.want || resolved != tc.wantResolved {
				t.Fatalf("resolveBorderBoxDimension()=(%v,%v), want (%v,%v)", got, resolved, tc.want, tc.wantResolved)
			}
		})
	}
}

func TestLayout_SizeConstraintsClampUsedWidthAndHeight(t *testing.T) {
	cases := []struct {
		name, style      string
		containingHeight float64
		heightSet        bool
		wantWidth        float64
		wantHeight       float64
	}{
		{name: "content_box_limits", style: "width:120pt;min-width:80pt;max-width:100pt;height:20pt;min-height:40pt;max-height:60pt", wantWidth: 100, wantHeight: 40},
		{name: "border_box_limits", style: "width:120pt;min-width:80pt;max-width:100pt;height:20pt;min-height:40pt;max-height:60pt;box-sizing:border-box;padding:4pt;border:2pt solid #000", wantWidth: 100, wantHeight: 40},
		{name: "percentage_limits", style: "width:120pt;max-width:50%;height:120pt;max-height:50%", containingHeight: 200, heightSet: true, wantWidth: 100, wantHeight: 100},
		{name: "minimum_wins_over_maximum", style: "width:20pt;min-width:90pt;max-width:60pt;height:20pt;min-height:90pt;max-height:60pt", wantWidth: 90, wantHeight: 90},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, err := boxes.Build(`<div style="` + tc.style + `"></div>`)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			box := root.Children[0]
			opts := testOptions()
			opts.space.containingSize.Block = sizeBasis{Value: tc.containingHeight, Definite: tc.heightSet}
			if got := usedWidth(box, 200); got != tc.wantWidth {
				t.Errorf("used width=%v, want %v", got, tc.wantWidth)
			}
			if got := usedHeight(box, 20, opts); got != tc.wantHeight {
				t.Errorf("used height=%v, want %v", got, tc.wantHeight)
			}
		})
	}
}

func TestLayout_PercentageHeightResolvesAgainstContainingBlock(t *testing.T) {
	cases := []struct {
		name, style      string
		containingHeight float64
		hasContaining    bool
		want             float64
	}{
		{name: "content_box", style: "height:25%;padding:5pt;border:1pt solid #000", containingHeight: 200, hasContaining: true, want: 62},
		{name: "unresolved_without_containing_height", style: "height:25%;padding:5pt;border:1pt solid #000", want: 32},
		{name: "border_box", style: "height:25%;box-sizing:border-box;padding:5pt;border:1pt solid #000", containingHeight: 200, hasContaining: true, want: 50},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, err := boxes.Build(`<div style="` + tc.style + `"></div>`)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			opts := testOptions()
			opts.space.containingSize.Block = sizeBasis{Value: tc.containingHeight, Definite: tc.hasContaining}
			if got := usedHeight(root.Children[0], 20, opts); got != tc.want {
				t.Fatalf("used height=%v, want %v", got, tc.want)
			}
		})
	}
}

func TestLayout_NestedPercentageHeightPropagatesContainingBlock(t *testing.T) {
	root, err := boxes.Build(`<div style="height:200pt"><div style="height:50%"><div style="height:50%;padding:5pt;border:1pt solid #000;background:#eee">child</div></div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	pages := Flow(root, testOptions())
	found := false
	for _, block := range pages[0].Blocks {
		if block.HasBg && block.Height == 62 {
			found = true
		}
	}
	if !found {
		t.Fatal("decorated nested percentage-height box with 62pt height was not found")
	}
}

func TestLayout_NestedContainersApplyHeightConstraintsToOuterBox(t *testing.T) {
	tests := []struct {
		name, style string
		want        float64
	}{
		{name: "minimum height grows container around content", style: "min-height:100pt", want: 112},
		{name: "maximum height clamps declared size", style: "height:100pt;max-height:60pt", want: 72},
		{name: "minimum height overrides smaller maximum", style: "min-height:80pt;max-height:40pt", want: 92},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := `<div style="` + tt.style + `;padding:5pt;border:1pt solid #000;background:#eee"><p style="margin:0">x</p></div><p style="margin:0">AFTER</p>`
			root, err := boxes.Build(html)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			pages := Flow(root, testOptions())
			var outer, after *PlacedBox
			for _, block := range pages[0].Blocks {
				if block.HasBg {
					outer = &block
					if !approx(block.Height, tt.want) {
						t.Fatalf("outer box height=%v, want %v", block.Height, tt.want)
					}
				}
				if block.Text == "AFTER" {
					after = &block
				}
			}
			if outer == nil {
				t.Fatal("decorated container was not placed")
			}
			if after == nil {
				t.Fatal("following block was not placed")
			}
			if after.Y < outer.Y+outer.Height {
				t.Fatalf("following block y=%v, want at or after outer bottom %v", after.Y, outer.Y+outer.Height)
			}
		})
	}
}
