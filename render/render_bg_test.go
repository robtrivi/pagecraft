package render

import (
	"math"
	"strings"
	"testing"

	"github.com/robtrivi/pagecraft-go/boxes"
	"github.com/robtrivi/pagecraft-go/layout"
)

// Background/borders step 3: a box with background and border emits a filled
// and stroked rectangle in the correct colors before its text.
func TestRender_DrawsBackgroundAndBorder(t *testing.T) {
	got, err := RenderHTMLToPDF(`<p style="background-color: #ff0000; border: 2px solid black">x</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := streamText(t, got)
	for _, want := range []string{"1 0 0 rg", " re f", "1.5 w", "0 0 0 RG", " re S"} {
		if !strings.Contains(text, want) {
			t.Errorf("the PDF does not contain %q", want)
		}
	}
}

func TestRender_BorderRadiusUsesCurvedBackgroundAndBorder(t *testing.T) {
	pdf, err := RenderHTMLToPDF(`<div style="margin:0;background-color:#dcfce7;border:2pt solid #16a34a;border-radius:12pt;padding:14pt">Rounded background</div>`)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	content := streamText(t, pdf)
	if !strings.Contains(content, "84 724.656 m") || !strings.Contains(content, " c\n") {
		t.Error("the background and border should use a curved path")
	}
	if strings.Contains(content, "re f") {
		t.Error("the rounded background must not be painted as a rectangle")
	}
}

func TestRender_BackgroundColorDoesNotLeakIntoText(t *testing.T) {
	pdf, err := RenderHTMLToPDF(`<p style="background-color: #ddd">Gray</p><p>Black</p>`)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	content := streamText(t, pdf)
	if !strings.Contains(content, "0.8666666666666667 0.8666666666666667 0.8666666666666667 rg") {
		t.Fatal("expected gray fill was not found")
	}
	if !strings.Contains(content, " re f\nQ\n0 0 0 rg\nBT /F1") {
		t.Fatal("background graphics state was not restored before drawing text")
	}
}

func TestRender_NormalizesAsymmetricBorderRadiiToFitBox(t *testing.T) {
	tests := []struct {
		name     string
		w, h     float64
		in, want [4]ellipseRadius
	}{
		{name: "fits", w: 100, h: 60, in: [4]ellipseRadius{{10, 10}, {20, 20}, {15, 15}, {5, 5}}, want: [4]ellipseRadius{{10, 10}, {20, 20}, {15, 15}, {5, 5}}},
		{name: "horizontal", w: 40, h: 60, in: [4]ellipseRadius{{30, 30}, {30, 30}, {10, 10}, {10, 10}}, want: [4]ellipseRadius{{20, 20}, {20, 20}, {6.6666666667, 6.6666666667}, {6.6666666667, 6.6666666667}}},
		{name: "vertical", w: 100, h: 30, in: [4]ellipseRadius{{40, 40}, {20, 20}, {10, 10}, {30, 30}}, want: [4]ellipseRadius{{17.1428571429, 17.1428571429}, {8.5714285714, 8.5714285714}, {4.2857142857, 4.2857142857}, {12.8571428571, 12.8571428571}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeEllipticalRadii(tc.w, tc.h, tc.in)
			for i := range got {
				if math.Abs(got[i].X-tc.want[i].X) > 0.000001 || math.Abs(got[i].Y-tc.want[i].Y) > 0.000001 {
					t.Errorf("radius[%d]=%+v, want %+v; all=%v", i, got[i], tc.want[i], got)
				}
			}
		})
	}
}

func TestRender_DrawsEachAsymmetricCornerRadius(t *testing.T) {
	var path strings.Builder
	drawRoundedEllipseRect(&path, 0, 0, 100, 50, [4]ellipseRadius{{20, 20}, {2, 2}, {10, 10}, {4, 4}})
	got := path.String()
	if !strings.HasPrefix(got, "20 0 m\n98 0 l\n") {
		t.Fatalf("left/right corner radii must differ: %s", got)
	}
	for _, want := range []string{"100 2 c\n", "90 50 c\n", "4 50 l\n", "0 20 l\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("path does not contain the expected segment %q: %s", want, got)
		}
	}
}

func TestRender_NormalizesEllipticalRadiiTogether(t *testing.T) {
	input := [4]ellipseRadius{{X: 80, Y: 30}, {X: 80, Y: 30}, {X: 10, Y: 10}, {X: 10, Y: 10}}
	got := normalizeEllipticalRadii(100, 50, input)
	want := [4]ellipseRadius{{X: 50, Y: 18.75}, {X: 50, Y: 18.75}, {X: 6.25, Y: 6.25}, {X: 6.25, Y: 6.25}}
	for i := range got {
		if math.Abs(got[i].X-want[i].X) > 0.000001 || math.Abs(got[i].Y-want[i].Y) > 0.000001 {
			t.Errorf("radii[%d]=%+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestRender_DrawsEllipticalCornerPath(t *testing.T) {
	var path strings.Builder
	drawRoundedEllipseRect(&path, 0, 0, 100, 50, [4]ellipseRadius{{X: 50, Y: 25}, {X: 50, Y: 25}, {X: 50, Y: 25}, {X: 50, Y: 25}})
	got := path.String()
	if !strings.HasPrefix(got, "50 0 m\n50 0 l\n") {
		t.Fatalf("curve must preserve different X/y radii: %s", got)
	}
	if !strings.Contains(got, "100 25 c\n") {
		t.Fatalf("vertical arc must end at 25pt: %s", got)
	}
}

func TestRender_PercentageRadiiResolveAgainstEachBoxAxis(t *testing.T) {
	percent := boxes.RadiusLength{Value: 50, Percent: true}
	box := layout.PlacedBox{BorderCornerRadii: [4]boxes.CornerRadius{
		{X: percent, Y: percent}, {X: percent, Y: percent}, {X: percent, Y: percent}, {X: percent, Y: percent},
	}}
	got := resolveBorderCornerRadii(box, 100, 50)
	for i, radius := range got {
		if radius.X != 50 || radius.Y != 25 {
			t.Errorf("corner[%d]=%+v, want X=50 y=25", i, radius)
		}
	}
}
