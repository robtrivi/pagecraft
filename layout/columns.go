package layout

import (
	"math"
	"sort"

	"github.com/robtrivi/pagecraft/boxes"
)

type columnItemLayout struct {
	lines     []inlineStream
	flowLines []inlineLine
	height    float64
}

type columnContentRun struct {
	kids           []*boxes.Box
	layouts        []columnItemLayout
	units          []fragmentUnit
	contentHeight  float64
	implicitBreaks int
	forcedPage     bool
}

type columnRowCursor struct {
	pageIndex int
	rowIndex  int
	top       float64
	bottom    float64
	options   Options
}

type columnWrapperGeometry struct {
	contentOffset float64
	contentWidth  float64
	boxOffsets    []float64
	boxWidths     []float64
}

func columnWrapperGeometryFor(wrappers []*boxes.Box, columnWidth float64) columnWrapperGeometry {
	geometry := columnWrapperGeometry{contentWidth: columnWidth, boxOffsets: make([]float64, len(wrappers)), boxWidths: make([]float64, len(wrappers))}
	for index, wrapper := range wrappers {
		available := geometry.contentWidth
		if wrapper.Style.Sizing.Width.IsAuto() {
			available = math.Max(0, available-wrapper.Style.MarginLeft-wrapper.Style.MarginRight)
		}
		width := usedWidth(wrapper, available)
		resolved := resolveHorizontalAutoMarginsWithWidth(wrapper, geometry.contentWidth, width)
		geometry.boxOffsets[index] = geometry.contentOffset + resolved.Style.MarginLeft
		geometry.boxWidths[index] = width
		inset := boxInsets(wrapper)
		geometry.contentOffset = geometry.boxOffsets[index] + inset.left
		geometry.contentWidth = math.Max(0, width-inset.left-inset.right)
	}
	return geometry
}

// Column sets flow content into equal-width fragmentainers separated by a gap.

// flowColumns places children in N columns from and and returns the cursor
// (the end of the tallest column).
func flowColumns(c *boxes.Box, y float64, pages *[]Page, opts Options, usable, bottom float64) float64 {
	y += c.Style.MarginTop
	items := columnItemsContainingSpanners(c)
	var wrapperSizing columnWrapperSizing
	for start := 0; start < len(items); {
		span := start
		for span < len(items) && !items[span].Style.ColumnSpanAll {
			span++
		}
		balance := c.Style.ColumnFill != boxes.ColumnFillAuto || span < len(items)
		segment := items[start:span]
		segmentStartY := y
		wrappers := commonColumnSpannerWrappers(c, segment)
		if len(segment) == 0 && span < len(items) {
			wrappers = columnSpannerWrapperPath(c, items[span])
		}
		segmentInsets := columnWrapperInsets(wrappers)
		segmentTop := y
		if start == 0 {
			segmentTop += segmentInsets.top + columnWrapperMarginTop(wrappers)
		}
		blockStarts := columnPageBlockCounts(*pages)
		segmentBottom := bottom
		if span == len(items) {
			segmentBottom -= segmentInsets.bottom + columnWrapperMarginBottom(wrappers)
		}
		gap := c.Style.ColumnGap + usable*c.Style.ColumnGapPercent/100
		columnCount := usedColumnCount(c.Style, usable, gap)
		columnWidth := (usable - float64(columnCount-1)*gap) / float64(columnCount)
		wrapperGeometry := columnWrapperGeometryFor(wrappers, columnWidth)
		y = flowColumnSegment(c, segment, segmentTop, pages, opts, usable, segmentBottom, wrappers, balance)
		if len(segment) == 0 && len(wrappers) > 0 {
			y = emitEmptyColumnWrapperFragment(wrappers, wrapperGeometry, segmentStartY, pages, opts, &wrapperSizing)
		} else if len(wrappers) > 0 {
			wrapColumnSegmentFragments(wrappers, wrapperGeometry, pages, blockStarts, segmentTop, opts, start == 0, span == len(items), &wrapperSizing)
			if span == len(items) {
				y += segmentInsets.bottom + columnWrapperMarginBottom(wrappers)
			}
		}
		if span == len(items) {
			break
		}
		y = flowColumnSpan(items[span], y, pages, opts, usable, bottom)
		start = span + 1
		if start < len(items) && items[span].Style.BreakAfter == boxes.BreakAfterPage {
			pageHasContent := (*pages)[len(*pages)-1].blockCount() > 0
			fragment := placeFragmentUnitForPage(fragmentUnit{breakBefore: fragmentBreakPage}, y, opts.top(), bottom, pageHasContent, pages, opts)
			y = fragment.y
		}
	}
	y += wrapperSizing.finish(opts)
	return y + c.Style.MarginBottom
}

func columnItemsContainingSpanners(container *boxes.Box) []*boxes.Box {
	var items []*boxes.Box
	for _, item := range nestedContainerItems(container) {
		if !item.Style.ColumnSpanAll && canUnwrapColumnSpannerPath(item) {
			items = append(items, columnItemsContainingSpanners(item)...)
			continue
		}
		items = append(items, item)
	}
	return items
}

func canUnwrapColumnSpannerPath(box *boxes.Box) bool {
	if box == nil || box.Style.Display != boxes.DisplayBlock || box.Style.Position != boxes.PositionStatic ||
		box.Style.Float != boxes.FloatNone || box.Style.Clear != boxes.ClearNone || isMultiColumn(box) ||
		box.Style.Display == boxes.DisplayFlex || box.Style.Display == boxes.DisplayGrid || isTableFormattingContext(box) ||
		box.Style.HasOpacity || clipsOverflow(box) ||
		box.Style.BreakBefore != boxes.BreakBeforeAuto || box.Style.BreakAfter != boxes.BreakAfterAuto || box.Style.BreakInside != boxes.BreakInsideAuto {
		return false
	}
	for _, child := range nestedContainerItems(box) {
		if child.Style.ColumnSpanAll || hasDescendantColumnSpanner(child) {
			return true
		}
	}
	return false
}

func hasDescendantColumnSpanner(box *boxes.Box) bool {
	for _, child := range box.Children {
		if child.Style.ColumnSpanAll || hasDescendantColumnSpanner(child) {
			return true
		}
	}
	return false
}

func commonColumnSpannerWrappers(container *boxes.Box, items []*boxes.Box) []*boxes.Box {
	if len(items) == 0 {
		return nil
	}
	var path []*boxes.Box
	for ancestor := items[0].Parent; ancestor != nil && ancestor != container; ancestor = ancestor.Parent {
		if canUnwrapColumnSpannerPath(ancestor) {
			path = append(path, ancestor)
		}
	}
	for left, right := 0, len(path)-1; left < right; left, right = left+1, right-1 {
		path[left], path[right] = path[right], path[left]
	}
	common := path[:0]
	for _, wrapper := range path {
		containsAll := true
		for _, item := range items {
			if !columnItemDescendsFrom(item, wrapper) {
				containsAll = false
				break
			}
		}
		if containsAll {
			common = append(common, wrapper)
		}
	}
	return common
}

func columnSpannerWrapperPath(container, spanner *boxes.Box) []*boxes.Box {
	var path []*boxes.Box
	for ancestor := spanner.Parent; ancestor != nil && ancestor != container; ancestor = ancestor.Parent {
		if canUnwrapColumnSpannerPath(ancestor) {
			path = append(path, ancestor)
		}
	}
	for left, right := 0, len(path)-1; left < right; left, right = left+1, right-1 {
		path[left], path[right] = path[right], path[left]
	}
	return path
}

func columnItemDescendsFrom(item, ancestor *boxes.Box) bool {
	for current := item; current != nil; current = current.Parent {
		if current == ancestor {
			return true
		}
	}
	return false
}

func columnWrapperInsets(wrappers []*boxes.Box) containerInsets {
	var result containerInsets
	for _, wrapper := range wrappers {
		inset := boxInsets(wrapper)
		result.top += inset.top
		result.right += inset.right
		result.bottom += inset.bottom
		result.left += inset.left
	}
	return result
}

func columnWrapperMarginTop(wrappers []*boxes.Box) float64 {
	var margin float64
	for _, wrapper := range wrappers {
		margin += wrapper.Style.MarginTop
	}
	return margin
}

func columnWrapperMarginBottom(wrappers []*boxes.Box) float64 {
	var margin float64
	for _, wrapper := range wrappers {
		margin += wrapper.Style.MarginBottom
	}
	return margin
}

func columnPageBlockCounts(pages []Page) []int {
	counts := make([]int, len(pages))
	for index := range pages {
		counts[index] = pages[index].blockCount()
	}
	return counts
}

type columnWrapperFragmentGroup struct {
	pageIndex int
	column    int
	start     int
	end       int
	left      float64
	top       float64
	bottom    float64
}

type columnWrapperSizing struct {
	heights map[*boxes.Box]float64
	last    map[*boxes.Box]*PlacedBox
}

func emitEmptyColumnWrapperFragment(wrappers []*boxes.Box, geometry columnWrapperGeometry, y float64, pages *[]Page, opts Options, sizing *columnWrapperSizing) float64 {
	starts := make([]float64, len(wrappers))
	cursor := y
	for index, wrapper := range wrappers {
		cursor += wrapper.Style.MarginTop
		starts[index] = cursor
		cursor += boxInsets(wrapper).top
	}
	heights := make([]float64, len(wrappers))
	for index := len(wrappers) - 1; index >= 0; index-- {
		wrapper := wrappers[index]
		contentHeight := math.Max(0, cursor-starts[index]-boxInsets(wrapper).top)
		height := cursor - starts[index]
		if hasHeightSizing(wrapper) {
			height = math.Max(height, usedHeight(wrapper, contentHeight, opts))
		}
		heights[index] = height
		if starts[index]+height > cursor {
			cursor = starts[index] + height
		}
	}
	page := &(*pages)[len(*pages)-1]
	var parent *fragmentOccurrence
	for index, wrapper := range wrappers {
		fragment := fragmentBox(wrapper, true, false)
		_, font := measureFor(opts, wrapper)
		block := placed(fragment, "", nil, opts.left()+geometry.boxOffsets[index], starts[index], geometry.boxWidths[index], heights[index], resolveSize(wrapper, opts), font)
		block.FragmentainerIndex = 1
		block.fragmentRef = newFragmentOccurrence(wrapper)
		if parent != nil {
			bindFragmentChild(parent, &block)
		}
		page.emitBlock(block)
		placedFragment := page.block(page.blockCount() - 1)
		sizing.add(wrapper, placedFragment.Height, nil)
		parent = block.fragmentRef
	}
	return cursor
}

func (sizing *columnWrapperSizing) add(wrapper *boxes.Box, height float64, last *PlacedBox) {
	if sizing.heights == nil {
		sizing.heights = make(map[*boxes.Box]float64)
		sizing.last = make(map[*boxes.Box]*PlacedBox)
	}
	sizing.heights[wrapper] += height
	if last != nil {
		sizing.last[wrapper] = last
	}
}

func (sizing *columnWrapperSizing) finish(opts Options) float64 {
	wrappers := make([]*boxes.Box, 0, len(sizing.heights))
	for wrapper := range sizing.heights {
		wrappers = append(wrappers, wrapper)
	}
	depth := func(wrapper *boxes.Box) int {
		depth := 0
		for current := wrapper; current != nil; current = current.Parent {
			depth++
		}
		return depth
	}
	sort.SliceStable(wrappers, func(i, j int) bool { return depth(wrappers[i]) > depth(wrappers[j]) })
	var extra float64
	for _, wrapper := range wrappers {
		height := sizing.heights[wrapper]
		if !hasHeightSizing(wrapper) || sizing.last[wrapper] == nil {
			continue
		}
		inset := boxInsets(wrapper)
		used := usedHeight(wrapper, math.Max(0, height-inset.top-inset.bottom), opts)
		delta := math.Max(0, used-height)
		if delta > 0 {
			sizing.last[wrapper].Height += delta
			extra += delta
			for ancestor := wrapper.Parent; ancestor != nil; ancestor = ancestor.Parent {
				if _, exists := sizing.heights[ancestor]; exists {
					sizing.heights[ancestor] += delta
				}
			}
		}
	}
	return extra
}

func wrapColumnSegmentFragments(wrappers []*boxes.Box, geometry columnWrapperGeometry, pages *[]Page, starts []int, segmentTop float64, opts Options, first, last bool, sizing *columnWrapperSizing) {
	if len(wrappers) == 0 {
		return
	}
	groupsByKey := make(map[[2]int]*columnWrapperFragmentGroup)
	var groups []*columnWrapperFragmentGroup
	for pageIndex := range *pages {
		start := 0
		if pageIndex < len(starts) {
			start = starts[pageIndex]
		}
		page := &(*pages)[pageIndex]
		for blockIndex := start; blockIndex < page.blockCount(); blockIndex++ {
			block := page.block(blockIndex)
			if block.source == nil || !columnItemDescendsFrom(block.source, wrappers[len(wrappers)-1]) {
				continue
			}
			key := [2]int{pageIndex, block.FragmentainerIndex}
			group := groupsByKey[key]
			if group == nil {
				group = &columnWrapperFragmentGroup{pageIndex: pageIndex, column: block.FragmentainerIndex, start: blockIndex, end: blockIndex + 1,
					left: block.X, top: block.Y, bottom: block.Y + block.Height}
				groupsByKey[key] = group
				groups = append(groups, group)
				continue
			}
			if blockIndex < group.start {
				group.start = blockIndex
			}
			group.end = blockIndex + 1
			group.left = math.Min(group.left, block.X)
			group.top = math.Min(group.top, block.Y)
			group.bottom = math.Max(group.bottom, block.Y+block.Height)
		}
	}
	sort.SliceStable(groups, func(i, j int) bool {
		if groups[i].pageIndex != groups[j].pageIndex {
			return groups[i].pageIndex > groups[j].pageIndex
		}
		return groups[i].start > groups[j].start
	})
	firstGroup, lastGroup := groups[0], groups[0]
	for _, group := range groups[1:] {
		if group.pageIndex < firstGroup.pageIndex || group.pageIndex == firstGroup.pageIndex && group.column < firstGroup.column {
			firstGroup = group
		}
		if group.pageIndex > lastGroup.pageIndex || group.pageIndex == lastGroup.pageIndex && group.column > lastGroup.column {
			lastGroup = group
		}
	}
	for _, group := range groups {
		page := &(*pages)[group.pageIndex]
		insertAt := group.start
		parent := (*fragmentOccurrence)(nil)
		top, bottom := segmentTop, group.bottom
		if group.pageIndex > len(starts)-1 {
			top = opts.forPage(page.Number).top()
		}
		columnX := group.left - geometry.contentOffset
		var lastFragments []*PlacedBox
		for index, wrapper := range wrappers {
			inset := boxInsets(wrapper)
			fragmentFirst := first && group == firstGroup
			fragmentLast := last && group == lastGroup
			if fragmentFirst {
				top -= inset.top
			}
			if fragmentLast {
				bottom += inset.bottom
			}
			fragment := fragmentBox(wrapper, fragmentFirst, fragmentLast)
			_, font := measureFor(opts, wrapper)
			block := placed(fragment, "", nil, columnX+geometry.boxOffsets[index], top, geometry.boxWidths[index], math.Max(0, bottom-top), resolveSize(wrapper, opts), font)
			block.FragmentainerIndex = group.column
			block.fragmentRef = newFragmentOccurrence(wrapper)
			if parent != nil {
				bindFragmentChild(parent, &block)
			}
			page.insertBlock(insertAt, block)
			placedFragment := page.block(insertAt)
			if group == lastGroup {
				lastFragments = append(lastFragments, placedFragment)
			} else {
				sizing.add(wrapper, placedFragment.Height, nil)
			}
			insertAt++
			parent = block.fragmentRef
		}
		if group == lastGroup {
			for index, wrapper := range wrappers {
				sizing.add(wrapper, lastFragments[index].Height, lastFragments[index])
			}
		}
		for blockIndex := insertAt; blockIndex < group.end+len(wrappers); blockIndex++ {
			child := page.block(blockIndex)
			if child.source != nil && columnItemDescendsFrom(child.source, wrappers[len(wrappers)-1]) && child.parentFragmentRef == nil {
				bindFragmentChild(parent, child)
			}
		}
	}
}

func flowColumnSegment(container *boxes.Box, kids []*boxes.Box, y float64, pages *[]Page, opts Options, usable, bottom float64, wrappers []*boxes.Box, balance bool) float64 {
	if len(kids) == 0 {
		return y
	}
	gap := container.Style.ColumnGap + usable*container.Style.ColumnGapPercent/100
	n := usedColumnCount(container.Style, usable, gap)
	colW := (usable - float64(n-1)*gap) / float64(n)
	wrapperGeometry := columnWrapperGeometryFor(wrappers, colW)
	contentColW := wrapperGeometry.contentWidth
	layouts := make([]columnItemLayout, len(kids))
	for index, item := range kids {
		if item.Style.Float != boxes.FloatNone {
			contentWidth := math.Max(0, usedWidth(item, contentColW)-horizontalInsets(item))
			lines := breakInlineStream(buildInlineStream(item, opts), contentWidth, wraps(item))
			height := usedHeight(item, inlineHeight(lines, item, opts), opts)
			if item.Image != nil && item.Style.Sizing.Height.IsAuto() {
				_, height = rasterDimensions(item, contentWidth, 0)
			}
			layouts[index] = columnItemLayout{lines: lines, height: height}
			continue
		}
		contentWidth := math.Max(0, contentColW-horizontalInsets(item))
		lines := breakInlineStream(buildInlineStream(item, opts), contentWidth, wraps(item))
		contentHeight := inlineHeight(lines, item, opts)
		height := usedHeight(item, contentHeight, opts)
		if item.Image != nil {
			_, imageHeight := rasterDimensions(item, contentWidth, 0)
			height = imageHeight + verticalInsets(item)
		}
		layouts[index] = columnItemLayout{lines: lines, height: height}
	}
	runs := splitColumnContentRuns(kids, layouts, opts)
	target := math.Max(0, bottom-y)
	if balance {
		target = balancedColumnRowTarget(runs, n, math.Max(0, bottom-y))
	}
	if !balance {
		end, _ := flowColumnRuns(container, runs, y, pages, opts, usable, bottom, n, gap, colW, wrapperGeometry, target, false)
		return end
	}
	maximumTarget := math.Max(target, math.Max(0, bottom-y))
	var acceptedEnd float64
	retryColumnBalance(target, maximumTarget, func(candidate float64) (float64, func()) {
		attemptPages, existingCounts := stageColumnAttemptPages(*pages)
		end, shortage := flowColumnRuns(container, runs, y, &attemptPages, opts, usable, bottom, n, gap, colW, wrapperGeometry, candidate, true)
		return shortage, func() {
			mergeColumnAttemptPages(pages, attemptPages, existingCounts)
			acceptedEnd = end
		}
	})
	return acceptedEnd
}

func retryColumnBalance(target, maximumTarget float64, attempt func(float64) (float64, func())) float64 {
	for {
		shortage, commit := attempt(target)
		if shortage <= fragmentEpsilon {
			commit()
			return target
		}
		next := math.Min(maximumTarget, target+shortage)
		if next <= target+fragmentEpsilon {
			commit()
			return target
		}
		target = next
	}
}

func stageColumnAttemptPages(pages []Page) ([]Page, []int) {
	staged := make([]Page, len(pages))
	existingCounts := make([]int, len(pages))
	for index, page := range pages {
		existingCounts[index] = page.blockCount()
		staged[index] = Page{Number: page.Number, Geometry: page.Geometry}
		if existingCounts[index] > 0 {
			// Keep existing pages occupied during the attempt without sharing their fragment nodes.
			staged[index].fragmentState = &pageFragmentState{
				paint: make([]*FragmentNode, existingCounts[index]),
			}
		}
	}
	return staged, existingCounts
}

func mergeColumnAttemptPages(pages *[]Page, staged []Page, existingCounts []int) {
	for len(*pages) < len(staged) {
		*pages = append(*pages, Page{Number: len(*pages) + 1})
	}
	for pageIndex := range staged {
		start := 0
		if pageIndex < len(existingCounts) {
			start = existingCounts[pageIndex]
		}
		page := &staged[pageIndex]
		blocks := make([]PlacedBox, 0, page.blockCount()-start)
		for blockIndex := start; blockIndex < page.blockCount(); blockIndex++ {
			blocks = append(blocks, *page.block(blockIndex))
		}
		for _, block := range blocks {
			(*pages)[pageIndex].emitBlock(block)
		}
	}
}

func flowColumnRuns(container *boxes.Box, runs []columnContentRun, y float64, pages *[]Page, opts Options, usable, bottom float64, n int, gap, colW float64, wrapperGeometry columnWrapperGeometry, target float64, balance bool) (float64, float64) {
	shortage := 0.0
	for _, run := range runs {
		if run.forcedPage {
			pageHasContent := (*pages)[len(*pages)-1].blockCount() > 0
			fragment := placeFragmentUnitForPage(fragmentUnit{breakBefore: fragmentBreakPage}, y, opts.top(), bottom, pageHasContent, pages, opts)
			y = fragment.y
		}
		runTarget := target
		if !balance {
			runTarget = math.Max(0, bottom-y)
		}
		runEnd, runShortage := flowColumnFragments(container, run.kids, run.layouts, y, pages, opts, usable, bottom, n, gap, colW, wrapperGeometry, runTarget, balance)
		y = runEnd
		if balance {
			shortage = math.Max(shortage, runShortage)
		}
	}
	return y, math.Max(0, shortage)
}

func splitColumnContentRuns(kids []*boxes.Box, layouts []columnItemLayout, opts Options) []columnContentRun {
	var runs []columnContentRun
	start := 0
	for index := 1; index <= len(kids); index++ {
		if index < len(kids) && !forcedPageBreakBetween(kids[index-1], kids[index]) {
			continue
		}
		if start < index {
			run := columnContentRun{
				kids:       kids[start:index],
				layouts:    layouts[start:index],
				forcedPage: start > 0,
			}
			run.units = columnBalanceUnits(run.kids, run.layouts, opts)
			for _, unit := range run.units {
				run.contentHeight += unit.height
			}
			runs = append(runs, run)
		}
		start = index
	}
	return runs
}

func isMultiColumn(b *boxes.Box) bool {
	return b != nil && (b.Style.ColCount > 1 || b.Style.HasColumnWidth)
}

func usedColumnCount(style boxes.ComputedStyle, availableWidth, gap float64) int {
	if !style.HasColumnWidth {
		if style.ColCount > 0 {
			return style.ColCount
		}
		return 1
	}
	width := math.Max(style.ColumnWidth, 0.75)
	count := int(math.Floor((availableWidth + gap) / (width + gap)))
	if count < 1 {
		count = 1
	}
	if style.ColCount > 0 && count > style.ColCount {
		count = style.ColCount
	}
	return count
}

func flowColumnFragments(container *boxes.Box, kids []*boxes.Box, layouts []columnItemLayout, y float64, pages *[]Page, opts Options, usable, bottom float64, n int, gap, colW float64, wrapperGeometry columnWrapperGeometry, target float64, balance bool) (float64, float64) {
	// A fragmentable paragraph may use the remaining space in one column;
	// subsequent children continue in source order in the next column.
	index, lineOffset, end := 0, 0, y
	previousBottomMargin, hasPreviousItem := 0.0, false
	firstPage := len(*pages) - 1
	pageBlockStarts := make([]int, len(*pages))
	for pageIndex := range *pages {
		pageBlockStarts[pageIndex] = (*pages)[pageIndex].blockCount()
	}
	shortage := math.Inf(1)
	row := columnRowCursor{pageIndex: firstPage, top: y, bottom: bottom, options: opts}
	for index < len(kids) {
		for col := 0; col < n && index < len(kids); col++ {
			columnX := opts.left() + float64(col)*(colW+gap)
			contentX := columnX + wrapperGeometry.contentOffset
			contentColW := wrapperGeometry.contentWidth
			yc, used := y, 0.0
			columnBottom := y
			var columnFloats []activeFloat
			if lineOffset > 0 && index < len(layouts) && len(layouts[index].flowLines) > 0 {
				item := layouts[index]
				var remainingWords inlineStream
				for _, line := range item.lines[lineOffset:] {
					remainingWords = append(remainingWords, line...)
				}
				contentWidth := math.Max(0, contentColW-horizontalInsets(kids[index]))
				item.lines = breakInlineStream(remainingWords, contentWidth, wraps(kids[index]))
				item.flowLines = nil
				layouts[index], lineOffset = item, 0
			} else if lineOffset == 0 && index < len(layouts) && len(layouts[index].flowLines) > 0 {
				item := layouts[index]
				contentWidth := math.Max(0, contentColW-horizontalInsets(kids[index]))
				item.lines = breakInlineStream(buildInlineStream(kids[index], opts), contentWidth, wraps(kids[index]))
				item.flowLines = nil
				layouts[index] = item
			}
			for index < len(kids) {
				it, item := kids[index], layouts[index]
				fs := resolveSize(it, opts)
				_, font := measureFor(opts, it)
				contentWidth := math.Max(0, contentColW-horizontalInsets(it))
				if it.Tag != "anonymous" && isMultiColumn(it) {
					marginGap := columnCollapsedMarginGap(previousBottomMargin, hasPreviousItem, it.Style.MarginTop)
					columnLimit := math.Max(y+target, columnBottom)
					if used > 0 && yc+item.height+marginGap+it.Style.MarginBottom > columnLimit+fragmentEpsilon {
						if col == n-1 {
							recordColumnShortage(&shortage, yc+item.height+marginGap+it.Style.MarginBottom-columnLimit)
						}
						break
					}
					columnStart := yc + marginGap - it.Style.MarginTop
					columnOpts := opts.withinPageHorizontalRange(contentX, contentColW)
					columnEnd := flowColumns(it, columnStart, pages, columnOpts, contentColW, bottom)
					used += columnEnd - yc
					yc = columnEnd
					columnBottom = math.Max(columnBottom, columnEnd)
					previousBottomMargin, hasPreviousItem = 0, false
					index++
					continue
				}
				if it.Style.Float != boxes.FloatNone {
					columnFloat := placeColumnFloat(it, yc, contentX, contentColW, col+1, pages, opts)
					columnFloats = append(columnFloats, columnFloat)
					columnBottom = math.Max(columnBottom, columnFloat.bottom)
					previousBottomMargin, hasPreviousItem = 0, false
					index++
					continue
				}
				if it.Style.Clear != boxes.ClearNone {
					clearedY := clearanceY(columnFloats, len(*pages)-1, it.Style.Clear, yc)
					if clearedY > yc {
						used += clearedY - yc
						yc = clearedY
						previousBottomMargin, hasPreviousItem = 0, false
					}
				}
				if columnTextCanSplit(it) && len(item.lines) > 0 {
					if lineOffset == 0 {
						marginGap := columnCollapsedMarginGap(previousBottomMargin, hasPreviousItem, it.Style.MarginTop)
						yc += marginGap
						used += marginGap
					}
					if lineOffset == 0 && len(columnFloats) > 0 {
						item.flowLines = layoutColumnLinesAroundFloats(it, contentX, contentColW, yc, columnFloats, len(*pages)-1, opts)
						item.lines = inlineLineWords(item.flowLines)
						layouts[index] = item
					}
					columnLimit := columnLineLimit(y+target, yc, columnFloats, it, opts)
					count := linesFittingColumn(item, lineOffset, yc, columnLimit, it, opts)
					left := len(item.lines) - lineOffset
					if count > 0 && count < left {
						units := columnTextLineFragmentUnits(item.lines[lineOffset:], it, opts)
						cut, _, retry := enforceWidowsOrphans(units, 0, count, 0, yc, y, nil)
						if cut > 0 {
							count = cut
						} else if retry && used > 0 {
							minimumLines := columnWidowOrphanMinimumLines(it, len(units))
							minimumHeight := measureFragmentRange(units, 0, minimumLines, nil)
							recordColumnShortage(&shortage, yc+minimumHeight-(y+target))
							break
						}
					}
					if col == n-1 && count < left && count > 0 {
						recordColumnShortage(&shortage, columnTextLineOverflow(item, lineOffset, lineOffset+count, yc, columnLimit, it, opts))
					}
					if col == n-1 && row.rowIndex == 0 && target < bottom-y-fragmentEpsilon && count < left {
						count = left
					} else if count > left {
						count = left
					}
					if count < 1 {
						if used > 0 {
							break
						}
						count = 1
					}
					fragment := item.lines[lineOffset : lineOffset+count]
					h := inlineHeight(fragment, it, opts) + verticalInsets(it)
					if len(item.flowLines) >= lineOffset+count {
						lastLine := item.flowLines[lineOffset+count-1]
						h = lastLine.y + lastLine.height - yc + it.Style.BorderBottom.Width + it.Style.PaddingBottom
					}
					if col == n-1 && lineOffset+count == len(item.lines) {
						recordColumnShortage(&shortage, yc+h-(y+target))
					}
					pageHasContent := (*pages)[len(*pages)-1].blockCount() > 0
					breakBefore := fragmentBreakAuto
					if lineOffset == 0 {
						if index > 0 && forcedPageBreakBetween(kids[index-1], it) {
							breakBefore = fragmentBreakPage
						} else {
							breakBefore = pageBreakValue(it.Style.BreakBefore)
						}
					}
					fragmentPlan := placeFragmentUnitForPage(fragmentUnit{height: h, breakBefore: breakBefore}, yc, opts.top(), bottom, pageHasContent, pages, opts)
					lineShift := fragmentPlan.y - yc
					yc = fragmentPlan.y
					pb := placed(it, flexFragmentText(fragment), nil, contentX, yc, contentColW, h, fs, font)
					pb.FragmentainerIndex = col + 1
					if len(item.flowLines) >= lineOffset+count {
						fragmentLines := item.flowLines[lineOffset : lineOffset+count]
						if lineShift != 0 {
							fragmentLines = shiftInlineLines(fragmentLines, lineShift)
						}
						pb.Runs = inlineLineRuns(it, fragmentLines, opts)
					} else {
						pb.Runs = assignRuns(it, fragment, contentX+it.Style.BorderLeft.Width+it.Style.PaddingLeft,
							contentWidth, yc+it.Style.BorderTop.Width+it.Style.PaddingTop, opts)
					}
					(*pages)[len(*pages)-1].emitBlock(pb)
					yc += h
					used += h
					lineOffset += count
					if lineOffset == len(item.lines) {
						yc += it.Style.MarginBottom
						used += it.Style.MarginBottom
						previousBottomMargin, hasPreviousItem = it.Style.MarginBottom, true
						lineOffset, index = 0, index+1
					} else {
						break
					}
					continue
				}
				marginGap := columnCollapsedMarginGap(previousBottomMargin, hasPreviousItem, it.Style.MarginTop)
				h := item.height + marginGap + it.Style.MarginBottom
				columnLimit := math.Max(y+target, columnBottom)
				if used > 0 && yc+h > columnLimit+fragmentEpsilon {
					if col == n-1 {
						recordColumnShortage(&shortage, yc+h-columnLimit)
					}
					break
				}
				yc += marginGap
				pageHasContent := (*pages)[len(*pages)-1].blockCount() > 0
				breakBefore := pageBreakValue(it.Style.BreakBefore)
				if index > 0 && forcedPageBreakBetween(kids[index-1], it) {
					breakBefore = fragmentBreakPage
				}
				fragmentPlan := placeFragmentUnitForPage(fragmentUnit{height: item.height, breakBefore: breakBefore}, yc, opts.top(), bottom, pageHasContent, pages, opts)
				yc = fragmentPlan.y
				pb := placed(it, it.Text, nil, contentX, yc, contentColW, item.height, fs, font)
				pb.FragmentainerIndex = col + 1
				pb.Runs = assignRuns(it, item.lines, contentX+it.Style.BorderLeft.Width+it.Style.PaddingLeft,
					contentWidth, yc+it.Style.BorderTop.Width+it.Style.PaddingTop, opts)
				(*pages)[len(*pages)-1].emitBlock(pb)
				yc += item.height + it.Style.MarginBottom
				used += h
				previousBottomMargin, hasPreviousItem = it.Style.MarginBottom, true
				index++
			}
			if columnBottom > end || yc > end {
				end = math.Max(columnBottom, yc)
			}
			if col == n-1 {
				recordColumnShortage(&shortage, math.Max(columnBottom, yc)-(y+target))
			}
		}
		if index < len(kids) {
			row = nextColumnRow(row, end, pages)
			opts, y, bottom = row.options, row.top, row.bottom
			end = row.top
			if container.Style.ColumnFill == boxes.ColumnFillAuto {
				target = math.Max(0, bottom-y)
			} else if balance {
				remainingKids, remainingLayouts := remainingColumnContent(kids, layouts, index, lineOffset, opts, wrapperGeometry.contentWidth)
				target = balancedColumnRowTarget(splitColumnContentRuns(remainingKids, remainingLayouts, opts), n, math.Max(0, bottom-y))
			}
		}
	}
	if container.Style.ColumnRuleWidth > 0 && gap > 0 {
		for pageIndex := firstPage; pageIndex < len(*pages); pageIndex++ {
			page := &(*pages)[pageIndex]
			startBlock := 0
			if pageIndex < len(pageBlockStarts) {
				startBlock = pageBlockStarts[pageIndex]
			}
			if startBlock == page.blockCount() {
				continue
			}
			pageOpts := opts
			if len(opts.PageRules) != 0 {
				pageOpts = opts.forPage(pageIndex + 1)
			}
			ruleTop := y
			if pageIndex != firstPage {
				ruleTop = pageOpts.top()
			}
			ruleBottom := ruleTop
			occupiedColumns := make([]bool, n)
			for blockIndex := startBlock; blockIndex < page.blockCount(); blockIndex++ {
				block := page.block(blockIndex)
				ruleBottom = math.Max(ruleBottom, block.Y+block.Height)
				if block.FragmentainerIndex > 0 && block.FragmentainerIndex <= n {
					occupiedColumns[block.FragmentainerIndex-1] = true
				}
			}
			for col := 0; col < n-1; col++ {
				if !occupiedColumns[col] || !occupiedColumns[col+1] {
					continue
				}
				x := pageOpts.left() + float64(col+1)*colW + float64(col)*gap + (gap-container.Style.ColumnRuleWidth)/2
				rule := PlacedBox{X: x, Y: ruleTop, Height: ruleBottom - ruleTop}
				rule.BorderLeft = boxes.BorderSide{Width: container.Style.ColumnRuleWidth, Color: container.Style.ColumnRuleColor}
				page.emitBlock(rule)
			}
		}
	}
	if math.IsInf(shortage, 1) {
		shortage = 0
	}
	return end, shortage
}

func remainingColumnContent(kids []*boxes.Box, layouts []columnItemLayout, index, lineOffset int, opts Options, columnWidth float64) ([]*boxes.Box, []columnItemLayout) {
	remainingKids := append([]*boxes.Box(nil), kids[index:]...)
	remainingLayouts := append([]columnItemLayout(nil), layouts[index:]...)
	if len(remainingKids) == 0 || lineOffset <= 0 || len(remainingLayouts) == 0 {
		return remainingKids, remainingLayouts
	}
	item := remainingKids[0]
	layout := remainingLayouts[0]
	if lineOffset >= len(layout.lines) {
		return remainingKids, remainingLayouts
	}
	var remainingWords inlineStream
	for _, line := range layout.lines[lineOffset:] {
		remainingWords = append(remainingWords, line...)
	}
	contentWidth := math.Max(0, columnWidth-horizontalInsets(item))
	layout.lines = breakInlineStream(remainingWords, contentWidth, wraps(item))
	layout.flowLines = nil
	layout.height = inlineHeight(layout.lines, item, opts) + verticalInsets(item)
	remainingLayouts[0] = layout
	resumed := *item
	resumed.Style.MarginTop = 0
	remainingKids[0] = &resumed
	return remainingKids, remainingLayouts
}

func balancedColumnRowTarget(runs []columnContentRun, columns int, available float64) float64 {
	if len(runs) == 0 || available <= 0 {
		return 0
	}
	contentHeight := 0.0
	for _, run := range runs {
		contentHeight += run.contentHeight
	}
	if contentHeight > float64(columns)*available+fragmentEpsilon {
		return available
	}
	target := distributeImplicitColumnBreaks(runs, columns)
	return raiseTargetForLegalBreaks(runs, target)
}

func nextColumnRow(current columnRowCursor, y float64, pages *[]Page) columnRowCursor {
	pageIndex := max(current.pageIndex, len(*pages)-1)
	pageOptions := current.options
	if len(pageOptions.PageRules) != 0 {
		pageOptions = pageOptions.forPage(pageIndex + 1)
	}
	pageHasContent := (*pages)[pageIndex].blockCount() > 0
	fragment := placeFragmentUnitForPage(
		fragmentUnit{breakBefore: fragmentBreakPage},
		y,
		pageOptions.top(),
		pageOptions.pageBottom(),
		pageHasContent,
		pages,
		pageOptions,
	)
	pageIndex = len(*pages) - 1
	if len(pageOptions.PageRules) != 0 {
		pageOptions = pageOptions.forPage(pageIndex + 1)
	}
	return columnRowCursor{
		pageIndex: pageIndex,
		rowIndex:  current.rowIndex + 1,
		top:       fragment.y,
		bottom:    pageOptions.pageBottom(),
		options:   pageOptions,
	}
}

func recordColumnShortage(shortage *float64, measured float64) {
	if measured > fragmentEpsilon && measured < *shortage {
		*shortage = measured
	}
}

func columnTextLineOverflow(item columnItemLayout, first, index int, y, limit float64, box *boxes.Box, opts Options) float64 {
	if index >= len(item.lines) {
		return 0
	}
	if len(item.flowLines) > index {
		line := item.flowLines[index]
		return math.Max(0, line.y+line.height-limit)
	}
	lineY := y + box.Style.BorderTop.Width + box.Style.PaddingTop
	for lineIndex := first; lineIndex < index; lineIndex++ {
		lineY += lineHeightW(item.lines[lineIndex], box, opts)
	}
	return math.Max(0, lineY+lineHeightW(item.lines[index], box, opts)-limit)
}

func distributeImplicitColumnBreaks(runs []columnContentRun, columns int) float64 {
	if len(runs) == 0 {
		return 0
	}
	if columns < 1 {
		columns = 1
	}
	// Every forced-break run starts with one column. Assign each remaining
	// column to the run whose current share is tallest, as Blink does.
	for found := len(runs); found < columns; found++ {
		tallest := 0
		for index := 1; index < len(runs); index++ {
			if runColumnBlockSize(runs[index]) > runColumnBlockSize(runs[tallest]) {
				tallest = index
			}
		}
		runs[tallest].implicitBreaks++
	}
	tallest := 0.0
	for _, run := range runs {
		tallest = math.Max(tallest, runColumnBlockSize(run))
	}
	return tallest
}

func runColumnBlockSize(run columnContentRun) float64 {
	columns := run.implicitBreaks + 1
	if columns < 1 {
		columns = 1
	}
	return math.Ceil(run.contentHeight/float64(columns)*64) / 64
}

func raiseTargetForLegalBreaks(runs []columnContentRun, target float64) float64 {
	unitCount := 0
	for _, run := range runs {
		unitCount += len(run.units)
		for _, unit := range run.units {
			target = math.Max(target, unit.height)
		}
	}
	for attempt := 0; attempt <= unitCount; attempt++ {
		shortage := math.Inf(1)
		for _, run := range runs {
			allowedColumns := run.implicitBreaks + 1
			columns, runShortage := columnsRequiredForRun(run.units, target)
			if columns > allowedColumns {
				shortage = math.Min(shortage, runShortage)
			}
		}
		if math.IsInf(shortage, 1) || shortage <= fragmentEpsilon {
			return target
		}
		// The unconstrained run estimate can fall between legal breaks. Retry
		// at the smallest observed shortage until each run fits its allocation.
		target += shortage
	}
	return target
}

func columnsRequiredForRun(units []fragmentUnit, target float64) (int, float64) {
	if len(units) == 0 {
		return 0, 0
	}
	columns, used, shortage := 1, 0.0, math.Inf(1)
	for index := 0; index < len(units); {
		unit := units[index]
		end, height := index+1, unit.height
		groupStart, groupEnd, keep := keepRange(units, index)
		if keep && groupStart == index {
			end = groupEnd
			height = measureFragmentRange(units, index, end, nil)
		}
		if height > target+fragmentEpsilon {
			shortage = math.Min(shortage, height-target)
		}
		if used > 0 && used+height > target+fragmentEpsilon {
			shortage = math.Min(shortage, used+height-target)
			columns++
			used = 0
		}
		used += height
		index = end
	}
	if math.IsInf(shortage, 1) {
		shortage = 0
	}
	return columns, shortage
}

func columnBalanceUnits(kids []*boxes.Box, layouts []columnItemLayout, opts Options) []fragmentUnit {
	var units []fragmentUnit
	previousBottomMargin, hasPreviousItem := 0.0, false
	breakPending := false
	nextGroup := 1
	for index, item := range kids {
		forcedBefore := pageBreakValue(item.Style.BreakBefore) == fragmentBreakPage ||
			(index > 0 && forcedPageBreakBetween(kids[index-1], item)) || breakPending
		if forcedBefore {
			previousBottomMargin, hasPreviousItem = 0, false
		}
		if item.Style.Float != boxes.FloatNone {
			previousBottomMargin, hasPreviousItem = 0, false
			breakPending = item.Style.BreakAfter == boxes.BreakAfterPage
			continue
		}
		layout := layouts[index]
		group := 0
		if item.Style.BreakInside == boxes.BreakInsideAvoid {
			group = nextGroup
			nextGroup++
		}
		breakBefore := pageBreakValue(item.Style.BreakBefore)
		breakAfter := fragmentBreakAuto
		switch item.Style.BreakAfter {
		case boxes.BreakAfterAvoid:
			breakAfter = fragmentBreakAvoid
		case boxes.BreakAfterPage:
			breakAfter = fragmentBreakPage
		}
		if columnTextCanSplit(item) && len(layout.lines) > 0 {
			for lineIndex, line := range layout.lines {
				height := lineHeightW(line, item, opts)
				if lineIndex == 0 {
					height += columnCollapsedMarginGap(previousBottomMargin, hasPreviousItem, item.Style.MarginTop)
				}
				if lineIndex == len(layout.lines)-1 {
					height += item.Style.MarginBottom
				}
				unit := fragmentUnit{height: height}
				if lineIndex == 0 {
					unit.breakBefore = breakBefore
				}
				if lineIndex == len(layout.lines)-1 {
					unit.breakAfter = breakAfter
				}
				if group != 0 {
					unit.group, unit.breakInsideAvoid = group, true
				}
				units = append(units, unit)
			}
		} else {
			gap := columnCollapsedMarginGap(previousBottomMargin, hasPreviousItem, item.Style.MarginTop)
			units = append(units, fragmentUnit{
				height: layout.height + gap + item.Style.MarginBottom,
				group:  group, breakInsideAvoid: group != 0,
				breakBefore: breakBefore, breakAfter: breakAfter,
			})
		}
		previousBottomMargin, hasPreviousItem = item.Style.MarginBottom, true
		breakPending = item.Style.BreakAfter == boxes.BreakAfterPage
	}
	return units
}

func columnCollapsedMarginGap(previous float64, hasPrevious bool, current float64) float64 {
	if !hasPrevious {
		return current
	}
	return collapseMargins(previous, current) - previous
}

func placeColumnFloat(item *boxes.Box, y, columnX, columnWidth float64, fragmentainer int, pages *[]Page, opts Options) activeFloat {
	if len(item.Children) > 0 {
		columnOpts := opts.withinPageHorizontalRange(columnX, columnWidth)
		firstBlock := (*pages)[len(*pages)-1].blockCount()
		placedFloat := flowFloat(item, y, pages, columnOpts, columnWidth, opts.PageHeight-opts.pageBottomMargin()-opts.space.marginBottom)
		placedFloat.x -= item.Style.MarginLeft
		placedFloat.top = y
		placedFloat.bottom += item.Style.MarginBottom
		for index := firstBlock; index < (*pages)[len(*pages)-1].blockCount(); index++ {
			(*pages)[len(*pages)-1].block(index).FragmentainerIndex = fragmentainer
		}
		return placedFloat
	}
	available := math.Max(0, columnWidth-item.Style.MarginLeft-item.Style.MarginRight)
	width := usedWidth(item, available)
	contentWidth := math.Max(0, width-horizontalInsets(item))
	lines := breakInlineStream(buildInlineStream(item, opts), contentWidth, wraps(item))
	contentHeight := inlineHeight(lines, item, opts)
	height := usedHeight(item, contentHeight, opts)
	if item.Style.Sizing.Height.IsAuto() && item.Image != nil {
		_, height = rasterDimensions(item, contentWidth, 0)
	}
	top := y + item.Style.MarginTop
	x := columnX + item.Style.MarginLeft
	left := item.Style.Float == boxes.FloatLeft
	if !left {
		x = columnX + columnWidth - width - item.Style.MarginRight
	}
	fs := resolveSize(item, opts)
	_, font := measureFor(opts, item)
	pb := placed(item, item.Text, item.Image, x, top, width, height, fs, font)
	pb.FragmentainerIndex = fragmentainer
	pb.Runs = assignRuns(item, lines, x+item.Style.BorderLeft.Width+item.Style.PaddingLeft,
		contentWidth, top+item.Style.BorderTop.Width+item.Style.PaddingTop, opts)
	(*pages)[len(*pages)-1].emitBlock(pb)
	return activeFloat{left: left, x: x - item.Style.MarginLeft, w: width + item.Style.MarginLeft + item.Style.MarginRight,
		top: top - item.Style.MarginTop, bottom: top + height + item.Style.MarginBottom, page: len(*pages) - 1}
}

func layoutColumnLinesAroundFloats(item *boxes.Box, columnX, columnWidth, top float64, floats []activeFloat, page int, opts Options) []inlineLine {
	contentX := columnX + item.Style.BorderLeft.Width + item.Style.PaddingLeft
	contentY := top + item.Style.BorderTop.Width + item.Style.PaddingTop
	contentWidth := math.Max(0, columnWidth-horizontalInsets(item))
	return layoutInlineLines(item, buildInlineStream(item, opts), contentX, contentWidth, contentY, floats, page, opts)
}

func linesFittingColumn(item columnItemLayout, first int, y, limit float64, box *boxes.Box, opts Options) int {
	if len(item.flowLines) >= first+1 {
		count := 0
		for index := first; index < len(item.flowLines); index++ {
			line := item.flowLines[index]
			if line.y+line.height > limit+fragmentEpsilon {
				break
			}
			count++
		}
		return count
	}
	lineY := y + box.Style.BorderTop.Width + box.Style.PaddingTop
	count := 0
	for index := first; index < len(item.lines); index++ {
		lineHeight := lineHeightW(item.lines[index], box, opts)
		if lineY+lineHeight > limit+fragmentEpsilon {
			break
		}
		count++
		lineY += lineHeight
	}
	return count
}

func columnTextLineFragmentUnits(lines []inlineStream, box *boxes.Box, opts Options) []fragmentUnit {
	units := make([]fragmentUnit, len(lines))
	for index, line := range lines {
		units[index] = fragmentUnit{
			height:  lineHeightW(line, box, opts),
			group:   1,
			orphans: box.Style.Orphans,
			widows:  box.Style.Widows,
		}
	}
	return units
}

func columnWidowOrphanMinimumLines(box *boxes.Box, total int) int {
	orphans, widows := box.Style.Orphans, box.Style.Widows
	if orphans <= 0 {
		orphans = 2
	}
	if widows <= 0 {
		widows = 2
	}
	minimum := orphans + widows
	if widows > 2 && orphans == 2 {
		minimum = 4
	}
	return min(total, minimum)
}

func columnLineLimit(targetEnd, lineY float64, floats []activeFloat, box *boxes.Box, opts Options) float64 {
	limit := targetEnd
	lineHeight := lineHeightFor(box, resolveSize(box, opts), opts)
	for _, fl := range floats {
		if fl.bottom <= lineY {
			continue
		}
		limit = math.Max(limit, math.Max(lineY, fl.top)+lineHeight)
	}
	return limit
}

func columnTextCanSplit(item *boxes.Box) bool {
	return item != nil && item.Text != "" && len(item.Children) == 0 && item.Image == nil &&
		item.SVG == nil && !item.Style.HasBg &&
		item.Style.PaddingTop == 0 && item.Style.PaddingRight == 0 && item.Style.PaddingBottom == 0 && item.Style.PaddingLeft == 0 &&
		item.Style.BorderTop.Width == 0 && item.Style.BorderRight.Width == 0 && item.Style.BorderBottom.Width == 0 && item.Style.BorderLeft.Width == 0 &&
		item.Style.BreakInside != boxes.BreakInsideAvoid
}

func flowColumnSpan(item *boxes.Box, y float64, pages *[]Page, opts Options, width, bottom float64) float64 {
	fs := resolveSize(item, opts)
	_, font := measureFor(opts, item)
	contentWidth := width - horizontalInsets(item)
	lines := breakInlineStream(buildInlineStream(item, opts), contentWidth, wraps(item))
	contentHeight := inlineHeight(lines, item, opts)
	boxHeight := contentHeight + verticalInsets(item)
	y += item.Style.MarginTop
	pageHasContent := (*pages)[len(*pages)-1].blockCount() > 0
	fragment := placeFragmentUnitForPage(fragmentUnit{height: boxHeight, breakBefore: pageBreakValue(item.Style.BreakBefore)}, y, opts.top(), bottom, pageHasContent, pages, opts)
	y = fragment.y
	x := opts.left()
	placedSpan := placed(item, item.Text, nil, x, y, width, boxHeight, fs, font)
	placedSpan.Runs = assignRuns(item, lines, x+item.Style.PaddingLeft+item.Style.BorderLeft.Width, contentWidth, y+item.Style.PaddingTop+item.Style.BorderTop.Width, opts)
	(*pages)[len(*pages)-1].emitBlock(placedSpan)
	return y + boxHeight + item.Style.MarginBottom
}
