package boxes

import (
	"math"
	"testing"
)

func TestBoxes_ParsesMixedGridTracks(t *testing.T) {
	root, err := Build(`<div style="display:grid; grid-template-columns:100px 1fr 2fr"><span>A</span><span>B</span><span>C</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	grid := root.Children[0]
	if grid.Style.GridCols != 3 || len(grid.Style.GridTracks) != 3 {
		t.Fatalf("tracks: cols=%d count=%d, want 3", grid.Style.GridCols, len(grid.Style.GridTracks))
	}
	want := []GridTrack{{Fixed: 75}, {Fraction: 1, MinAuto: true}, {Fraction: 2, MinAuto: true}}
	for i, track := range grid.Style.GridTracks {
		if math.Abs(track.Fixed-want[i].Fixed) > 1e-9 || math.Abs(track.Fraction-want[i].Fraction) > 1e-9 || track.MinAuto != want[i].MinAuto {
			t.Errorf("track %d: got %#v, want %#v", i, track, want[i])
		}
	}
}

func TestBoxes_ParsesPercentGridTracks(t *testing.T) {
	root, err := Build(`<div style="display:grid; grid-template-columns:25% 1fr"><span>A</span><span>B</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	grid := root.Children[0]
	if grid.Style.GridCols != 2 || len(grid.Style.GridTracks) != 2 {
		t.Fatalf("tracks: cols=%d count=%d, want 2", grid.Style.GridCols, len(grid.Style.GridTracks))
	}
	if grid.Style.GridTracks[0].Percent != 25 || grid.Style.GridTracks[1].Fraction != 1 {
		t.Errorf("tracks: got %#v, want percent=25 followed by 1fr", grid.Style.GridTracks)
	}
}

func TestBoxes_ParsesMinmaxGridTrack(t *testing.T) {
	root, err := Build(`<div style="display:grid; grid-template-columns:minmax(100px, 1fr) 1fr"><span>A</span><span>B</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	grid := root.Children[0]
	if grid.Style.GridCols != 2 || len(grid.Style.GridTracks) != 2 {
		t.Fatalf("tracks: cols=%d count=%d, want 2", grid.Style.GridCols, len(grid.Style.GridTracks))
	}
	if grid.Style.GridTracks[0].Min != 75 || grid.Style.GridTracks[0].Fraction != 1 || grid.Style.GridTracks[1].Fraction != 1 {
		t.Errorf("tracks: got %#v, want minmax(75pt,1fr) and 1fr", grid.Style.GridTracks)
	}
}

func TestBoxes_ParsesPercentMinmaxGridTrack(t *testing.T) {
	root, err := Build(`<div style="display:grid;grid-template-columns:minmax(45%,1fr)"><span>A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	track := root.Children[0].Style.GridTracks[0]
	if track.MinPercent != 45 || track.Fraction != 1 {
		t.Errorf("track: min%%/fraction=%v/%v, want 45/1", track.MinPercent, track.Fraction)
	}
}

func TestBoxes_ParsesFixedGridRows(t *testing.T) {
	root, err := Build(`<div style="display:grid; grid-template-columns:1fr 1fr; grid-template-rows:30pt 40pt"><span>A</span><span>B</span><span>C</span><span>D</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	grid := root.Children[0]
	if len(grid.Style.GridRowHeights) != 2 || grid.Style.GridRowHeights[0] != 30 || grid.Style.GridRowHeights[1] != 40 {
		t.Errorf("row heights: got %v, want [30 40] pt", grid.Style.GridRowHeights)
	}
}

func TestBoxes_ParsesGridAutoRows(t *testing.T) {
	root, err := Build(`<div style="display:grid; grid-template-columns:1fr 1fr; grid-auto-rows:24pt"><span>A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if got := root.Children[0].Style.GridAutoRowHeights; len(got) != 1 || got[0] != 24 {
		t.Errorf("grid-auto-rows: got %vpt, want [24]pt", got)
	}
}

func TestBoxes_ParsesZeroPercentGridAutoRowTrack(t *testing.T) {
	root, err := Build(`<div style="display:grid;grid-template-columns:1fr;grid-auto-rows:0.0%"><span>A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	box := root.Children[0]
	if len(box.Style.GridAutoRowHeights) != 1 || box.Style.GridAutoRowHeights[0] != 0 || len(box.Style.GridAutoRowFixedZeroTracks) != 1 || !box.Style.GridAutoRowFixedZeroTracks[0] {
		t.Fatalf("zero percentage track: heights=%v flags=%v", box.Style.GridAutoRowHeights, box.Style.GridAutoRowFixedZeroTracks)
	}
}

func TestBoxes_ParsesZeroLengthGridAutoRowTrack(t *testing.T) {
	root, err := Build(`<div style="display:grid;grid-template-columns:1fr;grid-auto-rows:0pt"><span>A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	box := root.Children[0]
	if len(box.Style.GridAutoRowHeights) != 1 || box.Style.GridAutoRowHeights[0] != 0 || len(box.Style.GridAutoRowFixedZeroTracks) != 1 || !box.Style.GridAutoRowFixedZeroTracks[0] {
		t.Fatalf("zero fixed track: heights=%v flags=%v", box.Style.GridAutoRowHeights, box.Style.GridAutoRowFixedZeroTracks)
	}
}

func TestBoxes_ParsesGridAutoRowsTrackList(t *testing.T) {
	root, err := Build(`<div style="display:grid; grid-template-columns:1fr; grid-auto-rows:24pt 36pt"><span>A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	want := []float64{24, 36}
	got := root.Children[0].Style.GridAutoRowHeights
	if len(got) != len(want) {
		t.Fatalf("grid-auto-rows heights: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("track implicit %d: got %v, want %v", i, got[i], want[i])
		}
	}
}

func TestBoxes_ParsesGridAutoFlowDense(t *testing.T) {
	root, err := Build(`<div style="display:grid; grid-template-columns:1fr 1fr 1fr; grid-auto-flow:row dense"><span>A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !root.Children[0].Style.GridAutoFlowDense {
		t.Error("grid-auto-flow:row dense was not recorded")
	}
}

func TestBoxes_ParsesGridAutoFlowColumn(t *testing.T) {
	root, err := Build(`<div style="display:grid; grid-template-rows:20pt 30pt; grid-auto-flow:column"><span>A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !root.Children[0].Style.GridAutoFlowColumn {
		t.Error("grid-auto-flow:column was not recorded")
	}
}

func TestBoxes_ParsesGridAutoColumnsTrackList(t *testing.T) {
	root, err := Build(`<div style="display:grid; grid-template-columns:1fr 1fr; grid-auto-columns:50pt 60pt"><span>A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	want := []float64{50, 60}
	got := root.Children[0].Style.GridAutoColumnWidths
	if len(got) != len(want) {
		t.Fatalf("widths grid-auto-columns: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("track implicit %d: got %v, want %v", i, got[i], want[i])
		}
	}
}

func TestBoxes_ParsesPercentGridAutoColumnTracks(t *testing.T) {
	root, err := Build(`<div style="display:grid;grid-template-columns:1fr;grid-auto-columns:50% 25%"><span>A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	want := []float64{50, 25}
	got := root.Children[0].Style.GridAutoColumnPercentages
	if len(got) != len(want) {
		t.Fatalf("percentages grid-auto-columns: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("track implicit %d: got %v, want %v", i, got[i], want[i])
		}
	}
}

func TestBoxes_ParsesZeroPercentGridAutoColumnTrack(t *testing.T) {
	root, err := Build(`<div style="display:grid;grid-template-columns:1fr;grid-auto-columns:0%"><span>A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	box := root.Children[0]
	if len(box.Style.GridAutoColumnPercentages) != 1 || box.Style.GridAutoColumnPercentages[0] != 0 || len(box.Style.GridAutoColumnPercentageTracks) != 1 || !box.Style.GridAutoColumnPercentageTracks[0] {
		t.Fatalf("zero percentage track: percentages=%v flags=%v", box.Style.GridAutoColumnPercentages, box.Style.GridAutoColumnPercentageTracks)
	}
}

func TestBoxes_ParsesGridAutoColumnFractionTracks(t *testing.T) {
	root, err := Build(`<div style="display:grid;grid-template-columns:100pt;grid-auto-columns:1fr 2fr"><span>A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	want := []float64{1, 2}
	got := root.Children[0].Style.GridAutoColumnFractions
	if len(got) != len(want) {
		t.Fatalf("factores grid-auto-columns: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("track implicit %d: got %v, want %v", i, got[i], want[i])
		}
	}
}

func TestBoxes_ParsesGridAutoColumnAutomaticMinimum(t *testing.T) {
	root, err := Build(`<div style="display:grid;grid-template-columns:100pt;grid-auto-columns:1fr minmax(0pt,2fr)"><span>A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	box := root.Children[0]
	if len(box.Style.GridAutoColumnAutomaticMinimums) != 2 || !box.Style.GridAutoColumnAutomaticMinimums[0] || box.Style.GridAutoColumnAutomaticMinimums[1] {
		t.Fatalf("minimos automaticos: got %v, want [true false]", box.Style.GridAutoColumnAutomaticMinimums)
	}
}

func TestBoxes_ParsesPercentMinmaxGridAutoColumnTracks(t *testing.T) {
	root, err := Build(`<div style="display:grid;grid-template-columns:100pt;grid-auto-columns:minmax(45%,1fr) minmax(0%,2fr)"><span>A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	box := root.Children[0]
	if len(box.Style.GridAutoColumnFractions) != 2 || len(box.Style.GridAutoColumnMinimumPercentages) != 2 {
		t.Fatalf("percentage minmax tracks: fractions=%v minima=%v", box.Style.GridAutoColumnFractions, box.Style.GridAutoColumnMinimumPercentages)
	}
	wantFractions, wantMinimums := []float64{1, 2}, []float64{45, 0}
	for i := range wantFractions {
		if box.Style.GridAutoColumnFractions[i] != wantFractions[i] || box.Style.GridAutoColumnMinimumPercentages[i] != wantMinimums[i] {
			t.Errorf("track %d: fraction/minimum%%=%v/%v, want %v/%v", i, box.Style.GridAutoColumnFractions[i], box.Style.GridAutoColumnMinimumPercentages[i], wantFractions[i], wantMinimums[i])
		}
	}
}

func TestBoxes_ParsesMinmaxGridAutoColumnTracks(t *testing.T) {
	root, err := Build(`<div style="display:grid;grid-template-columns:100pt;grid-auto-columns:minmax(100pt,1fr) minmax(0pt,2fr)"><span>A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	box := root.Children[0]
	wantFractions := []float64{1, 2}
	wantMinimums := []float64{100, 0}
	if len(box.Style.GridAutoColumnFractions) != len(wantFractions) || len(box.Style.GridAutoColumnMinimums) != len(wantMinimums) {
		t.Fatalf("tracks minmax: fractions=%v minimums=%v", box.Style.GridAutoColumnFractions, box.Style.GridAutoColumnMinimums)
	}
	for i := range wantFractions {
		if box.Style.GridAutoColumnFractions[i] != wantFractions[i] || box.Style.GridAutoColumnMinimums[i] != wantMinimums[i] {
			t.Errorf("track %d: fraction/minimum=%v/%v, want %v/%v", i, box.Style.GridAutoColumnFractions[i], box.Style.GridAutoColumnMinimums[i], wantFractions[i], wantMinimums[i])
		}
	}
}

func TestBoxes_ParsesZeroLengthGridAutoColumnTrack(t *testing.T) {
	root, err := Build(`<div style="display:grid;grid-template-columns:1fr;grid-auto-columns:0pt"><span>A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	box := root.Children[0]
	if len(box.Style.GridAutoColumnWidths) != 1 || box.Style.GridAutoColumnWidths[0] != 0 || len(box.Style.GridAutoColumnFixedZeroTracks) != 1 || !box.Style.GridAutoColumnFixedZeroTracks[0] {
		t.Fatalf("zero fixed track: widths=%v flags=%v", box.Style.GridAutoColumnWidths, box.Style.GridAutoColumnFixedZeroTracks)
	}
}

func TestBoxes_ParsesAutomaticGridAutoColumnTrack(t *testing.T) {
	root, err := Build(`<div style="display:grid; grid-template-columns:1fr 1fr; grid-auto-columns:auto 60pt"><span>A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	want := []float64{0, 60}
	got := root.Children[0].Style.GridAutoColumnWidths
	if len(got) != len(want) {
		t.Fatalf("grid-auto-columns: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("track implicit %d: got %v, want %v", i, got[i], want[i])
		}
	}
}

func TestBoxes_ParsesZeroLengthExplicitGridRow(t *testing.T) {
	root, err := Build(`<div style="display:grid;grid-template-rows:20pt 0pt 30pt"><span>A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	box := root.Children[0]
	if len(box.Style.GridRowHeights) != 3 || box.Style.GridRowHeights[1] != 0 || len(box.Style.GridRowFixedZeroTracks) != 3 || !box.Style.GridRowFixedZeroTracks[1] {
		t.Fatalf("rows explicits: heights=%v zero=%v", box.Style.GridRowHeights, box.Style.GridRowFixedZeroTracks)
	}
}

func TestBoxes_ParsesMixedAutomaticAndFixedGridRows(t *testing.T) {
	root, err := Build(`<div style="display:grid; grid-template-rows:auto 30pt"><span>A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	want := []float64{0, 30}
	got := root.Children[0].Style.GridRowHeights
	if len(got) != len(want) {
		t.Fatalf("row heights: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("height row %d: got %v, want %v", i, got[i], want[i])
		}
	}
}

func TestBoxes_ParsesPercentGridColumnGap(t *testing.T) {
	root, err := Build(`<div style="display:grid; grid-template-columns:1fr 1fr; column-gap:10%"><span>A</span><span>B</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	grid := root.Children[0]
	if grid.Style.ColumnGapPercent != 10 {
		t.Errorf("percentage column gap: got %v%%, want 10%%", grid.Style.ColumnGapPercent)
	}
}

func TestBoxes_ParsesGridColumnSpan(t *testing.T) {
	root, err := Build(`<div style="display:grid; grid-template-columns:1fr 1fr"><span style="grid-column:span 2">A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if got := root.Children[0].Children[0].Style.ColSpan; got != 2 {
		t.Errorf("grid-column span: got %d, want 2", got)
	}
}

func TestBoxes_ParsesGridRowSpan(t *testing.T) {
	root, err := Build(`<div style="display:grid; grid-template-columns:1fr 1fr; grid-template-rows:30pt 40pt"><span style="grid-row:span 2">A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if got := root.Children[0].Children[0].Style.GridRowSpan; got != 2 {
		t.Errorf("grid-row span: got %d, want 2", got)
	}
}

func TestBoxes_ParsesGridColumnLine(t *testing.T) {
	root, err := Build(`<div style="display:grid; grid-template-columns:1fr 1fr"><span style="grid-column:2">A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if got := root.Children[0].Children[0].Style.GridColumnLine; got != 2 {
		t.Errorf("grid-column line: got %d, want 2", got)
	}
}

func TestBoxes_ParsesGridColumnLineRange(t *testing.T) {
	root, err := Build(`<div style="display:grid; grid-template-columns:1fr 1fr 1fr"><span style="grid-column:1 / 3">A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	item := root.Children[0].Children[0]
	if item.Style.GridColumnLine != 1 || item.Style.ColSpan != 2 {
		t.Errorf("grid-column range: line=%d span=%d, want line=1 span=2", item.Style.GridColumnLine, item.Style.ColSpan)
	}
}

func TestBoxes_ParsesNegativeGridColumnLineRange(t *testing.T) {
	root, err := Build(`<div style="display:grid; grid-template-columns:1fr 1fr 1fr"><span style="grid-column:-3 / -1">A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	item := root.Children[0].Children[0]
	if item.Style.GridColumnLine != -3 || item.Style.GridColumnEndLine != -1 {
		t.Errorf("grid-column negative range: start=%d end=%d, want -3 / -1", item.Style.GridColumnLine, item.Style.GridColumnEndLine)
	}
}

func TestBoxes_ParsesNamedGridAreas(t *testing.T) {
	root, err := Build(`<div style='display:grid; grid-template-columns:1fr 1fr; grid-template-rows:30pt 40pt; grid-template-areas:"hero hero" "left right"'><span style="grid-area:hero">A</span><span style="grid-area:left">B</span><span style="grid-area:right">C</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	grid := root.Children[0]
	want := [][]string{{"hero", "hero"}, {"left", "right"}}
	if len(grid.Style.GridAreas) != len(want) {
		t.Fatalf("grid areas: got %#v, want %#v", grid.Style.GridAreas, want)
	}
	for row := range want {
		if len(grid.Style.GridAreas[row]) != len(want[row]) {
			t.Fatalf("grid area row %d: got %v, want %v", row, grid.Style.GridAreas[row], want[row])
		}
		for col := range want[row] {
			if grid.Style.GridAreas[row][col] != want[row][col] {
				t.Errorf("grid area [%d][%d]: got %q, want %q", row, col, grid.Style.GridAreas[row][col], want[row][col])
			}
		}
	}
	for i, name := range []string{"hero", "left", "right"} {
		if got := grid.Children[i].Style.GridAreaName; got != name {
			t.Errorf("item %d grid-area: got %q, want %q", i, got, name)
		}
	}
}

func TestBoxes_ParsesGridRowLineRange(t *testing.T) {
	root, err := Build(`<div style="display:grid; grid-template-columns:1fr 1fr; grid-template-rows:24pt 36pt 18pt"><span style="grid-row:2 / 4">A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	item := root.Children[0].Children[0]
	if item.Style.GridRowLine != 2 || item.Style.GridRowSpan != 2 {
		t.Errorf("grid-row range: line=%d span=%d, want line=2 span=2", item.Style.GridRowLine, item.Style.GridRowSpan)
	}
}

func TestBoxes_ParsesNegativeGridRowLineRange(t *testing.T) {
	root, err := Build(`<div style="display:grid; grid-template-columns:1fr 1fr; grid-template-rows:24pt 36pt 18pt"><span style="grid-row:-2 / -1">A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	item := root.Children[0].Children[0]
	if item.Style.GridRowLine != -2 || item.Style.GridRowEndLine != -1 {
		t.Errorf("grid-row negative range: start=%d end=%d, want -2 / -1", item.Style.GridRowLine, item.Style.GridRowEndLine)
	}
}

func TestBoxes_ExpandsRepeatedFixedGridTracks(t *testing.T) {
	root, err := Build(`<div style="display:grid; grid-template-columns:repeat(2, 100px)"><span>A</span><span>B</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	grid := root.Children[0]
	if grid.Style.GridCols != 2 || len(grid.Style.GridTracks) != 2 {
		t.Fatalf("tracks: cols=%d count=%d, want 2", grid.Style.GridCols, len(grid.Style.GridTracks))
	}
	for i, track := range grid.Style.GridTracks {
		if track.Fixed != 75 || track.Fraction != 0 {
			t.Errorf("track %d: got %#v, want {Fixed:75 Fraction:0}", i, track)
		}
	}
}

func TestBoxes_BoundsExcessiveRepeatedGridTracks(t *testing.T) {
	root, err := Build(`<div style="display:grid; grid-template-columns:repeat(1000000000, 1fr)"><span>A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if got := root.Children[0].Style.GridCols; got != 1 {
		t.Errorf("too many tracks: got %d, want safe fallback of 1 track", got)
	}
}
