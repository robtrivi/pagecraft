package layout

import "github.com/robtrivi/pagecraft/boxes"

func gridAutoColumnFraction(g *boxes.Box, index int) float64 {
	if len(g.Style.GridAutoColumnFractions) == 0 {
		return 0
	}
	return g.Style.GridAutoColumnFractions[index%len(g.Style.GridAutoColumnFractions)]
}

func gridAutoColumnMinimum(g *boxes.Box, index int, available float64) float64 {
	minimum := 0.0
	if len(g.Style.GridAutoColumnMinimums) > 0 {
		minimum = g.Style.GridAutoColumnMinimums[index%len(g.Style.GridAutoColumnMinimums)]
	}
	if len(g.Style.GridAutoColumnMinimumPercentages) > 0 {
		minimum += available * g.Style.GridAutoColumnMinimumPercentages[index%len(g.Style.GridAutoColumnMinimumPercentages)] / 100
	}
	return minimum
}

func gridResolveImplicitFractionColumns(g *boxes.Box, rows [][]gridItem, widths []float64, available float64, explicitColumns int, gap float64, opts Options, explicitAutomaticMinimums []float64) {
	implicitFactor := 0.0
	activeImplicit := make([]bool, len(widths))
	for index := range widths {
		factor := gridAutoColumnFraction(g, index)
		if factor > 0 {
			implicitFactor += factor
			activeImplicit[index] = true
		}
	}
	if implicitFactor == 0 {
		return
	}
	activeExplicit := make([]bool, explicitColumns)
	explicitFactor := 0.0
	used := gap * float64(explicitColumns+len(widths)-1)
	for index := 0; index < explicitColumns && index < len(g.Style.GridTracks); index++ {
		track := g.Style.GridTracks[index]
		if track.Fraction > 0 {
			explicitFactor += track.Fraction
			activeExplicit[index] = true
		} else {
			used += track.Fixed + available*track.Percent/100
		}
	}
	for _, width := range widths {
		used += width
	}
	free := available - used
	if free <= 0 {
		for index, active := range activeImplicit {
			if active {
				widths[index] = gridAutoColumnResolvedMinimum(g, rows, index, explicitColumns, available, opts)
			}
		}
		return
	}
	for {
		factorSum := implicitFactor + explicitFactor
		if factorSum == 0 {
			return
		}
		unit := free / factorSum
		frozeTrack := false
		for index, active := range activeExplicit {
			if !active || index >= len(g.Style.GridTracks) {
				continue
			}
			minimum := g.Style.GridTracks[index].Min + available*g.Style.GridTracks[index].MinPercent/100
			if index < len(explicitAutomaticMinimums) && explicitAutomaticMinimums[index] > minimum {
				minimum = explicitAutomaticMinimums[index]
			}
			if unit*g.Style.GridTracks[index].Fraction < minimum {
				free -= minimum
				explicitFactor -= g.Style.GridTracks[index].Fraction
				activeExplicit[index] = false
				frozeTrack = true
			}
		}
		for index, active := range activeImplicit {
			if !active {
				continue
			}
			minimum := gridAutoColumnResolvedMinimum(g, rows, index, explicitColumns, available, opts)
			if unit*gridAutoColumnFraction(g, index) < minimum {
				widths[index] = minimum
				free -= minimum
				implicitFactor -= gridAutoColumnFraction(g, index)
				activeImplicit[index] = false
				frozeTrack = true
			}
		}
		if !frozeTrack {
			unit = free / factorSum
			for index, active := range activeImplicit {
				if active {
					widths[index] = gridAutoColumnFraction(g, index) * unit
				}
			}
			return
		}
		if free <= 0 {
			for index, active := range activeImplicit {
				if active {
					widths[index] = gridAutoColumnResolvedMinimum(g, rows, index, explicitColumns, available, opts)
				}
			}
			return
		}
	}
}

func gridAutoColumnResolvedMinimum(g *boxes.Box, rows [][]gridItem, index, explicitColumns int, available float64, opts Options) float64 {
	minimum := gridAutoColumnMinimum(g, index, available)
	if gridAutoColumnHasAutomaticMinimum(g, index) {
		if contribution := gridAutoColumnMinContent(rows, explicitColumns+index, opts); contribution > minimum {
			minimum = contribution
		}
	}
	return minimum
}

func gridAutoColumnHasAutomaticMinimum(g *boxes.Box, index int) bool {
	return len(g.Style.GridAutoColumnAutomaticMinimums) > 0 && g.Style.GridAutoColumnAutomaticMinimums[index%len(g.Style.GridAutoColumnAutomaticMinimums)]
}

func gridAutoColumnMinContent(rows [][]gridItem, column int, opts Options) float64 {
	minimum := 0.0
	for _, row := range rows {
		for _, item := range row {
			if item.column != column || item.columnSpan != 1 {
				continue
			}
			width := intrinsicContribution(item.box, opts).borderMin
			if width > minimum {
				minimum = width
			}
		}
	}
	return minimum
}

func gridExplicitAutomaticMinimums(g *boxes.Box, rows [][]gridItem, columns int, opts Options) []float64 {
	minimums := make([]float64, columns)
	for _, row := range rows {
		for _, item := range row {
			column := item.column
			if item.columnSpan != 1 || column < 0 || column >= columns || column >= len(g.Style.GridTracks) || !g.Style.GridTracks[column].MinAuto {
				continue
			}
			minimum := intrinsicContribution(item.box, opts).borderMin
			if minimum > minimums[column] {
				minimums[column] = minimum
			}
		}
	}
	return minimums
}

func gridAutoColumnFixedZero(g *boxes.Box, index int) bool {
	if len(g.Style.GridAutoColumnFixedZeroTracks) == 0 {
		return false
	}
	return g.Style.GridAutoColumnFixedZeroTracks[index%len(g.Style.GridAutoColumnFixedZeroTracks)]
}

func gridAutoColumnPercentage(g *boxes.Box, index int) (float64, bool) {
	return gridAutoColumnPercentageValues(g.Style.GridAutoColumnPercentages, g.Style.GridAutoColumnPercentageTracks, index)
}

func gridAutoColumnPercentageValues(percentages []float64, percentageTracks []bool, index int) (float64, bool) {
	if len(percentages) == 0 {
		return 0, false
	}
	patternIndex := index % len(percentages)
	isPercentage := percentages[patternIndex] != 0
	if len(percentageTracks) > patternIndex {
		isPercentage = percentageTracks[patternIndex]
	}
	return percentages[patternIndex], isPercentage
}

func gridStretchImplicitAutoColumns(widths, pattern, percentages []float64, percentageTracks, fixedZeroTracks []bool, fractions []float64, explicitWidths []float64, gap, available float64) {
	autoTracks := make([]int, 0, len(widths))
	used := gap * float64(len(explicitWidths)+len(widths)-1)
	for _, width := range explicitWidths {
		used += width
	}
	for index, width := range widths {
		used += width
		patternIndex := 0
		if len(pattern) > 0 {
			patternIndex = index % len(pattern)
		}
		_, isPercentage := gridAutoColumnPercentageValues(percentages, percentageTracks, index)
		fixedZero := len(fixedZeroTracks) > 0 && fixedZeroTracks[index%len(fixedZeroTracks)]
		fraction := 0.0
		if len(fractions) > 0 {
			fraction = fractions[index%len(fractions)]
		}
		if (len(pattern) == 0 || pattern[patternIndex] == 0) && !isPercentage && !fixedZero && fraction == 0 {
			autoTracks = append(autoTracks, index)
		}
	}
	if len(autoTracks) == 0 || used >= available {
		return
	}
	share := (available - used) / float64(len(autoTracks))
	for _, index := range autoTracks {
		widths[index] += share
	}
}

func gridAutoColumnIntrinsicWidth(rows [][]gridItem, column int, opts Options) float64 {
	maxWidth := 0.0
	for _, row := range rows {
		for _, item := range row {
			if item.column != column || item.columnSpan != 1 {
				continue
			}
			width := intrinsicContribution(item.box, opts).borderMax
			if width > maxWidth {
				maxWidth = width
			}
		}
	}
	return maxWidth
}

func gridSpanAutoColumnContributions(g *boxes.Box, rows [][]gridItem, widths, pattern, percentages []float64, percentageTracks, fixedZeroTracks []bool, fractions, minimums []float64, explicitWidths, explicitMinimums []float64, firstImplicitColumn int, gap float64, opts Options) {
	for _, row := range rows {
		for _, item := range row {
			if item.columnSpan <= 1 || item.column+item.columnSpan <= firstImplicitColumn || item.column+item.columnSpan > firstImplicitColumn+len(widths) {
				continue
			}
			autoTracks := make([]int, 0, item.columnSpan)
			flexibleTracks := make([]int, 0, item.columnSpan)
			flexibleFraction := 0.0
			current := gap * float64(item.columnSpan-1)
			for column := item.column; column < item.column+item.columnSpan; column++ {
				if column < firstImplicitColumn {
					current += explicitWidths[column]
					if column < len(g.Style.GridTracks) && g.Style.GridTracks[column].Fraction > 0 {
						flexibleTracks = append(flexibleTracks, column)
						flexibleFraction += g.Style.GridTracks[column].Fraction
					}
					continue
				}
				index := column - firstImplicitColumn
				current += widths[index]
				patternIndex := 0
				if len(pattern) > 0 {
					patternIndex = index % len(pattern)
				}
				_, isPercentage := gridAutoColumnPercentageValues(percentages, percentageTracks, index)
				fixedZero := len(fixedZeroTracks) > 0 && fixedZeroTracks[index%len(fixedZeroTracks)]
				fraction := 0.0
				if len(fractions) > 0 {
					fraction = fractions[index%len(fractions)]
				}
				if (len(pattern) == 0 || pattern[patternIndex] == 0) && !isPercentage && !fixedZero && fraction == 0 {
					autoTracks = append(autoTracks, index)
				}
			}
			if len(autoTracks) == 0 && len(flexibleTracks) == 0 {
				continue
			}
			contribution := intrinsicContribution(item.box, opts).borderMin
			deficit := contribution - current
			if deficit <= 0 {
				continue
			}
			if len(flexibleTracks) == 0 {
				share := deficit / float64(len(autoTracks))
				for _, index := range autoTracks {
					widths[index] += share
				}
				continue
			}
			flexibleShare := deficit
			if flexibleFraction < 1 {
				flexibleShare *= flexibleFraction
			}
			for _, column := range flexibleTracks {
				fraction := g.Style.GridTracks[column].Fraction
				minimum := explicitWidths[column] + flexibleShare*fraction/flexibleFraction
				if minimum > explicitMinimums[column] {
					explicitMinimums[column] = minimum
				}
			}
			if flexibleFraction < 1 && len(autoTracks) > 0 {
				share := (deficit - flexibleShare) / float64(len(autoTracks))
				for _, index := range autoTracks {
					widths[index] += share
				}
			}
		}
	}
}

func gridTrackWidths(g *boxes.Box, usable float64, cols int, columnGap, implicitSpace float64, automaticMinimums []float64) []float64 {
	space := usable - columnGap*float64(cols-1) - implicitSpace
	if space < 0 {
		space = 0
	}
	if len(g.Style.GridTracks) != cols {
		width := space / float64(cols)
		widths := make([]float64, cols)
		for i := range widths {
			widths[i] = width
		}
		return widths
	}
	widths := make([]float64, cols)
	for i, track := range g.Style.GridTracks {
		widths[i] = track.Fixed + usable*track.Percent/100
		if track.Fraction == 0 {
			widths[i] += track.Min + usable*track.MinPercent/100
		}
	}
	free := space
	for i, track := range g.Style.GridTracks {
		if track.Fraction == 0 {
			free -= widths[i]
		}
	}
	if free < 0 {
		free = 0
	}
	active := make([]bool, cols)
	for i, track := range g.Style.GridTracks {
		active[i] = track.Fraction > 0
	}
	for {
		fractions := 0.0
		for i, track := range g.Style.GridTracks {
			if active[i] {
				fractions += track.Fraction
			}
		}
		if fractions == 0 {
			break
		}
		unit := free / fractions
		frozen := false
		for i, track := range g.Style.GridTracks {
			minimum := track.Min + usable*track.MinPercent/100
			if i < len(automaticMinimums) && automaticMinimums[i] > minimum {
				minimum = automaticMinimums[i]
			}
			if active[i] && unit*track.Fraction < minimum {
				widths[i] = minimum
				free -= minimum
				active[i] = false
				frozen = true
			}
		}
		if free < 0 {
			free = 0
		}
		if frozen {
			continue
		}
		for i, track := range g.Style.GridTracks {
			if active[i] {
				widths[i] = unit * track.Fraction
			}
		}
		break
	}
	return widths
}
