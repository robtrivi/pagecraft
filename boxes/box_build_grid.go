package boxes

import (
	"strconv"
	"strings"

	"github.com/robtrivi/pagecraft-go/css"
)

func applyGridProperties(b *Box, decls css.Declarations) {
	if gtc, ok := decls["grid-template-columns"]; ok {
		b.Style.GridCols = countTracks(gtc)
		if tracks, valid := parseGridTracks(gtc); valid {
			b.Style.GridTracks = tracks
			b.Style.GridCols = len(tracks)
		}
	}
	if span, ok := decls["column-span"]; ok && span == "all" {
		b.Style.ColumnSpanAll = true
	}
	if gtr, ok := decls["grid-template-rows"]; ok {
		if heights, fixedZeroTracks, valid := parseGridTemplateRows(gtr); valid {
			b.Style.GridRowHeights = heights
			b.Style.GridRowFixedZeroTracks = fixedZeroTracks
		}
	}
	if gar, ok := decls["grid-auto-rows"]; ok {
		if heights, fixedZeroTracks, valid := parseGridAutoRows(gar); valid {
			b.Style.GridAutoRowHeights = heights
			b.Style.GridAutoRowFixedZeroTracks = fixedZeroTracks
		}
	}
	if gac, ok := decls["grid-auto-columns"]; ok {
		if widths, percentages, percentageTracks, fixedZeroTracks, fractions, minimums, minimumPercentages, automaticMinimums, valid := parseGridAutoColumns(gac); valid {
			b.Style.GridAutoColumnWidths = widths
			b.Style.GridAutoColumnPercentages = percentages
			b.Style.GridAutoColumnPercentageTracks = percentageTracks
			b.Style.GridAutoColumnFixedZeroTracks = fixedZeroTracks
			b.Style.GridAutoColumnFractions = fractions
			b.Style.GridAutoColumnMinimums = minimums
			b.Style.GridAutoColumnMinimumPercentages = minimumPercentages
			b.Style.GridAutoColumnAutomaticMinimums = automaticMinimums
		}
	}
	if gta, ok := decls["grid-template-areas"]; ok {
		if areas, valid := parseGridTemplateAreas(gta); valid && (b.Style.GridCols == 0 || b.Style.GridCols == len(areas[0])) {
			b.Style.GridAreas = areas
			if b.Style.GridCols == 0 {
				b.Style.GridCols = len(areas[0])
			}
		}
	}
	if flow, ok := decls["grid-auto-flow"]; ok {
		parts := strings.Fields(flow)
		if flow == "dense" || (len(parts) == 2 && parts[0] == "row" && parts[1] == "dense") {
			b.Style.GridAutoFlowDense = true
		}
		if flow == "column" || (len(parts) == 2 && parts[0] == "column" && parts[1] == "dense") {
			b.Style.GridAutoFlowColumn = true
			b.Style.GridAutoFlowDense = len(parts) == 2 && parts[1] == "dense"
		}
	}
}

func applyGridItemStyle(b *Box, decls css.Declarations) {
	if gridColumn, ok := decls["grid-column"]; ok {
		parts := strings.Fields(gridColumn)
		if len(parts) == 2 && parts[0] == "span" {
			if n, err := strconv.Atoi(parts[1]); err == nil && n > 1 && n <= maxGridTracks {
				b.Style.ColSpan = n
			}
		} else if len(parts) == 3 && parts[1] == "/" {
			start, startErr := strconv.Atoi(parts[0])
			end, endErr := strconv.Atoi(parts[2])
			if startErr == nil && endErr == nil && validGridLineNumber(start) && validGridLineNumber(end) {
				b.Style.GridColumnLine = start
				b.Style.GridColumnEndLine = end
				if start > 0 && end > start {
					b.Style.ColSpan = end - start
				}
			}
		} else if len(parts) == 1 {
			if n, err := strconv.Atoi(parts[0]); err == nil && validGridLineNumber(n) {
				b.Style.GridColumnLine = n
			}
		}
	}
	if gridRow, ok := decls["grid-row"]; ok {
		parts := strings.Fields(gridRow)
		if len(parts) == 2 && parts[0] == "span" {
			if n, err := strconv.Atoi(parts[1]); err == nil && n > 1 && n <= maxGridTracks {
				b.Style.GridRowSpan = n
			}
		} else if len(parts) == 3 && parts[1] == "/" {
			start, startErr := strconv.Atoi(parts[0])
			end, endErr := strconv.Atoi(parts[2])
			if startErr == nil && endErr == nil && validGridLineNumber(start) && validGridLineNumber(end) {
				b.Style.GridRowLine = start
				b.Style.GridRowEndLine = end
				if start > 0 && end > start {
					b.Style.GridRowSpan = end - start
				}
			}
		}
	}
	if area, ok := decls["grid-area"]; ok && isGridAreaName(area) {
		b.Style.GridAreaName = area
	}
}
