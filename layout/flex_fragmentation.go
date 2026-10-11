package layout

import (
	"math"
	"strings"

	"github.com/robtrivi/pagecraft/boxes"
	"github.com/robtrivi/pagecraft/fonts"
)

func flexColumnItemCanFlow(container, item *boxes.Box, itemHeight, bottom float64, opts Options) bool {
	if hasHeightSizing(container) || clipsOverflow(container) || item.Image != nil || item.SVG != nil ||
		hasHeightSizing(item) || clipsOverflow(item) || item.Style.Position != boxes.PositionStatic || item.Style.Float != boxes.FloatNone ||
		item.Style.HasOpacity || container.Style.BreakInside == boxes.BreakInsideAvoid ||
		item.Style.BreakBefore != boxes.BreakBeforeAuto || item.Style.BreakAfter != boxes.BreakAfterAuto {
		return false
	}
	if item.Style.BreakInside == boxes.BreakInsideAvoid {
		if flexItemHasBlockFlow(item) {
			return false
		}
		freshPageTop := opts.top() + container.Style.MarginTop + container.Style.BorderTop.Width + container.Style.PaddingTop
		if breakInsideAvoidFits(item.Style.BreakInside, itemHeight, bottom-freshPageTop) {
			return false
		}
	}
	return flexItemHasBlockFlow(item) || canFragmentTextBlock(item) || item.Style.Display == boxes.DisplayInline && item.Tag != "li"
}

func rowCanFragment(f *boxes.Box, row flexRow, lines [][]inlineStream, alignments []boxes.FlexAlignment, rowHeight, freshPageCapacity float64, opts Options) bool {
	hasText := false
	for _, index := range row {
		item := f.Children[index]
		if breakInsideAvoidFits(item.Style.BreakInside, rowHeight, freshPageCapacity) {
			return false
		}
		imageCanFragment := item.Image != nil && (alignments[index] == boxes.AlignStretch || alignments[index] == boxes.AlignFlexStart)
		if flexItemHasBlockFlow(item) || (item.Image == nil && hasDeclaredHeight(item, opts)) || (item.Image != nil && !imageCanFragment) || flexFragmentHasRadius(item) || (alignments[index] != boxes.AlignStretch && alignments[index] != boxes.AlignFlexStart && alignments[index] != boxes.AlignCenter && alignments[index] != boxes.AlignFlexEnd && alignments[index] != boxes.AlignBaseline) {
			return false
		}
		if len(lines[index]) > 0 {
			hasText = true
		}
	}
	return hasText
}

func rowCanFlowParallelSubtrees(f *boxes.Box, row flexRow, alignments []boxes.FlexAlignment, rowHeight, freshPageCapacity float64, opts Options) bool {
	if len(row) == 0 || hasDeclaredHeight(f, opts) || clipsOverflow(f) || breakInsideAvoidFits(f.Style.BreakInside, rowHeight, freshPageCapacity) {
		return false
	}
	hasSubtree := false
	for _, index := range row {
		item := f.Children[index]
		alignment := alignments[index]
		if alignment != boxes.AlignFlexStart && alignment != boxes.AlignStretch && alignment != boxes.AlignCenter && alignment != boxes.AlignFlexEnd || item.SVG != nil ||
			alignments[index] == boxes.AlignStretch && item.Image != nil || hasDeclaredHeight(item, opts) || clipsOverflow(item) || flexFragmentHasRadius(item) ||
			breakInsideAvoidFits(item.Style.BreakInside, rowHeight, freshPageCapacity) {
			return false
		}
		hasSubtree = hasSubtree || flexItemHasBlockFlow(item)
	}
	return hasSubtree
}

// flowFlexSubtreesAcrossPages lays each flex item's block flow in its own
// fragment stream, then overlays streams by page. This lets independent item
// subtrees continue without treating the entire flex row as one tall box.
func flowFlexSubtreesAcrossPages(f *boxes.Box, row flexRow, y float64, widths []float64, lines [][]inlineStream, heights, fontSizes []float64, faces []*fonts.Face, alignments []boxes.FlexAlignment, pages *[]Page, opts Options, contentWidth, columnGap float64, bottom float64) (int, float64) {
	x := opts.left() + f.Style.MarginLeft + f.Style.BorderLeft.Width + f.Style.PaddingLeft
	rowWidth := columnGap * float64(max(0, len(row)-1))
	for _, index := range row {
		rowWidth += widths[index]
	}
	freeSpace := math.Max(0, contentWidth-rowWidth)
	switch f.Style.JustifyContent {
	case boxes.JustifyCenter:
		x += freeSpace / 2
	case boxes.JustifyFlexEnd:
		x += freeSpace
	case boxes.JustifySpaceBetween:
		if len(row) > 1 {
			columnGap += freeSpace / float64(len(row)-1)
		}
	}

	parallelItems := make([]parallelFlowItem, 0, len(row))
	for _, index := range row {
		item := f.Children[index]
		itemX, itemWidth := x, widths[index]
		parallelItems = append(parallelItems, parallelFlowItem{
			x: itemX, width: itemWidth, alignment: alignments[index],
			layout: func(itemPages *[]Page, itemOpts Options) float64 {
				if flexItemHasBlockFlow(item) {
					itemBox := withUsedBorderBoxWidth(item, itemWidth)
					return flowNestedContainer(itemBox, y, boxInsets(itemBox), nestedContainerItems(itemBox), false, itemPages, itemOpts, itemWidth)
				}
				if item.Image != nil {
					pageHasContent := (*itemPages)[0].blockCount() > 0
					fragment := placeFragmentUnitForPage(fragmentUnit{height: heights[index]}, y, opts.top(), bottom, pageHasContent, itemPages, opts)
					ensurePageIndex(itemPages, fragment.pageOffset)
					imageWidth, imageHeight := rasterDimensions(item, math.Max(0, itemWidth-horizontalInsets(item)), 0)
					(*itemPages)[fragment.pageOffset].emitBlock(placed(item, item.Text, item.Image, itemX, fragment.y, imageWidth, imageHeight, fontSizes[index], faces[index]))
					return fragment.y + fragment.height
				}
				return flowInlineBlockFragments(item, lines[index], itemX, itemWidth, y, bottom, itemPages, itemOpts)
			},
		})
		x += widths[index] + columnGap
	}
	return layoutParallelFlows(pages, y, opts.top(), opts, parallelItems)
}

func insertFlexContainerFragments(f *boxes.Box, pages *[]Page, firstPage, firstBlock int, top, width float64, endY float64, opts Options) {
	lastOffset := len(*pages) - 1 - firstPage
	for offset := 0; offset <= lastOffset; offset++ {
		pageIndex := firstPage + offset
		if pageIndex >= len(*pages) {
			break
		}
		page := &(*pages)[pageIndex]
		start := 0
		if offset == 0 {
			start = firstBlock
		}
		if start >= page.blockCount() {
			continue
		}
		fragmentTop := opts.top()
		if offset == 0 {
			fragmentTop = top
		}
		fragmentBottom := math.Inf(-1)
		for _, node := range page.paintNodes()[start:] {
			block := &node.Placed
			fragmentBottom = math.Max(fragmentBottom, block.Y+block.Height)
		}
		last := offset == lastOffset
		if last {
			fragmentBottom = math.Max(fragmentBottom, endY+f.Style.PaddingBottom+f.Style.BorderBottom.Width)
		}
		insertContainerDecoration(page, start, f, fragmentTop, width, math.Max(0, fragmentBottom-fragmentTop), offset == 0, last, opts)
	}
}

func placeFlexRowOnFreshPage(f *boxes.Box, rowHeight, y, bottom float64, pages *[]Page, opts Options) float64 {
	pageHasContent := (*pages)[len(*pages)-1].blockCount() > 0
	pageTop := opts.top() + f.Style.MarginTop + f.Style.BorderTop.Width + f.Style.PaddingTop
	fragment := placeFragmentUnitForPage(fragmentUnit{height: rowHeight}, y, pageTop, bottom, pageHasContent, pages, opts)
	return fragment.y
}

// flowFlexRowAcrossPages fragments oversized text rows at line boundaries.
// Rows that fit on a page still use the existing keep-together placement.
func flowFlexRowAcrossPages(f *boxes.Box, row flexRow, y float64, widths []float64, lines [][]inlineStream, heights, fontSizes []float64, faces []*fonts.Face, alignments []boxes.FlexAlignment, pages *[]Page, opts Options, contentWidth, columnGap float64, overflowClip *ClipRect, bottom float64) float64 {
	lineCount := 0
	for _, index := range row {
		lineCount = max(lineCount, len(lines[index]))
	}
	pageBase := len(*pages) - 1
	pageHasContent := (*pages)[len(*pages)-1].blockCount() > 0
	measure := func(firstLine, endLine int) float64 {
		return flexRowFragmentHeight(f, row, lines, heights, firstLine, endLine, firstLine == 0, endLine == lineCount, opts)
	}
	var fragments []fragmentRange
	if len(opts.PageRules) != 0 {
		fragments = planFragmentsByPage(make([]fragmentUnit, lineCount), y, pageHasContent, pageBoundsForOptions(opts, pageBase, f.Style.MarginTop), measure)
	} else {
		fragments = planFragments(make([]fragmentUnit, lineCount), y, opts.top()+f.Style.MarginTop, bottom, pageHasContent, measure)
	}
	for _, fragment := range fragments {
		ensurePageIndex(pages, pageBase+fragment.pageOffset)
		firstLine, lastLine := fragment.first, fragment.end
		fragmentHeight := fragment.height
		y = fragment.y
		fragmentLines := make([][]inlineStream, len(lines))
		fragmentItemHeights := make([]float64, len(heights))
		fragmentFlex := flexRowFragmentBox(f, firstLine == 0, lastLine == lineCount)
		for _, index := range row {
			if f.Children[index].Image != nil {
				if firstLine == 0 {
					fragmentItemHeights[index] = heights[index]
				}
			} else if firstLine < len(lines[index]) {
				end := min(lastLine, len(lines[index]))
				fragmentLines[index] = lines[index][firstLine:end]
				fragmentItemHeights[index] = inlineHeight(fragmentLines[index], fragmentFlex.Children[index], opts) + verticalInsets(fragmentFlex.Children[index])
			}
		}
		startBlock := (*pages)[len(*pages)-1].blockCount()
		flowFlexRow(fragmentFlex, row, y, fragmentHeight, widths, fragmentLines, fragmentItemHeights, fontSizes, faces, alignments, pages, opts, contentWidth, columnGap, overflowClip, false, firstLine == 0)
		page := &(*pages)[len(*pages)-1]
		blockOffset := 0
		for _, index := range row {
			if f.Children[index].Image != nil && firstLine > 0 {
				continue
			}
			block := page.block(startBlock + blockOffset)
			blockOffset++
			block.Text = flexFragmentText(fragmentLines[index])
			*block = fragmentPlacedBox(*block, firstLine == 0, lastLine == lineCount)
		}
		y += fragmentHeight
	}
	return y
}

func flexRowFragmentHeight(f *boxes.Box, row flexRow, lines [][]inlineStream, heights []float64, firstLine, endLine int, firstFragment, lastFragment bool, opts Options) float64 {
	height := 0.0
	for _, index := range row {
		item := f.Children[index]
		if item.Image != nil {
			if firstFragment {
				height = math.Max(height, heights[index])
			}
			continue
		}
		if firstLine >= len(lines[index]) {
			continue
		}
		end := min(endLine, len(lines[index]))
		itemHeight := inlineHeight(lines[index][firstLine:end], item, opts)
		if firstFragment {
			itemHeight += boxInsets(item).top
		}
		if lastFragment {
			itemHeight += boxInsets(item).bottom
		}
		height = math.Max(height, itemHeight)
	}
	return height
}

func flexRowFragmentBox(f *boxes.Box, firstFragment, lastFragment bool) *boxes.Box {
	fragment := *f
	fragment.Children = append([]*boxes.Box(nil), f.Children...)
	for i, child := range f.Children {
		fragment.Children[i] = fragmentBox(child, firstFragment, lastFragment)
	}
	return &fragment
}

func flexFragmentHasRadius(item *boxes.Box) bool {
	for _, radius := range item.Style.BorderCornerRadii {
		if radius.X.Value != 0 || radius.Y.Value != 0 {
			return true
		}
	}
	return false
}

func flexFragmentText(lines []inlineStream) string {
	var text strings.Builder
	for lineIndex, line := range lines {
		if lineIndex > 0 {
			text.WriteByte(' ')
		}
		for index, word := range line {
			if index > 0 && word.before.advance > 0 {
				text.WriteByte(' ')
			}
			text.WriteString(word.text)
		}
	}
	return strings.TrimSpace(text.String())
}
