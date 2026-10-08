package layout

import (
	"math"

	"github.com/robtrivi/pagecraft-go/boxes"
)

func overflowClipForBox(box *boxes.Box, x, y, width, height float64, opts Options, fixedHeight bool) *ClipRect {
	if !clipsOverflow(box) {
		return nil
	}
	clip := ClipRect{Width: opts.PageWidth, Height: opts.PageHeight}
	if clipsOverflowX(box) {
		clip.X = x + box.Style.BorderLeft.Width
		clip.Width = math.Max(0, width-box.Style.BorderLeft.Width-box.Style.BorderRight.Width)
	}
	if clipsOverflowY(box) && fixedHeight {
		clip.Y = y + box.Style.BorderTop.Width
		clip.Height = height - box.Style.BorderTop.Width - box.Style.BorderBottom.Width
		if clip.Height < 0 {
			clip.Height = 0
		}
	}
	return &clip
}
