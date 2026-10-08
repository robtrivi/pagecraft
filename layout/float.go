package layout

import (
	"math"

	"github.com/robtrivi/pagecraft-go/boxes"
)

// Float wrapping (v1): a float is placed at the margin without advancing the
// cursor (as in CSS); following content flows beside or below it through
// wrapPlacement. Floats stackvertically (drop in Flow).

// flowFloat places a float and returns it as active for wrapping.
func flowFloat(f *boxes.Box, y float64, pages *[]Page, opts Options, usable, bottom float64) activeFloat {
	if len(f.Children) > 0 && (hasBlockChild(f) || f.Image != nil || f.SVG != nil) {
		inset, _ := nestedContainerInsets(f)
		width := usedWidth(f, usable)
		if width > usable {
			width = usable
		}
		outerTop := y + f.Style.MarginTop
		floatOpts := opts
		remaining := usable - width - f.Style.MarginLeft - f.Style.MarginRight
		if remaining < 0 {
			remaining = 0
		}
		if f.Style.Float == boxes.FloatRight {
			floatOpts.space.marginLeft += remaining
		} else {
			floatOpts.space.marginRight += remaining
		}
		items := nestedContainerItems(f)
		end := flowNestedContainer(f, outerTop, inset, items, false, pages, floatOpts, width)
		x := opts.left() + f.Style.MarginLeft
		if f.Style.Float == boxes.FloatRight {
			x = opts.PageWidth - opts.right() - f.Style.MarginRight - width
		}
		return activeFloat{left: f.Style.Float == boxes.FloatLeft, x: x, w: width + f.Style.MarginLeft + f.Style.MarginRight, top: outerTop, bottom: end - f.Style.MarginBottom, page: len(*pages) - 1}
	}
	fs := resolveSize(f, opts)
	_, font := measureFor(opts, f)
	available := math.Max(0, usable-f.Style.MarginLeft-f.Style.MarginRight)
	width := usedWidth(f, available)
	if f.Style.Sizing.Width.IsAuto() {
		words := buildInlineStream(f, opts)
		maxLineWidth := 0.0
		for _, line := range breakInlineStream(words, math.MaxFloat64, false) {
			lineWidth := 0.0
			for _, word := range line {
				lineWidth += word.before.advance
				lineWidth += word.w
			}
			maxLineWidth = math.Max(maxLineWidth, lineWidth)
		}
		width = math.Min(available, maxLineWidth+horizontalInsets(f))
	}
	contentWidth := math.Max(0, width-horizontalInsets(f))
	words := buildInlineStream(f, opts)
	y += f.Style.MarginTop
	x := opts.left()
	if f.Style.Float == boxes.FloatRight {
		x = opts.PageWidth - opts.right() - width - f.Style.MarginRight
	} else {
		x += f.Style.MarginLeft
	}
	contentX := x + f.Style.BorderLeft.Width + f.Style.PaddingLeft
	contentY := y + f.Style.BorderTop.Width + f.Style.PaddingTop
	lines := layoutInlineLines(f, words, contentX, contentWidth, contentY, nil, len(*pages)-1, opts)
	height := usedHeight(f, inlineLinesHeight(lines), opts)
	if y+height > bottom && (*pages)[len(*pages)-1].blockCount() > 0 {
		*pages = append(*pages, Page{Number: len(*pages) + 1})
		y = opts.top() + f.Style.MarginTop
		contentY = y + f.Style.BorderTop.Width + f.Style.PaddingTop
		lines = layoutInlineLines(f, words, contentX, contentWidth, contentY, nil, len(*pages)-1, opts)
		height = usedHeight(f, inlineLinesHeight(lines), opts)
	}
	p := &(*pages)[len(*pages)-1]
	pb := placed(f, f.Text, nil, x, y, width, height, fs, font)
	pb.Runs = inlineLineRuns(f, lines, opts)
	p.emitBlock(pb)
	return activeFloat{left: f.Style.Float == boxes.FloatLeft, x: x - f.Style.MarginLeft, w: width + f.Style.MarginLeft + f.Style.MarginRight, top: y - f.Style.MarginTop, bottom: y + height + f.Style.MarginBottom, page: len(*pages) - 1}
}

// dropBelowFloats moves the cursor below active floats (vertical stacking in v1).
func dropBelowFloats(floats []activeFloat, page int, y float64) float64 {
	for _, f := range floats {
		if f.page == page && f.bottom > y {
			y = f.bottom
		}
	}
	return y
}

func clearanceY(floats []activeFloat, page int, clear boxes.ClearSide, y float64) float64 {
	for _, fl := range floats {
		if fl.page != page || (clear == boxes.ClearLeft && !fl.left) || (clear == boxes.ClearRight && fl.left) {
			continue
		}
		if fl.bottom > y {
			y = fl.bottom
		}
	}
	return y
}

// placeFormattingContextAroundFloats gives tables and other independent block
// formatting contexts the free rectangle beside active floats. Explicitly
// sized contexts that do not fit move below those floats, as Chromium does.
func placeFormattingContextAroundFloats(b *boxes.Box, y float64, floats []activeFloat, page int, opts Options, usable float64) (float64, Options, float64) {
	leftInset, rightInset := 0.0, 0.0
	contentLeft, contentRight := opts.left(), opts.PageWidth-opts.right()
	for _, fl := range floats {
		if fl.page != page || fl.top >= y+fragmentEpsilon || fl.bottom <= y+fragmentEpsilon {
			continue
		}
		if fl.left {
			leftInset = math.Max(leftInset, fl.x+fl.w-contentLeft)
		} else {
			rightInset = math.Max(rightInset, contentRight-fl.x)
		}
	}
	if leftInset <= fragmentEpsilon && rightInset <= fragmentEpsilon {
		return y, opts, usable
	}
	free := math.Max(0, usable-leftInset-rightInset)
	width := usedWidth(b, math.Max(0, usable-b.Style.MarginLeft-b.Style.MarginRight))
	if !b.Style.Sizing.Width.IsAuto() && width > free+fragmentEpsilon {
		return dropBelowFloats(floats, page, y), opts, usable
	}
	adjusted := opts
	adjusted.space.marginLeft += leftInset
	adjusted.space.marginRight += rightInset
	return y, adjusted, free
}
