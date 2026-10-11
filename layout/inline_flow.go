package layout

import (
	"math"

	"github.com/robtrivi/pagecraft/boxes"
	"github.com/robtrivi/pagecraft/fonts"
)

// inlineLine is a measured line in the containing block's coordinate space.
// Its inline opportunity can differ from adjacent lines when floats exclude
// part of the line box.
type inlineLine struct {
	words  inlineStream
	x      float64
	width  float64
	y      float64
	height float64
}

// inlineFlowCursor is the continuation state for a measured inline stream.
// A formatting context can request the next line in a different opportunity
// (for example after advancing to another column) without reconstructing text
// from painted runs.
type inlineFlowCursor struct {
	remaining inlineStream
}

func newInlineFlowCursor(words inlineStream) inlineFlowCursor {
	prepared := append(inlineStream(nil), words...)
	for index := range prepared {
		prepared[index].streamID = index + 1
	}
	return inlineFlowCursor{remaining: prepared}
}

func (cursor *inlineFlowCursor) next(box *boxes.Box, width float64) (inlineStream, bool) {
	if len(cursor.remaining) == 0 {
		return nil, false
	}
	prepared := splitInlineItemsForWidth(cursor.remaining, width, wraps(box))
	broken := breakInlineStream(prepared, width, wraps(box))
	if len(broken) == 0 || len(broken[0]) == 0 {
		return nil, false
	}
	line := broken[0]
	lastID := line[len(line)-1].streamID
	consumedRunes := 0
	for _, word := range line {
		if word.streamID == lastID {
			consumedRunes += len([]rune(word.text))
		}
	}
	for index, word := range cursor.remaining {
		if word.streamID != lastID {
			continue
		}
		runes := []rune(word.text)
		if consumedRunes >= len(runes) {
			cursor.remaining = cursor.remaining[index+1:]
		} else {
			word.text = string(runes[consumedRunes:])
			if word.font != nil {
				parts, ok := word.shaped.SplitAtRuneOffsets([]int{consumedRunes})
				if ok && len(parts) == 2 {
					word.shaped = parts[1]
				} else {
					word.shaped = word.font.Shape(word.text, word.fs)
				}
				word.w = word.shaped.Width + word.src.Style.LetterSpacing*float64(len([]rune(word.text)))
			} else if word.measure != nil {
				word.shaped = fonts.ShapedText{}
				word.w = word.measure(word.text, word.fs)
			}
			word.breakOffsets = inlineBreakOffsetsFor(word.text, word.shaped)
			cursor.remaining = append(inlineStream{word}, cursor.remaining[index+1:]...)
		}
		return line, true
	}
	cursor.remaining = nil
	return line, true
}

// layoutInlineLines places one inline stream into successive line
// opportunities. Blocks and columns share this path so hard breaks, wrapping,
// line metrics, and float exclusions have one interpretation.
func layoutInlineLines(box *boxes.Box, words inlineStream, x, width, y float64, floats []activeFloat, page int, opts Options) []inlineLine {
	cursor := newInlineFlowCursor(words)
	lines := make([]inlineLine, 0, len(words))
	baseX, right := x, x+width
	for len(cursor.remaining) > 0 {
		lineHeight := lineHeightFor(box, resolveSize(box, opts), opts)
		leftEdge, rightEdge, nextY := baseX, right, math.Inf(1)
		for _, fl := range floats {
			if fl.page != page || y >= fl.bottom || y+lineHeight <= fl.top {
				continue
			}
			if fl.left {
				leftEdge = math.Max(leftEdge, fl.x+fl.w)
			} else {
				rightEdge = math.Min(rightEdge, fl.x)
			}
			nextY = math.Min(nextY, fl.bottom)
		}
		available := math.Max(0, rightEdge-leftEdge)
		if available <= fragmentEpsilon && !math.IsInf(nextY, 1) {
			y = nextY
			continue
		}
		line, ok := cursor.next(box, available)
		if !ok {
			if math.IsInf(nextY, 1) {
				break
			}
			y = nextY
			continue
		}
		lineHeight = lineHeightW(line, box, opts)
		lines = append(lines, inlineLine{words: line, x: leftEdge, width: available, y: y, height: lineHeight})
		y += lineHeight
	}
	return lines
}

func inlineLineRuns(box *boxes.Box, lines []inlineLine, opts Options) [][]TextRun {
	var runs [][]TextRun
	for _, line := range lines {
		runs = append(runs, assignRuns(box, []inlineStream{line.words}, line.x, line.width, line.y, opts)...)
	}
	return runs
}

func shiftInlineLines(lines []inlineLine, dy float64) []inlineLine {
	shifted := append([]inlineLine(nil), lines...)
	for index := range shifted {
		shifted[index].y += dy
	}
	return shifted
}

func inlineLineWords(lines []inlineLine) []inlineStream {
	words := make([]inlineStream, len(lines))
	for i := range lines {
		words[i] = lines[i].words
	}
	return words
}

func inlineLinesHeight(lines []inlineLine) float64 {
	if len(lines) == 0 {
		return 0
	}
	return lines[len(lines)-1].y + lines[len(lines)-1].height - lines[0].y
}
