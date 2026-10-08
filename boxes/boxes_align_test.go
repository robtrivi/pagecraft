package boxes

import (
	"testing"
)

// Text-align step 1: left/center/right/justify; invalid or absent defaults to left.
func TestBoxes_ParsesTextAlign(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want string
	}{
		{"center", "center"}, {"right", "right"}, {"justify", "justify"},
		{"left", "left"}, {"", "left"}, {"diagonal", "left"},
	} {
		root, err := Build(`<p style="text-align: ` + tc.in + `">x</p>`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := root.Children[0].Style.Align; got != tc.want {
			t.Errorf("text-align %q: got %q, want %q", tc.in, got, tc.want)
		}
	}
}
