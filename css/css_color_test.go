package css

import (
	"testing"
)

// CSS colors convert to normalized RGBA. The cases cover supported syntax
// families and the invalid-value boundary of the implemented subset.
func TestCSS_ParseColor_FormatsAndAlpha(t *testing.T) {
	cases := []struct {
		in         string
		r, g, b, a float64
		ok         bool
	}{
		{"#f00", 1, 0, 0, 1, true},
		{"#ff0000", 1, 0, 0, 1, true},
		{"#f008", 1, 0, 0, 0x88 / 255.0, true},
		{"#ff000080", 1, 0, 0, 0x80 / 255.0, true},
		{"red", 1, 0, 0, 1, true},
		{"white", 1, 1, 1, 1, true},
		{"rgb(0,128,255)", 0, 128.0 / 255, 1, 1, true},
		{"rgba(255, 0, 0, 0.5)", 1, 0, 0, 0.5, true},
		{"rgba(0,0,0,1)", 0, 0, 0, 1, true},
		{"transparent", 0, 0, 0, 0, false},
		{"rgba(0,0,0,2)", 0, 0, 0, 0, false},
		{"not-a-color", 0, 0, 0, 0, false},
		{"", 0, 0, 0, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			c, ok := ParseColor(tc.in)
			if ok != tc.ok {
				t.Fatalf("ok: got %v, want %v", ok, tc.ok)
			}
			if !ok {
				return
			}
			if !close4(c, [4]float64{tc.r, tc.g, tc.b, tc.a}) {
				t.Errorf("got %v, want [%v %v %v %v]", c, tc.r, tc.g, tc.b, tc.a)
			}
		})
	}
}

func close4(c, w [4]float64) bool {
	const eps = 1e-9
	for i := range c {
		if d := c[i] - w[i]; d < -eps || d > eps {
			return false
		}
	}
	return true
}
