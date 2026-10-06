package text

import (
	"testing"
)

// Real Helvetica AFM widths in
// thousandths of an em and measurements in points. Adobe reference values:
// i=222, m=833, space=278, H=722, e with acute=556.
func TestText_Advance_KnownWidths(t *testing.T) {
	cases := []struct {
		r    rune
		want float64
	}{
		{'i', 222}, {'m', 833}, {' ', 278}, {'H', 722}, {'é', 556},
	}
	for _, tc := range cases {
		if got := Advance(tc.r); got != tc.want {
			t.Errorf("Advance(%q): got %v, want %v", tc.r, got, tc.want)
		}
	}
}

// Bold (official phvb8a.afm): L=611, m=889, H=722, e with acute=556.
// Not kerning in v1 (the separate bold KPX table is documented).
func TestText_BoldAdvances_KnownWidths(t *testing.T) {
	for _, tc := range []struct {
		r    rune
		want float64
	}{
		{'L', 611}, {'m', 889}, {'H', 722}, {'é', 556},
	} {
		if got := AdvanceBold(tc.r); got != tc.want {
			t.Errorf("AdvanceBold(%q): got %v, want %v", tc.r, got, tc.want)
		}
	}
}

func TestText_Kern_KnownPairs(t *testing.T) {
	for _, tc := range []struct {
		a, b rune
		want float64
	}{
		{'T', 'o', -120}, {'A', 'V', -70}, {'f', ',', -30}, {'H', 'i', 0},
	} {
		if got := Kern(tc.a, tc.b); got != tc.want {
			t.Errorf("Kern(%q,%q): got %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestText_MeasureComposesRunsAndScalesLinearly(t *testing.T) {
	measurements := []struct {
		name    string
		measure func(string, float64) float64
		kern    bool
	}{
		{"regular", Measure, true}, {"bold", MeasureBold, false},
	}
	pairs := [][2]string{{"", "text"}, {"A", "V"}, {"To", "gether"}, {"café", " au lait"}, {"text ", "with ☃"}}
	for _, font := range measurements {
		for _, pair := range pairs {
			left, right := pair[0], pair[1]
			boundaryKern := 0.0
			leftRunes, rightRunes := []rune(left), []rune(right)
			if font.kern && len(leftRunes) > 0 && len(rightRunes) > 0 {
				boundaryKern = Kern(leftRunes[len(leftRunes)-1], rightRunes[0]) / 1000
			}
			combined := left + right
			for _, size := range []float64{0, 6, 10, 18} {
				whole := font.measure(combined, size)
				parts := font.measure(left, size) + font.measure(right, size) + boundaryKern*size
				if !near(whole, parts) || !near(whole, font.measure(combined, 1)*size) {
					t.Errorf("%s Measure(%q+%q, %g)=%g; parts=%g and linear scale=%g", font.name, left, right, size, whole, parts, font.measure(combined, 1)*size)
				}
			}
		}
	}
}

func near(got, want float64) bool {
	const epsilon = 1e-9
	return got-want >= -epsilon && got-want <= epsilon
}
