package layout

import (
	"math"
	"strings"

	"github.com/robtrivi/pagecraft/boxes"
	"github.com/robtrivi/pagecraft/fonts"
)

type flexRow []int

// flowFlex distributes items across flexible tracks and preserves inline content
// of each item and breaks into rows when flex-wrap:wrap runs out of space.
func flowFlex(f *boxes.Box, y float64, pages *[]Page, opts Options, usable, bottom float64) float64 {
	if f.Style.MarginAutoLeft || f.Style.MarginAutoRight {
		resolved := *f
		width := usedWidth(f, usable-f.Style.MarginLeft-f.Style.MarginRight)
		free := math.Max(0, usable-f.Style.MarginLeft-f.Style.MarginRight-width)
		switch {
		case f.Style.MarginAutoLeft && f.Style.MarginAutoRight:
			resolved.Style.MarginLeft += free / 2
			resolved.Style.MarginRight += free / 2
		case f.Style.MarginAutoLeft:
			resolved.Style.MarginLeft += free
		case f.Style.MarginAutoRight:
			resolved.Style.MarginRight += free
		}
		resolved.Style.MarginAutoLeft, resolved.Style.MarginAutoRight = false, false
		return flowFlex(&resolved, y, pages, opts, usable, bottom)
	}
	items := f.Children
	if len(items) == 0 {
		if strings.TrimSpace(f.Text) == "" {
			return y
		}
		anonymous := *f
		anonymous.Tag = "#anonymous-flex-item"
		anonymous.Style.Display = boxes.DisplayBlock
		anonymous.Children = nil
		anonymous.Style.MarginTop, anonymous.Style.MarginRight, anonymous.Style.MarginBottom, anonymous.Style.MarginLeft = 0, 0, 0, 0
		anonymous.Style.PaddingTop, anonymous.Style.PaddingRight, anonymous.Style.PaddingBottom, anonymous.Style.PaddingLeft = 0, 0, 0, 0
		anonymous.Style.BorderWidth = 0
		anonymous.Style.BorderTop, anonymous.Style.BorderRight, anonymous.Style.BorderBottom, anonymous.Style.BorderLeft = boxes.BorderSide{}, boxes.BorderSide{}, boxes.BorderSide{}, boxes.BorderSide{}
		anonymous.Style.HasBg = false
		anonymous.Style.Sizing.Width, anonymous.Style.Sizing.Height = boxes.CSSDimension{}, boxes.CSSDimension{}
		words := buildInlineStream(&anonymous, opts)
		anonymous.Style.Sizing.Width = boxes.CSSDimension{Kind: boxes.DimensionLength, Value: inlineStreamNaturalWidth(words)}
		anonymous.Style.BoxSizing = boxes.BoxSizingBorderBox
		container := *f
		container.Text = ""
		container.Parts = nil
		container.Children = []*boxes.Box{&anonymous}
		return flowFlex(&container, y, pages, opts, usable, bottom)
	}
	if f.Style.FlexDirection == boxes.FlexDirectionColumn {
		return flowFlexColumn(f, y, pages, opts, usable, bottom)
	}
	containerWidth := usedWidth(f, usable-f.Style.MarginLeft-f.Style.MarginRight)
	contentWidth := math.Max(0, containerWidth-horizontalInsets(f))
	columnGap := f.Style.ColumnGap + contentWidth*f.Style.ColumnGapPercent/100
	rowGap := f.Style.RowGap + contentWidth*f.Style.RowGapPercent/100
	widths := make([]float64, len(items))
	for i, item := range items {
		widths[i] = flexBaseSize(item, contentWidth, opts)
	}
	minimums := make([]float64, len(items))
	hypotheticalWidths := make([]float64, len(items))
	for i, item := range items {
		minimums[i] = flexItemMinContentWidth(item, opts, contentWidth)
		hypotheticalWidths[i] = hypotheticalFlexSize(item, widths[i], minimums[i], contentWidth)
	}

	rows := make([]flexRow, 0, len(items))
	if f.Style.FlexWrap != boxes.FlexWrap {
		row := make(flexRow, len(items))
		for i := range items {
			row[i] = i
		}
		rows = append(rows, row)
	} else {
		row := flexRow{}
		rowWidth := 0.0
		for i, width := range hypotheticalWidths {
			needed := width
			if len(row) > 0 {
				needed += columnGap
			}
			if len(row) > 0 && rowWidth+needed > contentWidth+1e-7 {
				rows = append(rows, row)
				row, rowWidth = nil, 0
				needed = width
			}
			row = append(row, i)
			rowWidth += needed
		}
		if len(row) > 0 {
			rows = append(rows, row)
		}
	}
	for _, row := range rows {
		resolveFlexLine(widths, row, items, minimums, contentWidth, columnGap)
	}
	lines := make([][]inlineStream, len(items))
	itemHeights := make([]float64, len(items))
	fontSizes := make([]float64, len(items))
	faces := make([]*fonts.Face, len(items))
	alignments := make([]boxes.FlexAlignment, len(items))
	for i, item := range items {
		fs := resolveSize(item, opts)
		fontSizes[i] = fs
		_, faces[i] = measureFor(opts, item)
		alignments[i] = item.Style.AlignSelf
		if alignments[i] == boxes.AlignUnset || alignments[i] == boxes.AlignAuto {
			alignments[i] = f.Style.AlignItems
		}
		if alignments[i] == boxes.AlignUnset {
			alignments[i] = boxes.AlignStretch
		}
		itemContentWidth := math.Max(0, widths[i]-horizontalInsets(item))
		lines[i] = breakInlineStream(buildInlineStream(item, opts), itemContentWidth, wraps(item))
		itemHeights[i] = inlineHeight(lines[i], item, opts) + verticalInsets(item)
		if hasDeclaredHeight(item, opts) {
			itemHeights[i] = usedHeight(item, itemHeights[i]-verticalInsets(item), opts)
		}
		if item.Image != nil {
			_, itemHeights[i] = rasterDimensions(item, itemContentWidth, 0)
			itemHeights[i] += verticalInsets(item)
		} else if flexItemHasBlockFlow(item) {
			itemHeights[i] = measureFlexItemHeight(withUsedBorderBoxWidth(item, widths[i]), widths[i], opts)
		}
	}

	y += f.Style.MarginTop
	containerTop := y
	containerX := opts.left() + f.Style.MarginLeft
	y += f.Style.BorderTop.Width + f.Style.PaddingTop
	startPage := len(*pages) - 1
	startBlock := (*pages)[startPage].blockCount()
	fragmentPlan := planContainerFragments(f, containerTop, boxInsets(f), 1, opts)
	keepOverflowOnPage := fragmentPlan.constrainDescendants
	var overflowClip *ClipRect
	if clipsOverflow(f) {
		overflowClip = overflowClipForBox(f, containerX, containerTop, containerWidth, usedDeclaredHeight(f, opts), opts, hasDeclaredHeight(f, opts))
	}
	freshPageCapacity := bottom - (opts.top() + f.Style.MarginTop + f.Style.BorderTop.Width + f.Style.PaddingTop)
	if hasDeclaredHeight(f, opts) {
		minHeight := math.Max(0, usedDeclaredHeight(f, opts)-verticalInsets(f))
		for _, row := range rows {
			if len(row) == 0 {
				continue
			}
			rowHeight := minHeight
			for _, index := range row {
				rowHeight = math.Max(rowHeight, itemHeights[index])
			}
			if y+rowHeight > bottom && !keepOverflowOnPage && rowCanFlowParallelSubtrees(f, row, alignments, rowHeight, freshPageCapacity, opts) {
				_, endY := flowFlexSubtreesAcrossPages(f, row, y, widths, lines, itemHeights, fontSizes, faces, alignments, pages, opts, contentWidth, columnGap, bottom)
				y = endY + rowGap
				continue
			}
			if y+rowHeight > bottom && !keepOverflowOnPage && rowCanFragment(f, row, lines, alignments, rowHeight, freshPageCapacity, opts) {
				y = flowFlexRowAcrossPages(f, row, y, widths, lines, itemHeights, fontSizes, faces, alignments, pages, opts, contentWidth, columnGap, overflowClip, bottom)
				y += rowGap
				continue
			}
			if y+rowHeight > bottom && !keepOverflowOnPage {
				y = placeFlexRowOnFreshPage(f, rowHeight, y, bottom, pages, opts)
			}
			flowFlexRow(f, row, y, rowHeight, widths, lines, itemHeights, fontSizes, faces, alignments, pages, opts, contentWidth, columnGap, overflowClip, fragmentPlan.isolateDescendants, true)
			y += rowHeight + rowGap
		}
	} else {
		for _, row := range rows {
			rowHeight := 0.0
			maxBaseline, maxBelow := 0.0, 0.0
			for _, index := range row {
				rowHeight = math.Max(rowHeight, itemHeights[index])
				if alignments[index] == boxes.AlignBaseline {
					baseline := fontSizes[index]
					if faces[index] != nil {
						if offset := faces[index].BaselineOffset(fontSizes[index], lineHeightFor(items[index], fontSizes[index], opts)); offset > 0 {
							baseline = offset
						}
					}
					maxBaseline = math.Max(maxBaseline, baseline)
					maxBelow = math.Max(maxBelow, itemHeights[index]-baseline)
				}
			}
			rowHeight = math.Max(rowHeight, maxBaseline+maxBelow)
			if y+rowHeight > bottom && !keepOverflowOnPage && rowCanFlowParallelSubtrees(f, row, alignments, rowHeight, freshPageCapacity, opts) {
				_, endY := flowFlexSubtreesAcrossPages(f, row, y, widths, lines, itemHeights, fontSizes, faces, alignments, pages, opts, contentWidth, columnGap, bottom)
				y = endY + rowGap
				continue
			}
			if y+rowHeight > bottom && !keepOverflowOnPage && rowCanFragment(f, row, lines, alignments, rowHeight, freshPageCapacity, opts) {
				y = flowFlexRowAcrossPages(f, row, y, widths, lines, itemHeights, fontSizes, faces, alignments, pages, opts, contentWidth, columnGap, overflowClip, bottom)
				y += rowGap
				continue
			}
			if y+rowHeight > bottom && !keepOverflowOnPage {
				y = placeFlexRowOnFreshPage(f, rowHeight, y, bottom, pages, opts)
			}
			flowFlexRow(f, row, y, rowHeight, widths, lines, itemHeights, fontSizes, faces, alignments, pages, opts, contentWidth, columnGap, overflowClip, fragmentPlan.isolateDescendants, true)
			y += rowHeight + rowGap
		}
	}
	if f.Style.Overflow == boxes.OverflowHidden && hasDeclaredHeight(f, opts) {
		return containerTop + usedDeclaredHeight(f, opts) + f.Style.MarginBottom
	}
	contentEnd := y - rowGap
	outerEnd := contentEnd + f.Style.PaddingBottom + f.Style.BorderBottom.Width
	if len(*pages)-1 == startPage {
		if hasHeightSizing(f) {
			contentHeight := math.Max(0, contentEnd-containerTop-f.Style.PaddingTop-f.Style.BorderTop.Width)
			outerEnd = containerTop + usedHeight(f, contentHeight, opts)
		}
		page := &(*pages)[startPage]
		_, face := measureFor(opts, f)
		block := placed(f, "", nil, containerX, containerTop, containerWidth, math.Max(0, outerEnd-containerTop), resolveSize(f, opts), face)
		if block.HasBg || block.BorderWidth > 0 || (block.BorderTop.Width > 0 || block.BorderRight.Width > 0 || block.BorderBottom.Width > 0 || block.BorderLeft.Width > 0) {
			insertAt := startBlock
			block.fragmentRef = newFragmentOccurrence(f)
			for blockIndex := insertAt; blockIndex < page.blockCount(); blockIndex++ {
				bindDirectFragmentChild(block.fragmentRef, page.block(blockIndex))
			}
			page.insertBlock(insertAt, block)
		}
	} else {
		insertFlexContainerFragments(f, pages, startPage, startBlock, containerTop, containerWidth, contentEnd, opts)
	}
	return outerEnd + f.Style.MarginBottom
}

func flowFlexColumn(f *boxes.Box, y float64, pages *[]Page, opts Options, usable, bottom float64) float64 {
	containerWidth := usedWidth(f, usable-f.Style.MarginLeft-f.Style.MarginRight)
	contentWidth := math.Max(0, containerWidth-horizontalInsets(f))
	rowGap := f.Style.RowGap + contentWidth*f.Style.RowGapPercent/100
	type columnItem struct {
		width, contentWidth, height, fontSize float64
		lines                                 []inlineStream
		face                                  *fonts.Face
	}
	items := make([]columnItem, len(f.Children))
	contentHeight := 0.0
	for index, item := range f.Children {
		width := contentWidth
		if !item.Style.Sizing.Width.IsAuto() {
			width = math.Min(contentWidth, usedWidth(item, contentWidth))
		}
		width = clampUsedDimension(width, item.Style.Sizing.MinWidth, item.Style.Sizing.MaxWidth, definiteSize(contentWidth), horizontalInsets(item), item.Style.BoxSizing)
		itemContentWidth := math.Max(0, width-horizontalInsets(item))
		lines := breakInlineStream(buildInlineStream(item, opts), itemContentWidth, wraps(item))
		fontSize := resolveSize(item, opts)
		_, face := measureFor(opts, item)
		itemHeight := inlineHeight(lines, item, opts) + verticalInsets(item)
		if hasDeclaredHeight(item, opts) {
			itemHeight = usedHeight(item, itemHeight-verticalInsets(item), opts)
		}
		if item.Image != nil {
			imageWidth, imageHeight := rasterDimensions(item, itemContentWidth, 0)
			width = imageWidth + horizontalInsets(item)
			itemHeight = imageHeight + verticalInsets(item)
		}
		items[index] = columnItem{width: width, contentWidth: itemContentWidth, height: itemHeight, fontSize: fontSize, lines: lines, face: face}
		contentHeight += itemHeight
	}
	if len(items) > 1 {
		contentHeight += rowGap * float64(len(items)-1)
	}
	y += f.Style.MarginTop
	containerTop := y
	containerX := opts.left() + f.Style.MarginLeft
	y += f.Style.BorderTop.Width + f.Style.PaddingTop
	startPage := len(*pages) - 1
	firstBlock := (*pages)[startPage].blockCount()
	if hasDeclaredHeight(f, opts) {
		freeSpace := math.Max(0, usedDeclaredHeight(f, opts)-verticalInsets(f)-contentHeight)
		switch f.Style.JustifyContent {
		case boxes.JustifyCenter:
			y += freeSpace / 2
		case boxes.JustifyFlexEnd:
			y += freeSpace
		case boxes.JustifySpaceBetween:
			if len(items) > 1 {
				rowGap += freeSpace / float64(len(items)-1)
			}
		}
	}
	for index, item := range f.Children {
		metrics := items[index]
		width := metrics.width
		itemHeight := metrics.height
		if width < contentWidth && (f.Style.AlignItems == boxes.AlignUnset || f.Style.AlignItems == boxes.AlignStretch) && (item.Style.AlignSelf == boxes.AlignUnset || item.Style.AlignSelf == boxes.AlignAuto) {
			width = clampUsedDimension(contentWidth, item.Style.Sizing.MinWidth, item.Style.Sizing.MaxWidth, definiteSize(contentWidth), horizontalInsets(item), item.Style.BoxSizing)
		}
		x := containerX + f.Style.BorderLeft.Width + f.Style.PaddingLeft
		align := item.Style.AlignSelf
		if align == boxes.AlignUnset || align == boxes.AlignAuto {
			align = f.Style.AlignItems
		}
		if align == boxes.AlignUnset {
			align = boxes.AlignStretch
		}
		if width < contentWidth {
			switch align {
			case boxes.AlignCenter:
				x += (contentWidth - width) / 2
			case boxes.AlignFlexEnd:
				x += contentWidth - width
			}
		}
		if flexColumnItemCanFlow(f, item, itemHeight, bottom, opts) {
			itemOpts := opts.withinPageHorizontalRange(x, width)
			if flexItemHasBlockFlow(item) {
				itemBox := withUsedBorderBoxWidth(item, width)
				y = flowFlexBlockItem(itemBox, y, pages, itemOpts, width, bottom)
			} else {
				y = flowInlineBlockFragments(item, metrics.lines, x, width, y, bottom, pages, itemOpts)
			}
			if index < len(f.Children)-1 {
				y += rowGap
			}
			continue
		}
		if y+itemHeight > bottom {
			*pages = append(*pages, Page{Number: len(*pages) + 1})
			y = opts.top() + f.Style.MarginTop + f.Style.BorderTop.Width + f.Style.PaddingTop
		}
		block := placed(item, item.Text, item.Image, x, y, width, itemHeight, metrics.fontSize, metrics.face)
		if item.Image == nil {
			block.Runs = assignRuns(item, metrics.lines, x+item.Style.BorderLeft.Width+item.Style.PaddingLeft, math.Max(0, width-horizontalInsets(item)), y+item.Style.BorderTop.Width+item.Style.PaddingTop, opts)
		}
		(*pages)[len(*pages)-1].emitBlock(block)
		y += itemHeight
		if index < len(f.Children)-1 {
			y += rowGap
		}
	}
	outerHeight := usedHeight(f, contentHeight, opts)
	outerEnd := containerTop + outerHeight
	if !hasDeclaredHeight(f, opts) && (!hasHeightSizing(f) || len(*pages)-1 != startPage) {
		outerEnd = y + f.Style.PaddingBottom + f.Style.BorderBottom.Width
	}
	if len(*pages)-1 == startPage {
		page := &(*pages)[startPage]
		_, face := measureFor(opts, f)
		wrapper := placed(f, "", nil, containerX, containerTop, containerWidth, math.Max(0, outerEnd-containerTop), resolveSize(f, opts), face)
		if wrapper.HasBg || wrapper.BorderWidth > 0 || wrapper.BorderTop.Width > 0 || wrapper.BorderRight.Width > 0 || wrapper.BorderBottom.Width > 0 || wrapper.BorderLeft.Width > 0 {
			wrapper.fragmentRef = newFragmentOccurrence(f)
			for blockIndex := firstBlock; blockIndex < page.blockCount(); blockIndex++ {
				bindDirectFragmentChild(wrapper.fragmentRef, page.block(blockIndex))
			}
			page.insertBlock(firstBlock, wrapper)
		}
	} else {
		contentEnd := outerEnd - f.Style.PaddingBottom - f.Style.BorderBottom.Width
		insertFlexContainerFragments(f, pages, startPage, firstBlock, containerTop, containerWidth, contentEnd, opts)
	}
	return outerEnd + f.Style.MarginBottom
}

func flowFlexRow(f *boxes.Box, row flexRow, y, rowHeight float64, widths []float64, lines [][]inlineStream, heights, fontSizes []float64, faces []*fonts.Face, alignments []boxes.FlexAlignment, pages *[]Page, opts Options, contentWidth, columnGap float64, overflowClip *ClipRect, isolateNestedFlow, emitImages bool) {
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
	page := &(*pages)[len(*pages)-1]
	maxBaseline := 0.0
	for _, index := range row {
		if alignments[index] != boxes.AlignBaseline {
			continue
		}
		baseline := fontSizes[index]
		if faces[index] != nil {
			if offset := faces[index].BaselineOffset(fontSizes[index], lineHeightFor(f.Children[index], fontSizes[index], opts)); offset > 0 {
				baseline = offset
			}
		}
		maxBaseline = math.Max(maxBaseline, baseline)
	}
	for _, index := range row {
		item := f.Children[index]
		itemY, itemHeight := y, rowHeight
		switch alignments[index] {
		case boxes.AlignFlexStart:
			itemHeight = heights[index]
		case boxes.AlignCenter:
			itemY += (rowHeight - heights[index]) / 2
			itemHeight = heights[index]
		case boxes.AlignFlexEnd:
			itemY += rowHeight - heights[index]
			itemHeight = heights[index]
		case boxes.AlignBaseline:
			baseline := fontSizes[index]
			if faces[index] != nil {
				if offset := faces[index].BaselineOffset(fontSizes[index], lineHeightFor(item, fontSizes[index], opts)); offset > 0 {
					baseline = offset
				}
			}
			itemY += maxBaseline - baseline
			itemHeight = heights[index]
		}
		itemOverflowClip := overflowClipForBox(item, x, itemY, widths[index], itemHeight, opts, hasDeclaredHeight(item, opts))
		if flexItemHasBlockFlow(item) {
			itemBox := withUsedBorderBoxWidth(item, widths[index])
			itemOpts := opts.withinPageHorizontalRange(x, widths[index])
			if isolateNestedFlow {
				itemOpts.PageHeight = unboundedFragmentainerPageHeight(opts.PageHeight)
			}
			firstPage := len(*pages) - 1
			firstBlock := (*pages)[firstPage].blockCount()
			flowFlexBlockItem(itemBox, itemY, pages, itemOpts, widths[index], opts.pageBottom())
			if overflowClip != nil {
				for pageIndex := firstPage; pageIndex < len(*pages); pageIndex++ {
					blockIndex := 0
					if pageIndex == firstPage {
						blockIndex = firstBlock
					}
					for blockIndex < (*pages)[pageIndex].blockCount() {
						block := (*pages)[pageIndex].block(blockIndex)
						block.ClipRects = append(block.ClipRects, *overflowClip)
						blockIndex++
					}
				}
			}
		} else if item.Image != nil {
			if emitImages {
				contentWidth := widths[index] - horizontalInsets(item)
				imageWidth, imageHeight := rasterDimensions(item, contentWidth, 0)
				block := placed(item, item.Text, item.Image, x, itemY, imageWidth, imageHeight, fontSizes[index], faces[index])
				if overflowClip != nil {
					block.ClipRects = append(block.ClipRects, *overflowClip)
				}
				if itemOverflowClip != nil {
					block.ClipRects = append(block.ClipRects, *itemOverflowClip)
				}
				page.emitBlock(block)
			}
		} else {
			block := placed(item, item.Text, nil, x, itemY, widths[index], itemHeight, fontSizes[index], faces[index])
			contentX := x + item.Style.BorderLeft.Width + item.Style.PaddingLeft
			contentY := itemY + item.Style.BorderTop.Width + item.Style.PaddingTop
			contentWidth := widths[index] - horizontalInsets(item)
			block.Runs = assignRuns(item, lines[index], contentX, contentWidth, contentY, opts)
			if overflowClip != nil {
				block.ClipRects = append(block.ClipRects, *overflowClip)
			}
			if itemOverflowClip != nil {
				block.ClipRects = append(block.ClipRects, *itemOverflowClip)
			}
			page.emitBlock(block)
		}
		x += widths[index] + columnGap
	}
}
