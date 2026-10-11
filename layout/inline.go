package layout

import (
	"math"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/go-text/typesetting/bidi"
	"github.com/go-text/typesetting/language"
	"github.com/robtrivi/pagecraft/boxes"
	"github.com/robtrivi/pagecraft/fonts"
	"github.com/robtrivi/pagecraft/images"
)

// TextRun represents a positioned inline item: text or a replaced image.
type TextRun struct {
	Text                            string
	Shaped                          fonts.ShapedText
	Image                           *images.Raster
	ImageHeight                     float64
	X, Y                            float64 // Absolute X; Y is the line top.
	BaselineOffset                  float64 // Distance from Y to the baseline.
	Width                           float64
	LineH                           float64 // Height of the line.
	FontSize                        float64
	Bold, Italic, Underline, Strike bool
	HasColor                        bool
	Color                           [4]float64
	Link                            string
	Font                            *fonts.Face
	LetterSpacing                   float64
	Alpha                           float64 // effective alpha (color multiplied by opacity)
	GroupOpacity                    bool    // opacity comes from the composited block
	ListMarkerShape                 string
	LineY                           float64
	InlineDecorations               []InlineBoxDecoration
	source                          *boxes.Box
}

// InlineBoxDecoration is one painted line fragment attached to the first run
// of its inline source. Offsets are relative to that run so later placement
// translations keep text and decoration together.
type InlineBoxDecoration struct {
	Box              PlacedBox
	OffsetX, OffsetY float64
}

type inlineItemKind uint8

const (
	inlineText inlineItemKind = iota
	inlineImage
	inlineHardBreak
)

type inlineBreakKind uint8

const (
	inlineNoBreak inlineBreakKind = iota
	inlineBreakOpportunity
)

// inlineBoundary separates adjacent items. Advance and break permission are
// independent, as whitespace can contribute width inside a nowrap group.
type inlineBoundary struct {
	kind           inlineBreakKind
	advance        float64
	spaceCount     int
	leadingAdvance float64
}

type inlineStream []inlineItem

// inlineItem is a text run, replaced element, or explicit line break with
// resolved style and metrics.
type inlineItem struct {
	kind             inlineItemKind
	before           inlineBoundary
	text             string
	shaped           fonts.ShapedText
	src              *boxes.Box
	image            *images.Raster
	font             *fonts.Face
	measure          func(string, float64) float64
	fs               float64
	w                float64 // measured width
	imageH           float64
	inlineStartInset float64
	inlineEndInset   float64
	breakOffsets     []int
	markerShape      string
	streamID         int
	bidiLevel        bidi.Level
	bidiBaseRTL      bool
	hasBidiLevel     bool
}

// inlineDraft contains tokenizer state resolved into inlineBoundary before
// sizing and line construction consume the stream.
type inlineDraft struct {
	inlineItem
	candidateBreakOffsets []int
	sw                    float64
	spacesAfter           int
	sourceSpaceBefore     bool
	sourceSpaceAfter      bool
	bidiBoundary          bool
	hardBreak             bool
	leadSpaces            int
	hardStart             bool
	scope                 *inlineScope
}

// inlineScope is a shared persistent path to an inline box; drafts refer to
// it without allocating a copy of every ancestor chain per text token.
type inlineScope struct {
	box    *boxes.Box
	parent *inlineScope
	depth  int
}

// collectInlineDrafts flattens anonymous text, inline children, and replaced
// content while retaining only the temporary tokenizer context.
func collectInlineDrafts(b *boxes.Box, opts Options) []inlineDraft {
	return collectInlineDraftsWithScope(b, opts, &inlineScope{box: b})
}

func collectInlineDraftsWithScope(b *boxes.Box, opts Options, scope *inlineScope) []inlineDraft {
	var out []inlineDraft
	if len(b.Parts) == 0 {
		start := len(out)
		out = appendTextDrafts(out, b, b.Text, opts)
		setInlineScope(out[start:], scope)
		return out
	}
	for _, p := range b.Parts {
		if p.Child == nil {
			start := len(out)
			out = appendTextDrafts(out, b, p.Text, opts)
			setInlineScope(out[start:], scope)
			for i := start; i < len(out); i++ {
				out[i].markerShape = p.ListMarkerShape
			}
		} else if p.Child.Tag == "br" {
			out = appendHardBreakDraft(out, b, opts)
		} else if b.Tag == "li" && (p.Child.Tag == "ul" || p.Child.Tag == "ol") {
			continue
		} else if inlinePartChild(p.Child) {
			childScope := &inlineScope{box: p.Child, parent: scope, depth: scope.depth + 1}
			out = appendFlattenInlineDraftsInPlace(out, p.Child, 0, opts, childScope)
		} else {
			if len(out) > 0 {
				out[len(out)-1].hardBreak = true
			}
			out = appendTextDrafts(out, p.Child, p.Child.Text, opts)
			if len(out) > 0 {
				out[len(out)-1].hardBreak = true
			}
		}
	}
	normalizeCJKAdjacentSpacing(out)
	return out
}

func buildInlineStream(b *boxes.Box, opts Options) inlineStream {
	return resolveInlineStream(collectInlineDrafts(b, opts))
}

func isSimpleLTRText(text string) bool {
	for _, r := range text {
		if !unicode.IsGraphic(r) {
			return false
		}
		if r < utf8.RuneSelf {
			continue
		}
		script := language.LookupScript(r)
		if script != language.Latin && script != language.Common && script != language.Inherited {
			return false
		}
	}
	return true
}

func itemizeInlineDraftsByBidi(drafts []inlineDraft) []inlineDraft {
	type textRange struct{ start, end int }
	var result []inlineDraft
	for _, draft := range drafts {
		if draft.text == "" || draft.image != nil || draft.kind == inlineHardBreak {
			result = append(result, draft)
			continue
		}
		if isSimpleLTRText(draft.text) {
			result = append(result, draft)
			continue
		}
		runes := []rune(draft.text)
		paragraph := bidi.Paragraph{}
		bidiRuns := paragraph.Segment(runes, bidi.Neutral)
		var ranges []textRange
		for runIndex := 0; runIndex < bidiRuns.NumRuns(); runIndex++ {
			run := bidiRuns.Run(runIndex)
			start, script := run.Start, language.Common
			for runeIndex := run.Start; runeIndex < run.End; runeIndex++ {
				nextScript := language.LookupScript(runes[runeIndex])
				if !nextScript.Strong() {
					continue
				}
				if script.Strong() && nextScript != script {
					ranges = append(ranges, textRange{start, runeIndex})
					start = runeIndex
				}
				script = nextScript
			}
			if start < run.End {
				ranges = append(ranges, textRange{start, run.End})
			}
		}
		if len(ranges) < 2 {
			result = append(result, draft)
			continue
		}
		parts := make([]inlineDraft, 0, len(ranges))
		for index, textRange := range ranges {
			part := draft
			part.text = string(runes[textRange.start:textRange.end])
			part.bidiBoundary = true
			part.sourceSpaceBefore = index == 0 && draft.sourceSpaceBefore
			part.sourceSpaceAfter = index == len(ranges)-1 && draft.sourceSpaceAfter
			part.hardBreak = index == len(ranges)-1 && draft.hardBreak
			part.spacesAfter = 0
			part.candidateBreakOffsets = nil
			if draft.font != nil {
				part.shaped = draft.font.Shape(part.text, part.fs)
				part.w = part.shaped.Width + part.src.Style.LetterSpacing*float64(len([]rune(part.text)))
			} else {
				part.shaped = fonts.ShapedText{}
				part.w = part.measure(part.text, part.fs)
			}
			if wraps(part.src) {
				part.candidateBreakOffsets = inlineBreakOffsetsFor(part.text, part.shaped)
			}
			parts = append(parts, part)
		}
		parts[len(parts)-1].spacesAfter = draft.spacesAfter
		result = append(result, parts...)
	}
	return result
}

func assignInlineParagraphBidiLevels(drafts []inlineDraft) []inlineDraft {
	if len(drafts) == 0 {
		return drafts
	}
	simpleLTR := true
	for _, draft := range drafts {
		if draft.text != "" && !isSimpleLTRText(draft.text) {
			simpleLTR = false
			break
		}
	}
	if simpleLTR {
		for index := range drafts {
			drafts[index].bidiLevel = 0
			drafts[index].bidiBaseRTL = false
			drafts[index].hasBidiLevel = true
		}
		return drafts
	}
	var text strings.Builder
	starts := make([]int, len(drafts))
	runeCount := 0
	for index, draft := range drafts {
		if index > 0 {
			previous := drafts[index-1]
			if previous.hardBreak || previous.sourceSpaceAfter || draft.sourceSpaceBefore {
				text.WriteByte(' ')
				runeCount++
			}
		}
		starts[index] = runeCount
		if draft.text != "" {
			text.WriteString(draft.text)
			runeCount += len([]rune(draft.text))
		} else if draft.image != nil {
			text.WriteRune('\uFFFC')
			runeCount++
		}
	}
	paragraph := bidi.Paragraph{}
	segments := paragraph.Segment([]rune(text.String()), bidi.Neutral)
	if segments.NumRuns() == 0 {
		return drafts
	}
	baseRTL := segments.Run(0).Level%2 == 1
	runIndex, segment := 0, segments.Run(0)
	for index := range drafts {
		for runIndex+1 < segments.NumRuns() && starts[index] >= segment.End {
			runIndex++
			segment = segments.Run(runIndex)
		}
		drafts[index].bidiLevel = segment.Level
		drafts[index].bidiBaseRTL = baseRTL
		drafts[index].hasBidiLevel = true
	}
	return drafts
}

func shapeCompatibleInlineRuns(drafts []inlineDraft) []inlineDraft {
	for start := 0; start < len(drafts); {
		first := drafts[start]
		if first.image != nil || first.font == nil || first.text == "" {
			start++
			continue
		}
		end := start + 1
		for end < len(drafts) {
			previous, next := drafts[end-1], drafts[end]
			if previous.hardBreak || previous.bidiBoundary || next.bidiBoundary || previous.image != nil || next.image != nil || next.text == "" ||
				previous.font != first.font || next.font != first.font || previous.fs != first.fs || next.fs != first.fs ||
				previous.src.Style.LetterSpacing != first.src.Style.LetterSpacing || next.src.Style.LetterSpacing != first.src.Style.LetterSpacing ||
				previous.sourceSpaceAfter || next.sourceSpaceBefore {
				break
			}
			end++
		}
		if end-start > 1 {
			var text strings.Builder
			offsets := make([]int, 0, end-start-1)
			runeOffset := 0
			for index := start; index < end; index++ {
				text.WriteString(drafts[index].text)
				runeOffset += len([]rune(drafts[index].text))
				if index+1 < end {
					offsets = append(offsets, runeOffset)
				}
			}
			shaped := first.font.Shape(text.String(), first.fs)
			parts, ok := shaped.SplitAtRuneOffsets(offsets)
			if ok {
				for index, part := range parts {
					drafts[start+index].shaped = part
					drafts[start+index].w = part.Width + first.src.Style.LetterSpacing*float64(len([]rune(drafts[start+index].text)))
					if index+1 < len(parts) {
						drafts[start+index].spacesAfter = 0
					}
				}
			}
		}
		start = end
	}
	return drafts
}

func resolveInlineStream(drafts []inlineDraft) inlineStream {
	drafts = itemizeInlineDraftsByBidi(drafts)
	drafts = assignInlineParagraphBidiLevels(drafts)
	drafts = shapeCompatibleInlineRuns(drafts)
	stream := make(inlineStream, 0, len(drafts))
	for index, draft := range drafts {
		item := draft.inlineItem
		if index > 0 {
			previous := drafts[index-1]
			if previous.hardBreak {
				item.before = inlineBoundary{}
			} else {
				item.before.advance = float64(previous.spacesAfter) * previous.sw
				item.before.spaceCount = previous.spacesAfter
			}
			if draft.sourceSpaceBefore && !previous.hardBreak {
				item.before.advance = max(item.before.advance, draft.sw)
				item.before.spaceCount = max(item.before.spaceCount, 1)
			}
			if !previous.hardBreak && lineBreakOpportunityBetween(previous, draft) {
				item.before.kind = inlineBreakOpportunity
			}
		}
		if item.image != nil {
			item.kind = inlineImage
		} else {
			item.kind = inlineText
		}
		if draft.hardStart {
			item.before.leadingAdvance = float64(draft.leadSpaces) * draft.sw
		}
		item.breakOffsets = append([]int(nil), draft.candidateBreakOffsets...)
		stream = append(stream, item)
		if draft.hardBreak {
			stream = append(stream, inlineItem{kind: inlineHardBreak, src: item.src, fs: item.fs})
		}
	}
	return stream
}

func flattenInline(b *boxes.Box, opts Options) inlineStream {
	return flattenInlineWidth(b, 0, opts)
}

func flattenInlineWidth(b *boxes.Box, maxWidth float64, opts Options) inlineStream {
	scope := &inlineScope{box: b}
	return resolveInlineStream(collectFlattenInlineDraftsWithWidth(b, maxWidth, opts, scope))
}

func collectFlattenInlineDrafts(b *boxes.Box, opts Options, scope *inlineScope) []inlineDraft {
	return collectFlattenInlineDraftsWithWidth(b, 0, opts, scope)
}

func collectFlattenInlineDraftsWithWidth(b *boxes.Box, maxWidth float64, opts Options, scope *inlineScope) []inlineDraft {
	return appendFlattenInlineDraftsInPlace(nil, b, maxWidth, opts, scope)
}

func appendFlattenInlineDraftsInPlace(out []inlineDraft, b *boxes.Box, maxWidth float64, opts Options, scope *inlineScope) []inlineDraft {
	start := len(out)
	if b.Image != nil {
		word := inlineImageDraft(b, maxWidth, opts)
		word.scope = scope
		out = append(out, word)
	} else if len(b.Parts) == 0 {
		out = appendTextDrafts(out, b, b.Text, opts)
		setInlineScope(out[start:], scope)
	} else {
		for _, p := range b.Parts {
			if p.Child == nil {
				start := len(out)
				out = appendTextDrafts(out, b, p.Text, opts)
				setInlineScope(out[start:], scope)
			} else if p.Child.Tag == "br" {
				out = appendHardBreakDraft(out, b, opts)
			} else if inlinePartChild(p.Child) {
				childScope := &inlineScope{box: p.Child, parent: scope, depth: scope.depth + 1}
				out = appendFlattenInlineDraftsInPlace(out, p.Child, maxWidth, opts, childScope)
			} else {
				start := len(out)
				out = appendTextDrafts(out, p.Child, p.Child.Text, opts)
				setInlineScope(out[start:], scope)
			}
		}
	}
	if b.Style.Display == boxes.DisplayInline && len(out) > start {
		startInset := b.Style.BorderLeft.Width + b.Style.PaddingLeft
		endInset := b.Style.BorderRight.Width + b.Style.PaddingRight
		out[start].inlineStartInset += startInset
		out[start].w += startInset
		out[len(out)-1].inlineEndInset += endInset
		out[len(out)-1].w += endInset
	}
	normalizeCJKAdjacentSpacing(out[start:])
	return out
}

func setInlineScope(words []inlineDraft, scope *inlineScope) {
	for i := range words {
		words[i].scope = scope
	}
}

func normalizeCJKAdjacentSpacing(words []inlineDraft) {
	for index := 0; index+1 < len(words); index++ {
		if !words[index].sourceSpaceAfter && !words[index+1].sourceSpaceBefore && adjacentCJKLineCharacters(words[index].text, words[index+1].text) {
			words[index].spacesAfter = 0
		}
	}
}

func inlineImageDraft(box *boxes.Box, maxWidth float64, opts Options) inlineDraft {
	width, height := rasterDimensions(box, maxWidth, 0)
	return inlineDraft{inlineItem: inlineItem{
		src: box, image: box.Image, fs: resolveSize(box, opts),
		w: width, imageH: height,
	}, spacesAfter: 1}
}

func appendTextDrafts(out []inlineDraft, src *boxes.Box, text string, opts Options) []inlineDraft {
	fs := resolveSize(src, opts)
	if src.Style.VAlign == "sub" || src.Style.VAlign == "super" {
		fs *= 0.7
	}
	measure, font := measureFor(opts, src)
	mk := func(word string) inlineDraft {
		var shaped fonts.ShapedText
		width := measure(word, fs)
		if font != nil {
			shaped = font.Shape(word, fs)
			width = shaped.Width + src.Style.LetterSpacing*float64(len([]rune(word)))
		}
		return inlineDraft{inlineItem: inlineItem{
			text: word, shaped: shaped, src: src, font: font, measure: measure, fs: fs,
			w: width,
		}, sw: measure(" ", fs)}
	}
	switch src.Style.WhiteSpace {
	case "pre", "pre-wrap", "pre-line":
		return appendPreservedTextDrafts(out, src, text, opts, mk, src.Style.WhiteSpace != "pre-line")
	default: // Empty text and nowrap: collapse whitespace.
		cursor := 0
		for _, w := range strings.Fields(text) {
			wordStart := strings.Index(text[cursor:], w) + cursor
			wd := mk(w)
			wd.spacesAfter = 1
			wd.sourceSpaceBefore = wordStart > cursor
			wordEnd := wordStart + len(w)
			if wordEnd < len(text) {
				r, _ := utf8.DecodeRuneInString(text[wordEnd:])
				wd.sourceSpaceAfter = unicode.IsSpace(r)
			}
			if wraps(src) {
				wd.candidateBreakOffsets = inlineBreakOffsetsFor(w, wd.shaped)
			}
			out = append(out, wd)
			cursor = wordEnd
		}
		return out
	}
}

// appendPreserved tokenizes while retaining spaces and newlines.
func appendPreservedTextDrafts(out []inlineDraft, src *boxes.Box, text string, opts Options, mk func(string) inlineDraft, keepSpaces bool) []inlineDraft {
	segs := strings.Split(text, "\n")
	for si, seg := range segs {
		if si > 0 && len(out) > 0 {
			out[len(out)-1].hardBreak = true
		}
		if seg == "" {
			// An empty middle segment creates a blank line, but not at the start or end.
			if si > 0 && si < len(segs)-1 {
				wd := mk("")
				wd.hardStart = true
				out = append(out, wd)
			}
			continue
		}
		words, lead := splitSpaces(seg, keepSpaces)
		for wi, word := range words {
			wd := mk(word.text)
			wd.spacesAfter = word.spaces
			if wraps(src) {
				wd.candidateBreakOffsets = inlineBreakOffsetsFor(word.text, wd.shaped)
			}
			if wi == 0 {
				wd.leadSpaces = lead
				wd.hardStart = si > 0 || len(out) == 0
			}
			out = append(out, wd)
		}
	}
	return out
}

type preWord struct {
	text   string
	spaces int
}

// splitSpaces splits text into words and counts spaces (tab = 8).
// It returns the words and leading spaces in the segment.
func splitSpaces(seg string, keep bool) ([]preWord, int) {
	var words []preWord
	lead := 0
	j := 0
	count := func() int {
		n := 0
		for j < len(seg) && (seg[j] == ' ' || seg[j] == '\t') {
			if seg[j] == '\t' {
				n += 8
			} else {
				n++
			}
			j++
		}
		return n
	}
	lead = count()
	for j < len(seg) {
		k := j
		for k < len(seg) && seg[k] != ' ' && seg[k] != '\t' && seg[k] != '\n' {
			k++
		}
		word := seg[j:k]
		j = k
		sp := count()
		if !keep {
			sp = 1
		}
		words = append(words, preWord{word, sp})
	}
	if !keep && len(words) > 0 {
		words[len(words)-1].spaces = 0
	}
	return words, lead
}

// inlineHeight sums line heights (maximum font size per line; default if empty).
func inlineHeight(lines []inlineStream, b *boxes.Box, opts Options) float64 {
	if len(lines) == 0 {
		fs := resolveSize(b, opts)
		if b.Style.LineHeightPt > 0 {
			return b.Style.LineHeightPt
		}
		if b.Style.LineHeight > 0 {
			return fs * b.Style.LineHeight
		}
		return lineHeightFor(b, fs, opts)
	}
	h := 0.0
	for _, ln := range lines {
		h += lineHeightW(ln, b, opts)
	}
	return h
}

// appendBreak adds a <br> break to the previous word; without a usable
// previous word, it adds an empty word to create a blank line.
func appendHardBreakDraft(out []inlineDraft, b *boxes.Box, opts Options) []inlineDraft {
	if len(out) > 0 && !out[len(out)-1].hardBreak {
		out[len(out)-1].hardBreak = true
		return out
	}
	fs := resolveSize(b, opts)
	measure, font := measureFor(opts, b)
	return append(out, inlineDraft{inlineItem: inlineItem{src: b, font: font, fs: fs}, sw: measure(" ", fs), hardBreak: true, hardStart: true})
}

// boxOpacity returns the element opacity (1 when unset).
func boxOpacity(b *boxes.Box) float64 {
	if b.Style.HasOpacity {
		return b.Style.Opacity
	}
	return 1
}

// runAlpha combines the color alpha (1 when absent) with opacity.
func runAlpha(color [4]float64, hasColor bool, op float64) float64 {
	if hasColor {
		return color[3] * op
	}
	return op
}

// breakInlineStream composes the resolved stream by width; wrap=false only
// respects explicit hard-breakitems.
func breakInlineStream(stream inlineStream, width float64, wrap bool) []inlineStream {
	stream = splitInlineItemsForWidth(stream, width, wrap)
	var lines []inlineStream
	var cur inlineStream
	curW := 0.0
	flush := func() {
		if len(cur) > 0 {
			lines = append(lines, cur)
			cur = nil
			curW = 0
		}
	}
	for wordIndex, w := range stream {
		if w.kind == inlineHardBreak {
			if len(cur) == 0 && len(lines) > 0 {
				cur = append(cur, w)
			} else {
				cur = append(cur, w)
			}
			flush()
			continue
		}
		add := w.w
		if len(cur) > 0 {
			add += w.before.advance
			if wrap && w.before.kind == inlineBreakOpportunity {
				groupWidth := unbreakableInlineSequenceWidth(stream, wordIndex)
				if curW+add+groupWidth-w.w > width+fragmentEpsilon && groupWidth <= width+fragmentEpsilon {
					flush()
					add = w.w
				}
			}
			// Intrinsic sizing and line breaking add inter-word advances in a
			// different order. Ignore floating-point noise at an exact fit.
			if wrap && w.before.kind == inlineBreakOpportunity && curW+add > width+fragmentEpsilon {
				flush()
				add = w.w
			}
		}
		cur = append(cur, w)
		curW += add
	}
	flush()
	return lines
}

func unbreakableInlineSequenceWidth(words inlineStream, start int) float64 {
	if start < 0 || start >= len(words) {
		return 0
	}
	width := words[start].w
	for index := start + 1; index < len(words); index++ {
		current := words[index]
		if current.kind == inlineHardBreak || current.before.kind == inlineBreakOpportunity {
			break
		}
		width += current.before.advance + current.w
	}
	return width
}

func splitInlineItemsForWidth(words inlineStream, width float64, wrap bool) inlineStream {
	if !wrap {
		return words
	}
	var breakable inlineStream
	for _, item := range words {
		if item.kind == inlineText && item.w > width && len(item.breakOffsets) > 0 {
			breakable = append(breakable, splitInlineItemAtBreaks(item)...)
			continue
		}
		breakable = append(breakable, item)
	}
	return breakable
}

// lineHeightW returns the line height: the explicit value, or the maximum font size
// by the block factor (only normal flow uses its own factor in v1).
func lineHeightW(line inlineStream, block *boxes.Box, opts Options) float64 {
	maxfs := 0.0
	maxImageHeight := 0.0
	for _, w := range line {
		if w.fs > maxfs {
			maxfs = w.fs
		}
		maxImageHeight = max(maxImageHeight, w.imageH)
	}
	textHeight := lineHeightFor(block, maxfs, opts)
	if maxImageHeight == 0 {
		return textHeight
	}
	baseline := lineBaselineOffset(block, maxfs, textHeight, opts)
	descent := max(0, textHeight-baseline)
	return max(textHeight, max(maxImageHeight, baseline)+descent)
}

func lineHeightFor(block *boxes.Box, fs float64, opts Options) float64 {
	if block.Style.LineHeightPt > 0 {
		return block.Style.LineHeightPt
	}
	if block.Style.LineHeight > 0 {
		return fs * block.Style.LineHeight
	}
	if opts.LineHeight > 0 {
		return fs * opts.LineHeight
	}
	if face := fontFor(opts, block); face != nil {
		if height := face.NormalLineHeight(fs); height > 0 {
			return height
		}
	}
	// Chromium's normal line box for Helvetica resolved to Liberation Sans is
	// 1.112em (verified by the fragmentation regression fixture). Keep the explicit line-height
	// path above authoritative; this is only the base-14 fallback.
	return fs * 1.112
}

// singleRuns wraps lines in single-style runs, one run per line.
func singleRuns(src *boxes.Box, lines []string, x, yTop, lineH, fs float64, font *fonts.Face, measure func(string, float64) float64) [][]TextRun {
	runs := make([][]TextRun, 0, len(lines))
	y := yTop
	op := boxOpacity(src)
	baselineOffset := fs
	if font != nil {
		if offset := font.BaselineOffset(fs, lineH); offset > 0 {
			baselineOffset = offset
		}
	}
	for _, ln := range lines {
		var shaped fonts.ShapedText
		width := measure(ln, fs)
		if font != nil {
			shaped = font.Shape(ln, fs)
			width = shaped.Width + src.Style.LetterSpacing*float64(len([]rune(ln)))
		}
		runs = append(runs, []TextRun{{
			Text: ln, Shaped: shaped, X: x, Y: y, BaselineOffset: baselineOffset, Width: width, LineH: lineH, FontSize: fs,
			Bold: src.Style.Bold, Italic: src.Style.Italic, Underline: src.Style.Underline, Strike: src.Style.Strike, LetterSpacing: src.Style.LetterSpacing,
			HasColor: src.Style.HasColor, Color: src.Style.Color, Link: src.Link, Font: font,
			Alpha: runAlpha(src.Style.Color, src.Style.HasColor, op), GroupOpacity: src.Style.HasOpacity,
		}})
		y += lineH
	}
	return runs
}

func inlineLineBidiPositions(line inlineStream, origin, justifyGap float64) (map[int]float64, bool) {
	simpleLTR := true
	for _, item := range line {
		if item.kind != inlineHardBreak && !isSimpleLTRText(item.text) {
			simpleLTR = false
			break
		}
	}
	if simpleLTR {
		positions := make(map[int]float64, len(line))
		x := origin
		for index, item := range line {
			if item.kind == inlineHardBreak {
				continue
			}
			positions[index] = x
			x += item.w
			if index+1 < len(line) && line[index+1].kind != inlineHardBreak {
				x += line[index+1].before.advance + justifyGap
			}
		}
		return positions, false
	}
	type lineItem struct {
		index, start int
		level        bidi.Level
	}
	var text strings.Builder
	items := make([]lineItem, 0, len(line))
	runeCount := 0
	for index, item := range line {
		if item.kind == inlineHardBreak {
			continue
		}
		if index > 0 && len(items) > 0 && item.before.advance > 0 {
			text.WriteByte(' ')
			runeCount++
		}
		start := runeCount
		text.WriteString(item.text)
		runeCount += len([]rune(item.text))
		items = append(items, lineItem{index: index, start: start})
	}
	positions := make(map[int]float64, len(items))
	if len(items) == 0 {
		return positions, false
	}
	paragraph := bidi.Paragraph{}
	runes := []rune(text.String())
	segments := paragraph.Segment(runes, bidi.Neutral)
	if segments.NumRuns() == 0 {
		x := origin
		for _, item := range items {
			positions[item.index] = x
			x += line[item.index].w
		}
		return positions, false
	}
	baseRTL := segments.Run(0).Level%2 == 1
	resolvedParagraphLevels := true
	for _, item := range items {
		if !line[item.index].hasBidiLevel {
			resolvedParagraphLevels = false
			break
		}
	}
	if resolvedParagraphLevels {
		baseRTL = line[items[0].index].bidiBaseRTL
	}
	maxLevel, minOdd := bidi.Level(0), bidi.Level(126)
	runIndex := 0
	segment := segments.Run(0)
	for itemIndex := range items {
		for runIndex+1 < segments.NumRuns() && items[itemIndex].start >= segment.End {
			runIndex++
			segment = segments.Run(runIndex)
		}
		if resolvedParagraphLevels {
			items[itemIndex].level = line[items[itemIndex].index].bidiLevel
		} else if items[itemIndex].start >= segment.Start && items[itemIndex].start < segment.End {
			items[itemIndex].level = segment.Level
		}
		if items[itemIndex].level > maxLevel {
			maxLevel = items[itemIndex].level
		}
		if items[itemIndex].level%2 == 1 && items[itemIndex].level < minOdd {
			minOdd = items[itemIndex].level
		}
	}
	order := make([]int, len(items))
	for i := range order {
		order[i] = i
	}
	if minOdd <= maxLevel {
		for level := maxLevel; level >= minOdd; level-- {
			for i := 0; i < len(order); {
				if items[order[i]].level < level {
					i++
					continue
				}
				start := i
				for i < len(order) && items[order[i]].level >= level {
					i++
				}
				for left, right := start, i-1; left < right; left, right = left+1, right-1 {
					order[left], order[right] = order[right], order[left]
				}
			}
		}
	}
	x := origin
	for visualIndex, itemIndex := range order {
		item := line[items[itemIndex].index]
		positions[items[itemIndex].index] = x
		x += item.w
		if visualIndex+1 < len(order) {
			x += justifyGap
			nextIndex := items[order[visualIndex+1]].index
			if nextIndex == items[itemIndex].index+1 {
				x += line[nextIndex].before.advance
			}
			if items[itemIndex].index == nextIndex+1 {
				x += line[items[itemIndex].index].before.advance
			}
		}
	}
	return positions, baseRTL
}

// assignRuns positions runs and returns the text for each line.
func assignRuns(block *boxes.Box, lines []inlineStream, contentX, contentW, topY float64, opts Options) [][]TextRun {
	return assignRunsWithFinalLine(block, lines, contentX, contentW, topY, opts, true)
}

func assignRunsWithFinalLine(block *boxes.Box, lines []inlineStream, contentX, contentW, topY float64, opts Options, finalFragment bool) [][]TextRun {
	var runs [][]TextRun
	y := topY
	last := len(lines) - 1
	for li, ln := range lines {
		total := 0.0
		for i, w := range ln {
			if w.kind == inlineHardBreak {
				continue
			}
			total += w.w
			if i < len(ln)-1 {
				total += ln[i+1].before.advance
			}
		}
		shift, gapX := 0.0, 0.0
		switch block.Style.Align {
		case "center":
			if extra := contentW - total; extra > 0 {
				shift = extra / 2
			}
		case "right":
			if extra := contentW - total; extra > 0 {
				shift = extra
			}
		case "justify":
			if li != last || !finalFragment {
				if gaps := len(ln) - 1; gaps > 0 {
					if extra := contentW - total; extra > 0 {
						gapX = extra / float64(gaps)
					}
				}
			}
		}
		visualX, baseRTL := inlineLineBidiPositions(ln, contentX, gapX)
		if baseRTL && block.Style.Align == "" {
			if extra := contentW - total; extra > 0 {
				shift = extra
			}
		}
		for index := range visualX {
			visualX[index] += shift
		}
		var rr []TextRun
		lh := lineHeightW(ln, block, opts)
		baseFs := resolveSize(block, opts)
		baselineBlock, baselineFs := block, baseFs
		if len(ln) > 0 && ln[0].src != nil && ln[0].src != block && ln[0].src.Style.Display == boxes.DisplayBlock {
			lineSource := ln[0].src
			allFromLineSource := true
			for _, word := range ln[1:] {
				if word.src != lineSource {
					allFromLineSource = false
					break
				}
			}
			if allFromLineSource {
				baselineBlock = lineSource
				baselineFs = resolveSize(lineSource, opts)
			}
		}
		baselineOffset := lineBaselineOffset(baselineBlock, baselineFs, lh, opts)
		imageShift := 0.0
		for _, word := range ln {
			if word.kind == inlineHardBreak {
				continue
			}
			if word.image != nil {
				imageShift = max(imageShift, word.imageH-baselineOffset)
			}
		}
		for i, w := range ln {
			if w.kind == inlineHardBreak {
				continue
			}
			x := visualX[i]
			if i == 0 {
				x += w.before.leadingAdvance
			}
			if w.image != nil {
				imageY := y + max(0, imageShift) + baselineOffset - w.imageH
				switch w.src.Style.VAlign {
				case "top":
					imageY = y
				case "middle":
					imageY = y + (lh-w.imageH)/2
				case "bottom":
					imageY = y + lh - w.imageH
				}
				rr = append(rr, TextRun{
					Image: w.image, ImageHeight: w.imageH, X: x, Y: imageY, Width: w.w,
					LineH: lh, LineY: y, FontSize: w.fs, Alpha: boxOpacity(w.src), GroupOpacity: w.src.Style.HasOpacity, source: w.src,
				})
				continue
			}
			s := w.src
			runX := x + w.inlineStartInset
			runWidth := math.Max(0, w.w-w.inlineStartInset-w.inlineEndInset)
			// Vertical offset in top-down coordinates: compensate for the
			// shrinkage (base-fs) plus the spacing adjustment in em units of the base.
			ry := y + max(0, imageShift)
			switch s.Style.VAlign {
			case "super":
				ry += (baseFs - w.fs) - 0.33*baseFs
			case "sub":
				ry += (baseFs - w.fs) + 0.2*baseFs
			}
			rr = append(rr, TextRun{
				Text: w.text, Shaped: w.shaped, X: runX, Y: ry, BaselineOffset: baselineOffset, Width: runWidth, LineH: lh, FontSize: w.fs,
				Bold: s.Style.Bold, Italic: s.Style.Italic, Underline: s.Style.Underline, Strike: s.Style.Strike, LetterSpacing: s.Style.LetterSpacing,
				HasColor: s.Style.HasColor, Color: s.Style.Color, Link: s.Link, Font: w.font,
				Alpha: runAlpha(s.Style.Color, s.Style.HasColor, boxOpacity(s)), GroupOpacity: s == block && block.Style.HasOpacity, ListMarkerShape: w.markerShape,
				LineY: y, source: s,
			})
		}
		appendInlineBoxDecorations(block, ln, rr, y, lh, opts)
		runs = append(runs, rr)
		y += lineHeightW(ln, block, opts)
	}
	return runs
}

func appendInlineBoxDecorations(block *boxes.Box, words inlineStream, runs []TextRun, y, lineHeight float64, opts Options) {
	if len(runs) == 0 || len(words) == 0 {
		return
	}
	var candidates []*boxes.Box
	var collect func(*boxes.Box)
	collect = func(parent *boxes.Box) {
		for _, child := range parent.Children {
			if child.Style.Display == boxes.DisplayInline && hasInlineBoxPaint(child) {
				candidates = append(candidates, child)
			}
			collect(child)
		}
	}
	collect(block)
	for _, candidate := range candidates {
		first, last := -1, -1
		for index, run := range runs {
			if inlineSourceWithin(run.source, candidate) {
				if first < 0 {
					first = index
				}
				last = index
			}
		}
		if first < 0 {
			continue
		}
		candidateWords := buildInlineStream(candidate, opts)
		starts, ends := false, false
		for _, word := range candidateWords {
			if word.text != "" {
				starts = strings.HasPrefix(runs[first].Text, word.text)
				break
			}
		}
		for index := len(candidateWords) - 1; index >= 0; index-- {
			if candidateWords[index].text != "" {
				ends = strings.HasSuffix(runs[last].Text, candidateWords[index].text)
				break
			}
		}
		leftInset, rightInset := candidate.Style.PaddingLeft+candidate.Style.BorderLeft.Width, candidate.Style.PaddingRight+candidate.Style.BorderRight.Width
		topInset, bottomInset := candidate.Style.PaddingTop+candidate.Style.BorderTop.Width, candidate.Style.PaddingBottom+candidate.Style.BorderBottom.Width
		left, right := runs[first].X, runs[last].X+runs[last].Width
		if starts {
			left -= leftInset
		}
		if ends {
			right += rightInset
		}
		decoration := placed(candidate, "", nil, left, y-topInset, math.Max(0, right-left), lineHeight+topInset+bottomInset, resolveSize(candidate, opts), fontFor(opts, candidate))
		decoration.BorderWidth = 0
		if !starts {
			decoration.BorderLeft.Width = 0
		}
		if !ends {
			decoration.BorderRight.Width = 0
		}
		runs[first].InlineDecorations = append(runs[first].InlineDecorations, InlineBoxDecoration{
			Box: decoration, OffsetX: left - runs[first].X, OffsetY: -topInset,
		})
	}
}

func hasInlineBoxPaint(box *boxes.Box) bool {
	return box.Style.HasBg || box.Style.BorderWidth > 0 || box.Style.BorderTop.Width > 0 || box.Style.BorderRight.Width > 0 || box.Style.BorderBottom.Width > 0 || box.Style.BorderLeft.Width > 0
}

func inlineSourceWithin(source, ancestor *boxes.Box) bool {
	if source == nil || ancestor == nil {
		return false
	}
	if source == ancestor {
		return true
	}
	ancestorPath, sourcePath := ancestor.SourcePath(), source.SourcePath()
	if len(ancestorPath) == 0 || len(sourcePath) <= len(ancestorPath) {
		return false
	}
	for index, part := range ancestorPath {
		if sourcePath[index] != part {
			return false
		}
	}
	return true
}

func lineBaselineOffset(block *boxes.Box, fs, lineHeight float64, opts Options) float64 {
	if face := fontFor(opts, block); face != nil {
		if offset := face.BaselineOffset(fs, lineHeight); offset > 0 {
			return offset
		}
	}
	return fs
}
