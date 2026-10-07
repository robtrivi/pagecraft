package svg

import (
	"testing"
)

// Transform tests apply every supported SVG transform family to a known point.
// The expected coordinates are standards-defined examples; keeping them in a
// shared contract makes composition and individual operations fail alike.
func TestSVG_Transform_ParseAndApply(t *testing.T) {
	cases := []struct {
		name, transform string
		x, y            float64
		wantX, wantY    float64
	}{
		{"translate and scale composition", "translate(10, 20) scale(2)", 5, 5, 20, 30},
		{"rotation", "rotate(90)", 1, 0, 0, 1},
		{"matrix and skew", "matrix(1, 0, 0, 1, 3, 4) skewX(45) skewY(0)", 2, 5, 10, 9},
		{"rotation around center", "rotate(90, 10, 20)", 11, 20, 10, 21},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, err := ParseTransform(tc.transform)
			if err != nil {
				t.Fatalf("ParseTransform(%q): %v", tc.transform, err)
			}
			if x, y := m.Apply(tc.x, tc.y); !svgPointNear(x, tc.wantX) || !svgPointNear(y, tc.wantY) {
				t.Errorf("Apply(%g,%g) = (%g,%g), want (%g,%g)", tc.x, tc.y, x, y, tc.wantX, tc.wantY)
			}
		})
	}
}

func svgPointNear(got, want float64) bool {
	const epsilon = 1e-9
	return got-want >= -epsilon && got-want <= epsilon
}

func TestSVG_ViewBoxTransform_PreservesAspectRatio(t *testing.T) {
	doc, err := Parse([]byte(`<svg width="200" height="200" viewBox="10 20 100 50"/>`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	m := doc.ViewBoxTransform(200, 200)
	x, y := m.Apply(10, 20)
	if x != 0 || y != 50 {
		t.Errorf("viewBox origin maps to (%v,%v), want (0,50)", x, y)
	}
	x, y = m.Apply(110, 70)
	if x != 200 || y != 150 {
		t.Errorf("viewBox end maps to (%v,%v), want (200,150)", x, y)
	}
}

func TestSVG_ViewBoxTransform_RespectsPreserveAspectRatio(t *testing.T) {
	for _, tc := range []struct {
		name, value string
		x, y        float64
	}{
		{"none", "none", 0, 0},
		{"minimum", "xMinYMin meet", 0, 0},
		{"maximum slice", "xMaxYMax slice", -200, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := Parse([]byte(`<svg width="200" height="200" viewBox="10 20 100 50" preserveAspectRatio="` + tc.value + `"/>`))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			x, y := doc.ViewBoxTransform(200, 200).Apply(10, 20)
			if x != tc.x || y != tc.y {
				t.Errorf("viewBox origin maps to (%v,%v), want (%v,%v)", x, y, tc.x, tc.y)
			}
		})
	}
}

func TestSVG_Transform_Invalid(t *testing.T) {
	for _, s := range []string{"", "spin(1)", "translate()", "scale(1,2,3)"} {
		if _, err := ParseTransform(s); err == nil {
			t.Errorf("expected error: %q", s)
		}
	}
}
