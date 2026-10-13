package layout

import (
	"github.com/robtrivi/pagecraft/boxes"
	"github.com/robtrivi/pagecraft/fonts"
)

// Table layout resolves intrinsic or explicit auto width and distributes
// columns. Colspan/rowspan reserve grid tracks. Normal rows stay together;
// rows larger than the printable area fragment each cell independently.

// tableCell represents a table cell and its spans.
type tableCell struct {
	b    *boxes.Box
	row  *boxes.Box
	span int
}

type tableDoneCell struct {
	b                     *boxes.Box
	col                   int
	span                  int
	rowSpan               int
	lines                 []tableCellLine
	after                 float64
	fs                    float64
	font                  *fonts.Face
	table                 *boxes.Box
	row                   *boxes.Box
	group                 *boxes.Box
	tableFirst, tableLast bool
	groupFirst, groupLast bool
	collapsedBorders      collapsedTableCellBorders
	borderModelReady      bool
}

type tableSpanningCell struct {
	page, block, endRow int
	height, contentH    float64
	align               string
	segments            map[int]int
}

func isTableFormattingContext(box *boxes.Box) bool {
	return box != nil && (box.Tag == "table" || box.Style.Display == boxes.DisplayTable)
}

func isTableRow(box *boxes.Box) bool {
	return box != nil && (box.Tag == "tr" || box.Style.Display == boxes.DisplayTableRow)
}

func isTableCell(box *boxes.Box) bool {
	return box != nil && (box.Tag == "td" || box.Tag == "th" || box.Style.Display == boxes.DisplayTableCell)
}

func isTableRowGroup(box *boxes.Box) bool {
	if box == nil {
		return false
	}
	switch box.Tag {
	case "thead", "tbody", "tfoot":
		return true
	}
	switch box.Style.Display {
	case boxes.DisplayTableRowGroup, boxes.DisplayTableHeaderGroup, boxes.DisplayTableFooterGroup:
		return true
	default:
		return false
	}
}

func isTableColumn(box *boxes.Box) bool {
	return box != nil && (box.Tag == "col" || box.Style.Display == boxes.DisplayTableColumn)
}

func isTableColumnGroup(box *boxes.Box) bool {
	return box != nil && (box.Tag == "colgroup" || box.Style.Display == boxes.DisplayTableColumnGroup)
}

func tableGridFromRows(rows []*boxes.Box) ([][]tableCell, int) {
	grid := make([][]tableCell, 0, len(rows))
	cols := 0
	for _, row := range rows {
		cells := tableCells(row)
		gridRow := make([]tableCell, 0, len(cells))
		columnCount := 0
		for _, cell := range cells {
			span := max(1, cell.Style.ColSpan)
			gridRow = append(gridRow, tableCell{b: cell, row: row, span: span})
			columnCount += span
		}
		cols = max(cols, columnCount)
		grid = append(grid, gridRow)
	}
	return grid, cols
}

func tableCellBorderInsets(cell *boxes.Box, collapsed bool) (top, right, bottom, left float64) {
	factor := 1.0
	if collapsed {
		factor = 0.5
	}
	return cell.Style.BorderTop.Width * factor, cell.Style.BorderRight.Width * factor,
		cell.Style.BorderBottom.Width * factor, cell.Style.BorderLeft.Width * factor
}

type collapsedTableCellBorders struct {
	top, right, bottom, left                                     boxes.BorderSide
	skipTopPaint, skipRightPaint, skipBottomPaint, skipLeftPaint bool
}

// collapsedCellBorderCandidates collects the borders that can contribute to
// one cell's grid edges. Row, group, and table vertical borders only apply at
// the outside of the row; their horizontal borders span every cell.
func collapsedCellBorderCandidates(cell, row, group, table *boxes.Box, firstCell, lastCell, tableFirst, tableLast, groupFirst, groupLast bool, columnLeft, columnRight []boxes.BorderSide) collapsedTableCellBorders {
	style := func(box *boxes.Box) boxes.ComputedStyle {
		if box == nil {
			return boxes.ComputedStyle{}
		}
		return box.Style
	}
	cellStyle, rowStyle, groupStyle, tableStyle := style(cell), style(row), style(group), style(table)
	resolve := func(sides ...boxes.BorderSide) boxes.BorderSide {
		winner := boxes.BorderSide{}
		for _, side := range sides {
			if collapsedBorderBeats(side, winner) {
				winner = side
			}
		}
		return winner
	}
	result := collapsedTableCellBorders{
		top:    resolve(cellStyle.BorderTop, rowStyle.BorderTop),
		bottom: resolve(cellStyle.BorderBottom, rowStyle.BorderBottom),
	}
	if groupFirst {
		result.top = resolve(result.top, groupStyle.BorderTop)
	}
	if groupLast {
		result.bottom = resolve(result.bottom, groupStyle.BorderBottom)
	}
	if tableFirst {
		result.top = resolve(result.top, tableStyle.BorderTop)
	}
	if tableLast {
		result.bottom = resolve(result.bottom, tableStyle.BorderBottom)
	}
	rightCandidates := []boxes.BorderSide{cellStyle.BorderRight}
	leftCandidates := []boxes.BorderSide{cellStyle.BorderLeft}
	if lastCell {
		rightCandidates = append(rightCandidates, rowStyle.BorderRight, groupStyle.BorderRight)
	}
	if firstCell {
		leftCandidates = append(leftCandidates, rowStyle.BorderLeft, groupStyle.BorderLeft)
	}
	// Blink resolves column groups before columns, then applies table borders
	// last. Keep the source order here so equal width and style ties follow that
	// origin precedence.
	rightCandidates = append(rightCandidates, columnRight...)
	leftCandidates = append(leftCandidates, columnLeft...)
	if lastCell {
		rightCandidates = append(rightCandidates, tableStyle.BorderRight)
	}
	if firstCell {
		leftCandidates = append(leftCandidates, tableStyle.BorderLeft)
	}
	result.right = resolve(rightCandidates...)
	result.left = resolve(leftCandidates...)
	return result
}

func collapsedBorderBeats(candidate, current boxes.BorderSide) bool {
	if candidate.Style == "hidden" {
		return current.Style != "hidden"
	}
	if current.Style == "hidden" {
		return false
	}
	if candidate.Width != current.Width {
		return candidate.Width > current.Width
	}
	styleRank := func(style string) int {
		switch style {
		case "double":
			return 8
		case "solid":
			return 7
		case "dashed":
			return 6
		case "dotted":
			return 5
		case "ridge":
			return 4
		case "outset":
			return 3
		case "groove":
			return 2
		case "inset":
			return 1
		default:
			return 0
		}
	}
	return styleRank(candidate.Style) > styleRank(current.Style)
}

func resolveCollapsedRowBorders(cells []tableDoneCell) []collapsedTableCellBorders {
	resolved := make([]collapsedTableCellBorders, len(cells))
	for index, cell := range cells {
		if cell.borderModelReady {
			resolved[index] = cell.collapsedBorders
		} else {
			resolved[index] = collapsedCellBorderCandidates(cell.b, cell.row, cell.group, cell.table,
				index == 0, index == len(cells)-1, cell.tableFirst, cell.tableLast, cell.groupFirst, cell.groupLast, nil, nil)
		}
	}
	for index := 1; index < len(cells); index++ {
		left, right := cells[index-1], cells[index]
		if left.col+left.span != right.col {
			continue
		}
		winner := resolved[index-1].right
		if collapsedBorderBeats(resolved[index].left, winner) {
			winner = resolved[index].left
		}
		resolved[index-1].right, resolved[index].left = winner, winner
		resolved[index].skipLeftPaint = true
	}
	return resolved
}

type tableColumnBorderSources struct {
	column, group        *boxes.Box
	groupStart, groupEnd bool
}

func tableColumnBorders(table *boxes.Box, cols int) []tableColumnBorderSources {
	sources := make([]tableColumnBorderSources, cols)
	column := 0
	for _, child := range table.Children {
		if !isTableColumn(child) && !isTableColumnGroup(child) {
			continue
		}
		if isTableColumnGroup(child) && len(child.Children) > 0 {
			start := column
			for _, col := range child.Children {
				if !isTableColumn(col) {
					continue
				}
				span := max(1, col.Style.TableColumnSpan)
				for track := 0; track < span && column < cols; track++ {
					sources[column] = tableColumnBorderSources{column: col, group: child}
					column++
				}
			}
			end := column - 1
			if start < cols && end >= start {
				sources[start].groupStart = true
				sources[end].groupEnd = true
			}
			continue
		}
		span := max(1, child.Style.TableColumnSpan)
		start := column
		for track := 0; track < span && column < cols; track++ {
			if isTableColumnGroup(child) {
				sources[column] = tableColumnBorderSources{group: child}
			} else {
				sources[column] = tableColumnBorderSources{column: child}
			}
			column++
		}
		end := column - 1
		if isTableColumnGroup(child) && start < cols && end >= start {
			sources[start].groupStart = true
			sources[end].groupEnd = true
		}
	}
	return sources
}

func tableCellColumnBorderCandidates(sources []tableColumnBorderSources, start, span int) (left, right []boxes.BorderSide) {
	if start >= len(sources) || span <= 0 {
		return nil, nil
	}
	end := min(len(sources)-1, start+span-1)
	currentStart, currentEnd := sources[start], sources[end]
	if currentStart.groupStart && currentStart.group != nil {
		left = append(left, currentStart.group.Style.BorderLeft)
	}
	if start > 0 {
		previous := sources[start-1]
		if previous.groupEnd && previous.group != nil {
			left = append(left, previous.group.Style.BorderRight)
		}
	}
	if currentStart.column != nil {
		left = append(left, currentStart.column.Style.BorderLeft)
	}
	if start > 0 && sources[start-1].column != nil {
		left = append(left, sources[start-1].column.Style.BorderRight)
	}
	if currentEnd.groupEnd && currentEnd.group != nil {
		right = append(right, currentEnd.group.Style.BorderRight)
	}
	if end+1 < len(sources) {
		next := sources[end+1]
		if next.groupStart && next.group != nil {
			right = append(right, next.group.Style.BorderLeft)
		}
	}
	if currentEnd.column != nil {
		right = append(right, currentEnd.column.Style.BorderRight)
	}
	if end+1 < len(sources) && sources[end+1].column != nil {
		right = append(right, sources[end+1].column.Style.BorderLeft)
	}
	return left, right
}

// collapsedTableBorderModel resolves each shared grid edge once and stores the
// winner on both cells that meet at the edge. Keeping this model at table scope
// avoids painting independent borders from adjacent rows over one another.
func collapsedTableBorderModel(table *boxes.Box, rows []*boxes.Box, grid [][]tableCell, cols int) map[*boxes.Box]collapsedTableCellBorders {
	columnBorders := tableColumnBorders(table, cols)
	borderRows := make([][]tableDoneCell, len(grid))
	occupied := make([][]bool, len(grid))
	for row := range occupied {
		occupied[row] = make([]bool, cols)
	}
	for rowIndex, row := range grid {
		column := 0
		for _, cell := range row {
			for column < cols && occupied[rowIndex][column] {
				column++
			}
			if column >= cols {
				break
			}
			span := min(max(1, cell.span), cols-column)
			group := tableRowGroup(cell.row)
			done := tableDoneCell{
				b: cell.b, col: column, span: span, rowSpan: max(1, cell.b.Style.RowSpan),
				table: table, row: cell.row, group: group,
				tableFirst: rowIndex == 0, tableLast: rowIndex == len(grid)-1,
				groupFirst: rowIndex == 0 || tableRowGroup(rows[rowIndex-1]) != group,
				groupLast:  rowIndex == len(grid)-1 || tableRowGroup(rows[rowIndex+1]) != group,
			}
			columnLeft, columnRight := tableCellColumnBorderCandidates(columnBorders, column, span)
			done.collapsedBorders = collapsedCellBorderCandidates(cell.b, cell.row, group, table,
				column == 0, column+span == cols, done.tableFirst, done.tableLast, done.groupFirst, done.groupLast, columnLeft, columnRight)
			done.borderModelReady = true
			borderRows[rowIndex] = append(borderRows[rowIndex], done)
			endRow := min(len(grid), rowIndex+done.rowSpan)
			for coveredRow := rowIndex; coveredRow < endRow; coveredRow++ {
				for coveredColumn := column; coveredColumn < column+span; coveredColumn++ {
					occupied[coveredRow][coveredColumn] = true
				}
			}
			column += span
		}
	}

	resolvedRows := make([][]collapsedTableCellBorders, len(borderRows))
	for rowIndex, cells := range borderRows {
		resolvedRows[rowIndex] = resolveCollapsedRowBorders(cells)
	}
	for rowIndex := 0; rowIndex+1 < len(borderRows); rowIndex++ {
		for upperIndex := range borderRows[rowIndex] {
			upper := &borderRows[rowIndex][upperIndex]
			if upper.rowSpan != 1 {
				continue
			}
			for lowerIndex := range borderRows[rowIndex+1] {
				lower := &borderRows[rowIndex+1][lowerIndex]
				if upper.col >= lower.col+lower.span || lower.col >= upper.col+upper.span {
					continue
				}
				upperBorder := resolvedRows[rowIndex][upperIndex].bottom
				lowerBorder := resolvedRows[rowIndex+1][lowerIndex].top
				winner := upperBorder
				if collapsedBorderBeats(lowerBorder, winner) {
					winner = lowerBorder
				}
				resolvedRows[rowIndex][upperIndex].bottom = winner
				resolvedRows[rowIndex+1][lowerIndex].top = winner
				resolvedRows[rowIndex+1][lowerIndex].skipTopPaint = true
			}
		}
	}

	resolved := make(map[*boxes.Box]collapsedTableCellBorders)
	for rowIndex, cells := range borderRows {
		for cellIndex, cell := range cells {
			resolved[cell.b] = resolvedRows[rowIndex][cellIndex]
		}
	}
	return resolved
}

func collapsedRowHeight(cells []tableDoneCell, opts Options) float64 {
	borders := resolveCollapsedRowBorders(cells)
	height := 0.0
	for index, cell := range cells {
		top := borders[index].top.Width * 0.5
		bottom := borders[index].bottom.Width * 0.5
		cellHeight := tableCellAllContentHeight(cell, opts) + cell.b.Style.PaddingTop + cell.b.Style.PaddingBottom + top + bottom
		height = max(height, cellHeight)
	}
	return height
}

func applyCollapsedCellBorders(block *PlacedBox, borders collapsedTableCellBorders) {
	block.BorderWidth = 0
	block.CollapsedTableCell = true
	block.BorderTop, block.BorderRight, block.BorderBottom, block.BorderLeft = borders.top, borders.right, borders.bottom, borders.left
	block.CollapsedBorderPaintSuppressed = [4]bool{
		borders.skipTopPaint, borders.skipRightPaint, borders.skipBottomPaint, borders.skipLeftPaint,
	}
}

// flowTable lays out t's rows from and and returns the resulting cursor.
func flowTable(t *boxes.Box, y float64, pages *[]Page, opts Options, usable, bottom float64) float64 {
	head, body := tableRows(t)
	rows := append(head, body...)
	headCount := len(head)
	grid, cols := tableGridFromRows(rows)
	if cols == 0 {
		return y
	}
	collapsedBorderModel := map[*boxes.Box]collapsedTableCellBorders(nil)
	if t.Style.BorderCollapse {
		collapsedBorderModel = collapsedTableBorderModel(t, rows, grid, cols)
	}
	tableWidth := tableUsedWidth(t, grid, cols, usable, opts)
	t = resolveHorizontalAutoMarginsWithWidth(t, usable, tableWidth)
	hSpacing, vSpacing := 0.0, 0.0
	if !t.Style.BorderCollapse {
		hSpacing, vSpacing = t.Style.TableBorderSpacingX, t.Style.TableBorderSpacingY
	}
	trackWidth := max(0, tableWidth-hSpacing*float64(cols+1))
	var colW []float64
	if t.Style.TableLayoutFixed && t.Style.Sizing.Width.Kind != boxes.DimensionAuto {
		colW = fixedTableColumnWidths(t, grid, cols, trackWidth)
	} else {
		colW = columnWidthsForTable(t, grid, cols, trackWidth, opts)
	}
	tableX := opts.left() + t.Style.MarginLeft
	colX := make([]float64, cols+1)
	colX[0] = hSpacing
	for i := 0; i < cols; i++ {
		colX[i+1] = colX[i] + colW[i] + hSpacing
	}
	cellW := func(col, span int) float64 {
		w := 0.0
		for s := 0; s < span && col+s < cols; s++ {
			w += colW[col+s]
		}
		return w
	}
	occ := [][]bool{}
	ensure := func(r int) {
		for len(occ) <= r {
			occ = append(occ, make([]bool, cols))
		}
	}
	var spanning []*tableSpanningCell
	currentRow := 0
	var headDone [][]tableDoneCell
	var headRh []float64
	if opts.tableOpacityCache == nil {
		opts.tableOpacityCache = make(map[*boxes.Box]OpacityGroup)
	}
	canRepeat := len(head) > 0
	y += vSpacing
	placeCells := func(done []tableDoneCell, rh, y float64, repeated bool) float64 {
		p := &(*pages)[len(*pages)-1]
		collapsedBorders := resolveCollapsedRowBorders(done)
		for cellIndex, d := range done {
			pb := placed(d.b, d.b.Text,
				nil, tableX+colX[d.col], y, cellW(d.col, d.span), rh, d.fs, d.font)
			if t.Style.BorderCollapse {
				applyCollapsedCellBorders(&pb, collapsedBorders[cellIndex])
			}
			applyTablePaint(&pb, d, tableX, tableWidth, y, rh, true, true, repeated, cellIndex == 0, cellIndex == len(done)-1, t.Style.BorderCollapse, opts)
			contentH := tableCellAllContentHeight(d, opts) + d.b.Style.PaddingTop + d.b.Style.PaddingBottom
			borderTop, _, borderBottom, borderLeft := tableCellBorderInsets(d.b, t.Style.BorderCollapse)
			if t.Style.BorderCollapse {
				borderTop = collapsedBorders[cellIndex].top.Width * 0.5
				borderBottom = collapsedBorders[cellIndex].bottom.Width * 0.5
			}
			contentBoxH := rh - borderTop - borderBottom
			offset := 0.0
			if d.rowSpan <= 1 {
				switch d.b.Style.VAlign {
				case "middle":
					offset = (contentBoxH - contentH) / 2
				case "bottom":
					offset = contentBoxH - contentH
				}
			}
			contentX := tableX + colX[d.col] + borderLeft + d.b.Style.PaddingLeft
			contentY := y + borderTop + d.b.Style.PaddingTop + offset
			pb.Runs = assignTableCellRuns(d, d.lines, contentX, cellW(d.col, d.span)-horizontalInsets(d.b), contentY, opts)
			pb.fragmentRef = newFragmentOccurrence(d.b)
			p.emitBlock(pb)
			blockIndex := p.blockCount() - 1
			appendPageDecorations(p, blockIndex, tableCellBlockDecorations(d, d.lines, 0, contentX, contentY, opts))
			appendNestedTableBlocks(p, d.lines, contentX, contentY, opts, p.block(blockIndex).fragmentRef)
			if d.rowSpan > 1 {
				spanning = append(spanning, &tableSpanningCell{
					page: len(*pages) - 1, block: blockIndex, endRow: currentRow + d.rowSpan,
					height: rh, contentH: contentH, align: d.b.Style.VAlign,
					segments: map[int]int{len(*pages) - 1: blockIndex},
				})
			}
		}
		return y + rh
	}
	for ri, row := range grid {
		ensure(ri)
		col := 0
		var done []tableDoneCell
		rh := 0.0
		for _, rc := range row {
			for col < cols && occ[ri][col] {
				col++
			}
			if col >= cols {
				break
			}
			span := rc.span
			if span < 1 {
				span = 1
			}
			if col+span > cols {
				span = cols - col
			}
			for s := 0; s < span; s++ {
				occ[ri][col+s] = true
			}
			if rs := rc.b.Style.RowSpan; rs > 1 {
				for k := 1; k < rs; k++ {
					ensure(ri + k)
					for s := 0; s < span && col+s < cols; s++ {
						occ[ri+k][col+s] = true
					}
				}
			}
			fs := resolveSize(rc.b, opts)
			font := fontFor(opts, rc.b)
			contentW := cellW(col, span) - rc.b.Style.PaddingLeft - rc.b.Style.PaddingRight
			borderTop, borderRight, borderBottom, borderLeft := tableCellBorderInsets(rc.b, t.Style.BorderCollapse)
			contentW -= borderLeft + borderRight
			if contentW < 0 {
				contentW = 0
			}
			lines, after := tableCellContent(rc.b, contentW, opts)
			group := tableRowGroup(rc.row)
			doneCell := tableDoneCell{
				b: rc.b, col: col, span: span, lines: lines, after: after, fs: fs, font: font,
				table: t, row: rc.row, group: group,
				tableFirst: ri == 0, tableLast: ri == len(grid)-1,
				groupFirst: ri == 0 || tableRowGroup(rows[ri-1]) != group,
				groupLast:  ri == len(grid)-1 || tableRowGroup(rows[ri+1]) != group,
			}
			if t.Style.BorderCollapse {
				doneCell.collapsedBorders = collapsedBorderModel[rc.b]
				doneCell.borderModelReady = true
			}
			if h := tableCellAllContentHeight(doneCell, opts) + rc.b.Style.PaddingTop + rc.b.Style.PaddingBottom + borderTop + borderBottom; h > rh {
				rh = h
			}
			rowSpan := rc.b.Style.RowSpan
			if rowSpan < 1 {
				rowSpan = 1
			}
			doneCell.rowSpan = rowSpan
			done = append(done, doneCell)
			col += span
		}
		if t.Style.BorderCollapse {
			rh = collapsedRowHeight(done, opts)
		}
		isHead := ri < headCount
		if isHead {
			headDone = append(headDone, done)
			headRh = append(headRh, rh)
		}
		currentRow = ri
		repeatHeaderHeight := 0.0
		for _, headerHeight := range headRh {
			repeatHeaderHeight += headerHeight
		}
		repeatHeaders := !isHead && canRepeat && repeatHeaderHeight <= bottom-opts.top()+fragmentEpsilon
		rowBreakBefore := pageBreakValue(rows[ri].Style.BreakBefore)
		rowHasSpan := false
		for _, cell := range done {
			rowHasSpan = rowHasSpan || cell.rowSpan > 1
		}
		rowCanSplit := maxTableCellLines(done) > 0 && rh > bottom-opts.top()-repeatHeaderHeight-vSpacing+fragmentEpsilon
		if rowCanSplit {
			var activeSpans []*tableSpanningCell
			for _, span := range spanning {
				if span.endRow > ri {
					activeSpans = append(activeSpans, span)
				}
			}
			y = fragmentTableRow(done, y, opts.top(), bottom, rowBreakBefore, pages, opts, colW, t.Style.BorderCollapse, headDone, headRh, repeatHeaders, activeSpans, ri, &spanning, !rowHasSpan && len(activeSpans) == 0)
			spanning = finalizeTableSpanningRows(spanning, ri, pages, t.Style.BorderCollapse)
			y += vSpacing
			continue
		}
		rowUnit := fragmentUnit{height: rh}
		rowUnit.breakBefore = rowBreakBefore
		pageHasContent := (*pages)[len(*pages)-1].blockCount() > 0
		rowFragment := placeFragmentUnitForPage(rowUnit, y, opts.top(), bottom, pageHasContent, pages, opts)
		if rowFragment.pageOffset > 0 {
			y = rowFragment.y + vSpacing
			// Repeat headers when a new table page starts.
			if repeatHeaders && repeatHeaderHeight <= bottom-opts.top()+fragmentEpsilon {
				for hi := range headDone {
					if y+headRh[hi] > bottom {
						break
					}
					y = placeCells(headDone[hi], headRh[hi], y, true) + vSpacing
				}
			}
		}
		var activeSpans []*tableSpanningCell
		for _, span := range spanning {
			if span.endRow > ri {
				activeSpans = append(activeSpans, span)
			}
		}
		rowY := y
		y = placeCells(done, rh, rowY, false)
		y += vSpacing
		if len(activeSpans) > 0 {
			pageIndex := len(*pages) - 1
			extendTableSpanFragments(activeSpans, pageIndex, ri, rowY, rh, true, pages)
			spanning = finalizeTableSpanningRows(spanning, ri, pages, t.Style.BorderCollapse)
		}
	}
	return y
}

func tableUsedWidth(table *boxes.Box, grid [][]tableCell, cols int, available float64, opts Options) float64 {
	available = max(0, available-table.Style.MarginLeft-table.Style.MarginRight)
	var width float64
	basis := definiteSize(available)
	if declaredWidth, resolved := resolveDimension(table.Style.Sizing.Width, basis); resolved {
		width = declaredWidth
		if table.Style.Sizing.Width.IsLength() {
			width = max(0, width-table.Style.MarginLeft-table.Style.MarginRight)
		}
	} else {
		minimum, maximum := tableIntrinsicWidths(grid, cols, opts)
		intrinsic := tableIntrinsicColumns(grid, cols, opts)
		for index, constraint := range tableColumnDeclaredConstraints(table, grid, cols) {
			if constraint.IsLength() {
				maximum += max(0, constraint.Value-intrinsic[index].max)
			}
		}
		if !table.Style.BorderCollapse {
			spacing := table.Style.TableBorderSpacingX * float64(cols+1)
			minimum += spacing
			maximum += spacing
		}
		width = max(minimum, min(maximum, available))
	}
	if table.Style.TableLayoutFixed && table.Style.Sizing.Width.Kind != boxes.DimensionAuto {
		minimum := fixedTableColumnMinimumWidth(table, grid, cols)
		if !table.Style.BorderCollapse {
			minimum += table.Style.TableBorderSpacingX * float64(cols+1)
		}
		width = max(width, minimum)
	}
	return clampUsedDimension(width, table.Style.Sizing.MinWidth, table.Style.Sizing.MaxWidth, basis, 0, table.Style.BoxSizing)
}

func finalizeTableSpanningRows(spanning []*tableSpanningCell, rowIndex int, pages *[]Page, collapsed bool) []*tableSpanningCell {
	active := spanning[:0]
	for _, span := range spanning {
		if span.endRow != rowIndex+1 {
			active = append(active, span)
			continue
		}
		if len(span.segments) == 1 {
			placedCell := (*pages)[span.page].block(span.block)
			placedCell.Height = span.height
			alignTableSpanContent(span, placedCell, collapsed)
		}
	}
	return active
}

func alignTableSpanContent(span *tableSpanningCell, placedCell *PlacedBox, collapsed bool) {
	offset := 0.0
	borderTop, _, borderBottom, _ := tableCellBorderInsets(&boxes.Box{Style: boxes.ComputedStyle{BorderTop: placedCell.BorderTop, BorderBottom: placedCell.BorderBottom}}, collapsed)
	switch span.align {
	case "middle":
		offset = (span.height - borderTop - borderBottom - span.contentH) / 2
	case "bottom":
		offset = span.height - borderTop - borderBottom - span.contentH
	}
	for line := range placedCell.Runs {
		for run := range placedCell.Runs[line] {
			placedCell.Runs[line][run].Y += offset
		}
	}
}

// tableRows returns direct table rows and rows inside a row group. Header and
// footer groups retain their browser-defined order around the body rows.
func tableRows(t *boxes.Box) (head, body []*boxes.Box) {
	var foot []*boxes.Box
	var anonymousChildren []*boxes.Box
	flushAnonymousRow := func() {
		if len(anonymousChildren) == 0 {
			return
		}
		body = append(body, anonymousTableRow(t, anonymousChildren))
		anonymousChildren = nil
	}
	for _, c := range t.Children {
		switch {
		case isTableRow(c):
			flushAnonymousRow()
			body = append(body, c)
		case c.Tag == "thead" || c.Style.Display == boxes.DisplayTableHeaderGroup:
			flushAnonymousRow()
			head = append(head, tableRowsInGroup(c)...)
		case c.Tag == "tfoot" || c.Style.Display == boxes.DisplayTableFooterGroup:
			flushAnonymousRow()
			foot = append(foot, tableRowsInGroup(c)...)
		case c.Tag == "tbody" || c.Style.Display == boxes.DisplayTableRowGroup:
			flushAnonymousRow()
			body = append(body, tableRowsInGroup(c)...)
		case isTableColumn(c) || isTableColumnGroup(c) || c.Style.Display == boxes.DisplayTableCaption:
			flushAnonymousRow()
		default:
			anonymousChildren = append(anonymousChildren, c)
		}
	}
	flushAnonymousRow()
	return head, append(body, foot...)
}

func tableRowsInGroup(group *boxes.Box) []*boxes.Box {
	var rows, anonymousChildren []*boxes.Box
	flushAnonymousRow := func() {
		if len(anonymousChildren) == 0 {
			return
		}
		rows = append(rows, anonymousTableRow(group, anonymousChildren))
		anonymousChildren = nil
	}
	for _, child := range group.Children {
		switch {
		case isTableRow(child):
			flushAnonymousRow()
			rows = append(rows, child)
		default:
			anonymousChildren = append(anonymousChildren, child)
		}
	}
	flushAnonymousRow()
	return rows
}

func anonymousTableRow(parent *boxes.Box, children []*boxes.Box) *boxes.Box {
	row := &boxes.Box{Tag: "anonymous-table-row", Style: boxes.ComputedStyle{Display: boxes.DisplayTableRow}, Parent: parent}
	row.Children = anonymousTableCells(row, children)
	return row
}

func anonymousTableCells(parent *boxes.Box, children []*boxes.Box) []*boxes.Box {
	var cells []*boxes.Box
	var anonymousCellChildren []*boxes.Box
	flushAnonymousCell := func() {
		if len(anonymousCellChildren) == 0 {
			return
		}
		cell := &boxes.Box{
			Tag:      "anonymous-table-cell",
			Style:    boxes.ComputedStyle{Display: boxes.DisplayTableCell},
			Parent:   parent,
			Children: anonymousCellChildren,
		}
		for _, child := range anonymousCellChildren {
			cell.Parts = append(cell.Parts, boxes.Part{Child: child})
		}
		cells = append(cells, cell)
		anonymousCellChildren = nil
	}
	for _, child := range children {
		if isTableCell(child) {
			flushAnonymousCell()
			cells = append(cells, child)
			continue
		}
		anonymousCellChildren = append(anonymousCellChildren, child)
	}
	flushAnonymousCell()
	return cells
}

func tableCells(r *boxes.Box) []*boxes.Box {
	return anonymousTableCells(r, r.Children)
}

func tableRowGroup(row *boxes.Box) *boxes.Box {
	if row == nil || row.Parent == nil {
		return nil
	}
	if isTableRowGroup(row.Parent) {
		return row.Parent
	}
	return nil
}

func applyTablePaint(block *PlacedBox, cell tableDoneCell, x, width, y, height float64,
	firstFragment, lastFragment, repeated, firstCell, lastCell, collapsed bool,
	opts Options) {
	groups := append([]OpacityGroup(nil), opts.groupStack...)
	for _, source := range []*boxes.Box{cell.table, cell.group, cell.row} {
		if source == nil || !source.Style.HasOpacity {
			continue
		}
		group, ok := opts.tableOpacityCache[source]
		if !ok {
			if opts.groupSequence == nil {
				continue
			}
			*opts.groupSequence++
			group = OpacityGroup{ID: *opts.groupSequence, Opacity: source.Style.Opacity}
			opts.tableOpacityCache[source] = group
		}
		groups = append(groups, group)
	}
	ensureOpacityGroups(block, groups, opts.groupSequence)

	if firstCell {
		for _, source := range []*boxes.Box{cell.table, cell.group, cell.row} {
			if source == nil || !source.Style.HasBg {
				continue
			}
			block.TablePaintUnderlays = append(block.TablePaintUnderlays, TablePaintLayer{
				Source: source, Bounds: FragmentBounds{X: x, Y: y, Width: width, Height: height}, Edges: TablePaintAllEdges,
			})
		}
	}
	if !lastCell || collapsed {
		return
	}
	addBorderLayer := func(source *boxes.Box, edges TablePaintEdges) {
		if source == nil || !hasTableBorder(source.Style) || edges == 0 {
			return
		}
		block.TablePaintOverlays = append(block.TablePaintOverlays, TablePaintLayer{
			Source: source, Bounds: FragmentBounds{X: x, Y: y, Width: width, Height: height}, Edges: edges,
		})
	}
	tableEdges := TablePaintLeft | TablePaintRight
	if cell.tableFirst && firstFragment {
		tableEdges |= TablePaintTop
	}
	if cell.tableLast && lastFragment {
		tableEdges |= TablePaintBottom
	}
	addBorderLayer(cell.table, tableEdges)
	if cell.group != nil {
		groupEdges := TablePaintLeft | TablePaintRight
		if (cell.groupFirst && firstFragment) || repeated && firstFragment {
			groupEdges |= TablePaintTop
		}
		if (cell.groupLast && lastFragment) || repeated && lastFragment {
			groupEdges |= TablePaintBottom
		}
		addBorderLayer(cell.group, groupEdges)
	}
	rowEdges := TablePaintLeft | TablePaintRight
	if firstFragment {
		rowEdges |= TablePaintTop
	}
	if lastFragment {
		rowEdges |= TablePaintBottom
	}
	addBorderLayer(cell.row, rowEdges)
}

func hasTableBorder(style boxes.ComputedStyle) bool {
	return style.BorderWidth > 0 || style.BorderTop.Width > 0 || style.BorderRight.Width > 0 ||
		style.BorderBottom.Width > 0 || style.BorderLeft.Width > 0
}
