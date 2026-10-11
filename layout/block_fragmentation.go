package layout

import (
	"math"
	"strings"

	"github.com/robtrivi/pagecraft/boxes"
)

func canFragmentTextBlock(b *boxes.Box) bool {
	if b.Style.Display != boxes.DisplayBlock || b.Tag == "li" || b.Style.Position != boxes.PositionStatic || b.Style.Float != boxes.FloatNone ||
		b.Image != nil || b.SVG != nil || b.Style.HasOpacity {
		return false
	}
	return true
}

func flowInlineBlockFragments(block *boxes.Box, lines []inlineStream, x, width, y, bottom float64, pages *[]Page, opts Options) float64 {
	inset := boxInsets(block)
	lineHeights := make([]float64, len(lines))
	for index, line := range lines {
		lineHeights[index] = lineHeightW(line, block, opts)
	}
	height := func(first, end int) float64 {
		total := 0.0
		for _, lineHeight := range lineHeights[first:end] {
			total += lineHeight
		}
		if first == 0 {
			total += inset.top
		}
		if end == len(lines) {
			total += inset.bottom
		}
		return total
	}
	pageBase := len(*pages) - 1
	pageHasContent := (*pages)[len(*pages)-1].blockCount() > 0
	var fragments []fragmentRange
	if len(opts.PageRules) != 0 {
		bounds := func(offset int) (float64, float64) {
			pageOpts := opts.forPage(pageBase + offset + 1)
			return pageOpts.top(), pageOpts.pageBottom()
		}
		fragments = planFragmentsByPage(blockLineFragmentUnits(block, len(lines)), y, pageHasContent, bounds, height)
	} else {
		fragments = planFragments(blockLineFragmentUnits(block, len(lines)), y, opts.top(), bottom, pageHasContent, height)
	}
	for _, fragmentRange := range fragments {
		ensurePageIndex(pages, pageBase+fragmentRange.pageOffset)
		fragmentLines := lines[fragmentRange.first:fragmentRange.end]
		first, last := fragmentRange.first == 0, fragmentRange.end == len(lines)
		fragmentBlock := fragmentBox(block, first, last)
		contentY := fragmentRange.y
		if first {
			contentY += inset.top
		}
		runs := assignRunsWithFinalLine(block, fragmentLines, x+inset.left, width-inset.left-inset.right, contentY, opts, last)
		fragment := placed(fragmentBlock, inlineFragmentText(fragmentLines), nil, x, fragmentRange.y, width, fragmentRange.height, resolveSize(block, opts), fontFor(opts, block))
		fragment = fragmentPlacedBox(fragment, first, last)
		fragment.Runs = runs
		(*pages)[pageBase+fragmentRange.pageOffset].emitBlock(fragment)
	}
	if len(fragments) == 0 {
		return y
	}
	last := fragments[len(fragments)-1]
	return last.y + last.height
}

// flowResponsiveInlineFragments rebreaks a paragraph against each destination page's width and height.
func flowResponsiveInlineFragments(block *boxes.Box, stream inlineStream, y float64, pages *[]Page, opts Options) float64 {
	inset := boxInsets(block)
	pageIndex := len(*pages) - 1
	firstFragment := true
	for len(stream) > 0 {
		pageOpts := opts.forPage(pageIndex + 1)
		ensurePageIndex(pages, pageIndex)
		pageTop, pageBottom := pageOpts.top(), pageOpts.pageBottom()
		if pageIndex != len(*pages)-1 {
			pageIndex = len(*pages) - 1
		}
		if pageIndex >= 0 && (*pages)[pageIndex].blockCount() == 0 && y < pageTop {
			y = pageTop
		}
		available := math.Max(0, pageOpts.PageWidth-pageOpts.left()-pageOpts.right()-block.Style.MarginLeft-block.Style.MarginRight)
		width := usedWidth(block, available)
		contentWidth := math.Max(0, width-inset.left-inset.right)
		lines := breakInlineStream(stream, contentWidth, wraps(block))
		if len(lines) == 0 {
			return y
		}
		lineHeights := make([]float64, len(lines))
		for i := range lines {
			lineHeights[i] = lineHeightW(lines[i], block, pageOpts)
		}
		fit, extent := 0, 0.0
		for fit < len(lines) {
			extra := lineHeights[fit]
			if firstFragment && fit == 0 {
				extra += inset.top
			}
			if fit+1 == len(lines) {
				extra += inset.bottom
			}
			if y+extent+extra > pageBottom+fragmentEpsilon {
				break
			}
			extent += extra
			fit++
		}
		if fit == 0 {
			if (*pages)[pageIndex].blockCount() > 0 || y > pageTop+fragmentEpsilon {
				pageIndex++
				ensurePageIndex(pages, pageIndex)
				y = pageOpts.forPage(pageIndex + 1).top()
				continue
			}
			fit, extent = 1, lineHeights[0]
			if firstFragment {
				extent += inset.top
			}
			if len(lines) == 1 {
				extent += inset.bottom
			}
		}
		if fit < len(lines) && fit == 1 && len(lines) > 1 && (*pages)[pageIndex].blockCount() > 0 {
			pageIndex++
			ensurePageIndex(pages, pageIndex)
			y = opts.forPage(pageIndex + 1).top()
			continue
		}
		if fit < len(lines) && len(lines)-fit == 1 && fit > 2 {
			fit--
			extent -= lineHeights[fit]
		}
		last := fit == len(lines)
		fragmentLines := lines[:fit]
		boxX := pageOpts.left() + block.Style.MarginLeft
		box := fragmentBox(block, firstFragment, last)
		contentY := y
		if firstFragment {
			contentY += inset.top
		}
		runs := assignRunsWithFinalLine(block, fragmentLines, boxX+inset.left, contentWidth, contentY, pageOpts, last)
		fragment := placed(box, inlineFragmentText(fragmentLines), nil, boxX, y, width, extent, resolveSize(block, pageOpts), fontFor(pageOpts, block))
		fragment = fragmentPlacedBox(fragment, firstFragment, last)
		fragment.Runs = runs
		(*pages)[pageIndex].emitBlock(fragment)
		if last {
			return y + extent
		}
		var rest inlineStream
		for _, line := range lines[fit:] {
			rest = append(rest, line...)
		}
		stream = rest
		firstFragment = false
		pageIndex++
		ensurePageIndex(pages, pageIndex)
		y = opts.forPage(pageIndex + 1).top()
	}
	return y
}

func blockLineFragmentUnits(block *boxes.Box, lineCount int) []fragmentUnit {
	units := make([]fragmentUnit, lineCount)
	for i := range units {
		units[i] = fragmentUnit{group: 1, orphans: block.Style.Orphans, widows: block.Style.Widows}
	}
	return units
}

func inlineFragmentText(lines []inlineStream) string {
	textLines := make([]string, len(lines))
	for index, line := range lines {
		var text strings.Builder
		for index, word := range line {
			if index > 0 && word.before.advance > 0 {
				text.WriteByte(' ')
			}
			text.WriteString(word.text)
		}
		textLines[index] = strings.TrimRight(text.String(), " ")
	}
	return strings.Join(textLines, "\n")
}

// nestedContainerInsets returns the insets of a decorated container whose
// children are all blocks.
