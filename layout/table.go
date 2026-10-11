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
}

type tableSpanningCell struct {
	page, block, endRow int
	height, contentH    float64
	align               string
	segments            map[int]int
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

// flowTable lays out t's rows from and and returns the resulting cursor.
func flowTable(t *boxes.Box, y float64, pages *[]Page, opts Options, usable, bottom float64) float64 {
	head, body := tableRows(t)
	rows := append(head, body...)
	headCount := len(head)
	grid, cols := tableGridFromRows(rows)
	if cols == 0 {
		return y
	}
	tableWidth := tableUsedWidth(t, grid, cols, usable, opts)
	colW := columnWidths(grid, cols, tableWidth, opts)
	tableX := opts.left() + t.Style.MarginLeft
	colX := make([]float64, cols+1)
	for i := 0; i < cols; i++ {
		colX[i+1] = colX[i] + colW[i]
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
	placeCells := func(done []tableDoneCell, rh, y float64, repeated bool) float64 {
		p := &(*pages)[len(*pages)-1]
		for cellIndex, d := range done {
			pb := placed(d.b, d.b.Text,
				nil, tableX+colX[d.col], y, cellW(d.col, d.span), rh, d.fs, d.font)
			applyTablePaint(&pb, d, tableX, tableWidth, y, rh, true, true, repeated, cellIndex == 0, cellIndex == len(done)-1, t.Style.BorderCollapse, opts)
			contentH := tableCellAllContentHeight(d, opts) + d.b.Style.PaddingTop + d.b.Style.PaddingBottom
			borderTop, _, borderBottom, borderLeft := tableCellBorderInsets(d.b, t.Style.BorderCollapse)
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
		rowCanSplit := maxTableCellLines(done) > 0 && rh > bottom-opts.top()-repeatHeaderHeight+fragmentEpsilon
		if rowCanSplit {
			var activeSpans []*tableSpanningCell
			for _, span := range spanning {
				if span.endRow > ri {
					activeSpans = append(activeSpans, span)
				}
			}
			y = fragmentTableRow(done, y, opts.top(), bottom, rowBreakBefore, pages, opts, colW, t.Style.BorderCollapse, headDone, headRh, repeatHeaders, activeSpans, ri, &spanning, !rowHasSpan && len(activeSpans) == 0)
			spanning = finalizeTableSpanningRows(spanning, ri, pages, t.Style.BorderCollapse)
			continue
		}
		rowUnit := fragmentUnit{height: rh}
		rowUnit.breakBefore = rowBreakBefore
		pageHasContent := (*pages)[len(*pages)-1].blockCount() > 0
		rowFragment := placeFragmentUnitForPage(rowUnit, y, opts.top(), bottom, pageHasContent, pages, opts)
		if rowFragment.pageOffset > 0 {
			y = rowFragment.y
			// Repeat headers when a new table page starts.
			if repeatHeaders && repeatHeaderHeight <= bottom-opts.top()+fragmentEpsilon {
				for hi := range headDone {
					if y+headRh[hi] > bottom {
						break
					}
					y = placeCells(headDone[hi], headRh[hi], y, true)
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
		y = placeCells(done, rh, y, false)
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
		width = max(minimum, min(maximum, available))
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

// tableRows returns direct <tr> elements and rows inside thead/tbody/tfoot
// (the HTML parser inserts tbody even when it is absent from the source).
func tableRows(t *boxes.Box) (head, body []*boxes.Box) {
	for _, c := range t.Children {
		switch c.Tag {
		case "tr":
			body = append(body, c)
		case "thead":
			for _, r := range c.Children {
				if r.Tag == "tr" {
					head = append(head, r)
				}
			}
		case "tbody", "tfoot":
			for _, r := range c.Children {
				if r.Tag == "tr" {
					body = append(body, r)
				}
			}
		}
	}
	return head, body
}

func tableCells(r *boxes.Box) []*boxes.Box {
	var cells []*boxes.Box
	for _, c := range r.Children {
		if c.Tag == "td" || c.Tag == "th" {
			cells = append(cells, c)
		}
	}
	return cells
}

func tableRowGroup(row *boxes.Box) *boxes.Box {
	if row == nil || row.Parent == nil {
		return nil
	}
	switch row.Parent.Tag {
	case "thead", "tbody", "tfoot":
		return row.Parent
	default:
		return nil
	}
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
