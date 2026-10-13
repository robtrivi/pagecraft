package layout

import (
	"math"

	"github.com/robtrivi/pagecraft/boxes"
	"github.com/robtrivi/pagecraft/css"
)

// sizeBasis carries the containing block size together with whether CSS permits
// percentage resolution against it. Zero is a valid definite size.
type sizeBasis struct {
	Value    float64
	Definite bool
}

// sizeContext keeps the logical inline and block bases distinct while layout
// moves through nested formatting contexts.
type sizeContext struct {
	Inline sizeBasis
	Block  sizeBasis
}

func definiteSize(value float64) sizeBasis { return sizeBasis{Value: value, Definite: true} }

func placeFixed(fb *boxes.Box, opts Options, usable float64) PlacedBox {
	fs := resolveSize(fb, opts)
	_, font := measureFor(opts, fb)
	x := opts.pageAreaLeftMargin() + fb.Style.Left
	yy := opts.pageAreaTopMargin() + fb.Style.Top
	if fb.Image != nil {
		w, h := rasterDimensions(fb, usable, 0)
		if fb.Style.HasRight {
			x = opts.PageWidth - opts.pageAreaRightMargin() - fb.Style.Right - w
		}
		if fb.Style.HasBottom {
			yy = opts.PageHeight - opts.pageAreaBottomMargin() - fb.Style.Bottom - h
		}
		pb := placed(fb, fb.Text, fb.Image, x, yy, w, h, fs, font)
		return pb
	}
	width := usedWidth(fb, usable)
	contentW := width - horizontalInsets(fb)
	if contentW < 0 {
		contentW = 0
	}
	words := buildInlineStream(fb, opts)
	linesW := breakInlineStream(words, contentW, wraps(fb))
	h := usedHeight(fb, inlineHeight(linesW, fb, opts), opts)
	if fb.Style.HasRight {
		x = opts.PageWidth - opts.pageAreaRightMargin() - fb.Style.Right - width
	}
	if fb.Style.HasBottom {
		yy = opts.PageHeight - opts.pageAreaBottomMargin() - fb.Style.Bottom - h
	}
	runs := assignRuns(fb, linesW, x+fb.Style.BorderLeft.Width+fb.Style.PaddingLeft, contentW, yy+fb.Style.BorderTop.Width+fb.Style.PaddingTop, opts)
	pb := placed(fb, fb.Text, nil, x, yy, width, h, fs, font)
	pb.Runs = runs
	return pb
}

func rasterDimensions(box *boxes.Box, maxWidth, maxHeight float64) (float64, float64) {
	image := box.Image
	intrinsicWidth := float64(image.Width) * css.PxToPt
	intrinsicHeight := float64(image.Height) * css.PxToPt
	width, hasWidth := box.Style.ImageWidth, box.Style.ImageWidth > 0
	height, hasHeight := box.Style.ImageHeight, box.Style.ImageHeight > 0
	if !hasWidth && box.Style.ImageWidthPct > 0 && maxWidth > 0 {
		width, hasWidth = maxWidth*box.Style.ImageWidthPct/100, true
	}
	if !hasHeight && box.Style.ImageHeightPct > 0 && maxHeight > 0 {
		height, hasHeight = maxHeight*box.Style.ImageHeightPct/100, true
	}
	switch {
	case hasWidth && hasHeight:
		return width, height
	case hasWidth:
		return width, width * intrinsicHeight / intrinsicWidth
	case hasHeight:
		return height * intrinsicWidth / intrinsicHeight, height
	default:
		scale := 1.0
		if maxWidth > 0 && intrinsicWidth > maxWidth {
			scale = maxWidth / intrinsicWidth
		}
		if maxHeight > 0 && intrinsicHeight*scale > maxHeight {
			scale = maxHeight / intrinsicHeight
		}
		return intrinsicWidth * scale, intrinsicHeight * scale
	}
}

// wraps reports whether the block wraps lines (anything except pre and nowrap).
func wraps(b *boxes.Box) bool {
	return b.Style.WhiteSpace != "pre" && b.Style.WhiteSpace != "nowrap"
}

// activeFloat is a float that narrows flow across its vertical range.
type activeFloat struct {
	left        bool
	x, w        float64
	top, bottom float64
	page        int
}

// liChildren returns direct <li> children; lists flow item by item, not as a block.
func liChildren(list *boxes.Box) []*boxes.Box {
	var out []*boxes.Box
	var visit func(*boxes.Box, int)
	visit = func(parent *boxes.Box, depth int) {
		for _, child := range parent.Children {
			if child.Tag != "li" {
				continue
			}
			item := child
			if depth > 0 {
				copy := *child
				copy.Style.PaddingLeft += 18 * float64(depth)
				item = &copy
			}
			out = append(out, item)
			for _, nested := range child.Children {
				if nested.Tag == "ul" || nested.Tag == "ol" {
					visit(nested, depth+1)
				}
			}
		}
	}
	visit(list, 0)
	return out
}

// isKeepable reports whether a block participates in keep-with-next (plain text).
func isKeepable(b *boxes.Box) bool {
	if b.Image != nil || b.Style.Float != boxes.FloatNone || b.Style.Position == boxes.PositionAbsolute {
		return false
	}
	if b.Style.Display == boxes.DisplayFlex || b.Style.Display == boxes.DisplayGrid || b.Style.ColCount > 1 {
		return false
	}
	return b.Tag != "table" && b.Tag != "ul" && b.Tag != "ol"
}

// floatsOnPage filters floats on the current page; v1 does not carry them across pages.
func floatsOnPage(floats []activeFloat, page int) []activeFloat {
	var out []activeFloat
	for _, f := range floats {
		if f.page == page {
			out = append(out, f)
		}
	}
	return out
}

// wrapPlacementW chooses placement around active floats using word
// (redistribute when narrowed). Without overlap, return the full width unchanged.
func usedWidth(b *boxes.Box, available float64) float64 {
	insets := horizontalInsets(b)
	basis := definiteSize(available)
	width, resolved := resolveBorderBoxDimension(b.Style.Sizing.Width, basis, insets, b.Style.BoxSizing)
	if !resolved {
		width = available
	}
	return clampUsedDimension(width, b.Style.Sizing.MinWidth, b.Style.Sizing.MaxWidth, basis, insets, b.Style.BoxSizing)
}

// resolveHorizontalAutoMargins applies the CSS block rule: free space
// remaining containing-block width is assigned to the auto horizontal margins.
func resolveHorizontalAutoMargins(b *boxes.Box, available float64) *boxes.Box {
	if b == nil || b.Style.Float != boxes.FloatNone || b.Style.Position != boxes.PositionStatic || (!b.Style.MarginAutoLeft && !b.Style.MarginAutoRight) {
		return b
	}
	width := usedWidth(b, math.Max(0, available-b.Style.MarginLeft-b.Style.MarginRight))
	return resolveHorizontalAutoMarginsWithWidth(b, available, width)
}

func resolveHorizontalAutoMarginsWithWidth(b *boxes.Box, available, width float64) *boxes.Box {
	if b == nil || (!b.Style.MarginAutoLeft && !b.Style.MarginAutoRight) {
		return b
	}
	resolved := *b
	free := math.Max(0, available-b.Style.MarginLeft-b.Style.MarginRight-width)
	switch {
	case b.Style.MarginAutoLeft && b.Style.MarginAutoRight:
		resolved.Style.MarginLeft += free / 2
		resolved.Style.MarginRight += free / 2
	case b.Style.MarginAutoLeft:
		resolved.Style.MarginLeft += free
	case b.Style.MarginAutoRight:
		resolved.Style.MarginRight += free
	}
	resolved.Style.MarginAutoLeft, resolved.Style.MarginAutoRight = false, false
	return &resolved
}

func resolvedDeclaredHeight(b *boxes.Box, opts Options) (float64, bool) {
	return resolveDimension(b.Style.Sizing.Height, opts.space.containingSize.Block)
}

func resolveDimension(dimension boxes.CSSDimension, basis sizeBasis) (float64, bool) {
	switch dimension.Kind {
	case boxes.DimensionLength:
		return dimension.Value, true
	case boxes.DimensionPercentage:
		if basis.Definite {
			return dimension.Value * basis.Value / 100, true
		}
	}
	return 0, false
}

func resolveBorderBoxDimension(dimension boxes.CSSDimension, basis sizeBasis, insets float64, boxSizing boxes.BoxSizing) (float64, bool) {
	value, resolved := resolveDimension(dimension, basis)
	if !resolved {
		return 0, false
	}
	if boxSizing != boxes.BoxSizingBorderBox {
		value += insets
	}
	return value, true
}

func clampUsedDimension(value float64, minimum, maximum boxes.CSSDimension, basis sizeBasis, insets float64, boxSizing boxes.BoxSizing) float64 {
	minValue, hasMin := resolveBorderBoxDimension(minimum, basis, insets, boxSizing)
	maxValue, hasMax := resolveBorderBoxDimension(maximum, basis, insets, boxSizing)
	if hasMax && value > maxValue {
		value = maxValue
	}
	if hasMin && value < minValue {
		value = minValue
	}
	return math.Max(0, value)
}

func hasDeclaredHeight(b *boxes.Box, opts Options) bool {
	_, ok := resolvedDeclaredHeight(b, opts)
	return ok
}

func usedDeclaredHeight(b *boxes.Box, opts Options) float64 {
	basis := opts.space.containingSize.Block
	declared, ok := resolveBorderBoxDimension(b.Style.Sizing.Height, basis, verticalInsets(b), b.Style.BoxSizing)
	if !ok {
		return 0
	}
	return clampUsedDimension(declared, b.Style.Sizing.MinHeight, b.Style.Sizing.MaxHeight, basis, verticalInsets(b), b.Style.BoxSizing)
}

func usedHeight(b *boxes.Box, contentHeight float64, opts Options) float64 {
	height := contentHeight + verticalInsets(b)
	if hasDeclaredHeight(b, opts) && usedDeclaredHeight(b, opts) > height {
		height = usedDeclaredHeight(b, opts)
	}
	return clampUsedDimension(height, b.Style.Sizing.MinHeight, b.Style.Sizing.MaxHeight, opts.space.containingSize.Block, verticalInsets(b), b.Style.BoxSizing)
}
