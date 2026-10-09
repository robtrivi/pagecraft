package layout

import "github.com/robtrivi/pagecraft-go/boxes"

// tableCellBlockDecorations derives fragment boxes from the same line ranges used
// for cell sizing and pagination. The block ancestry preserves nested wrapper
// geometry while each source box gets its own background and fragment-specific borders.
func tableCellBlockDecorations(cell tableDoneCell, lines []tableCellLine, lineStart int, x, top float64, opts Options) []PlacedBox {
	type blockRange struct {
		box        *boxes.Box
		first, end int
	}
	sourceRanges := make(map[*boxes.Box]blockRange)
	for index, line := range cell.lines {
		for _, block := range line.blocks {
			rangeForBlock := sourceRanges[block]
			rangeForBlock.box = block
			if rangeForBlock.end == 0 {
				rangeForBlock.first = index
			}
			rangeForBlock.end = index + 1
			sourceRanges[block] = rangeForBlock
		}
	}
	var ranges []blockRange
	byBox := make(map[*boxes.Box]int)
	for index, line := range lines {
		for _, block := range line.blocks {
			if rangeIndex, ok := byBox[block]; ok {
				ranges[rangeIndex].end = index + 1
			} else {
				byBox[block] = len(ranges)
				ranges = append(ranges, blockRange{box: block, first: index, end: index + 1})
			}
		}
	}
	var decorations []PlacedBox
	for _, blockRange := range ranges {
		block := blockRange.box
		index := blockRange.first
		line := lines[index]
		lineY := top
		for prior := 0; prior < index; prior++ {
			lineY += lines[prior].gapBefore + tableCellLineHeight(lines[prior], opts)
		}
		lineY += line.gapBefore
		sourceRange := sourceRanges[block]
		chainIndex := -1
		for chain, candidate := range line.blocks {
			if candidate == block {
				chainIndex = chain
				break
			}
		}
		if chainIndex < 0 {
			continue
		}
		left := block.Style.MarginLeft + block.Style.BorderLeft.Width + block.Style.PaddingLeft
		right := block.Style.MarginRight + block.Style.BorderRight.Width + block.Style.PaddingRight
		beforeTop := block.Style.MarginTop
		for _, descendant := range line.blocks[chainIndex+1:] {
			left += descendant.Style.MarginLeft + descendant.Style.BorderLeft.Width + descendant.Style.PaddingLeft
			right += descendant.Style.MarginRight + descendant.Style.BorderRight.Width + descendant.Style.PaddingRight
			beforeTop += descendant.Style.MarginTop + descendant.Style.BorderTop.Width + descendant.Style.PaddingTop
		}
		boxX := x + line.xOffset - left
		boxWidth := line.width + left + right
		boxY := lineY - block.Style.BorderTop.Width - block.Style.PaddingTop
		if sourceRange.first == lineStart+blockRange.first {
			boxY -= beforeTop - block.Style.MarginTop
		} else {
			boxY = lineY
		}
		boxBottom := lineY + tableCellLineHeight(line, opts)
		for current := index + 1; current < blockRange.end; current++ {
			boxBottom += lines[current].gapBefore + tableCellLineHeight(lines[current], opts)
		}
		fragmentEnd := lineStart + len(lines)
		blockEndsHere := sourceRange.end > 0 && sourceRange.end <= fragmentEnd
		if blockEndsHere {
			insideBottom := 0.0
			for _, descendant := range line.blocks[chainIndex+1:] {
				insideBottom += descendant.Style.MarginBottom + descendant.Style.BorderBottom.Width + descendant.Style.PaddingBottom
			}
			boxBottom += insideBottom + block.Style.PaddingBottom + block.Style.BorderBottom.Width
		}
		placedBlock := placed(block, "", nil, boxX, boxY, boxWidth, max(0, boxBottom-boxY), resolveSize(block, opts), fontFor(opts, block))
		placedBlock.Runs = nil
		placedBlock = fragmentPlacedBox(placedBlock, sourceRange.first == lineStart+blockRange.first, blockEndsHere)
		decorations = append(decorations, placedBlock)
	}
	return decorations
}
