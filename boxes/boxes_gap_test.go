package boxes

import (
	"math"
	"testing"
)

func TestBoxes_ParsesGapAndLonghands(t *testing.T) {
	cases := []struct {
		name, style string
		wantRow     float64
		wantColumn  float64
	}{
		{"one_value", "gap: 10px", 7.5, 7.5},
		{"two_values", "gap: 4px 8pt", 3, 8},
		{"longhands_override", "gap: 4px 8pt; row-gap: 6pt; column-gap: 10px", 6, 7.5},
		{"negative_is_ignored", "gap: -2px", 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, err := Build(`<div style="display: grid; ` + tc.style + `"><span>A</span></div>`)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			grid := root.Children[0]
			if math.Abs(grid.Style.RowGap-tc.wantRow) > 1e-9 || math.Abs(grid.Style.ColumnGap-tc.wantColumn) > 1e-9 {
				t.Errorf("gaps: row=%v column=%v, want %v %v", grid.Style.RowGap, grid.Style.ColumnGap, tc.wantRow, tc.wantColumn)
			}
		})
	}
}
