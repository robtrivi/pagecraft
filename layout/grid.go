package layout

import (
	"math"

	"github.com/robtrivi/pagecraft/boxes"
	"github.com/robtrivi/pagecraft/fonts"
)

// Grid supports row/column auto-flow with fixed px/pt tracks, fr and
// minmax(fixed, fr) tracks, automatic tracks, spans, explicit lines, and named
// areas. Each row stays intact on one page.

// flowGrid places children in grid tracks and returns the cursor.
type gridItem struct {
	box        *boxes.Box
	column     int
	columnSpan int
	rowSpan    int
}

type gridRowLayout struct {
	height         float64
	lines          [][]string
	sizes          []float64
	faces          []*fonts.Face
	widths         []float64
	contentHeights []float64
	nested         []bool
	children       [][]*FragmentNode
}

func flowGrid(g *boxes.Box, y float64, pages *[]Page, opts Options, usable, bottom float64) float64 {
	declaredCols := g.Style.GridCols
	if declaredCols < 1 {
		declaredCols = 1
	}
	cols := declaredCols
	if len(g.Children) == 0 {
		return y
	}
	insets := boxInsets(g)
	containerWidth := usedWidth(g, usable-g.Style.MarginLeft-g.Style.MarginRight)
	contentWidth := math.Max(0, containerWidth-insets.left-insets.right)
	contentX := opts.left() + g.Style.MarginLeft + insets.left
	flowColumn := g.Style.GridAutoFlowColumn
	flowRows := len(g.Style.GridRowHeights)
	if flowRows == 0 {
		flowRows = len(g.Style.GridAreas)
		if flowRows == 0 {
			flowRows = 1
		}
	}
	if flowColumn {
		implicitCapacity := len(g.Children)
		if implicitCapacity > 10000-declaredCols {
			implicitCapacity = 10000 - declaredCols
		}
		cols += implicitCapacity
	}
	columnGap := g.Style.ColumnGap + contentWidth*g.Style.ColumnGapPercent/100
	rowGap := g.Style.RowGap + contentWidth*g.Style.RowGapPercent/100
	var rows [][]gridItem
	var occupied [][]bool
	rowIndex, column := 0, 0
	for _, child := range g.Children {
		span := child.Style.ColSpan
		if span < 1 {
			span = 1
		}
		requestedColumn := -1
		if child.Style.GridColumnEndLine != 0 {
			startLine, startOK := resolveGridLine(child.Style.GridColumnLine, declaredCols)
			endLine, endOK := resolveGridLine(child.Style.GridColumnEndLine, declaredCols)
			if startOK && endOK && startLine < declaredCols && endLine > startLine && endLine <= declaredCols {
				requestedColumn = startLine
				span = endLine - startLine
			} else {
				span = 1
			}
		} else if child.Style.GridColumnLine != 0 {
			line, valid := resolveGridLine(child.Style.GridColumnLine, declaredCols)
			if valid && line < declaredCols && line+span <= cols {
				requestedColumn = line
			}
		}
		if span > cols {
			span = cols
		}
		rowSpan := child.Style.GridRowSpan
		if rowSpan < 1 {
			rowSpan = 1
		}
		requestedRow := -1
		if child.Style.GridRowEndLine != 0 {
			startLine, startOK := resolveGridLine(child.Style.GridRowLine, len(g.Style.GridRowHeights))
			endLine, endOK := resolveGridLine(child.Style.GridRowEndLine, len(g.Style.GridRowHeights))
			if startOK && endOK && startLine < len(g.Style.GridRowHeights) && endLine > startLine && endLine <= len(g.Style.GridRowHeights) {
				requestedRow = startLine
				rowSpan = endLine - startLine
			} else {
				rowSpan = 1
			}
		} else if child.Style.GridRowLine != 0 {
			line, valid := resolveGridLine(child.Style.GridRowLine, len(g.Style.GridRowHeights))
			if valid && line < len(g.Style.GridRowHeights) && line+rowSpan <= len(g.Style.GridRowHeights) {
				requestedRow = line
			}
		}
		if areaRow, areaColumn, areaRowSpan, areaColumnSpan, found := namedGridArea(g.Style.GridAreas, child.Style.GridAreaName); found && areaColumn+areaColumnSpan <= declaredCols {
			requestedRow = areaRow
			requestedColumn = areaColumn
			rowSpan = areaRowSpan
			span = areaColumnSpan
		}
		if g.Style.GridAutoFlowDense && requestedRow < 0 {
			rowIndex, column = 0, 0
		}
		for {
			candidateRow := rowIndex
			if requestedRow >= 0 {
				candidateRow = requestedRow
			}
			candidateColumn := column
			if requestedColumn >= 0 {
				candidateColumn = requestedColumn
			}
			if candidateColumn+span > cols {
				if requestedRow >= 0 {
					requestedRow = -1
				} else if flowColumn {
					flowColumn = false
					rowIndex++
					column = 0
				} else {
					rowIndex++
					column = 0
				}
				continue
			}
			for len(occupied) < candidateRow+rowSpan {
				occupied = append(occupied, make([]bool, cols))
			}
			free := true
			for row := candidateRow; row < candidateRow+rowSpan && free; row++ {
				for col := candidateColumn; col < candidateColumn+span; col++ {
					if occupied[row][col] {
						free = false
						break
					}
				}
			}
			if free {
				for len(rows) <= candidateRow {
					rows = append(rows, nil)
				}
				rows[candidateRow] = append(rows[candidateRow], gridItem{box: child, column: candidateColumn, columnSpan: span, rowSpan: rowSpan})
				for row := candidateRow; row < candidateRow+rowSpan; row++ {
					for col := candidateColumn; col < candidateColumn+span; col++ {
						occupied[row][col] = true
					}
				}
				if requestedRow >= 0 {
					rowIndex = candidateRow
				}
				if flowColumn {
					rowIndex = candidateRow + rowSpan
					column = candidateColumn
					if rowIndex+rowSpan > flowRows {
						rowIndex = 0
						column = candidateColumn + span
					}
				} else {
					column = candidateColumn + span
					if column == cols {
						rowIndex++
						column = 0
					}
				}
				break
			}
			if requestedRow >= 0 {
				column++
				if column+span > cols {
					requestedRow = -1
					rowIndex = candidateRow
					column = 0
				}
			} else if requestedColumn >= 0 {
				rowIndex++
				column = 0
			} else if flowColumn {
				rowIndex++
				if rowIndex+rowSpan > flowRows {
					rowIndex = 0
					column++
				}
			} else {
				column++
			}
		}
	}
	usedCols := declaredCols
	for _, row := range rows {
		for _, item := range row {
			if end := item.column + item.columnSpan; end > usedCols {
				usedCols = end
			}
		}
	}
	implicitColumns := usedCols - declaredCols
	implicitWidths := make([]float64, implicitColumns)
	implicitSpace := 0.0
	for col := 0; col < implicitColumns; col++ {
		patternIndex := 0
		if len(g.Style.GridAutoColumnWidths) > 0 {
			patternIndex = col % len(g.Style.GridAutoColumnWidths)
			implicitWidths[col] = g.Style.GridAutoColumnWidths[patternIndex]
		}
		percentage, isPercentage := gridAutoColumnPercentage(g, col)
		implicitWidths[col] += contentWidth * percentage / 100
		if implicitWidths[col] == 0 && !isPercentage && !gridAutoColumnFixedZero(g, col) && gridAutoColumnFraction(g, col) == 0 {
			implicitWidths[col] = gridAutoColumnIntrinsicWidth(rows, declaredCols+col, opts)
		}
	}
	explicitAutomaticMinimums := gridExplicitAutomaticMinimums(g, rows, declaredCols, opts)
	explicitWidths := gridTrackWidths(g, contentWidth, declaredCols, columnGap, 0, explicitAutomaticMinimums)
	explicitMinimums := make([]float64, declaredCols)
	gridSpanAutoColumnContributions(g, rows, implicitWidths, g.Style.GridAutoColumnWidths, g.Style.GridAutoColumnPercentages, g.Style.GridAutoColumnPercentageTracks, g.Style.GridAutoColumnFixedZeroTracks, g.Style.GridAutoColumnFractions, g.Style.GridAutoColumnMinimums, explicitWidths, explicitMinimums, declaredCols, columnGap, opts)
	gridResolveImplicitFractionColumns(g, rows, implicitWidths, contentWidth, declaredCols, columnGap, opts, explicitAutomaticMinimums)
	for _, width := range implicitWidths {
		implicitSpace += width
	}
	implicitSpace += columnGap * float64(implicitColumns)
	trackWidths := gridTrackWidths(g, contentWidth, declaredCols, columnGap, implicitSpace, explicitAutomaticMinimums)
	for column, minimum := range explicitMinimums {
		if minimum > trackWidths[column] {
			trackWidths[column] = minimum
		}
	}
	gridStretchImplicitAutoColumns(implicitWidths, g.Style.GridAutoColumnWidths, g.Style.GridAutoColumnPercentages, g.Style.GridAutoColumnPercentageTracks, g.Style.GridAutoColumnFixedZeroTracks, g.Style.GridAutoColumnFractions, trackWidths, columnGap, contentWidth)
	trackWidths = append(trackWidths, implicitWidths...)
	trackOffsets := make([]float64, usedCols)
	for i := 1; i < usedCols; i++ {
		trackOffsets[i] = trackOffsets[i-1] + trackWidths[i-1] + columnGap
	}
	for len(rows) < len(occupied) || len(rows) < len(g.Style.GridRowHeights) || len(rows) < len(g.Style.GridAreas) {
		rows = append(rows, nil)
	}
	layouts := make([]gridRowLayout, len(rows))
	for rowIndex, row := range rows {
		metrics := gridRowLayout{
			lines:          make([][]string, len(row)),
			sizes:          make([]float64, len(row)),
			faces:          make([]*fonts.Face, len(row)),
			widths:         make([]float64, len(row)),
			contentHeights: make([]float64, len(row)),
			nested:         make([]bool, len(row)),
			children:       make([][]*FragmentNode, len(row)),
		}
		for i, item := range row {
			it := item.box
			colW := 0.0
			for col := item.column; col < item.column+item.columnSpan; col++ {
				colW += trackWidths[col]
				if col > item.column {
					colW += columnGap
				}
			}
			metrics.widths[i] = colW
			fs := resolveSize(it, opts)
			metrics.sizes[i] = fs
			measure, face := measureFor(opts, it)
			metrics.faces[i] = face
			metrics.lines[i] = BreakLines(it.Text, colW, measure, fs)
			contentHeight := float64(len(metrics.lines[i])) * lineHeightFor(it, fs, opts)
			if len(it.Children) > 0 {
				metrics.children[i], contentHeight = layoutGridItemChildren(it, colW, opts)
				metrics.nested[i] = true
			}
			if it.Image != nil {
				_, contentHeight = gridRasterDimensions(it, colW, g, rowIndex, item.rowSpan, rowGap)
			}
			contentHeight = gridItemTrackContribution(it, contentHeight, metrics.nested[i], opts)
			metrics.contentHeights[i] = contentHeight
			if item.rowSpan == 1 && contentHeight > metrics.height {
				metrics.height = contentHeight
			}
		}
		if rowIndex < len(g.Style.GridRowHeights) {
			if gridRowIsFixedZero(g, rowIndex) {
				metrics.height = 0
			} else if g.Style.GridRowHeights[rowIndex] > 0 {
				metrics.height = g.Style.GridRowHeights[rowIndex]
			}
		} else if rowIndex >= len(g.Style.GridRowHeights) && len(g.Style.GridAutoRowHeights) > 0 {
			if gridRowIsFixedZero(g, rowIndex) {
				metrics.height = 0
			} else if height := gridImplicitRowHeight(g, rowIndex); height > 0 {
				metrics.height = height
			}
		}
		layouts[rowIndex] = metrics
	}
	for rowIndex, row := range rows {
		for i, item := range row {
			if item.rowSpan <= 1 {
				continue
			}
			spanHeight := rowGap * float64(item.rowSpan-1)
			autoRows := 0
			for row := rowIndex; row < rowIndex+item.rowSpan; row++ {
				spanHeight += layouts[row].height
				if gridRowIsAuto(g, row) {
					autoRows++
				}
			}
			deficit := layouts[rowIndex].contentHeights[i] - spanHeight
			if deficit > 0 && autoRows > 0 {
				share := deficit / float64(autoRows)
				for row := rowIndex; row < rowIndex+item.rowSpan; row++ {
					if gridRowIsAuto(g, row) {
						layouts[row].height += share
					}
				}
			}
		}
	}
	// Definite and minimum sizes create free space that automatic tracks
	// absorb. The maximum size limits flow without shrinking overflowing tracks.
	naturalHeight := rowGap * float64(max(0, len(layouts)-1))
	autoRows := 0
	for rowIndex := range layouts {
		naturalHeight += layouts[rowIndex].height
		if gridRowIsAuto(g, rowIndex) {
			autoRows++
		}
	}
	usedOuterHeight := usedHeight(g, naturalHeight, opts)
	targetContentHeight := math.Max(naturalHeight, usedOuterHeight-verticalInsets(g))
	freeHeight := targetContentHeight - naturalHeight
	if freeHeight > 0 && autoRows > 0 {
		share := freeHeight / float64(autoRows)
		for rowIndex := range layouts {
			if gridRowIsAuto(g, rowIndex) {
				layouts[rowIndex].height += share
			}
		}
	}
	y += g.Style.MarginTop
	containerTop := y
	startPage := len(*pages) - 1
	startBlock := (*pages)[startPage].blockCount()
	y += insets.top
	var overflowClip *ClipRect
	if clipsOverflow(g) {
		clip := ClipRect{Width: opts.PageWidth, Height: opts.PageHeight}
		if clipsOverflowX(g) {
			clip.X = opts.left() + g.Style.MarginLeft + g.Style.BorderLeft.Width
			clip.Width = math.Max(0, usedWidth(g, usable-g.Style.MarginLeft-g.Style.MarginRight)-g.Style.BorderLeft.Width-g.Style.BorderRight.Width)
		}
		if clipsOverflowY(g) && hasDeclaredHeight(g, opts) {
			clip.Y = containerTop + g.Style.BorderTop.Width
			clip.Height = usedDeclaredHeight(g, opts) - g.Style.BorderTop.Width - g.Style.BorderBottom.Width
			if clip.Height < 0 {
				clip.Height = 0
			}
		}
		overflowClip = &clip
	}
	spanningThrough := -1
	fragmentedThrough := -1
	freshPageCapacity := bottom - (opts.top() + g.Style.MarginTop + insets.top)
	for rowIndex, row := range rows {
		if rowIndex > 0 {
			if rowIndex <= fragmentedThrough {
				continue
			}
			y += rowGap
		}
		metrics := layouts[rowIndex]
		rowExtent := metrics.height
		for _, item := range row {
			if item.rowSpan > 1 {
				spanHeight := rowGap * float64(item.rowSpan-1)
				for r := rowIndex; r < rowIndex+item.rowSpan; r++ {
					spanHeight += layouts[r].height
				}
				if spanHeight > rowExtent {
					rowExtent = spanHeight
				}
			}
		}
		if y+rowExtent > bottom && rowIndex > spanningThrough && !(clipsOverflowY(g) && hasDeclaredHeight(g, opts)) {
			if gridRowCanFlowSubtrees(g, row, metrics, rowIndex, rowExtent, freshPageCapacity, opts) {
				y = flowGridSubtreesAcrossPages(g, row, metrics, rowIndex, rowGap, trackOffsets, y, pages, opts, bottom)
				continue
			}
			switch decideGridRowBreak(gridRowBreakContext{
				grid: g, rows: rows, layouts: layouts, row: row, metrics: metrics,
				rowIndex: rowIndex, rowGap: rowGap, rowExtent: rowExtent,
				freshPageCapacity: freshPageCapacity, available: bottom - y, opts: opts,
			}) {
			case gridRowBreakFragment:
				y = flowGridRowAcrossPages(g, row, metrics, rowIndex, y, trackOffsets, pages, opts, overflowClip, bottom)
				for _, item := range row {
					if item.rowSpan > 1 {
						fragmentedThrough = max(fragmentedThrough, rowIndex+item.rowSpan-1)
						spanningThrough = max(spanningThrough, rowIndex+item.rowSpan-1)
					}
				}
				continue
			case gridRowBreakMove:
				pageHasContent := (*pages)[len(*pages)-1].blockCount() > 0
				rowFragment := placeFragmentUnitForPage(fragmentUnit{height: rowExtent}, y, opts.top()+g.Style.MarginTop, bottom, pageHasContent, pages, opts)
				y = rowFragment.y
			}
		}
		p := &(*pages)[len(*pages)-1]
		for i, item := range row {
			it := item.box
			x := contentX + trackOffsets[item.column]
			if it.Image != nil {
				imageWidth, imageHeight := gridRasterDimensions(it, metrics.widths[i], g, rowIndex, item.rowSpan, rowGap)
				block := placed(it, it.Text, it.Image, x, y, imageWidth, imageHeight, metrics.sizes[i], metrics.faces[i])
				if overflowClip != nil {
					block.ClipRects = append(block.ClipRects, *overflowClip)
				}
				p.emitBlock(block)
				continue
			}
			itemHeight := metrics.height
			if item.rowSpan > 1 {
				itemHeight = rowGap * float64(item.rowSpan-1)
				for r := rowIndex; r < rowIndex+item.rowSpan; r++ {
					itemHeight += layouts[r].height
				}
				if rowIndex+item.rowSpan-1 > spanningThrough {
					spanningThrough = rowIndex + item.rowSpan - 1
				}
			}
			pb := placed(it, it.Text,
				nil, x, y, metrics.widths[i], itemHeight, metrics.sizes[i], metrics.faces[i])
			if metrics.nested[i] {
				pb.Text = ""
				p.emitBlock(pb)
				for _, child := range metrics.children[i] {
					translatePlacedBox(&child.Placed, x, y)
					if overflowClip != nil {
						child.Placed.ClipRects = append(child.Placed.ClipRects, *overflowClip)
					}
					p.emitBlock(child.Placed)
				}
				continue
			}
			measure, _ := measureFor(opts, it)
			pb.Runs = singleRuns(it, metrics.lines[i], x, y, lineHeightFor(it, metrics.sizes[i], opts), metrics.sizes[i], metrics.faces[i], measure)
			if overflowClip != nil {
				pb.ClipRects = append(pb.ClipRects, *overflowClip)
			}
			p.emitBlock(pb)
		}
		y += metrics.height
		if rowIndex >= spanningThrough && !(clipsOverflowY(g) && hasDeclaredHeight(g, opts)) {
			y = advanceGridCursor(y, opts.top()+g.Style.MarginTop, bottom, pages)
		}
	}
	contentHeight := math.Max(0, y-containerTop-insets.top)
	outerHeight := usedHeight(g, contentHeight, opts)
	if hasDeclaredHeight(g, opts) {
		outerHeight = usedDeclaredHeight(g, opts)
	}
	fragmented := len(*pages)-1 > startPage
	if fragmented {
		outerHeight = 0
	}
	outerEnd := containerTop + outerHeight
	if !fragmented && outerEnd > bottom+fragmentEpsilon && !(clipsOverflowY(g) && hasDeclaredHeight(g, opts)) {
		return continueGridOverflow(g, pages, startPage, startBlock, containerTop, bottom, outerEnd, containerWidth, opts)
	}
	endY := y + insets.bottom
	insertGridContainerFragments(g, pages, startPage, startBlock, containerTop, containerWidth, outerHeight, endY, opts.top()+g.Style.MarginTop, bottom, opts)
	if fragmented {
		return y + insets.bottom + g.Style.MarginBottom
	}
	return outerEnd + g.Style.MarginBottom
}

// layoutGridItemChildren lays out a grid item's normal block/inline subtree
// inside its track width. Grid placement owns the outer item box; the regular
// flow layout owns its descendants and their styles/decorations.
func layoutGridItemChildren(item *boxes.Box, width float64, opts Options) ([]*FragmentNode, float64) {
	if item.Style.Display == boxes.DisplayFlex || item.Style.Display == boxes.DisplayGrid || item.Style.ColCount > 1 || isTableFormattingContext(item) {
		return layoutGridItemFormattingContext(item, width, opts)
	}
	inset := boxInsets(item)
	childRoot := &boxes.Box{Tag: "root", Style: boxes.ComputedStyle{Display: boxes.DisplayBlock}, Children: nestedContainerItems(item), Fonts: opts.Faces, FontVariants: opts.FontVariants}
	if len(childRoot.Children) == 0 {
		return nil, inset.top + inset.bottom
	}
	childOpts := opts
	childOpts.PageWidth = width
	childOpts.PageHeight = unboundedFragmentainerPageHeight(opts.PageHeight)
	childOpts.Margin = 0
	childOpts.space = childOpts.space.withPageMargin(0, true).
		withMargins(inset.left, inset.right, 0, 0).
		withStartY(inset.top)
	childPages, end := flowLayout(childRoot, childOpts)
	if len(childPages) == 0 {
		return nil, inset.top + inset.bottom
	}
	return append([]*FragmentNode(nil), childPages[0].paintNodes()...), math.Max(inset.top+inset.bottom, end+inset.bottom)
}

func layoutGridItemFormattingContext(item *boxes.Box, width float64, opts Options) ([]*FragmentNode, float64) {
	resolvedItem := withUsedBorderBoxWidth(item, width)
	childRoot := &boxes.Box{Tag: "root", Style: boxes.ComputedStyle{Display: boxes.DisplayBlock}, Children: []*boxes.Box{resolvedItem}, Fonts: opts.Faces, FontVariants: opts.FontVariants}
	childOpts := opts
	childOpts.PageWidth = width
	childOpts.PageHeight = unboundedFragmentainerPageHeight(opts.PageHeight)
	childOpts.Margin = 0
	childOpts.space = childOpts.space.withPageMargin(0, true).
		withMargins(0, 0, 0, 0).
		withStartY(0)
	childPages, end := flowLayout(childRoot, childOpts)
	if len(childPages) == 0 {
		return nil, 0
	}
	blocks := make([]*FragmentNode, 0, childPages[0].blockCount())
	for _, node := range childPages[0].paintNodes() {
		if node.Placed.source != resolvedItem {
			blocks = append(blocks, node)
		}
	}
	contentHeight := math.Max(0, end-resolvedItem.Style.MarginTop-resolvedItem.Style.MarginBottom)
	return blocks, contentHeight
}
