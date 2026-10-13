package layout

import (
	"math"

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

// Column sets flow content into equal-width fragmentainers separated by a gap.

// flowColumns places children in N columns from and and returns the cursor
// (the end of the tallest column).
func flowColumns(c *boxes.Box, y float64, pages *[]Page, opts Options, usable, bottom float64) float64 {
	y += c.Style.MarginTop
	items := nestedContainerItems(c)
	for start := 0; start < len(items); {
		span := start
		for span < len(items) && !items[span].Style.ColumnSpanAll {
			span++
		}
		balance := c.Style.ColumnFill != boxes.ColumnFillAuto || span < len(items)
		y = flowColumnSegment(c, items[start:span], y, pages, opts, usable, bottom, balance)
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
	return y + c.Style.MarginBottom
}

func flowColumnSegment(container *boxes.Box, kids []*boxes.Box, y float64, pages *[]Page, opts Options, usable, bottom float64, balance bool) float64 {
	if len(kids) == 0 {
		return y
	}
	gap := container.Style.ColumnGap + usable*container.Style.ColumnGapPercent/100
	n := usedColumnCount(container.Style, usable, gap)
	colW := (usable - float64(n-1)*gap) / float64(n)
	layouts := make([]columnItemLayout, len(kids))
	for index, item := range kids {
		if item.Style.Float != boxes.FloatNone {
			contentWidth := math.Max(0, usedWidth(item, colW)-horizontalInsets(item))
			lines := breakInlineStream(buildInlineStream(item, opts), contentWidth, wraps(item))
			height := usedHeight(item, inlineHeight(lines, item, opts), opts)
			if item.Image != nil && item.Style.Sizing.Height.IsAuto() {
				_, height = rasterDimensions(item, contentWidth, 0)
			}
			layouts[index] = columnItemLayout{lines: lines, height: height}
			continue
		}
		contentWidth := math.Max(0, colW-horizontalInsets(item))
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
		target = distributeImplicitColumnBreaks(runs, n)
		target = raiseTargetForLegalBreaks(runs, target)
	}
	if !balance {
		end, _ := flowColumnRuns(container, runs, y, pages, opts, usable, bottom, n, gap, colW, target, false)
		return end
	}
	maximumTarget := math.Max(target, math.Max(0, bottom-y))
	var acceptedEnd float64
	retryColumnBalance(target, maximumTarget, func(candidate float64) (float64, func()) {
		attemptPages, existingCounts := stageColumnAttemptPages(*pages)
		end, shortage := flowColumnRuns(container, runs, y, &attemptPages, opts, usable, bottom, n, gap, colW, candidate, true)
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

func flowColumnRuns(container *boxes.Box, runs []columnContentRun, y float64, pages *[]Page, opts Options, usable, bottom float64, n int, gap, colW, target float64, balance bool) (float64, float64) {
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
		runEnd, runShortage := flowColumnFragments(container, run.kids, run.layouts, y, pages, opts, usable, bottom, n, gap, colW, runTarget)
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

func flowColumnFragments(container *boxes.Box, kids []*boxes.Box, layouts []columnItemLayout, y float64, pages *[]Page, opts Options, usable, bottom float64, n int, gap, colW, target float64) (float64, float64) {
	// A fragmentable paragraph may use the remaining space in one column;
	// subsequent children continue in source order in the next column.
	index, lineOffset, end := 0, 0, y
	continuationX, continuationY := opts.left(), y
	continuationFragmentainer := 1
	previousBottomMargin, hasPreviousItem := 0.0, false
	rulePage := len(*pages) - 1
	shortage := math.Inf(1)
	for col := 0; col < n && index < len(kids); col++ {
		columnX := opts.left() + float64(col)*(colW+gap)
		continuationFragmentainer = col + 1
		yc, used := y, 0.0
		columnBottom := y
		var columnFloats []activeFloat
		if lineOffset > 0 && index < len(layouts) && len(layouts[index].flowLines) > 0 {
			item := layouts[index]
			var remainingWords inlineStream
			for _, line := range item.lines[lineOffset:] {
				remainingWords = append(remainingWords, line...)
			}
			contentWidth := math.Max(0, colW-horizontalInsets(kids[index]))
			item.lines = breakInlineStream(remainingWords, contentWidth, wraps(kids[index]))
			item.flowLines = nil
			layouts[index], lineOffset = item, 0
		} else if lineOffset == 0 && index < len(layouts) && len(layouts[index].flowLines) > 0 {
			item := layouts[index]
			contentWidth := math.Max(0, colW-horizontalInsets(kids[index]))
			item.lines = breakInlineStream(buildInlineStream(kids[index], opts), contentWidth, wraps(kids[index]))
			item.flowLines = nil
			layouts[index] = item
		}
		for index < len(kids) {
			it, item := kids[index], layouts[index]
			fs := resolveSize(it, opts)
			_, font := measureFor(opts, it)
			contentWidth := math.Max(0, colW-horizontalInsets(it))
			if it.Style.Float != boxes.FloatNone {
				columnFloat := placeColumnFloat(it, yc, columnX, colW, col+1, pages, opts)
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
					item.flowLines = layoutColumnLinesAroundFloats(it, columnX, colW, yc, columnFloats, len(*pages)-1, opts)
					item.lines = inlineLineWords(item.flowLines)
					layouts[index] = item
				}
				columnLimit := columnLineLimit(y+target, yc, columnFloats, it, opts)
				count := linesFittingColumn(item, lineOffset, yc, columnLimit, it, opts)
				left := len(item.lines) - lineOffset
				if col == n-1 && count < left && count > 0 {
					recordColumnShortage(&shortage, columnTextLineOverflow(item, lineOffset, lineOffset+count, yc, columnLimit, it, opts))
				}
				if col == n-1 || count > left {
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
				pb := placed(it, flexFragmentText(fragment), nil, columnX, yc, colW, h, fs, font)
				pb.FragmentainerIndex = col + 1
				if len(item.flowLines) >= lineOffset+count {
					fragmentLines := item.flowLines[lineOffset : lineOffset+count]
					if lineShift != 0 {
						fragmentLines = shiftInlineLines(fragmentLines, lineShift)
					}
					pb.Runs = inlineLineRuns(it, fragmentLines, opts)
				} else {
					pb.Runs = assignRuns(it, fragment, columnX+it.Style.BorderLeft.Width+it.Style.PaddingLeft,
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
			pb := placed(it, it.Text, nil, columnX, yc, colW, item.height, fs, font)
			pb.FragmentainerIndex = col + 1
			pb.Runs = assignRuns(it, item.lines, columnX+it.Style.BorderLeft.Width+it.Style.PaddingLeft,
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
		continuationX, continuationY = columnX, yc
	}
	if index < len(kids) {
		// Keep unconsumed content: continue it below this segment if the
		// declared column count was insufficient for the available lines.
		for index < len(kids) {
			it := kids[index]
			fs := resolveSize(it, opts)
			_, font := measureFor(opts, it)
			item := layouts[index]
			contentWidth := math.Max(0, colW-horizontalInsets(it))
			marginGap := columnCollapsedMarginGap(previousBottomMargin, hasPreviousItem, it.Style.MarginTop)
			continuationY += marginGap
			pb := placed(it, it.Text, nil, continuationX, continuationY, colW, item.height, fs, font)
			pb.FragmentainerIndex = continuationFragmentainer
			pb.Runs = assignRuns(it, item.lines, continuationX+it.Style.BorderLeft.Width+it.Style.PaddingLeft, contentWidth, continuationY+it.Style.BorderTop.Width+it.Style.PaddingTop, opts)
			(*pages)[len(*pages)-1].emitBlock(pb)
			continuationY += item.height + it.Style.MarginBottom
			previousBottomMargin, hasPreviousItem = it.Style.MarginBottom, true
			if continuationY > end {
				end = continuationY
			}
			index++
		}
	}
	if container.Style.ColumnRuleWidth > 0 && gap > 0 {
		for col := 0; col < n-1; col++ {
			x := opts.left() + float64(col+1)*colW + float64(col)*gap + (gap-container.Style.ColumnRuleWidth)/2
			rule := PlacedBox{X: x, Y: y, Height: end - y}
			rule.BorderLeft = boxes.BorderSide{Width: container.Style.ColumnRuleWidth, Color: container.Style.ColumnRuleColor}
			(*pages)[rulePage].emitBlock(rule)
		}
	}
	if math.IsInf(shortage, 1) {
		shortage = 0
	}
	return end, shortage
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
