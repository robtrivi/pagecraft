package layout

import (
	"testing"

	"github.com/robtrivi/pagecraft-go/boxes"
)

func TestLayout_PageBreakBetweenBoxesCombinesBeforeAndAfter(t *testing.T) {
	tests := []struct {
		name       string
		before     boxes.BreakBeforeMode
		after      boxes.BreakAfterMode
		wantForced bool
	}{
		{name: "automatic boundary"},
		{name: "forced before", before: boxes.BreakBeforePage, wantForced: true},
		{name: "forced after", after: boxes.BreakAfterPage, wantForced: true},
		{name: "both forced values still make one boundary", before: boxes.BreakBeforePage, after: boxes.BreakAfterPage, wantForced: true},
		{name: "avoid does not force a break", after: boxes.BreakAfterAvoid},
		{name: "forced break overrides avoid", before: boxes.BreakBeforePage, after: boxes.BreakAfterAvoid, wantForced: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			previous := &boxes.Box{}
			previous.Style.BreakAfter = test.after
			next := &boxes.Box{}
			next.Style.BreakBefore = test.before
			if got := forcedPageBreakBetween(previous, next); got != test.wantForced {
				t.Errorf("forcedPageBreakBetween()=%t, want %t", got, test.wantForced)
			}
		})
	}
}

func TestLayout_BreakInsideAvoidKeepsContentOnlyWhenItFitsFreshPage(t *testing.T) {
	tests := []struct {
		name     string
		mode     boxes.BreakInsideMode
		extent   float64
		capacity float64
		wantKeep bool
	}{
		{name: "automatic", extent: 8, capacity: 10},
		{name: "avoid fits", mode: boxes.BreakInsideAvoid, extent: 8, capacity: 10, wantKeep: true},
		{name: "avoid fits exactly", mode: boxes.BreakInsideAvoid, extent: 10, capacity: 10, wantKeep: true},
		{name: "avoid cannot fit", mode: boxes.BreakInsideAvoid, extent: 11, capacity: 10},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := breakInsideAvoidFits(test.mode, test.extent, test.capacity); got != test.wantKeep {
				t.Errorf("breakInsideAvoidFits()=%t, want %t", got, test.wantKeep)
			}
		})
	}
}

func TestLayout_PlanFragmentsFitsMeasuredUnitsIntoAvailablePages(t *testing.T) {
	lineHeights := []float64{3, 3, 3, 3, 3}
	units := make([]fragmentUnit, len(lineHeights))
	fragments := planFragments(units, 7, 2, 12, false, func(first, end int) float64 {
		height := 0.0
		for _, lineHeight := range lineHeights[first:end] {
			height += lineHeight
		}
		return height
	})
	want := []fragmentRange{
		{first: 0, end: 1, pageOffset: 0, y: 7, height: 3},
		{first: 1, end: 4, pageOffset: 1, y: 2, height: 9},
		{first: 4, end: 5, pageOffset: 2, y: 2, height: 3},
	}
	assertFragmentRanges(t, fragments, want)
}

func TestLayout_FragmentContinuationResumesTheSamePagePlan(t *testing.T) {
	units := []fragmentUnit{{height: 4}, {height: 7}, {height: 3}}
	want := planFragments(units, 2, 1, 10, true, nil)

	token := fragmentBreakToken{nextUnit: 0, pageOffset: 0, y: 2, pageHasContent: true}
	var got []fragmentRange
	for token.nextUnit < len(units) {
		fragment, next := planNextFragment(units, token, 1, 10, nil)
		if next.nextUnit <= token.nextUnit {
			t.Fatalf("continuation did not advance: before=%+v after=%+v", token, next)
		}
		got = append(got, fragment)
		token = next
	}

	assertFragmentRanges(t, got, want)
	if token.nextUnit != len(units) || token.pageOffset != 2 || token.y != 4 || !token.pageHasContent {
		t.Errorf("final continuation=%+v, want end of unit 3 on page 3 at y=4", token)
	}
}

func TestLayout_PlanFragmentsRetriesForMinimumLinesBeforeBreak(t *testing.T) {
	units := make([]fragmentUnit, 5)
	for i := range units {
		units[i] = fragmentUnit{group: 1, orphans: 2, widows: 2}
	}
	fragments := planFragments(units, 7, 2, 12, false, func(first, end int) float64 {
		return float64(end-first) * 3
	})
	want := []fragmentRange{
		{first: 0, end: 3, pageOffset: 1, y: 2, height: 9},
		{first: 3, end: 5, pageOffset: 2, y: 2, height: 6},
	}
	assertFragmentRanges(t, fragments, want)
}

func TestLayout_PlanFragmentsHonorsForcedPageBreak(t *testing.T) {
	fragments := planFragments([]fragmentUnit{
		{height: 3},
		{height: 3, breakBefore: fragmentBreakPage},
	}, 0, 0, 10, false, nil)
	want := []fragmentRange{
		{first: 0, end: 1, pageOffset: 0, y: 0, height: 3},
		{first: 1, end: 2, pageOffset: 1, y: 0, height: 3},
	}
	assertFragmentRanges(t, fragments, want)
}

func TestLayout_PlanFragmentsForcedBreakOverridesWidowConstraint(t *testing.T) {
	fragments := planFragments([]fragmentUnit{
		{height: 2, group: 1, orphans: 2, widows: 2, breakAfter: fragmentBreakPage},
		{height: 2, group: 1, orphans: 2, widows: 2},
	}, 3, 0, 10, false, nil)
	want := []fragmentRange{
		{first: 0, end: 1, pageOffset: 0, y: 3, height: 2},
		{first: 1, end: 2, pageOffset: 1, y: 0, height: 2},
	}
	assertFragmentRanges(t, fragments, want)
}

func TestLayout_PlanFragmentsMovesWholeUnitToFreshPage(t *testing.T) {
	fragments := planFragments([]fragmentUnit{{height: 6}}, 3, 0, 8, false, nil)
	want := []fragmentRange{{first: 0, end: 1, pageOffset: 1, y: 0, height: 6}}
	assertFragmentRanges(t, fragments, want)
}

func TestLayout_PlanFragmentsKeepsAvoidGroupTogetherWhenItFitsFreshPage(t *testing.T) {
	fragments := planFragments([]fragmentUnit{
		{height: 4},
		{height: 4, group: 1, breakInsideAvoid: true},
		{height: 4, group: 1, breakInsideAvoid: true},
	}, 0, 0, 10, false, nil)
	want := []fragmentRange{
		{first: 0, end: 1, pageOffset: 0, y: 0, height: 4},
		{first: 1, end: 3, pageOffset: 1, y: 0, height: 8},
	}
	assertFragmentRanges(t, fragments, want)
}

func TestLayout_PlanFragmentsKeepsAvoidBoundaryTogetherWhenItFitsFreshPage(t *testing.T) {
	tests := []struct {
		name  string
		left  fragmentUnit
		right fragmentUnit
	}{
		{name: "before", left: fragmentUnit{height: 4}, right: fragmentUnit{height: 4, breakBefore: fragmentBreakAvoid}},
		{name: "after", left: fragmentUnit{height: 4, breakAfter: fragmentBreakAvoid}, right: fragmentUnit{height: 4}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fragments := planFragments([]fragmentUnit{test.left, test.right}, 3, 0, 10, false, nil)
			want := []fragmentRange{{first: 0, end: 2, pageOffset: 1, y: 0, height: 8}}
			assertFragmentRanges(t, fragments, want)
		})
	}
}

func TestLayout_PlanFragmentsRelaxesAvoidForOversizedGroup(t *testing.T) {
	fragments := planFragments([]fragmentUnit{
		{height: 6, group: 1, breakInsideAvoid: true},
		{height: 6, group: 1, breakInsideAvoid: true},
	}, 0, 0, 10, false, nil)
	want := []fragmentRange{
		{first: 0, end: 1, pageOffset: 0, y: 0, height: 6},
		{first: 1, end: 2, pageOffset: 1, y: 0, height: 6},
	}
	assertFragmentRanges(t, fragments, want)
}

func TestLayout_PlanFragmentsHonorsWidowsAndOrphans(t *testing.T) {
	units := make([]fragmentUnit, 6)
	for i := range units {
		units[i] = fragmentUnit{height: 1, group: 1, orphans: 2, widows: 2}
	}
	fragments := planFragments(units, 3, 0, 4, false, nil)
	want := []fragmentRange{
		{first: 0, end: 4, pageOffset: 1, y: 0, height: 4},
		{first: 4, end: 6, pageOffset: 2, y: 0, height: 2},
	}
	assertFragmentRanges(t, fragments, want)
}

func assertFragmentRanges(t *testing.T, got, want []fragmentRange) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("fragment count=%d, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].first != want[i].first || got[i].end != want[i].end || got[i].pageOffset != want[i].pageOffset || got[i].y != want[i].y || got[i].height != want[i].height {
			t.Errorf("fragment %d=%+v, want %+v", i, got[i], want[i])
		}
	}
}
