package boxes

import "testing"

func TestBoxes_ComputedFlexDistributionUsesTypedValues(t *testing.T) {
	root, err := Build(`<div style="display:flex;flex-wrap:wrap;flex-direction:column;justify-content:space-between;align-items:center"><span style="align-self:flex-end">A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	container := root.Children[0]
	if container.Style.FlexWrap != FlexWrap || container.Style.FlexDirection != FlexDirectionColumn ||
		container.Style.JustifyContent != JustifySpaceBetween || container.Style.AlignItems != AlignCenter {
		t.Fatalf("computed flex container modes = %+v", container.Style)
	}
	if got := container.Children[0].Style.AlignSelf; got != AlignFlexEnd {
		t.Fatalf("computed align-self=%q, want %q", got, AlignFlexEnd)
	}
}

func TestBoxes_ParsesFlexAlignItems(t *testing.T) {
	cases := []struct {
		name, css, want string
	}{
		{"center", "center", "center"},
		{"flex_end", "flex-end", "flex-end"},
		{"unsupported", "space-between", ""},
		{"unset", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, err := Build(`<div style="display: flex; align-items: ` + tc.css + `"><span>A</span></div>`)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			if got := root.Children[0].Style.AlignItems; got != FlexAlignment(tc.want) {
				t.Errorf("align-items=%q, want %q", got, tc.want)
			}
		})
	}
}

func TestBoxes_ParsesFlexAlignItemsFlexStart(t *testing.T) {
	root, err := Build(`<div style="display: flex; align-items: flex-start"><span>A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if got := root.Children[0].Style.AlignItems; got != AlignFlexStart {
		t.Errorf("align-items=%q, want flex-start", got)
	}
}

func TestBoxes_ParsesFlexAlignItemsBaseline(t *testing.T) {
	root, err := Build(`<div style="display: flex; align-items: baseline"><span>A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if got := root.Children[0].Style.AlignItems; got != AlignBaseline {
		t.Errorf("align-items=%q, want baseline", got)
	}
}

func TestBoxes_ParsesFlexAlignSelf(t *testing.T) {
	cases := []struct {
		name, css, want string
	}{
		{"start", "flex-start", "flex-start"},
		{"center", "center", "center"},
		{"end", "flex-end", "flex-end"},
		{"baseline", "baseline", "baseline"},
		{"stretch", "stretch", "stretch"},
		{"auto", "auto", "auto"},
		{"invalid", "space-between", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, err := Build(`<div style="display: flex"><span style="align-self: ` + tc.css + `">A</span></div>`)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			if got := root.Children[0].Children[0].Style.AlignSelf; got != FlexAlignment(tc.want) {
				t.Errorf("align-self=%q, want %q", got, tc.want)
			}
		})
	}
}

func TestBoxes_ParsesFlexWrapAndPercentageItemWidth(t *testing.T) {
	root, err := Build(`<div style="display:flex; flex-wrap:wrap"><div style="width:28%">item</div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if root.Children[0].Style.FlexWrap != FlexWrap {
		t.Errorf("flex-wrap: got %q, want wrap", root.Children[0].Style.FlexWrap)
	}
	if root.Children[0].Children[0].Style.Sizing.Width.Value != 28 {
		t.Errorf("width: got %v%%, want 28%%", root.Children[0].Children[0].Style.Sizing.Width.Value)
	}
}

func TestBoxes_ParsesFlexDirectionAndJustifyContent(t *testing.T) {
	root, err := Build(`<div style="display:flex;flex-direction:column;justify-content:space-between"><span>A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	container := root.Children[0]
	if container.Style.FlexDirection != FlexDirectionColumn || container.Style.JustifyContent != JustifySpaceBetween {
		t.Errorf("flex direction=%q justify=%q", container.Style.FlexDirection, container.Style.JustifyContent)
	}
}

func TestBoxes_ParsesFlexOneAsGrowingZeroBasis(t *testing.T) {
	root, err := Build(`<div style="display:flex"><span style="flex:1">A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	item := root.Children[0].Children[0]
	if item.Style.FlexGrow != 1 || item.Style.FlexBasis != (FlexBasis{Kind: FlexBasisPercentage, Value: 0}) {
		t.Errorf("flex:1 expansion: grow=%v basis=%+v, want 1/0%%", item.Style.FlexGrow, item.Style.FlexBasis)
	}
}

func TestBoxes_FlexShorthandParsesFactorsAndBasis(t *testing.T) {
	tests := []struct {
		name, value  string
		grow, shrink float64
		basis        FlexBasis
	}{
		{name: "number shorthand", value: "2", grow: 2, shrink: 1, basis: FlexBasis{Kind: FlexBasisPercentage}},
		{name: "explicit zero percentage", value: "1 1 0%", grow: 1, shrink: 1, basis: FlexBasis{Kind: FlexBasisPercentage}},
		{name: "fixed basis", value: "0 1 64pt", grow: 0, shrink: 1, basis: FlexBasis{Kind: FlexBasisLength, Value: 64}},
		{name: "percentage basis", value: "1 0 25%", grow: 1, shrink: 0, basis: FlexBasis{Kind: FlexBasisPercentage, Value: 25}},
		{name: "basis before factors", value: "64pt 0 1", grow: 0, shrink: 1, basis: FlexBasis{Kind: FlexBasisLength, Value: 64}},
		{name: "basis only uses shorthand initial factors", value: "32pt", grow: 1, shrink: 1, basis: FlexBasis{Kind: FlexBasisLength, Value: 32}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, err := Build(`<div style="display:flex"><span style="flex:` + tt.value + `">A</span></div>`)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			style := root.Children[0].Children[0].Style
			if style.FlexGrow != tt.grow || style.FlexShrink != tt.shrink || style.FlexBasis != tt.basis {
				t.Errorf("flex:%s = grow %v shrink %v basis %+v; want %v %v %+v", tt.value, style.FlexGrow, style.FlexShrink, style.FlexBasis, tt.grow, tt.shrink, tt.basis)
			}
		})
	}
}

func TestBoxes_FlexShrinkLonghandOverridesShorthandDefault(t *testing.T) {
	root, err := Build(`<div style="display:flex"><span style="flex:1;flex-shrink:0">A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	item := root.Children[0].Children[0]
	if item.Style.FlexGrow != 1 || item.Style.FlexBasis != (FlexBasis{Kind: FlexBasisPercentage}) {
		t.Errorf("flex shorthand: grow=%v basis=%+v, want 1/0%%", item.Style.FlexGrow, item.Style.FlexBasis)
	}
	if !item.Style.FlexShrinkSet || item.Style.FlexShrink != 0 {
		t.Errorf("explicit flex-shrink was lost: set=%v value=%v, want true/0", item.Style.FlexShrinkSet, item.Style.FlexShrink)
	}
}

func TestBoxes_ParsesMinWidthLengthAndPercentage(t *testing.T) {
	root, err := Build(`<div><span style="min-width:12pt"></span><span style="min-width:25%"></span><span style="min-width:0"></span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	items := root.Children[0].Children
	if !items[0].Style.Sizing.MinWidth.IsLength() || items[0].Style.Sizing.MinWidth.Value != 12 {
		t.Errorf("min-width:12pt = %+v, want 12pt", items[0].Style.Sizing.MinWidth)
	}
	if !items[1].Style.Sizing.MinWidth.IsPercentage() || items[1].Style.Sizing.MinWidth.Value != 25 {
		t.Errorf("min-width:25%% = %+v, want 25%%", items[1].Style.Sizing.MinWidth)
	}
	if !items[2].Style.Sizing.MinWidth.IsLength() || items[2].Style.Sizing.MinWidth.Value != 0 {
		t.Errorf("min-width:0 = %+v, want specified zero", items[2].Style.Sizing.MinWidth)
	}
}
