package boxes

import (
	"testing"
)

// Backgrounds/borders step 2: background-color and uniform border from inline
// style. Only `border: Npx style color` is covered (style is ignored except none).
func TestBoxes_ParsesBackgroundAndBorder(t *testing.T) {
	root, err := Build(`<p style="background-color: #ff0000; border: 2px solid black">x</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := root.Children[0]
	if !b.Style.HasBg || b.Style.Bg != [4]float64{1, 0, 0, 1} {
		t.Errorf("bg: got %v %v, want true [1 0 0]", b.Style.HasBg, b.Style.Bg)
	}
	if b.Style.BorderWidth != 1.5 || b.Style.Border != [4]float64{0, 0, 0, 1} {
		t.Errorf("border: got %v %v, want 1.5 [0 0 0]", b.Style.BorderWidth, b.Style.Border)
	}
}

func TestBoxes_NoBackgroundOrBorder_Defaults(t *testing.T) {
	root, err := Build(`<p>x</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := root.Children[0]
	if b.Style.HasBg || b.Style.BorderWidth != 0 {
		t.Errorf("defaults: got HasBg=%v BorderWidth=%v, want false 0", b.Style.HasBg, b.Style.BorderWidth)
	}
}

func TestBoxes_BorderNone_Ignored(t *testing.T) {
	root, err := Build(`<p style="border: none">x</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w := root.Children[0].Style.BorderWidth; w != 0 {
		t.Errorf("border none: got %v, want 0", w)
	}
}

func TestBoxes_ParsesUniformBorderRadius(t *testing.T) {
	root, err := Build(`<div style="border-radius:12pt">x</div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	want := [4]CornerRadius{
		{X: RadiusLength{Value: 12}, Y: RadiusLength{Value: 12}},
		{X: RadiusLength{Value: 12}, Y: RadiusLength{Value: 12}},
		{X: RadiusLength{Value: 12}, Y: RadiusLength{Value: 12}},
		{X: RadiusLength{Value: 12}, Y: RadiusLength{Value: 12}},
	}
	if got := root.Children[0].Style.BorderCornerRadii; got != want {
		t.Errorf("border-radius corners: got %v, want %v", got, want)
	}
}

func TestBoxes_ParsesBackgroundColorShorthand(t *testing.T) {
	tests := []struct {
		name, style string
		want        [4]float64
	}{
		{name: "hex", style: "background: #123456", want: [4]float64{float64(0x12) / 255, float64(0x34) / 255, float64(0x56) / 255, 1}},
		{name: "name", style: "background: red", want: [4]float64{1, 0, 0, 1}},
		{name: "rgb", style: "background: rgb(12, 34, 56)", want: [4]float64{12.0 / 255, 34.0 / 255, 56.0 / 255, 1}},
		{name: "combined shorthand", style: "background: rgb(12, 34, 56) no-repeat center", want: [4]float64{12.0 / 255, 34.0 / 255, 56.0 / 255, 1}},
		{name: "longhand wins", style: "background: blue; background-color: red", want: [4]float64{1, 0, 0, 1}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root, err := Build(`<div style="` + tc.style + `">x</div>`)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			box := root.Children[0]
			if !box.Style.HasBg || box.Style.Bg != tc.want {
				t.Errorf("background: got %v %v, want true %v", box.Style.HasBg, box.Style.Bg, tc.want)
			}
		})
	}
}

func TestBoxes_ParsesAsymmetricBorderRadiusShorthand(t *testing.T) {
	tests := []struct {
		name, value string
		want        [4]float64
	}{
		{name: "one", value: "1pt", want: [4]float64{1, 1, 1, 1}},
		{name: "two", value: "1pt 2pt", want: [4]float64{1, 2, 1, 2}},
		{name: "three", value: "1pt 2pt 3pt", want: [4]float64{1, 2, 3, 2}},
		{name: "four", value: "22pt 4pt 14pt 2pt", want: [4]float64{22, 4, 14, 2}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root, err := Build(`<div style="border-radius:` + tc.value + `">x</div>`)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			got := root.Children[0].Style.BorderCornerRadii
			for i, radius := range got {
				if radius.X.Value != tc.want[i] || radius.Y.Value != tc.want[i] {
					t.Errorf("corner %d=%+v, want x/y=%v", i, radius, tc.want[i])
				}
			}
		})
	}
}

func TestBoxes_ParsesPercentageAndEllipticalBorderRadius(t *testing.T) {
	percent := RadiusLength{Value: 50, Percent: true}
	pt10 := RadiusLength{Value: 10}
	pt20 := RadiusLength{Value: 20}
	tests := []struct {
		name, value string
		want        [4]CornerRadius
	}{
		{name: "percentage", value: "50%", want: [4]CornerRadius{{X: percent, Y: percent}, {X: percent, Y: percent}, {X: percent, Y: percent}, {X: percent, Y: percent}}},
		{name: "elliptical", value: "10pt 20pt / 50% 25%", want: [4]CornerRadius{{X: pt10, Y: percent}, {X: pt20, Y: RadiusLength{Value: 25, Percent: true}}, {X: pt10, Y: percent}, {X: pt20, Y: RadiusLength{Value: 25, Percent: true}}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root, err := Build(`<div style="border-radius:` + tc.value + `">x</div>`)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			if got := root.Children[0].Style.BorderCornerRadii; got != tc.want {
				t.Errorf("BorderCornerRadii=%v, want %v", got, tc.want)
			}
		})
	}
}

func TestBoxes_ParsesBoxSizing(t *testing.T) {
	root, err := Build(`<div style="box-sizing:border-box;width:30%;padding:9pt;border:1pt solid black">x</div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if got := root.Children[0].Style.BoxSizing; got != BoxSizingBorderBox {
		t.Errorf("BoxSizing=%q, want border-box", got)
	}
}
