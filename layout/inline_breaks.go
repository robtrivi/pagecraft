package layout

import (
	"sort"
	"strings"
	"unicode"

	"github.com/go-text/typesetting/segmenter"
	"github.com/robtrivi/pagecraft/boxes"
	"github.com/robtrivi/pagecraft/fonts"
)

// lineBreakOpportunityBetween resolves to a draft boundary once. Sizing and line
// construction consume the resulting inlineBoundary rather than re-deriving it.
func lineBreakOpportunityBetween(previous, next inlineDraft) bool {
	if ancestor := commonInlineScope(previous.scope, next.scope); ancestor != nil {
		if ancestor.Style.WhiteSpace == "pre" || ancestor.Style.WhiteSpace == "nowrap" {
			return false
		}
	}
	if previous.spacesAfter <= 0 {
		canWrap := previous.src == nil || next.src == nil || wraps(previous.src) || wraps(next.src)
		return canWrap && unicodeLineBreakOpportunity(previous.text, next.text)
	}
	if previous.src == next.src {
		return previous.src == nil || wraps(previous.src)
	}
	return previous.src == nil || next.src == nil || wraps(previous.src) || wraps(next.src)
}

func commonInlineScope(left, right *inlineScope) *boxes.Box {
	for left != nil && right != nil && left.depth > right.depth {
		left = left.parent
	}
	for left != nil && right != nil && right.depth > left.depth {
		right = right.parent
	}
	for left != nil && right != nil && left != right {
		left, right = left.parent, right.parent
	}
	if left == nil || right == nil {
		return nil
	}
	return left.box
}

func unicodeLineBreakOpportunity(previous, next string) bool {
	if previous == "" || next == "" {
		return false
	}
	combined := previous + next
	boundary := len([]rune(previous))
	var lineBreaks segmenter.Segmenter
	lineBreaks.InitWithString(combined)
	iterator := lineBreaks.LineIterator()
	for iterator.Next() {
		line := iterator.Line()
		if line.Offset+len(line.Text) == boundary {
			return true
		}
	}
	return false
}

func inlineBreakOffsets(text string) []int {
	runes := []rune(text)
	boundaries := inlineGraphemeBoundaries(runes)
	offsetSet := make(map[int]struct{})
	var lineBreaks segmenter.Segmenter
	lineBreaks.Init(runes)
	iterator := lineBreaks.LineIterator()
	for iterator.Next() {
		line := iterator.Line()
		offset := line.Offset + len(line.Text)
		if offset > 0 && offset < len(runes) && boundaries[offset] && !forbiddenCJKBoundary(runes, offset) {
			offsetSet[offset] = struct{}{}
		}
	}
	for index, current := range runes {
		if (current == '-' || current == '\u2010') && boundaries[index+1] {
			offsetSet[index+1] = struct{}{}
		}
	}
	for index := range runes {
		if isSignedNumberBoundary(runes, index+1) {
			delete(offsetSet, index+1)
		}
	}
	offsets := make([]int, 0, len(offsetSet))
	for offset := range offsetSet {
		offsets = append(offsets, offset)
	}
	sort.Ints(offsets)
	if len(offsets) == 0 {
		return nil
	}
	return offsets
}

func forbiddenCJKBoundary(runes []rune, offset int) bool {
	if offset <= 0 || offset >= len(runes) {
		return false
	}
	previous, next := runes[offset-1], runes[offset]
	return isCJKOpeningPunctuation(previous) || isCJKOpeningPunctuation(next) || isCJKClosingPunctuation(next)
}

func isCJKOpeningPunctuation(r rune) bool {
	return strings.ContainsRune("「『（［｛〈《【〔〖〘〚", r)
}

func isCJKClosingPunctuation(r rune) bool {
	return strings.ContainsRune("、。，．！？：；」』）］｝〉》】〕〗〙〛", r)
}

func isSignedNumberBoundary(runes []rune, offset int) bool {
	if offset < 1 || offset >= len(runes) || (runes[offset-1] != '-' && runes[offset-1] != '\u2010') || !unicode.IsDigit(runes[offset]) {
		return false
	}
	if offset == 1 {
		return true
	}
	previous := runes[offset-2]
	return !unicode.IsLetter(previous) && !unicode.IsNumber(previous)
}

func adjacentCJKLineCharacters(previous, next string) bool {
	previousRunes, nextRunes := []rune(previous), []rune(next)
	return len(previousRunes) > 0 && len(nextRunes) > 0 && isCJKLineCharacter(previousRunes[len(previousRunes)-1]) && isCJKLineCharacter(nextRunes[0])
}

func isCJKLineCharacter(r rune) bool {
	return unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul)
}

func inlineBreakOffsetsFor(text string, shaped fonts.ShapedText) []int {
	offsets := inlineBreakOffsets(text)
	if len(shaped.Glyphs) == 0 {
		return offsets
	}
	safe := offsets[:0]
	for _, offset := range offsets {
		if _, ok := shaped.SplitAtRuneOffsets([]int{offset}); ok {
			safe = append(safe, offset)
		}
	}
	return safe
}

// inlineGraphemeBoundaries returns rune offsets at which text can be split
// without separating combining sequences, variation selectors, emoji ZWJ
// sequences, modifiers, or a pair of regional indicators.
func inlineGraphemeBoundaries(runes []rune) []bool {
	boundaries := make([]bool, len(runes)+1)
	boundaries[0], boundaries[len(runes)] = true, true
	regionalIndicators := 0
	for index := 1; index < len(runes); index++ {
		previous, current := runes[index-1], runes[index]
		allowed := !unicode.Is(unicode.Mn, current) && !unicode.Is(unicode.Mc, current) && !unicode.Is(unicode.Me, current) &&
			!isVariationSelector(current) && !isEmojiModifier(current) && previous != '\u200d' && current != '\u200d'
		if isRegionalIndicator(previous) && isRegionalIndicator(current) {
			if index == 1 || !isRegionalIndicator(runes[index-2]) {
				regionalIndicators = 1
			} else {
				regionalIndicators++
			}
			allowed = regionalIndicators%2 == 0
		} else {
			regionalIndicators = 0
		}
		boundaries[index] = allowed
	}
	return boundaries
}

func isVariationSelector(r rune) bool {
	return r >= 0xfe00 && r <= 0xfe0f || r >= 0xe0100 && r <= 0xe01ef
}

func isEmojiModifier(r rune) bool { return r >= 0x1f3fb && r <= 0x1f3ff }

func isRegionalIndicator(r rune) bool { return r >= 0x1f1e6 && r <= 0x1f1ff }

func splitInlineItemAtBreaks(word inlineItem) inlineStream {
	if len(word.breakOffsets) == 0 {
		return inlineStream{word}
	}
	runes := []rune(word.text)
	var shapedParts []fonts.ShapedText
	if word.font != nil {
		var ok bool
		shapedParts, ok = word.shaped.SplitAtRuneOffsets(word.breakOffsets)
		if !ok {
			return inlineStream{word}
		}
	}
	parts := make(inlineStream, 0, len(word.breakOffsets)+1)
	start := 0
	appendPart := func(end int) {
		part := word
		if len(parts) > 0 {
			part.before = inlineBoundary{kind: inlineBreakOpportunity}
		}
		part.text = string(runes[start:end])
		if part.font != nil {
			part.shaped = shapedParts[len(parts)]
			part.w = part.shaped.Width + part.src.Style.LetterSpacing*float64(len([]rune(part.text)))
		} else if part.measure != nil {
			part.shaped = fonts.ShapedText{}
			part.w = part.measure(part.text, part.fs)
		}
		part.breakOffsets = nil
		parts = append(parts, part)
		start = end
	}
	for _, offset := range word.breakOffsets {
		if offset > start && offset <= len(runes) {
			appendPart(offset)
		}
	}
	if start < len(runes) {
		appendPart(len(runes))
	}
	return parts
}
