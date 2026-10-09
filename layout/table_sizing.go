package layout

import (
	"github.com/robtrivi/pagecraft-go/boxes"
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

func columnWidths(grid [][]tableCell, cols int, usable float64, opts Options) []float64 {
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
	percentages := firstRowColumnPercentages(grid, cols)
	pctTotal := 0.0
	for _, percentage := range percentages {
		pctTotal += percentage
	}
	if pctTotal > 100 {
		for index := range percentages {
			percentages[index] *= 100 / pctTotal
		}
		pctTotal = 100
	}
	if pctTotal > 0 {
		for index, percentage := range percentages {
			if percentage > 0 {
				widths[index] = max(widths[index], usable*percentage/100)
			}
		}
		remaining = usable - sumTableColumnWidths(widths)
		if remaining <= 0 {
			return widths
		}
	}

	capacities := make([]float64, cols)
	capacityTotal := 0.0
	for index := range widths {
		if percentages[index] > 0 {
			continue
		}
		capacities[index] = max(0, intrinsic[index].max-widths[index])
		capacityTotal += capacities[index]
	}
	if capacityTotal > 0 {
		fraction := min(1, remaining/capacityTotal)
		for index, capacity := range capacities {
			widths[index] += capacity * fraction
		}
		remaining = usable - sumTableColumnWidths(widths)
		if remaining <= 0 {
			return widths
		}
	}

	// The small min-to-max gap must not absorb all spare width. Once auto
	// columns reach max-content, distribute the excess by max-content width.
	weights := make([]float64, cols)
	weightTotal := 0.0
	for index := range widths {
		if percentages[index] == 0 {
			weights[index] = intrinsic[index].max
			weightTotal += weights[index]
		}
	}
	if weightTotal == 0 {
		for index := range weights {
			if percentages[index] == 0 {
				weights[index] = 1
				weightTotal++
			}
		}
	}
	if weightTotal == 0 {
		for index := range weights {
			weights[index] = 1
		}
		weightTotal = float64(cols)
	}
	for index, weight := range weights {
		widths[index] += remaining * weight / weightTotal
	}
	return widths
}

func firstRowColumnPercentages(grid [][]tableCell, cols int) []float64 {
	percentages := make([]float64, cols)
	if len(grid) == 0 {
		return percentages
	}
	column := 0
	for _, cell := range grid[0] {
		span := min(max(1, cell.span), cols-column)
		if span <= 0 {
			break
		}
		if cell.b.Style.Sizing.Width.IsPercentage() {
			for offset := 0; offset < span; offset++ {
				percentages[column+offset] = cell.b.Style.Sizing.Width.Value / float64(span)
			}
		}
		column += span
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
