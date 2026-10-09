package layout

import "github.com/robtrivi/pagecraft-go/boxes"

type tableCellLineRange struct {
	first int
	end   int
}

func maxTableCellLines(cells []tableDoneCell) int {
	count := 0
	for _, cell := range cells {
		count = max(count, len(cell.lines))
	}
	return count
}

func fragmentTableRow(done []tableDoneCell, y, pageTop, pageBottom float64, breakBefore fragmentBreak, pages *[]Page, opts Options, colWidths []float64, collapsed bool, repeatedHeaders [][]tableDoneCell, repeatedHeaderHeights []float64, repeatHeaders bool, spanning []*tableSpanningCell, rowIndex int, allSpans *[]*tableSpanningCell, alignCompletedCellsTop bool) float64 {
	lineOffsets := make([]int, len(done))
	rowSpans := make([]*tableSpanningCell, len(done))
	hasLines := false
	for _, cell := range done {
		hasLines = hasLines || len(cell.lines) > 0
	}
	if !hasLines {
		return y
	}
	repeatAtStart := false
	if breakBefore == fragmentBreakPage {
		pageHasContent := (*pages)[len(*pages)-1].blockCount() > 0
		forced := placeFragmentUnit(fragmentUnit{breakBefore: breakBefore}, y, pageTop, pageBottom, pageHasContent, pages)
		y = forced.y
		repeatAtStart = forced.pageOffset > 0
	}
	firstRowFragment := true
	startPage := len(*pages) - 1
	rowPageHasContent := false
	for {
		pageIndex := len(*pages) - 1
		if pageIndex > startPage || repeatAtStart {
			y = pageTop
			if repeatHeaders {
				for index, header := range repeatedHeaders {
					y = placeTableRowCells(header, fullTableCellRanges(header), repeatedHeaderHeights[index], y, true, true, pages, opts, colWidths, collapsed, true, false)
				}
			}
			repeatAtStart = false
		}
		available := pageBottom - y
		ranges := make([]tableCellLineRange, len(done))
		progress := false
		for index, cell := range done {
			ranges[index].first = lineOffsets[index]
			ranges[index].end = lineOffsets[index]
			lineCount := len(cell.lines)
			if lineOffsets[index] >= lineCount {
				continue
			}
			height := 0.0
			if firstRowFragment {
				top, _, _, _ := tableCellBorderInsets(cell.b, collapsed)
				height = cell.b.Style.PaddingTop + top
			}
			for lineIndex := lineOffsets[index]; lineIndex < lineCount; lineIndex++ {
				line := cell.lines[lineIndex]
				candidate := height + line.gapBefore + tableCellLineHeight(line, opts)
				if lineIndex == lineCount-1 {
					_, _, bottomBorder, _ := tableCellBorderInsets(cell.b, collapsed)
					candidate += cell.after + cell.b.Style.PaddingBottom + bottomBorder
				}
				if candidate > available+fragmentEpsilon {
					break
				}
				height = candidate
				ranges[index].end++
				progress = true
			}
		}

		rowComplete := true
		for index, cell := range done {
			if ranges[index].end < len(cell.lines) {
				rowComplete = false
				break
			}
		}
		if rowComplete {
			for index, cell := range done {
				if ranges[index].end == ranges[index].first {
					continue
				}
				if tableCellFragmentHeight(cell, ranges[index], firstRowFragment, true, collapsed, opts) > available+fragmentEpsilon {
					ranges[index].end--
					rowComplete = false
					progress = true
				}
			}
		}
		progress = false
		for index := range ranges {
			progress = progress || ranges[index].end > ranges[index].first
		}
		if !progress {
			if rowPageHasContent {
				*pages = append(*pages, Page{Number: len(*pages) + 1})
				rowPageHasContent = false
				continue
			}
			// An unbreakable line taller than the page must still advance.
			for index, cell := range done {
				if lineOffsets[index] < len(cell.lines) {
					ranges[index].end++
					progress = true
					break
				}
			}
			rowComplete = true
			for index, cell := range done {
				if ranges[index].end < len(cell.lines) {
					rowComplete = false
					break
				}
			}
		}

		rowHeight := 0.0
		for index, cell := range done {
			rowHeight = max(rowHeight, tableCellFragmentHeight(cell, ranges[index], firstRowFragment, rowComplete, collapsed, opts))
		}
		extendTableSpanFragments(spanning, pageIndex, rowIndex, y, rowHeight, rowComplete, pages)
		firstBlockIndex := (*pages)[pageIndex].blockCount()
		placeTableRowCells(done, ranges, rowHeight, y, firstRowFragment, rowComplete, pages, opts, colWidths, collapsed, false, alignCompletedCellsTop && !rowComplete)
		for index, cell := range done {
			if cell.rowSpan <= 1 {
				continue
			}
			if rowSpans[index] == nil {
				rowSpans[index] = &tableSpanningCell{
					page: pageIndex, block: firstBlockIndex + index, endRow: rowIndex + cell.rowSpan,
					align: cell.b.Style.VAlign, segments: make(map[int]int),
				}
				*allSpans = append(*allSpans, rowSpans[index])
			}
			rowSpans[index].segments[pageIndex] = firstBlockIndex + index
			rowSpans[index].height += rowHeight
		}
		y += rowHeight
		rowPageHasContent = true
		for index := range done {
			lineOffsets[index] = ranges[index].end
		}
		if rowComplete {
			return y
		}
		*pages = append(*pages, Page{Number: len(*pages) + 1})
		rowPageHasContent = false
		firstRowFragment = false
	}
}

func extendTableSpanFragments(spanning []*tableSpanningCell, pageIndex, rowIndex int, y, rowHeight float64, rowComplete bool, pages *[]Page) {
	for _, span := range spanning {
		spanComplete := rowComplete && span.endRow == rowIndex+1
		if blockIndex, ok := span.segments[pageIndex]; ok {
			segment := (*pages)[pageIndex].block(blockIndex)
			segment.Height += rowHeight
			segment.BorderBottom = boxes.BorderSide{}
			if spanComplete && segment.source != nil {
				segment.BorderBottom = segment.source.Style.BorderBottom
			}
		} else {
			original := *(*pages)[span.page].block(span.block)
			for segmentPage, segmentBlock := range span.segments {
				(*pages)[segmentPage].block(segmentBlock).BorderBottom = boxes.BorderSide{}
			}
			segment := original
			segment.Y = y
			segment.Height = rowHeight
			segment.Text = ""
			segment.Runs = nil
			segment.BorderTop = boxes.BorderSide{}
			segment.BorderBottom = boxes.BorderSide{}
			if spanComplete && segment.source != nil {
				segment.BorderBottom = segment.source.Style.BorderBottom
			}
			segment = fragmentPlacedBox(segment, false, spanComplete)
			segment.source = original.source
			segment.repeated = false
			segment.fragmentNode = nil
			segment.fragmentRef = newFragmentOccurrence(segment.source)
			blockIndex = (*pages)[pageIndex].blockCount()
			(*pages)[pageIndex].emitBlock(segment)
			span.segments[pageIndex] = blockIndex
		}
		span.height += rowHeight
	}
}

func tableCellFragmentHeight(cell tableDoneCell, lines tableCellLineRange, firstFragment, lastFragment, collapsed bool, opts Options) float64 {
	height := tableCellRangeHeight(cell, lines.first, lines.end, lastFragment, opts)
	borderTop, _, borderBottom, _ := tableCellBorderInsets(cell.b, collapsed)
	if firstFragment {
		height += cell.b.Style.PaddingTop + borderTop
	}
	if lastFragment {
		height += cell.b.Style.PaddingBottom + borderBottom
	}
	return height
}

func fullTableCellRanges(cells []tableDoneCell) []tableCellLineRange {
	ranges := make([]tableCellLineRange, len(cells))
	for index, cell := range cells {
		ranges[index].end = len(cell.lines)
	}
	return ranges
}

func placeTableRowCells(cells []tableDoneCell, ranges []tableCellLineRange, rowHeight, y float64, firstFragment, lastFragment bool, pages *[]Page, opts Options, colWidths []float64, collapsed, repeated, alignCompletedCellTop bool) float64 {
	page := &(*pages)[len(*pages)-1]
	for index, cell := range cells {
		lines := cell.lines[ranges[index].first:ranges[index].end]
		style := fragmentBox(cell.b, firstFragment, lastFragment)
		x := opts.left()
		for column := 0; column < cell.col && column < len(colWidths); column++ {
			x += colWidths[column]
		}
		width := 0.0
		for column := cell.col; column < cell.col+cell.span && column < len(colWidths); column++ {
			width += colWidths[column]
		}
		borderTop, _, borderBottom, borderLeft := tableCellBorderInsets(style, collapsed)
		font := fontFor(opts, style)
		contentWidth := max(0, width-horizontalInsets(style))
		text := inlineFragmentText(tableCellWords(lines))
		placedCell := placed(style, text, nil, x, y, width, rowHeight, cell.fs, font)
		placedCell.source = cell.b
		placedCell = fragmentPlacedBox(placedCell, firstFragment, lastFragment)
		offset := 0.0
		contentHeight := tableCellRangeHeight(cell, ranges[index].first, ranges[index].end, lastFragment, opts) + style.Style.PaddingTop + style.Style.PaddingBottom
		if cell.rowSpan <= 1 {
			switch style.Style.VAlign {
			case "middle":
				if !alignCompletedCellTop || ranges[index].end < len(cell.lines) {
					offset = (rowHeight - borderTop - borderBottom - contentHeight) / 2
				}
			case "bottom":
				if !alignCompletedCellTop || ranges[index].end < len(cell.lines) {
					offset = rowHeight - borderTop - borderBottom - contentHeight
				}
			}
		}
		contentX := x + borderLeft + style.Style.PaddingLeft
		contentY := y + borderTop + style.Style.PaddingTop + offset
		placedCell.Runs = assignTableCellRuns(cell, lines, contentX, contentWidth, contentY, opts)
		placedCell.repeated = repeated
		placedCell.fragmentRef = newFragmentOccurrence(cell.b)
		page.emitBlock(placedCell)
		blockIndex := page.blockCount() - 1
		appendPageDecorations(page, blockIndex, tableCellBlockDecorations(cell, lines, ranges[index].first, contentX, contentY, opts))
		appendNestedTableBlocks(page, lines, contentX, contentY, opts, page.block(blockIndex).fragmentRef)
	}
	return y + rowHeight
}
