package css

import (
	"testing"
)

// Page step 1: @page size (keyword and landscape) and simple margins.
// Units: pt, px (1:1 in v1), cm, mm, and in.
func TestCSS_ParsePageRule_ResolvesPageSizeAndMargin(t *testing.T) {
	cases := []struct {
		name, css     string
		width, height float64
		margin        float64
		valid         bool
	}{
		{"A4 landscape and margin", `@page { size: A4 landscape; margin: 2cm }`, 842, 595, 2 * 72 / 2.54, true},
		{"Letter default margin", `@page { size: Letter }`, 612, 792, -1, true},
		{"margin preserves page size", `@page { margin: 2cm }`, 0, 0, 2 * 72 / 2.54, true},
		{"A6 portrait", `@page { size: A6 }`, 105 * 72 / 25.4, 148 * 72 / 25.4, -1, true},
		{"stylesheet without page rule", `p { color: red }`, 0, 0, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			width, height, margin, ok := ParsePageRule(tc.css)
			if ok != tc.valid {
				t.Fatalf("valid: got %v, want %v", ok, tc.valid)
			}
			if !ok {
				return
			}
			if !pageValueNear(width, tc.width) || !pageValueNear(height, tc.height) || !pageValueNear(margin, tc.margin) {
				t.Errorf("got %vx%v margin=%v, want %vx%v margin=%v", width, height, margin, tc.width, tc.height, tc.margin)
			}
		})
	}
}

func pageValueNear(got, want float64) bool {
	const epsilon = 1e-9
	return got-want >= -epsilon && got-want <= epsilon
}
