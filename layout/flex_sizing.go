package layout

import (
	"math"

	"github.com/robtrivi/pagecraft/boxes"
)

func inlineStreamNaturalWidth(words inlineStream) float64 {
	return maxContentInlineStream(words)
}

func flexBaseSize(item *boxes.Box, availableWidth float64, opts Options) float64 {
	insets := horizontalInsets(item)
	basis := item.Style.FlexBasis
	var base float64
	switch basis.Kind {
	case boxes.FlexBasisLength:
		base = basis.Value
	case boxes.FlexBasisPercentage:
		base = availableWidth * basis.Value / 100
	case boxes.FlexBasisAuto:
		if !item.Style.Sizing.Width.IsAuto() {
			base, _ = resolveBorderBoxDimension(item.Style.Sizing.Width, definiteSize(availableWidth), insets, item.Style.BoxSizing)
		} else {
			base = inlineStreamNaturalWidth(buildInlineStream(item, opts)) + insets
		}
	case boxes.FlexBasisContent:
		base = inlineStreamNaturalWidth(buildInlineStream(item, opts)) + insets
	default:
		base = inlineStreamNaturalWidth(buildInlineStream(item, opts)) + insets
	}
	if basis.Kind == boxes.FlexBasisLength || basis.Kind == boxes.FlexBasisPercentage {
		if item.Style.BoxSizing != boxes.BoxSizingBorderBox {
			base += insets
		}
	}
	return base
}

func flexItemMinContentWidth(item *boxes.Box, opts Options, availableWidth float64) float64 {
	if minimum, ok := resolveBorderBoxDimension(item.Style.Sizing.MinWidth, definiteSize(availableWidth), horizontalInsets(item), item.Style.BoxSizing); ok {
		return math.Max(0, minimum)
	}
	minimum := 0.0
	if !flexItemHasZeroAutomaticMinWidth(item) {
		minimum = intrinsicContribution(item, opts).contentMin
		for _, part := range item.Parts {
			if part.Child != nil && !inlinePartChild(part.Child) {
				childMinimum := flexItemMinContentWidth(part.Child, opts, math.Max(0, availableWidth-horizontalInsets(item)))
				minimum = math.Max(minimum, childMinimum)
			}
		}
	}
	minimum += horizontalInsets(item)
	if maximum, ok := flexMaximumBorderBoxWidth(item, availableWidth); ok {
		minimum = math.Min(minimum, maximum)
	}
	return minimum
}

func flexItemHasZeroAutomaticMinWidth(item *boxes.Box) bool {
	return overflowAxisIsScrollable(item, true)
}

func flexMaximumBorderBoxWidth(item *boxes.Box, availableWidth float64) (float64, bool) {
	maximum, ok := resolveBorderBoxDimension(item.Style.Sizing.MaxWidth, definiteSize(availableWidth), horizontalInsets(item), item.Style.BoxSizing)
	if !ok {
		return 0, false
	}
	if minimum, hasMinimum := resolveBorderBoxDimension(item.Style.Sizing.MinWidth, definiteSize(availableWidth), horizontalInsets(item), item.Style.BoxSizing); hasMinimum {
		maximum = math.Max(maximum, minimum)
	}
	return math.Max(horizontalInsets(item), maximum), true
}

func resolveFlexLine(widths []float64, row flexRow, items []*boxes.Box, minimums []float64, availableWidth, gap float64) {
	if len(row) == 0 {
		return
	}
	lineItems := make([]*boxes.Box, len(row))
	bases := make([]float64, len(row))
	hypothetical := make([]float64, len(row))
	growFactors := make([]float64, len(row))
	shrinkFactors := make([]float64, len(row))
	hypotheticalTotal := gap * float64(len(row)-1)
	for local, index := range row {
		item := items[index]
		lineItems[local] = item
		bases[local] = widths[index]
		hypothetical[local] = hypotheticalFlexSize(item, widths[index], minimums[index], availableWidth)
		growFactors[local] = item.Style.FlexGrow
		shrinkFactors[local] = item.Style.FlexShrink
		if !item.Style.FlexShrinkSet {
			shrinkFactors[local] = 1
		}
		hypotheticalTotal += hypothetical[local]
	}
	useGrow := hypotheticalTotal < availableWidth-1e-7
	targets := append([]float64(nil), bases...)
	frozen := make([]bool, len(row))
	for i := range lineItems {
		factor := shrinkFactors[i]
		if useGrow {
			factor = growFactors[i]
		}
		if factor == 0 || useGrow && bases[i] > hypothetical[i]+1e-7 || !useGrow && bases[i] < hypothetical[i]-1e-7 {
			targets[i], frozen[i] = hypothetical[i], true
		}
	}
	initialFree := flexLineFreeSpace(availableWidth, gap, bases, targets, frozen)
	for iteration := 0; iteration < len(row)+1; iteration++ {
		remaining := flexLineFreeSpace(availableWidth, gap, bases, targets, frozen)
		factorSum := 0.0
		for i := range row {
			if frozen[i] {
				continue
			}
			if useGrow {
				factorSum += growFactors[i]
			} else {
				factorSum += shrinkFactors[i]
			}
		}
		if factorSum > 0 && factorSum < 1 {
			partial := initialFree * factorSum
			if math.Abs(partial) < math.Abs(remaining) {
				remaining = partial
			}
		}
		scaledShrinkSum := 0.0
		if !useGrow {
			for i := range row {
				if !frozen[i] {
					scaledShrinkSum += shrinkFactors[i] * math.Max(0, bases[i]-horizontalInsets(lineItems[i]))
				}
			}
		}
		clamped := make([]float64, len(row))
		unclamped := make([]float64, len(row))
		violation := 0.0
		for i, item := range lineItems {
			if frozen[i] {
				continue
			}
			candidate := bases[i]
			if useGrow && factorSum > 0 {
				candidate += remaining * growFactors[i] / factorSum
			} else if !useGrow && scaledShrinkSum > 0 {
				weight := shrinkFactors[i] * math.Max(0, bases[i]-horizontalInsets(item))
				candidate -= math.Abs(remaining) * weight / scaledShrinkSum
			}
			unclamped[i] = candidate
			clamped[i] = clampFlexTarget(item, candidate, minimums[row[i]], availableWidth)
			violation += clamped[i] - candidate
		}
		if math.Abs(violation) <= 1e-7 {
			for i := range frozen {
				if !frozen[i] {
					targets[i], frozen[i] = clamped[i], true
				}
			}
			break
		}
		froze := false
		for i := range row {
			if frozen[i] {
				continue
			}
			adjustment := clamped[i] - unclamped[i]
			if violation > 0 && adjustment > 1e-7 || violation < 0 && adjustment < -1e-7 || math.Abs(violation) <= 1e-7 {
				targets[i], frozen[i], froze = clamped[i], true, true
			}
		}
		if !froze {
			break
		}
	}
	for local, index := range row {
		widths[index] = targets[local]
	}
}

func hypotheticalFlexSize(item *boxes.Box, base, minimum, availableWidth float64) float64 {
	return clampFlexTarget(item, math.Max(base, minimum), minimum, availableWidth)
}

func clampFlexTarget(item *boxes.Box, target, minimum, availableWidth float64) float64 {
	insets := horizontalInsets(item)
	target = math.Max(target, insets)
	target = math.Max(target, minimum)
	if maximum, ok := flexMaximumBorderBoxWidth(item, availableWidth); ok {
		target = math.Max(minimum, math.Min(target, maximum))
	}
	return target
}

func flexLineFreeSpace(availableWidth, gap float64, bases, targets []float64, frozen []bool) float64 {
	free := availableWidth - gap*float64(max(0, len(bases)-1))
	for i := range bases {
		if frozen[i] {
			free -= targets[i]
		} else {
			free -= bases[i]
		}
	}
	return free
}

func flexItemHasBlockFlow(item *boxes.Box) bool {
	if item.Style.Display == boxes.DisplayFlex || item.Style.Display == boxes.DisplayGrid || isMultiColumn(item) || isTableFormattingContext(item) {
		return true
	}
	for _, part := range item.Parts {
		if part.Child != nil && !inlinePartChild(part.Child) {
			return true
		}
	}
	return false
}

func measureFlexItemHeight(item *boxes.Box, width float64, opts Options) float64 {
	measureOpts := opts
	measureOpts.PageWidth, measureOpts.PageHeight, measureOpts.Margin = width, unboundedFragmentainerPageHeight(opts.PageHeight), 0
	measureOpts.space = measureOpts.space.withPageMargin(0, true).
		withMargins(0, 0, 0, 0)
	measureOpts.space.startY, measureOpts.space.startYSet = 0, false
	pages := []Page{{Number: 1}}
	return flowFlexBlockItem(item, 0, &pages, measureOpts, width, measureOpts.pageBottom())
}

// flowFlexBlockItem preserves nested formatting contexts instead of flattening
// their children into ordinary block flow.
func flowFlexBlockItem(item *boxes.Box, y float64, pages *[]Page, opts Options, usable, bottom float64) float64 {
	if isTableFormattingContext(item) {
		y += item.Style.MarginTop
		y = flowTable(item, y, pages, opts, usable, bottom)
		return y + item.Style.MarginBottom
	}
	if item.Style.Display == boxes.DisplayFlex {
		return flowFlex(item, y, pages, opts, usable, bottom)
	}
	if item.Style.Display == boxes.DisplayGrid {
		return flowGrid(item, y, pages, opts, usable, bottom)
	}
	if isMultiColumn(item) {
		return flowColumns(item, y, pages, opts, usable, bottom)
	}
	return flowNestedContainer(item, y, boxInsets(item), nestedContainerItems(item), false, pages, opts, usable)
}

func withUsedBorderBoxWidth(item *boxes.Box, width float64) *boxes.Box {
	resolved := *item
	resolved.Style.Sizing.Width = boxes.CSSDimension{Kind: boxes.DimensionLength, Value: width}
	if resolved.Style.BoxSizing != boxes.BoxSizingBorderBox {
		resolved.Style.Sizing.Width.Value = math.Max(0, width-horizontalInsets(item))
	}
	return &resolved
}
