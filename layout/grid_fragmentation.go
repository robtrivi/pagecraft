package layout

import (
	"math"
	"strings"

	"github.com/robtrivi/pagecraft-go/boxes"
)

func gridRowContentFits(row []gridItem, metrics gridRowLayout, available float64) bool {
	if len(row) == 0 || available < 0 {
		return false
	}
	contentHeight := 0.0
	for i, item := range row {
		if item.rowSpan > 1 {
			return false
		}
		if item.box.Image == nil && len(metrics.lines[i]) == 0 && len(item.box.Children) == 0 && metrics.contentHeights[i] == 0 {
			return false
		}
		contentHeight = math.Max(contentHeight, metrics.contentHeights[i])
	}
	return contentHeight <= available+fragmentEpsilon
}

type gridRowBreakAction uint8

const (
	gridRowBreakKeep gridRowBreakAction = iota
	gridRowBreakMove
	gridRowBreakFragment
)

type gridRowBreakContext struct {
	grid              *boxes.Box
	rows              [][]gridItem
	layouts           []gridRowLayout
	row               []gridItem
	metrics           gridRowLayout
	rowIndex          int
	rowGap            float64
	rowExtent         float64
	freshPageCapacity float64
	available         float64
	opts              Options
}

func decideGridRowBreak(context gridRowBreakContext) gridRowBreakAction {
	if gridRowContentFits(context.row, context.metrics, context.available) {
		return gridRowBreakKeep
	}
	if gridRowCanFragment(context.grid, context.rows, context.layouts, context.row, context.metrics, context.rowIndex, context.rowGap, context.rowExtent, context.freshPageCapacity, context.opts) {
		return gridRowBreakFragment
	}
	return gridRowBreakMove
}

func gridRowCanFlowSubtrees(g *boxes.Box, row []gridItem, metrics gridRowLayout, rowIndex int, rowExtent, freshPageCapacity float64, opts Options) bool {
	if len(row) == 0 || !gridRowIsAuto(g, rowIndex) || hasDeclaredHeight(g, opts) || clipsOverflow(g) ||
		breakInsideAvoidFits(g.Style.BreakInside, rowExtent, freshPageCapacity) {
		return false
	}
	hasSubtree := false
	for index, item := range row {
		if item.rowSpan != 1 || hasDeclaredHeight(item.box, opts) || clipsOverflow(item.box) || breakInsideAvoidFits(item.box.Style.BreakInside, rowExtent, freshPageCapacity) {
			return false
		}
		hasSubtree = hasSubtree || metrics.nested[index]
	}
	return hasSubtree
}

// flowGridSubtreesAcrossPages lays each grid item's content through the shared
// page flow and merges the resulting cell fragments into their common rows.
func flowGridSubtreesAcrossPages(g *boxes.Box, row []gridItem, metrics gridRowLayout, rowIndex int, rowGap float64, trackOffsets []float64, y float64, pages *[]Page, opts Options, bottom float64) float64 {
	contentOriginX := opts.left() + g.Style.MarginLeft + boxInsets(g).left
	parallelItems := make([]parallelFlowItem, 0, len(row))
	for index, item := range row {
		it := item.box
		width := metrics.widths[index]
		x := contentOriginX + trackOffsets[item.column]
		alignment := boxes.AlignStretch
		if it.Image != nil {
			alignment = boxes.AlignFlexStart
		}
		itemX, itemWidth, itemIndex := x, width, index
		parallelItems = append(parallelItems, parallelFlowItem{
			x: itemX, width: itemWidth, alignment: alignment,
			layout: func(itemPages *[]Page, itemOpts Options) float64 {
				switch {
				case metrics.nested[itemIndex]:
					itemBox := withUsedBorderBoxWidth(it, itemWidth)
					return flowNestedContainer(itemBox, y, boxInsets(itemBox), nestedContainerItems(itemBox), false, itemPages, itemOpts, itemWidth)
				case it.Image != nil:
					itemHeight := metrics.contentHeights[itemIndex]
					pageHasContent := (*itemPages)[0].blockCount() > 0
					fragment := placeFragmentUnitForPage(fragmentUnit{height: itemHeight}, y, opts.top()+g.Style.MarginTop, bottom, pageHasContent, itemPages, opts)
					ensurePageIndex(itemPages, fragment.pageOffset)
					imageWidth, imageHeight := gridRasterDimensions(it, itemWidth, g, rowIndex, 1, rowGap)
					(*itemPages)[fragment.pageOffset].emitBlock(placed(it, it.Text, it.Image, itemX, fragment.y, imageWidth, imageHeight, metrics.sizes[itemIndex], metrics.faces[itemIndex]))
					return fragment.y + fragment.height
				default:
					words := buildInlineStream(it, opts)
					lines := breakInlineStream(words, math.Max(0, itemWidth-horizontalInsets(it)), wraps(it))
					return flowInlineBlockFragments(it, lines, itemX, itemWidth, y, bottom, itemPages, itemOpts)
				}
			},
		})
	}
	_, endY := layoutParallelFlows(pages, y, opts.top(), opts, parallelItems)
	return endY
}

func advanceGridCursor(y, pageTop, pageBottom float64, pages *[]Page) float64 {
	pageCapacity := pageBottom - pageTop
	if y <= pageBottom+fragmentEpsilon || pageCapacity <= fragmentEpsilon {
		return y
	}
	pageCount := int(math.Ceil((y - pageBottom) / pageCapacity))
	ensurePageIndex(pages, len(*pages)-1+pageCount)
	return y - float64(pageCount)*pageCapacity
}

func insertGridContainerFragments(g *boxes.Box, pages *[]Page, firstPage, firstBlock int, top, width, height, endY, pageTop, pageBottom float64, opts Options) {
	lastPage := len(*pages) - 1
	hasEmptyContinuation := false
	for pageIndex := firstPage + 1; pageIndex <= lastPage; pageIndex++ {
		if (*pages)[pageIndex].blockCount() == 0 {
			hasEmptyContinuation = true
			break
		}
	}
	for pageIndex := firstPage; pageIndex <= lastPage; pageIndex++ {
		page := &(*pages)[pageIndex]
		blockStart := 0
		if pageIndex == firstPage {
			blockStart = firstBlock
		}
		first, last := pageIndex == firstPage, pageIndex == lastPage
		if hasEmptyContinuation {
			fragmentTop, fragmentBottom := pageTop, pageBottom
			if first {
				fragmentTop = top
			}
			if last {
				fragmentBottom = endY
			}
			insertGridDecoration(page, blockStart, g, fragmentTop, width, math.Max(0, fragmentBottom-fragmentTop), first, last, opts)
			continue
		}
		if blockStart >= page.blockCount() {
			continue
		}
		fragmentTop, fragmentBottom := math.Inf(1), math.Inf(-1)
		for _, node := range page.paintNodes()[blockStart:] {
			block := &node.Placed
			fragmentTop = math.Min(fragmentTop, block.Y)
			fragmentBottom = math.Max(fragmentBottom, block.Y+block.Height)
		}
		if first {
			fragmentTop = top
		}
		if last {
			fragmentBottom += verticalInsets(g) - boxInsets(g).top
		}
		fragmentHeight := math.Max(0, fragmentBottom-fragmentTop)
		if first && last {
			fragmentHeight = height
		}
		insertGridDecoration(page, blockStart, g, fragmentTop, width, fragmentHeight, first, last, opts)
	}
}

func continueGridOverflow(g *boxes.Box, pages *[]Page, firstPage, firstBlock int, top, bottom, outerEnd, width float64, opts Options) float64 {
	insertGridDecoration(&(*pages)[firstPage], firstBlock, g, top, width, bottom-top, true, false, opts)
	remaining := outerEnd - bottom
	pageTop := opts.top() + g.Style.MarginTop
	pageCapacity := bottom - pageTop
	pageIndex := firstPage + 1
	lastY := pageTop
	for remaining > fragmentEpsilon && pageCapacity > fragmentEpsilon {
		fragmentHeight := math.Min(remaining, pageCapacity)
		last := remaining <= pageCapacity+fragmentEpsilon
		ensurePageIndex(pages, pageIndex)
		insertGridDecoration(&(*pages)[pageIndex], 0, g, pageTop, width, fragmentHeight, false, last, opts)
		remaining -= fragmentHeight
		lastY = pageTop + fragmentHeight
		pageIndex++
	}
	return lastY + g.Style.MarginBottom
}

func insertGridDecoration(page *Page, index int, g *boxes.Box, y, width, height float64, first, last bool, opts Options) {
	insertContainerDecoration(page, index, g, y, width, height, first, last, opts)
}

func insertContainerDecoration(page *Page, index int, box *boxes.Box, y, width, height float64, first, last bool, opts Options) {
	fragment := fragmentBox(box, first, last)
	_, face := measureFor(opts, box)
	decoration := placed(fragment, "", nil, opts.left()+box.Style.MarginLeft, y, width, height, resolveSize(box, opts), face)
	decoration = fragmentPlacedBox(decoration, first, last)
	if !hasPlacedDecoration(decoration) {
		return
	}
	decoration.fragmentRef = newFragmentOccurrence(box)
	for blockIndex := index; blockIndex < page.blockCount(); blockIndex++ {
		bindDirectFragmentChild(decoration.fragmentRef, page.block(blockIndex))
	}
	page.insertBlock(index, decoration)
}

func hasPlacedDecoration(block PlacedBox) bool {
	return block.HasBg || block.BorderWidth > 0 || block.BorderTop.Width > 0 || block.BorderRight.Width > 0 || block.BorderBottom.Width > 0 || block.BorderLeft.Width > 0
}

func gridRowCanFragment(g *boxes.Box, rows [][]gridItem, layouts []gridRowLayout, row []gridItem, metrics gridRowLayout, rowIndex int, rowGap, rowExtent, freshPageCapacity float64, opts Options) bool {
	if len(row) == 0 {
		return false
	}
	contentHeight, spanHeight := 0.0, 0.0
	span := 1
	hasText := false
	for i, item := range row {
		it := item.box
		if breakInsideAvoidFits(it.Style.BreakInside, rowExtent, freshPageCapacity) {
			return false
		}
		if it.Image != nil || hasDeclaredHeight(it, opts) || verticalInsets(it) != 0 || len(metrics.lines[i]) == 0 {
			return false
		}
		if item.rowSpan > span {
			span = item.rowSpan
		}
		hasText = true
		contentHeight = math.Max(contentHeight, metrics.contentHeights[i])
	}
	if !hasText || rowIndex+span > len(layouts) {
		return false
	}
	if span == 1 {
		return gridRowIsAuto(g, rowIndex) && math.Abs(metrics.height-contentHeight) < 1e-7
	}
	if len(row) != 1 {
		return false
	}
	spanHeight = rowGap * float64(span-1)
	for coveredRow := rowIndex; coveredRow < rowIndex+span; coveredRow++ {
		if !gridRowIsAuto(g, coveredRow) {
			return false
		}
		if coveredRow > rowIndex && len(rows[coveredRow]) > 0 {
			return false
		}
		spanHeight += layouts[coveredRow].height
	}
	return math.Abs(spanHeight-contentHeight) < 1e-7
}

func flowGridRowAcrossPages(g *boxes.Box, row []gridItem, metrics gridRowLayout, rowIndex int, y float64, trackOffsets []float64, pages *[]Page, opts Options, overflowClip *ClipRect, bottom float64) float64 {
	lineCount := 0
	for _, lines := range metrics.lines {
		lineCount = max(lineCount, len(lines))
	}
	lineHeights := make([]float64, lineCount)
	for lineIndex := 0; lineIndex < lineCount; lineIndex++ {
		for i, item := range row {
			if lineIndex < len(metrics.lines[i]) {
				lineHeights[lineIndex] = math.Max(lineHeights[lineIndex], lineHeightFor(item.box, metrics.sizes[i], opts))
			}
		}
	}
	pageBase := len(*pages) - 1
	pageHasContent := (*pages)[len(*pages)-1].blockCount() > 0
	contentOriginX := opts.left() + g.Style.MarginLeft + boxInsets(g).left
	measure := func(firstLine, endLine int) float64 {
		height := 0.0
		for _, lineHeight := range lineHeights[firstLine:endLine] {
			height += lineHeight
		}
		return height
	}
	var fragments []fragmentRange
	if len(opts.PageRules) != 0 {
		fragments = planFragmentsByPage(make([]fragmentUnit, lineCount), y, pageHasContent, pageBoundsForOptions(opts, pageBase, g.Style.MarginTop), measure)
	} else {
		fragments = planFragments(make([]fragmentUnit, lineCount), y, opts.top()+g.Style.MarginTop, bottom, pageHasContent, measure)
	}
	for _, fragment := range fragments {
		ensurePageIndex(pages, pageBase+fragment.pageOffset)
		firstLine, lastLine := fragment.first, fragment.end
		fragmentHeight := fragment.height
		y = fragment.y
		page := &(*pages)[pageBase+fragment.pageOffset]
		for i, item := range row {
			end := min(lastLine, len(metrics.lines[i]))
			if firstLine >= end {
				continue
			}
			fragmentLines := metrics.lines[i][firstLine:end]
			it := item.box
			x := contentOriginX + trackOffsets[item.column]
			block := placed(it, strings.Join(fragmentLines, " "), nil, x, y, metrics.widths[i], fragmentHeight, metrics.sizes[i], metrics.faces[i])
			measure, _ := measureFor(opts, it)
			block.Runs = singleRuns(it, fragmentLines, x, y, lineHeightFor(it, metrics.sizes[i], opts), metrics.sizes[i], metrics.faces[i], measure)
			if overflowClip != nil {
				block.ClipRects = append(block.ClipRects, *overflowClip)
			}
			page.emitBlock(block)
		}
		y += fragmentHeight
	}
	return y
}
