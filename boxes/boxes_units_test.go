package boxes

import "testing"

func TestBoxes_ParsesPhysicalCSSUnitsForBoxDimensions(t *testing.T) {
	root, err := Build(`<div style="width:32mm;height:1in;padding:10mm;margin-left:1cm"></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	box := root.Children[0]
	if !box.Style.Sizing.Width.IsLength() || box.Style.Sizing.Width.Value < 90.7 || box.Style.Sizing.Width.Value > 90.8 {
		t.Errorf("width:32mm=%g (present=%v), want 90.709pt", box.Style.Sizing.Width.Value, box.Style.Sizing.Width.IsLength())
	}
	if !box.Style.Sizing.Height.IsLength() || box.Style.Sizing.Height.Value != 72 {
		t.Errorf("height:1in=%g (present=%v), want 72pt", box.Style.Sizing.Height.Value, box.Style.Sizing.Height.IsLength())
	}
	if box.Style.PaddingTop < 28.34 || box.Style.PaddingTop > 28.35 || box.Style.MarginLeft < 28.34 || box.Style.MarginLeft > 28.35 {
		t.Errorf("physical insets: padding=%g margin=%g, want 28.346pt", box.Style.PaddingTop, box.Style.MarginLeft)
	}
}

func TestBoxes_ParsesAutoHorizontalMargins(t *testing.T) {
	root, err := Build(`<div style="margin:4pt auto"><span>x</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	box := root.Children[0]
	if !box.Style.MarginAutoLeft || !box.Style.MarginAutoRight {
		t.Errorf("margin auto flags: left=%v right=%v, want both", box.Style.MarginAutoLeft, box.Style.MarginAutoRight)
	}
}

func TestBoxes_ConvertsAllAbsoluteCSSUnitsToPoints(t *testing.T) {
	cases := []struct {
		name, length string
	}{
		{"px", "96px"}, {"pt", "72pt"}, {"in", "1in"},
		{"cm", "2.54cm"}, {"mm", "25.4mm"}, {"pc", "6pc"}, {"Q", "101.6Q"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, err := Build(`<div style="width:` + tc.length + `"></div>`)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			box := root.Children[0]
			if !box.Style.Sizing.Width.IsLength() || box.Style.Sizing.Width.Value < 71.999 || box.Style.Sizing.Width.Value > 72.001 {
				t.Errorf("width:%s = %gpt (present=%v), want 72pt", tc.length, box.Style.Sizing.Width.Value, box.Style.Sizing.Width.IsLength())
			}
		})
	}
}
