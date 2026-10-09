package layout

import "github.com/robtrivi/pagecraft-go/boxes"

const fragmentEpsilon = 1e-7

type fragmentBreak uint8

const (
	fragmentBreakAuto fragmentBreak = iota
	fragmentBreakAvoid
	fragmentBreakPage
)

// fragmentUnit is one indivisible layout unit, usually a line or a table row.
// Units in the same group can share keep and widow/orphan constraints.
type fragmentUnit struct {
	height           float64
	group            int
	breakBefore      fragmentBreak
	breakAfter       fragmentBreak
	breakInsideAvoid bool
	orphans          int
	widows           int
}

func pageBreakValue(value boxes.BreakBeforeMode) fragmentBreak {
	if value == boxes.BreakBeforePage {
		return fragmentBreakPage
	}
	return fragmentBreakAuto
}

// forcedPageBreakBetween resolves the supported forced page-break values at a
// sibling boundary. Layout modes still decide which boxes form that boundary.
func forcedPageBreakBetween(previous, next *boxes.Box) bool {
	return (previous != nil && previous.Style.BreakAfter == boxes.BreakAfterPage) ||
		(next != nil && pageBreakValue(next.Style.BreakBefore) == fragmentBreakPage)
}

// breakInsideAvoidFits reports whether an avoided unit fits in a fresh page.
// An oversized unit must be allowed to fragment so layout can make progress.
func breakInsideAvoidFits(mode boxes.BreakInsideMode, extent, freshPageCapacity float64) bool {
	return mode == boxes.BreakInsideAvoid && extent <= freshPageCapacity+fragmentEpsilon
}

type fragmentRange struct {
	first, end int
	pageOffset int
	y, height  float64
}

// fragmentBreakToken records the next unplaced unit and the fragmentainer
// geometry needed to resume planning at that boundary.
type fragmentBreakToken struct {
	nextUnit       int
	pageOffset     int
	y              float64
	pageHasContent bool
}

// planFragments applies page break constraints to the measured sequence. Layout
// modes provide their own units and sizes; this function owns page advancement.
func planFragments(units []fragmentUnit, y, pageTop, pageBottom float64, pageHasContent bool, measure func(first, end int) float64) []fragmentRange {
	fragments := make([]fragmentRange, 0, 1)
	token := fragmentBreakToken{y: y, pageHasContent: pageHasContent}
	for token.nextUnit < len(units) {
		fragment, next := planNextFragment(units, token, pageTop, pageBottom, measure)
		fragments = append(fragments, fragment)
		token = next
	}
	return fragments
}

// planNextFragment chooses one fragment and returns the state required to
// continue at the next unit. Break retries stay inside this operation so the
// caller only observes accepted fragment boundaries.
func planNextFragment(units []fragmentUnit, token fragmentBreakToken, pageTop, pageBottom float64, measure func(first, end int) float64) (fragmentRange, fragmentBreakToken) {
	for {
		first := token.nextUnit
		if forcedBreakBefore(units, first) && (token.pageHasContent || token.y > pageTop+fragmentEpsilon) {
			token.pageOffset++
			token.y = pageTop
			token.pageHasContent = false
		}
		if moveKeepGroupToFreshPage(units, first, token.y, pageTop, pageBottom, token.pageHasContent, measure) {
			token.pageOffset++
			token.y = pageTop
			token.pageHasContent = false
			continue
		}
		if (token.pageHasContent || token.y > pageTop+fragmentEpsilon) && measureFragmentRange(units, first, first+1, measure) > pageBottom-token.y+fragmentEpsilon {
			token.pageOffset++
			token.y = pageTop
			token.pageHasContent = false
			continue
		}

		available := pageBottom - token.y
		end, height := first, 0.0
		for end < len(units) {
			if end > first && forcedBreakBefore(units, end) {
				break
			}
			if end > first {
				groupStart, groupEnd, keep := keepRange(units, end)
				if groupStart < end {
					groupStart = end
				}
				groupHeight := measureFragmentRange(units, groupStart, groupEnd, measure)
				if keep && groupHeight <= pageBottom-pageTop+fragmentEpsilon && height+groupHeight > available+fragmentEpsilon {
					break
				}
			}
			candidate := measureFragmentRange(units, first, end+1, measure)
			if end > first && candidate > available+fragmentEpsilon {
				break
			}
			height = candidate
			end++
			if units[end-1].breakAfter == fragmentBreakPage {
				break
			}
		}
		if end == first { // An oversized unit must still make progress.
			height = units[first].height
			end++
		}

		if adjusted, adjustedHeight, retry := enforceWidowsOrphans(units, first, end, height, token.y, pageTop, measure); retry {
			token.pageOffset++
			token.y = pageTop
			token.pageHasContent = false
			continue
		} else if adjusted > 0 {
			end, height = adjusted, adjustedHeight
		}

		fragment := fragmentRange{first: first, end: end, pageOffset: token.pageOffset, y: token.y, height: height}
		next := fragmentBreakToken{
			nextUnit:       end,
			pageOffset:     token.pageOffset,
			y:              token.y + height,
			pageHasContent: true,
		}
		if end < len(units) {
			next.pageOffset++
			next.y = pageTop
			next.pageHasContent = false
		}
		return fragment, next
	}
}

func placeFragmentUnit(unit fragmentUnit, y, pageTop, pageBottom float64, pageHasContent bool, pages *[]Page) fragmentRange {
	fragment := planFragments([]fragmentUnit{unit}, y, pageTop, pageBottom, pageHasContent, nil)[0]
	ensurePageIndex(pages, len(*pages)-1+fragment.pageOffset)
	return fragment
}

func forcedBreakBefore(units []fragmentUnit, index int) bool {
	return units[index].breakBefore == fragmentBreakPage || (index > 0 && units[index-1].breakAfter == fragmentBreakPage)
}

func keepBoundary(units []fragmentUnit, index int) bool {
	if index <= 0 || index >= len(units) {
		return false
	}
	left, right := units[index-1], units[index]
	return right.breakBefore == fragmentBreakAvoid || left.breakAfter == fragmentBreakAvoid ||
		(left.group != 0 && left.group == right.group && (left.breakInsideAvoid || right.breakInsideAvoid))
}

func keepRange(units []fragmentUnit, index int) (int, int, bool) {
	start, end := index, index+1
	for start > 0 && keepBoundary(units, start) && !forcedBreakBefore(units, start) {
		start--
	}
	for end < len(units) && keepBoundary(units, end) && !forcedBreakBefore(units, end) {
		end++
	}
	kept := end-start > 1 || (units[index].group != 0 && units[index].breakInsideAvoid)
	return start, end, kept
}

func moveKeepGroupToFreshPage(units []fragmentUnit, start int, y, top, bottom float64, pageHasContent bool, measure func(first, end int) float64) bool {
	groupStart, groupEnd, keep := keepRange(units, start)
	if (y <= top+fragmentEpsilon && !pageHasContent) || !keep {
		return false
	}
	if groupStart < start {
		groupStart = start
	}
	groupHeight := measureFragmentRange(units, groupStart, groupEnd, measure)
	return groupHeight <= bottom-top+fragmentEpsilon && groupHeight > bottom-y+fragmentEpsilon
}

func measureFragmentRange(units []fragmentUnit, first, end int, measure func(first, end int) float64) float64 {
	if measure != nil {
		return measure(first, end)
	}
	height := 0.0
	for _, unit := range units[first:end] {
		height += unit.height
	}
	return height
}

func enforceWidowsOrphans(units []fragmentUnit, first, end int, height, y, top float64, measure func(first, end int) float64) (int, float64, bool) {
	if end >= len(units) || forcedBreakBefore(units, end) || units[end-1].group == 0 || units[end-1].group != units[end].group {
		return 0, height, false
	}
	group := units[end].group
	groupStart, groupEnd := end, end+1
	for groupStart > first && units[groupStart-1].group == group {
		groupStart--
	}
	for groupEnd < len(units) && units[groupEnd].group == group {
		groupEnd++
	}
	orphans, widows := units[end-1].orphans, units[end-1].widows
	if orphans <= 0 {
		orphans = 2
	}
	if widows <= 0 {
		widows = 2
	}
	if candidate, candidateHeight, ok := widowOrphanCut(units, first, end, group, groupStart, groupEnd, orphans, widows, measure); ok {
		return candidate, candidateHeight, false
	}
	// Match Chromium's useful fallback when the widows value leaves no
	// valid break: preserve two lines on each side if the content permits it.
	if widows > 2 && orphans == 2 {
		if candidate, candidateHeight, ok := widowOrphanCut(units, first, end, group, groupStart, groupEnd, 2, 2, measure); ok {
			return candidate, candidateHeight, false
		}
	}
	if y > top+fragmentEpsilon {
		return 0, height, true
	}
	return 0, height, false // Relax constraints on an otherwise empty page.
}

func widowOrphanCut(units []fragmentUnit, first, end, group, groupStart, groupEnd, orphans, widows int, measure func(first, end int) float64) (int, float64, bool) {
	for candidate := end; candidate > first; candidate-- {
		if units[candidate-1].group != group || candidate == groupEnd {
			continue
		}
		before, after := candidate-groupStart, groupEnd-candidate
		if before >= orphans && after >= widows {
			candidateHeight := measureFragmentRange(units, first, candidate, measure)
			return candidate, candidateHeight, true
		}
	}
	return 0, 0, false
}

func ensurePageIndex(pages *[]Page, pageIndex int) {
	for len(*pages) <= pageIndex {
		*pages = append(*pages, Page{Number: len(*pages) + 1})
	}
}
