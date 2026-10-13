package layout

import (
	"sort"

	"github.com/robtrivi/pagecraft/boxes"
)

type tableColumnIntrinsic struct {
	min float64
	max float64
}

func tableIntrinsicWidths(grid [][]tableCell, cols int, opts Options) (minimum, maximum float64) {
	for _, column := range tableIntrinsicColumns(grid, cols, opts) {
		minimum += column.min
		maximum += column.max
	}
	return minimum, maximum
}

func tableIntrinsicColumns(grid [][]tableCell, cols int, opts Options) []tableColumnIntrinsic {
	columns := make([]tableColumnIntrinsic, cols)
	occupied := make([][]bool, len(grid))
	for rowIndex := range occupied {
		occupied[rowIndex] = make([]bool, cols)
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
			span := min(cell.span, cols-column)
			cellMin, cellMax := tableCellIntrinsicWidths(cell.b, opts)
			distributeTableSpanConstraint(columns, column, span, cellMin, false)
			distributeTableSpanConstraint(columns, column, span, cellMax, true)
			endRow := min(len(grid), rowIndex+max(1, cell.b.Style.RowSpan))
			for coveredRow := rowIndex; coveredRow < endRow; coveredRow++ {
				for coveredColumn := column; coveredColumn < column+span; coveredColumn++ {
					occupied[coveredRow][coveredColumn] = true
				}
			}
			column += span
		}
	}
	for index := range columns {
		if columns[index].max < columns[index].min {
			columns[index].max = columns[index].min
		}
	}
	return columns
}

func distributeTableSpanConstraint(columns []tableColumnIntrinsic, start, span int, required float64, maximum bool) {
	current := 0.0
	for index := start; index < start+span; index++ {
		if maximum {
			current += columns[index].max
		} else {
			current += columns[index].min
		}
	}
	if required <= current || span <= 0 {
		return
	}
	extra := (required - current) / float64(span)
	for index := start; index < start+span; index++ {
		if maximum {
			columns[index].max += extra
		} else {
			columns[index].min += extra
		}
	}
}

func tableCellIntrinsicWidths(cell *boxes.Box, opts Options) (minimum, maximum float64) {
	contribution := intrinsicContribution(cell, opts)
	return contribution.borderMin, contribution.borderMax
}

// fixedTableColumnWidths follows Blink's fixed algorithm: column declarations
// establish tracks first, only cells in the first row may establish remaining
// tracks, and content in later rows never participates in column sizing.
func fixedTableColumnDeclarations(table *boxes.Box, grid [][]tableCell, cols int) []boxes.CSSDimension {
	declared := make([]boxes.CSSDimension, cols)
	assigned := make([]bool, cols)
	apply := func(start, span int, dimension boxes.CSSDimension, divide bool) {
		if dimension.Kind == boxes.DimensionAuto || span <= 0 || start >= cols {
			return
		}
		end := min(cols, start+span)
		share := dimension
		if divide {
			share.Value /= float64(span)
		}
		for index := start; index < end; index++ {
			if !assigned[index] {
				declared[index], assigned[index] = share, true
			}
		}
	}

	column := 0
	for _, child := range table.Children {
		if !isTableColumn(child) && !isTableColumnGroup(child) {
			continue
		}
		if isTableColumnGroup(child) && len(child.Children) > 0 {
			for _, col := range child.Children {
				if !isTableColumn(col) {
					continue
				}
				span := max(1, col.Style.TableColumnSpan)
				apply(column, span, col.Style.Sizing.Width, false)
				column += span
			}
			continue
		}
		span := max(1, child.Style.TableColumnSpan)
		apply(column, span, child.Style.Sizing.Width, false)
		column += span
	}

	if len(grid) > 0 {
		column = 0
		for _, cell := range grid[0] {
			if column >= cols {
				break
			}
			span := min(max(1, cell.span), cols-column)
			if span <= 0 {
				break
			}
			apply(column, span, cell.b.Style.Sizing.Width, true)
			column += span
		}
	}
	return declared
}

func fixedTableColumnMinimumWidth(table *boxes.Box, grid [][]tableCell, cols int) float64 {
	minimum := 0.0
	for _, dimension := range fixedTableColumnDeclarations(table, grid, cols) {
		if dimension.IsLength() {
			minimum += dimension.Value
		}
	}
	return minimum
}

func fixedTableColumnWidths(table *boxes.Box, grid [][]tableCell, cols int, width float64) []float64 {
	if cols <= 0 {
		return nil
	}
	declared := fixedTableColumnDeclarations(table, grid, cols)

	widths := make([]float64, cols)
	autoColumns := make([]int, 0, cols)
	totalFixed, totalPercent, totalPercentWidth := 0.0, 0.0, 0.0
	for index, dimension := range declared {
		switch dimension.Kind {
		case boxes.DimensionLength:
			widths[index] = dimension.Value
			totalFixed += widths[index]
		case boxes.DimensionPercentage:
			widths[index] = width * dimension.Value / 100
			totalPercent += dimension.Value
			totalPercentWidth += widths[index]
		default:
			autoColumns = append(autoColumns, index)
		}
	}
	totalWidth := totalFixed + totalPercentWidth
	if len(autoColumns) > 0 && totalWidth <= width {
		remaining := width - totalWidth
		share := remaining / float64(len(autoColumns))
		for _, index := range autoColumns {
			widths[index] = share
		}
		return widths
	}
	if totalWidth < width && totalFixed > 0 {
		for index, dimension := range declared {
			if dimension.IsLength() {
				widths[index] *= width / totalWidth
			}
		}
		totalFixed = 0
		for index, dimension := range declared {
			if dimension.IsLength() {
				totalFixed += widths[index]
			}
		}
	}
	if totalPercent > 0 {
		remainingForPercent := max(0, width-totalFixed)
		totalPercentWidth = 0
		for index, dimension := range declared {
			if dimension.IsPercentage() {
				widths[index] = remainingForPercent * dimension.Value / totalPercent
				totalPercentWidth += widths[index]
			}
		}
	}
	totalWidth = totalFixed + totalPercentWidth
	if totalWidth < width {
		remaining := width - totalWidth
		for index := len(widths) - 1; index >= 0 && remaining > 0; index-- {
			share := remaining / float64(index+1)
			widths[index] += share
			remaining -= share
		}
	}
	return widths
}

func columnWidthsForTable(table *boxes.Box, grid [][]tableCell, cols int, usable float64, opts Options) []float64 {
	if cols <= 0 {
		return nil
	}
	intrinsic := tableIntrinsicColumns(grid, cols, opts)
	widths := make([]float64, cols)
	for index := range widths {
		widths[index] = intrinsic[index].min
	}
	remaining := usable - sumTableColumnWidths(widths)
	if remaining <= 0 {
		return widths
	}
	percentages := tableColumnPercentages(grid, cols, intrinsic)
	declared := tableColumnDeclaredConstraints(table, grid, cols)
	fixed := make([]float64, cols)
	for index, constraint := range declared {
		if constraint.IsPercentage() {
			percentages[index] = max(percentages[index], constraint.Value)
		} else if constraint.IsLength() && percentages[index] == 0 {
			fixed[index] = constraint.Value
			widths[index] = max(widths[index], constraint.Value)
		}
	}
	pctTotal := 0.0
	for _, percentage := range percentages {
		pctTotal += percentage
	}
	// Blink starts every track at min-content, then applies percentage
	// constraints relative to the table's content width.
	if pctTotal > 0 && remaining > 0 {
		for index, percentage := range percentages {
			if percentage > 0 {
				widths[index] = max(widths[index], usable*percentage/100)
			}
		}
		if pctTotal > 100 {
			// Blink trims percentage over-allocation from the last percentage
			// columns first, without shrinking a column below its min-content width.
			excess := usable * (pctTotal - 100) / 100
			for index := len(widths) - 1; index >= 0 && excess > 0; index-- {
				if percentages[index] == 0 {
					continue
				}
				reduction := min(excess, widths[index]-intrinsic[index].min)
				widths[index] -= reduction
				excess -= reduction
			}
		}
	}

	remaining = usable - sumTableColumnWidths(widths)
	if remaining < 0 {
		// The content may force the percentage targets beyond the table width.
		// Match Blink's shrink order: auto tracks first, then percentage tracks,
		// never going below the min-content width of a track.
		deficit := -remaining
		deficit = shrinkTableColumns(widths, intrinsic, percentages, false, deficit)
		shrinkTableColumns(widths, intrinsic, percentages, true, deficit)
		return widths
	}

	hasAuto, hasFixed := false, false
	for index, percentage := range percentages {
		hasAuto = hasAuto || percentage == 0 && fixed[index] == 0
		hasFixed = hasFixed || fixed[index] > 0
	}
	if hasAuto && remaining > 0 {
		// The automatic algorithm assigns remaining width to auto columns in
		// proportion to their max-content widths. This allocation is not capped
		// at max-content; max-content supplies the distribution weights.
		distributeTableAutoWidths(widths, intrinsic, percentages, fixed, remaining)
	} else if hasFixed && remaining > 0 {
		weight := 0.0
		for index := range widths {
			if fixed[index] > 0 {
				weight += intrinsic[index].max
			}
		}
		for index := range widths {
			if fixed[index] > 0 {
				share := remaining / float64(countTableFixed(fixed))
				if weight > 0 {
					share = remaining * intrinsic[index].max / weight
				}
				widths[index] += share
			}
		}
	} else if remaining > 0 {
		if pctTotal > 0 && pctTotal < 100 {
			// If there are no auto columns, Blink gives leftover width to percentage
			// columns, weighted by their declared percentages.
			for index, percentage := range percentages {
				if percentage > 0 {
					widths[index] += remaining * percentage / pctTotal
				}
			}
		}
	}
	return widths
}

func countTableFixed(fixed []float64) int {
	count := 0
	for _, width := range fixed {
		if width > 0 {
			count++
		}
	}
	return count
}

func tableColumnDeclaredConstraints(table *boxes.Box, grid [][]tableCell, cols int) []boxes.CSSDimension {
	constraints := make([]boxes.CSSDimension, cols)
	column := 0
	for _, child := range table.Children {
		if !isTableColumn(child) && !isTableColumnGroup(child) {
			continue
		}
		assign := func(span int, dimension boxes.CSSDimension) {
			for end := min(cols, column+span); column < end; column++ {
				constraints[column] = dimension
			}
		}
		if isTableColumnGroup(child) && len(child.Children) > 0 {
			for _, col := range child.Children {
				if !isTableColumn(col) {
					continue
				}
				dimension := col.Style.Sizing.Width
				if dimension.Kind == boxes.DimensionAuto {
					dimension = child.Style.Sizing.Width
				}
				assign(max(1, col.Style.TableColumnSpan), dimension)
			}
			continue
		}
		assign(max(1, child.Style.TableColumnSpan), child.Style.Sizing.Width)
	}
	occupied := make([][]bool, len(grid))
	for row := range occupied {
		occupied[row] = make([]bool, cols)
	}
	for rowIndex, row := range grid {
		column = 0
		for _, cell := range row {
			for column < cols && occupied[rowIndex][column] {
				column++
			}
			if column >= cols {
				break
			}
			span := min(max(1, cell.span), cols-column)
			dimension := cell.b.Style.Sizing.Width
			if dimension.IsLength() {
				share := dimension
				if span > 1 {
					share.Value /= float64(span)
				}
				for index := column; index < column+span; index++ {
					current := constraints[index]
					if current.Kind == boxes.DimensionAuto || current.Kind == share.Kind && share.Value > current.Value {
						constraints[index] = share
					}
				}
			}
			endRow := min(len(grid), rowIndex+max(1, cell.b.Style.RowSpan))
			for coveredRow := rowIndex; coveredRow < endRow; coveredRow++ {
				for coveredColumn := column; coveredColumn < column+span; coveredColumn++ {
					occupied[coveredRow][coveredColumn] = true
				}
			}
			column += span
		}
	}
	return constraints
}

func distributeTableAutoWidths(widths []float64, intrinsic []tableColumnIntrinsic, percentages, fixed []float64, remaining float64) {
	active := make([]bool, len(widths))
	maxWeight := 0.0
	for index, percentage := range percentages {
		if percentage == 0 && fixed[index] == 0 {
			maxWeight += intrinsic[index].max
		}
	}
	pool, totalWeight := remaining, maxWeight
	for index, percentage := range percentages {
		if percentage > 0 || fixed[index] > 0 {
			continue
		}
		if maxWeight > 0 && intrinsic[index].max == 0 {
			continue // Blink allocates empty auto tracks after non-empty ones.
		}
		active[index] = true
		pool += widths[index]
		if maxWeight == 0 {
			totalWeight++
		}
	}
	if totalWeight == 0 {
		return
	}
	for totalWeight > 0 {
		clamped := false
		for index, isActive := range active {
			if !isActive {
				continue
			}
			weight := intrinsic[index].max
			if maxWeight == 0 {
				weight = 1
			}
			share := pool * weight / totalWeight
			if share < widths[index] {
				pool -= widths[index]
				totalWeight -= weight
				active[index] = false
				clamped = true
			}
		}
		if clamped {
			continue
		}
		for index, isActive := range active {
			if !isActive {
				continue
			}
			weight := intrinsic[index].max
			if maxWeight == 0 {
				weight = 1
			}
			widths[index] = pool * weight / totalWeight
		}
		return
	}
}

func shrinkTableColumns(widths []float64, intrinsic []tableColumnIntrinsic, percentages []float64, percentageColumns bool, deficit float64) float64 {
	beyondMin := 0.0
	for index, width := range widths {
		if (percentages[index] > 0) == percentageColumns {
			beyondMin += max(0, width-intrinsic[index].min)
		}
	}
	if beyondMin == 0 {
		return deficit
	}
	for index := len(widths) - 1; index >= 0 && deficit > 0; index-- {
		if (percentages[index] > 0) != percentageColumns {
			continue
		}
		capacity := max(0, widths[index]-intrinsic[index].min)
		reduction := min(capacity, deficit*capacity/beyondMin)
		widths[index] -= reduction
		deficit -= reduction
		beyondMin -= capacity
	}
	return deficit
}

func tableColumnPercentages(grid [][]tableCell, cols int, intrinsic []tableColumnIntrinsic) []float64 {
	percentages := make([]float64, cols)
	if len(grid) == 0 || cols == 0 {
		return percentages
	}
	type spanConstraint struct {
		start, span int
		percentage  float64
	}
	var spans []spanConstraint
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
			if cell.b.Style.Sizing.Width.IsPercentage() {
				if span == 1 {
					percentages[column] = max(percentages[column], cell.b.Style.Sizing.Width.Value)
				} else {
					spans = append(spans, spanConstraint{column, span, cell.b.Style.Sizing.Width.Value})
				}
			}
			endRow := min(len(grid), rowIndex+max(1, cell.b.Style.RowSpan))
			for coveredRow := rowIndex; coveredRow < endRow; coveredRow++ {
				for coveredColumn := column; coveredColumn < column+span; coveredColumn++ {
					occupied[coveredRow][coveredColumn] = true
				}
			}
			column += span
		}
	}
	sort.SliceStable(spans, func(i, j int) bool { return spans[i].span < spans[j].span })
	for _, constraint := range spans {
		total, allPercent := 0.0, true
		for index := constraint.start; index < constraint.start+constraint.span; index++ {
			total += percentages[index]
			allPercent = allPercent && percentages[index] > 0
		}
		if allPercent || total >= constraint.percentage {
			continue
		}
		missing := constraint.percentage - total
		weightTotal := 0.0
		for index := constraint.start; index < constraint.start+constraint.span; index++ {
			if percentages[index] == 0 {
				weightTotal += intrinsic[index].max
			}
		}
		if weightTotal == 0 {
			count := 0
			for index := constraint.start; index < constraint.start+constraint.span; index++ {
				if percentages[index] == 0 {
					count++
				}
			}
			for index := constraint.start; index < constraint.start+constraint.span; index++ {
				if percentages[index] == 0 {
					percentages[index] = missing / float64(count)
				}
			}
			continue
		}
		for index := constraint.start; index < constraint.start+constraint.span && weightTotal > 0; index++ {
			if percentages[index] > 0 {
				continue
			}
			share := missing * intrinsic[index].max / weightTotal
			percentages[index] = share
			missing -= share
			weightTotal -= intrinsic[index].max
		}
	}
	return percentages
}

func sumTableColumnWidths(widths []float64) float64 {
	total := 0.0
	for _, width := range widths {
		total += width
	}
	return total
}
