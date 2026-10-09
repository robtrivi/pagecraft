package layout

import (
	"sort"

	"github.com/robtrivi/pagecraft-go/boxes"
)

type tableCellLine struct {
	words     inlineStream
	owner     *boxes.Box
	blocks    []*boxes.Box
	nested    []*FragmentNode
	height    float64
	gapBefore float64
	xOffset   float64
	width     float64
}

type tableCellSegment struct {
	owner                   *boxes.Box
	blocks                  []*boxes.Box
	nested                  []*FragmentNode
	height                  float64
	lines                   []inlineStream
	marginTop, marginBottom float64
	insetTop, insetBottom   float64
	xOffset, contentWidth   float64
}

func tableCellContent(cell *boxes.Box, contentWidth float64, opts Options) ([]tableCellLine, float64) {
	segments := tableCellParts(cell, cell.Parts, contentWidth, opts)
	if len(cell.Parts) == 0 {
		segments = []tableCellSegment{tableCellBlockSegment(cell, contentWidth, opts)}
	}
	var lines []tableCellLine
	pendingBottomMargin, pendingBottomInset := 0.0, 0.0
	for _, segment := range segments {
		gap := pendingBottomInset + collapseMargins(pendingBottomMargin, segment.marginTop) + segment.insetTop
		if len(segment.lines) == 0 {
			pendingBottomMargin = collapseMargins(pendingBottomMargin, segment.marginBottom)
			pendingBottomInset += segment.insetTop + segment.insetBottom
			continue
		}
		for index, words := range segment.lines {
			lineGap := 0.0
			if index == 0 {
				lineGap = gap
			}
			lines = append(lines, tableCellLine{
				words: words, owner: segment.owner, blocks: segment.blocks, gapBefore: lineGap,
				nested: segment.nested, height: segment.height,
				xOffset: segment.xOffset, width: segment.contentWidth,
			})
		}
		pendingBottomMargin = segment.marginBottom
		pendingBottomInset = segment.insetBottom
	}
	return lines, pendingBottomMargin + pendingBottomInset
}

func tableCellParts(owner *boxes.Box, parts []boxes.Part, width float64, opts Options) []tableCellSegment {
	var segments []tableCellSegment
	var inlineParts []boxes.Part
	flushInline := func() {
		if len(inlineParts) == 0 {
			return
		}
		words := tableCellInlineStream(owner, inlineParts, width, opts)
		segments = append(segments, tableCellSegment{
			owner: owner, lines: breakInlineStream(words, width, wraps(owner)), contentWidth: width,
		})
		inlineParts = nil
	}
	for _, part := range parts {
		if part.Child == nil || inlinePartChild(part.Child) {
			inlineParts = append(inlineParts, part)
			continue
		}
		flushInline()
		if part.Child.Tag == "table" {
			segments = append(segments, nestedTableCellSegments(part.Child, width, opts)...)
		} else {
			segments = append(segments, tableCellBlockSegments(part.Child, width, opts)...)
		}
	}
	flushInline()
	return segments
}

// nestedTableCellSegments measures a nested table in an isolated fragmentainer.
// Its rows become indivisible line units in the parent cell, allowing the outer
// row fragmenter to continue the nested grid at its own row boundaries.
func nestedTableCellSegments(table *boxes.Box, width float64, opts Options) []tableCellSegment {
	contentWidth := max(0, width-table.Style.MarginLeft-table.Style.MarginRight)
	measureOpts := opts
	measureOpts.PageWidth = contentWidth
	measureOpts.PageHeight = 1_000_000
	measureOpts.Margin = 0
	measureOpts.space = measureOpts.space.withPageMargin(0, true).withStartY(0)
	pages := []Page{{Number: 1}}
	height := flowTable(table, 0, &pages, measureOpts, contentWidth, measureOpts.PageHeight)
	if len(pages) == 0 || pages[0].blockCount() == 0 {
		return nil
	}
	rows := make(map[float64][]*FragmentNode)
	var rowTops []float64
	for _, node := range pages[0].paintNodes() {
		block := &node.Placed
		if _, exists := rows[block.Y]; !exists {
			rowTops = append(rowTops, block.Y)
		}
		rows[block.Y] = append(rows[block.Y], node)
	}
	sort.Float64s(rowTops)
	segments := make([]tableCellSegment, 0, len(rowTops))
	for index, rowTop := range rowTops {
		rowHeight := 0.0
		if index+1 < len(rowTops) {
			rowHeight = rowTops[index+1] - rowTop
		} else {
			rowHeight = max(0, height-rowTop)
		}
		rowBlocks := rows[rowTop]
		for blockIndex := range rowBlocks {
			rowHeight = max(rowHeight, rowBlocks[blockIndex].Placed.Y+rowBlocks[blockIndex].Placed.Height-rowTop)
			translatePlacedBox(&rowBlocks[blockIndex].Placed, 0, -rowTop)
		}
		segment := tableCellSegment{
			owner: table, nested: rowBlocks, height: rowHeight, lines: []inlineStream{nil},
			contentWidth: contentWidth,
		}
		if index == 0 {
			segment.marginTop = table.Style.MarginTop
		}
		if index == len(rowTops)-1 {
			segment.marginBottom = table.Style.MarginBottom
		}
		segments = append(segments, segment)
	}
	return segments
}

func tableCellBlockSegments(block *boxes.Box, width float64, opts Options) []tableCellSegment {
	contentWidth := max(0, width-horizontalInsets(block)-block.Style.MarginLeft-block.Style.MarginRight)
	hasBlockChild := false
	for _, part := range block.Parts {
		if part.Child != nil && !inlinePartChild(part.Child) {
			hasBlockChild = true
			break
		}
	}
	if !hasBlockChild {
		words := buildInlineStream(block, opts)
		return []tableCellSegment{{
			owner: block, blocks: []*boxes.Box{block}, lines: breakInlineStream(words, contentWidth, wraps(block)),
			marginTop:    block.Style.MarginTop,
			marginBottom: block.Style.MarginBottom,
			insetTop:     block.Style.BorderTop.Width + block.Style.PaddingTop,
			insetBottom:  block.Style.BorderBottom.Width + block.Style.PaddingBottom,
			xOffset:      block.Style.MarginLeft + block.Style.BorderLeft.Width + block.Style.PaddingLeft,
			contentWidth: contentWidth,
		}}
	}
	segments := tableCellParts(block, block.Parts, contentWidth, opts)
	if len(segments) == 0 {
		return nil
	}
	first, last := &segments[0], &segments[len(segments)-1]
	first.marginTop = collapseMargins(block.Style.MarginTop, first.marginTop)
	first.insetTop += block.Style.BorderTop.Width + block.Style.PaddingTop
	last.marginBottom = collapseMargins(last.marginBottom, block.Style.MarginBottom)
	last.insetBottom += block.Style.BorderBottom.Width + block.Style.PaddingBottom
	for index := range segments {
		segments[index].blocks = append([]*boxes.Box{block}, segments[index].blocks...)
		segments[index].xOffset += block.Style.MarginLeft + block.Style.BorderLeft.Width + block.Style.PaddingLeft
		segments[index].contentWidth = contentWidth
	}
	return segments
}

func tableCellBlockSegment(block *boxes.Box, width float64, opts Options) tableCellSegment {
	return tableCellBlockSegments(block, width, opts)[0]
}

func tableCellInlineStream(owner *boxes.Box, parts []boxes.Part, width float64, opts Options) inlineStream {
	var words []inlineDraft
	for _, part := range parts {
		if part.Child == nil {
			start := len(words)
			words = appendTextDrafts(words, owner, part.Text, opts)
			for index := start; index < len(words); index++ {
				words[index].markerShape = part.ListMarkerShape
			}
		} else if part.Child.Tag == "br" {
			words = appendHardBreakDraft(words, owner, opts)
		} else if part.Child.Image != nil {
			words = append(words, inlineImageDraft(part.Child, width, opts))
		} else {
			ownerScope := &inlineScope{box: owner}
			childScope := &inlineScope{box: part.Child, parent: ownerScope, depth: 1}
			words = append(words, collectFlattenInlineDraftsWithWidth(part.Child, width, opts, childScope)...)
		}
	}
	return resolveInlineStream(words)
}

func tableCellLineHeight(line tableCellLine, opts Options) float64 {
	if line.nested != nil {
		return line.height
	}
	return lineHeightW(line.words, line.owner, opts)
}

func appendNestedTableBlocks(page *Page, lines []tableCellLine, x, y float64, opts Options, parent *fragmentOccurrence) {
	for _, line := range lines {
		y += line.gapBefore
		for _, nested := range line.nested {
			translatePlacedBox(&nested.Placed, x+line.xOffset, y)
			bindFragmentChild(parent, &nested.Placed)
			page.emitBlock(nested.Placed)
		}
		y += tableCellLineHeight(line, opts)
	}
}

func appendPageDecorations(page *Page, ownerPaintIndex int, decorations []PlacedBox) {
	owner := *page.block(ownerPaintIndex)
	parents := map[*boxes.Box]*fragmentOccurrence{owner.source: owner.fragmentRef}
	for decorationIndex, decoration := range decorations {
		decoration.fragmentRef = newFragmentOccurrence(decoration.source)
		if parent := parents[decoration.sourceParent]; parent != nil {
			decoration.parentFragmentRef = parent
		}
		page.emitDecoration(ownerPaintIndex, decorationIndex, decoration)
		parents[decoration.source] = decoration.fragmentRef
	}
}

func translatePlacedBox(block *PlacedBox, dx, dy float64) {
	block.X += dx
	block.Y += dy
	for lineIndex := range block.Runs {
		for runIndex := range block.Runs[lineIndex] {
			block.Runs[lineIndex][runIndex].X += dx
			block.Runs[lineIndex][runIndex].Y += dy
		}
	}
	for index := range block.ClipRects {
		block.ClipRects[index].X += dx
		block.ClipRects[index].Y += dy
	}
	shiftChildDecorations(block.fragmentNode, dx, dy)
}

func tableCellRangeHeight(cell tableDoneCell, first, end int, lastFragment bool, opts Options) float64 {
	height := 0.0
	for index := first; index < end; index++ {
		height += cell.lines[index].gapBefore + tableCellLineHeight(cell.lines[index], opts)
	}
	if lastFragment {
		height += cell.after
	}
	if len(cell.lines) == 0 {
		height += inlineHeight(nil, cell.b, opts)
	}
	return height
}

func tableCellAllContentHeight(cell tableDoneCell, opts Options) float64 {
	return tableCellRangeHeight(cell, 0, len(cell.lines), true, opts)
}

func tableCellWords(lines []tableCellLine) []inlineStream {
	words := make([]inlineStream, len(lines))
	for index, line := range lines {
		words[index] = line.words
	}
	return words
}

func assignTableCellRuns(cell tableDoneCell, lines []tableCellLine, x, width, top float64, opts Options) [][]TextRun {
	runs := make([][]TextRun, 0, len(lines))
	y := top
	for _, line := range lines {
		y += line.gapBefore
		if line.nested != nil {
			runs = append(runs, nil)
			y += line.height
			continue
		}
		lineWidth := min(line.width, max(0, width-line.xOffset))
		lineRuns := assignRuns(line.owner, []inlineStream{line.words}, x+line.xOffset, lineWidth, y, opts)
		if len(lineRuns) > 0 {
			runs = append(runs, lineRuns[0])
		}
		y += tableCellLineHeight(line, opts)
	}
	return runs
}
