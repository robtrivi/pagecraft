package layout

import (
	"github.com/robtrivi/pagecraft/boxes"
)

// Absolute positioning resolves offsets against the current containing block,
// outside normal flow (it does not move the cursor).

// flowAbsolute places the box and returns the cursor unchanged.
func flowAbsolute(f *boxes.Box, y float64, pages *[]Page, opts Options, usable float64) float64 {
	fs := resolveSize(f, opts)
	measure, font := measureFor(opts, f)
	p := &(*pages)[len(*pages)-1]
	x, originY := opts.left(), opts.top()
	containingWidth := usable
	containingHeight := opts.pageBottom() - opts.top()
	if opts.space.absoluteOriginSet {
		x, originY = opts.space.absoluteOriginX, opts.space.absoluteOriginY
		if opts.space.absoluteContainingWidth > 0 {
			containingWidth = opts.space.absoluteContainingWidth
		}
		containingHeight = opts.space.absoluteContainingHeight
	}
	if f.Image != nil {
		width, height := rasterDimensions(f, containingWidth, 0)
		if f.Style.HasRight {
			x += containingWidth - f.Style.Right - width
		} else {
			x += f.Style.Left
		}
		top := absoluteBlockTop(f, originY, containingHeight, height)
		p.emitBlock(placed(f, f.Text, f.Image, x, top, width, height, fs, font))
		return y
	}
	width := usedWidth(f, containingWidth)
	contentWidth := width - horizontalInsets(f)
	if contentWidth < 0 {
		contentWidth = 0
	}
	lines := BreakLines(f.Text, contentWidth, measure, fs)
	x += f.Style.Left
	if f.Style.HasRight {
		x += containingWidth - f.Style.Right - width
	}
	lineH := lineHeightFor(f, fs, opts)
	height := float64(len(lines))*lineH + verticalInsets(f)
	if hasDeclaredHeight(f, opts) {
		height = usedDeclaredHeight(f, opts)
	}
	top := absoluteBlockTop(f, originY, containingHeight, height)
	pb := placed(f, f.Text,
		nil, x, top, width, height, fs, font)
	contentX := x + f.Style.BorderLeft.Width + f.Style.PaddingLeft
	contentY := top + f.Style.BorderTop.Width + f.Style.PaddingTop
	pb.Runs = singleRuns(f, lines, contentX, contentY, lineH, fs, font, measure)
	p.emitBlock(pb)
	return y
}

func absoluteBlockTop(box *boxes.Box, originY, containingHeight, height float64) float64 {
	if box.Style.HasBottom && !box.Style.HasTop && containingHeight > 0 {
		return originY + containingHeight - box.Style.Bottom - height
	}
	return originY + box.Style.Top
}
