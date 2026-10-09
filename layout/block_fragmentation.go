package layout

import (
	"strings"

	"github.com/robtrivi/pagecraft-go/boxes"
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
	fragments := planFragments(blockLineFragmentUnits(block, len(lines)), y, opts.top(), bottom, pageHasContent, height)
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
