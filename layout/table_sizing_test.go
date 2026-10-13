package layout

import (
	"math"
	"testing"
)

func TestLayout_AutoTableDistributesEmptyTracksEvenly(t *testing.T) {
	widths := []float64{0, 0}
	intrinsic := []tableColumnIntrinsic{{}, {}}
	percentages := []float64{0, 0}
	fixed := []float64{0, 0}

	distributeTableAutoWidths(widths, intrinsic, percentages, fixed, 96)

	for index, got := range widths {
		if math.Abs(got-48) > 0.001 {
			t.Errorf("track %d width=%g, want 48pt", index, got)
		}
	}
}
