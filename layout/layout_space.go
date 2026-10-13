package layout

// layoutSpace carries geometric state for one layout invocation. Services and
// document-wide configuration remain in Options; nested flows derive a value.
type layoutSpace struct {
	pageMargin               float64 // legacy scalar retained for old internal callers
	pageMarginLeft           float64
	pageMarginRight          float64
	pageMarginTop            float64
	pageMarginBottom         float64
	pageMarginSet            bool
	bodyMargin               float64
	containingSize           sizeContext
	startY                   float64
	startYSet                bool
	marginLeft               float64
	marginRight              float64
	marginTop                float64
	marginBottom             float64
	absoluteOriginX          float64
	absoluteOriginY          float64
	absoluteOriginSet        bool
	absoluteContainingWidth  float64
	absoluteContainingHeight float64
}

func (space layoutSpace) withContainingBlock(size sizeContext) layoutSpace {
	space.containingSize = size
	return space
}

func (space layoutSpace) withMargins(left, right, top, bottom float64) layoutSpace {
	space.marginLeft = left
	space.marginRight = right
	space.marginTop = top
	space.marginBottom = bottom
	return space
}

func (space layoutSpace) withStartY(y float64) layoutSpace {
	space.startY = y
	space.startYSet = true
	return space
}

func (space layoutSpace) withPageMargin(margin float64, set bool) layoutSpace {
	space.pageMargin = margin
	space.pageMarginLeft, space.pageMarginRight = margin, margin
	space.pageMarginTop, space.pageMarginBottom = margin, margin
	space.pageMarginSet = set
	return space
}

func (space layoutSpace) withPageMargins(left, right, top, bottom float64) layoutSpace {
	space.pageMarginLeft, space.pageMarginRight = left, right
	space.pageMarginTop, space.pageMarginBottom = top, bottom
	space.pageMargin = left
	space.pageMarginSet = true
	return space
}

func (space layoutSpace) withBodyMargin(margin float64) layoutSpace {
	space.bodyMargin = margin
	return space
}

func (space layoutSpace) withAbsoluteContainingBlock(x, y, width, height float64) layoutSpace {
	space.absoluteOriginX = x
	space.absoluteOriginY = y
	space.absoluteOriginSet = true
	space.absoluteContainingWidth = width
	space.absoluteContainingHeight = height
	return space
}
