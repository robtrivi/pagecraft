package layout

import (
	"math"

	"github.com/robtrivi/pagecraft-go/boxes"
)

func gridItemTrackContribution(item *boxes.Box, contentHeight float64, contentIncludesInsets bool, opts Options) float64 {
	contentBoxHeight := contentHeight
	if contentIncludesInsets {
		contentBoxHeight = math.Max(0, contentHeight-verticalInsets(item))
	}
	return math.Max(contentHeight, usedHeight(item, contentBoxHeight, opts))
}

func gridRowIsAuto(g *boxes.Box, row int) bool {
	return !gridRowIsFixedZero(g, row) && (row >= len(g.Style.GridRowHeights) || g.Style.GridRowHeights[row] == 0) && gridRowHeight(g, row) == 0
}

func gridFixedAreaHeight(g *boxes.Box, firstRow, rowSpan int, gap float64) (float64, bool) {
	height := gap * float64(rowSpan-1)
	for row := firstRow; row < firstRow+rowSpan; row++ {
		rowHeight := 0.0
		if row < len(g.Style.GridRowHeights) {
			rowHeight = g.Style.GridRowHeights[row]
		} else {
			rowHeight = gridImplicitRowHeight(g, row)
		}
		if rowHeight < 0 || (rowHeight == 0 && !gridRowIsFixedZero(g, row)) {
			return 0, false
		}
		height += rowHeight
	}
	return height, true
}

func gridRasterDimensions(item *boxes.Box, columnWidth float64, g *boxes.Box, row, rowSpan int, rowGap float64) (float64, float64) {
	maxWidth, maxHeight := 0.0, 0.0
	if item.Style.ImageWidthPct > 0 {
		maxWidth = columnWidth
	}
	if item.Style.ImageHeightPct > 0 {
		if height, definite := gridFixedAreaHeight(g, row, rowSpan, rowGap); definite {
			maxHeight = height
		}
	}
	return rasterDimensions(item, maxWidth, maxHeight)
}

func gridRowIsFixedZero(g *boxes.Box, row int) bool {
	if row < len(g.Style.GridRowHeights) {
		return row < len(g.Style.GridRowFixedZeroTracks) && g.Style.GridRowFixedZeroTracks[row]
	}
	if len(g.Style.GridAutoRowFixedZeroTracks) == 0 {
		return false
	}
	index := (row - len(g.Style.GridRowHeights)) % len(g.Style.GridAutoRowFixedZeroTracks)
	return g.Style.GridAutoRowFixedZeroTracks[index]
}

func gridRowHeight(g *boxes.Box, row int) float64 {
	if row < len(g.Style.GridRowHeights) {
		return g.Style.GridRowHeights[row]
	}
	return gridImplicitRowHeight(g, row)
}

func gridImplicitRowHeight(g *boxes.Box, row int) float64 {
	if len(g.Style.GridAutoRowHeights) == 0 {
		return 0
	}
	index := (row - len(g.Style.GridRowHeights)) % len(g.Style.GridAutoRowHeights)
	return g.Style.GridAutoRowHeights[index]
}

func namedGridArea(areas [][]string, name string) (row, column, rowSpan, columnSpan int, ok bool) {
	if name == "" {
		return 0, 0, 0, 0, false
	}
	top, bottom, left, right := len(areas), -1, -1, -1
	for r, cells := range areas {
		for c, cell := range cells {
			if cell != name {
				continue
			}
			if r < top {
				top = r
			}
			if r > bottom {
				bottom = r
			}
			if left < 0 || c < left {
				left = c
			}
			if c > right {
				right = c
			}
		}
	}
	if bottom < top || right < left {
		return 0, 0, 0, 0, false
	}
	return top, left, bottom - top + 1, right - left + 1, true
}

func resolveGridLine(line, tracks int) (int, bool) {
	if line > 0 {
		index := line - 1
		return index, index <= tracks
	}
	if line < 0 {
		index := tracks + line + 1
		return index, index >= 0 && index <= tracks
	}
	return 0, false
}
