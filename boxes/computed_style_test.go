package boxes

import "testing"

func TestBoxes_ComputedStyleLivesOutsideBoxContent(t *testing.T) {
	root, err := Build(`<div style="display:flex;margin-left:7pt;font:italic 13pt/1.5 Helvetica">plain <span>nested</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	box := root.Children[0]
	style := box.Style
	if style.Display != "flex" || style.MarginLeft != 7 || style.FontSize != 13 || !style.Italic {
		t.Fatalf("computed style incomplete: %+v", style)
	}
	if box.Text != "plain nested" || len(box.Children) != 1 || box.Children[0].Tag != "span" {
		t.Fatalf("box content changed when separating the style: text=%q children=%+v", box.Text, box.Children)
	}
}

func TestBoxes_ComputedLayoutModesUseTypedValues(t *testing.T) {
	root, err := Build(`<div style="display:flex;position:static;float:right;overflow-x:hidden;box-sizing:border-box">content</div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	style := root.Children[0].Style
	if style.Display != DisplayFlex || style.Position != PositionStatic || style.Float != FloatRight ||
		style.OverflowX != OverflowHidden || style.BoxSizing != BoxSizingBorderBox {
		t.Fatalf("computed layout modes = %+v", style)
	}
	contentBox, err := Build(`<div style="box-sizing:content-box">content</div>`)
	if err != nil {
		t.Fatalf("Build content-box: %v", err)
	}
	if got := contentBox.Children[0].Style.BoxSizing; got != BoxSizingContentBox {
		t.Fatalf("content-box mode=%q, want initial mode %q", got, BoxSizingContentBox)
	}
}

func TestBoxes_ComputedDimensionsPreserveAutoLengthAndPercentage(t *testing.T) {
	root, err := Build(`<div><span></span><span style="width:0pt;height:0pt"></span><span style="width:50%"></span><span style="width:0%"></span><span style="height:25%"></span><span style="width:25%;max-width:80pt;min-height:10%"></span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	children := root.Children[0].Children
	if got := children[0].Style.Sizing.Width.Kind; got != DimensionAuto {
		t.Errorf("unspecified width kind=%v, want auto", got)
	}
	if got := children[1].Style.Sizing.Width; got != (CSSDimension{Kind: DimensionLength, Value: 0}) {
		t.Errorf("zero width=%+v, want specified zero length", got)
	}
	if got := children[1].Style.Sizing.Height; got != (CSSDimension{Kind: DimensionLength, Value: 0}) {
		t.Errorf("zero height=%+v, want specified zero length", got)
	}
	if got := children[2].Style.Sizing.Width; got != (CSSDimension{Kind: DimensionPercentage, Value: 50}) {
		t.Errorf("percentage width=%+v, want 50%%", got)
	}
	if got := children[3].Style.Sizing.Width; got != (CSSDimension{Kind: DimensionPercentage, Value: 0}) {
		t.Errorf("zero percentage width=%+v, want specified 0%%", got)
	}
	if got := children[4].Style.Sizing.Height; got != (CSSDimension{Kind: DimensionPercentage, Value: 25}) {
		t.Errorf("percentage height=%+v, want 25%%", got)
	}
	if got := children[5].Style.Sizing.MaxWidth; got != (CSSDimension{Kind: DimensionLength, Value: 80}) {
		t.Errorf("max-width=%+v, want 80pt", got)
	}
	if got := children[5].Style.Sizing.MinHeight; got != (CSSDimension{Kind: DimensionPercentage, Value: 10}) {
		t.Errorf("min-height=%+v, want 10%%", got)
	}
}

func TestBoxes_ComputedMinimumWidthUsesDimensionModel(t *testing.T) {
	root, err := Build(`<div><span></span><span style="min-width:0pt"></span><span style="min-width:25%"></span><span style="min-width:0%"></span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	children := root.Children[0].Children
	want := []CSSDimension{
		{},
		{Kind: DimensionLength},
		{Kind: DimensionPercentage, Value: 25},
		{Kind: DimensionPercentage},
	}
	for i, expected := range want {
		if got := children[i].Style.Sizing.MinWidth; got != expected {
			t.Errorf("min-width[%d]=%+v, want %+v", i, got, expected)
		}
	}
}
