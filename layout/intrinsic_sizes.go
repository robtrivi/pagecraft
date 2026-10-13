package layout

import (
	"strings"
	"unicode"

	"github.com/robtrivi/pagecraft/boxes"
)

type intrinsicInlineWidths struct {
	minContent float64
	maxContent float64
}

// intrinsicBoxContribution keeps content-box and border-box contributions
// together so each layout mode applies insets once and uses the same text/image
// measurement contract.
type intrinsicBoxContribution struct {
	contentMin float64
	contentMax float64
	borderMin  float64
	borderMax  float64
}

func intrinsicContribution(box *boxes.Box, opts Options) intrinsicBoxContribution {
	content := intrinsicInlineWidths{}
	if box.Image != nil {
		width, _ := rasterDimensions(box, 0, 0)
		content.minContent, content.maxContent = width, width
	} else {
		content = intrinsicContentWidths(box, opts)
	}
	insets := horizontalInsets(box)
	return intrinsicBoxContribution{
		contentMin: content.minContent,
		contentMax: content.maxContent,
		borderMin:  content.minContent + insets,
		borderMax:  content.maxContent + insets,
	}
}

// intrinsicContentWidths follows formatting-context boundaries. Consecutive
// inline parts share a line; block children contribute their own widths.
func intrinsicContentWidths(box *boxes.Box, opts Options) intrinsicInlineWidths {
	if isTableFormattingContext(box) {
		return intrinsicTableContentWidths(box, opts)
	}
	hasBlockPart := false
	for _, part := range box.Parts {
		if part.Child != nil && !inlinePartChild(part.Child) {
			hasBlockPart = true
			break
		}
	}
	if !hasBlockPart {
		return inlineContentWidths(box, opts)
	}
	var widths intrinsicInlineWidths
	var inlineParts []boxes.Part
	flushInline := func() {
		if len(inlineParts) == 0 {
			return
		}
		inlineBox := *box
		inlineBox.Parts = inlineParts
		inlineBox.Children = nil
		partWidths := inlineContentWidths(&inlineBox, opts)
		widths.minContent = max(widths.minContent, partWidths.minContent)
		widths.maxContent = max(widths.maxContent, partWidths.maxContent)
		inlineParts = nil
	}
	for _, part := range box.Parts {
		if part.Child == nil || inlinePartChild(part.Child) {
			inlineParts = append(inlineParts, part)
			continue
		}
		flushInline()
		child := part.Child
		contribution := intrinsicContribution(child, opts)
		margins := child.Style.MarginLeft + child.Style.MarginRight
		widths.minContent = max(widths.minContent, contribution.borderMin+margins)
		widths.maxContent = max(widths.maxContent, contribution.borderMax+margins)
	}
	flushInline()
	return widths
}

func intrinsicTableContentWidths(table *boxes.Box, opts Options) intrinsicInlineWidths {
	head, body := tableRows(table)
	grid, cols := tableGridFromRows(append(head, body...))
	minimum, maximum := tableIntrinsicWidths(grid, cols, opts)
	return intrinsicInlineWidths{minContent: minimum, maxContent: maximum}
}

// inlineContentWidths is the shared text/image contribution used by Flex,
// Grid, and table sizing. It measures the same styled tokens as line layout.
func inlineContentWidths(box *boxes.Box, opts Options) intrinsicInlineWidths {
	words := buildInlineStream(box, opts)
	return intrinsicInlineWidths{
		minContent: minContentInlineStream(words, opts),
		maxContent: maxContentInlineStream(words),
	}
}

func minContentInlineStream(words inlineStream, opts Options) float64 {
	minimum := 0.0
	segment := make(inlineStream, 0, len(words))
	finish := func() {
		minimum = max(minimum, minContentSegmentWidth(segment, opts))
		segment = segment[:0]
	}
	for index := range words {
		word := &words[index]
		if word.kind == inlineHardBreak {
			finish()
			continue
		}
		if word.before.kind == inlineBreakOpportunity {
			finish()
		}
		segment = append(segment, *word)
	}
	finish()
	return minimum
}

func minContentSegmentWidth(words inlineStream, opts Options) float64 {
	if len(words) == 0 {
		return 0
	}
	source := words[0].src
	if source != nil && !wraps(source) {
		var text strings.Builder
		width := 0.0
		for index := range words {
			word := &words[index]
			if word.kind == inlineHardBreak {
				width += measureIntrinsicText(source, text.String(), words[0].fs, opts)
				text.Reset()
				continue
			}
			if word.image != nil {
				width += measureIntrinsicText(source, text.String(), words[0].fs, opts)
				text.Reset()
				width += word.w
				continue
			}
			if word.before.spaceCount > 0 {
				text.WriteString(strings.Repeat(" ", word.before.spaceCount))
			}
			text.WriteString(word.text)
		}
		return width + measureIntrinsicText(source, text.String(), words[0].fs, opts)
	}
	width := 0.0
	for _, word := range words {
		width += minContentInlineItem(word, opts)
	}
	return width
}

func maxContentInlineStream(words inlineStream) float64 {
	maximum, line := 0.0, 0.0
	for _, word := range words {
		if word.kind == inlineHardBreak {
			maximum = max(maximum, line)
			line = 0
			continue
		}
		line += word.before.advance
		line += word.w
	}
	return max(maximum, line)
}

func minContentInlineItem(word inlineItem, opts Options) float64 {
	if word.image != nil || word.text == "" || word.src == nil || !wraps(word.src) {
		return word.w
	}
	if len(word.breakOffsets) > 0 {
		minimum := 0.0
		for _, part := range splitInlineItemAtBreaks(word) {
			minimum = max(minimum, minContentInlineItem(part, opts))
		}
		return minimum
	}
	if !strings.ContainsFunc(word.text, func(r rune) bool { return unicode.Is(unicode.Han, r) }) {
		return word.w
	}
	minimum, segment := 0.0, strings.Builder{}
	flush := func() {
		if segment.Len() == 0 {
			return
		}
		minimum = max(minimum, measureIntrinsicSegment(word, segment.String(), opts))
		segment.Reset()
	}
	for _, char := range word.text {
		if unicode.Is(unicode.Han, char) {
			flush()
			minimum = max(minimum, measureIntrinsicSegment(word, string(char), opts))
			continue
		}
		segment.WriteRune(char)
	}
	flush()
	return minimum
}

func measureIntrinsicSegment(word inlineItem, value string, opts Options) float64 {
	return measureIntrinsicText(word.src, value, word.fs, opts)
}

func measureIntrinsicText(source *boxes.Box, value string, fontSize float64, opts Options) float64 {
	measure, font := measureFor(opts, source)
	if font != nil {
		return font.Shape(value, fontSize).Width + source.Style.LetterSpacing*float64(len([]rune(value)))
	}
	return measure(value, fontSize)
}
