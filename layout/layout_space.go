package layout

// layoutSpace carries geometric state for one layout invocation. Services and
// document-wide configuration remain in Options; nested flows derive a value.
type layoutSpace struct {
	pageMargin              float64
	pageMarginSet           bool
	containingSize          sizeContext
	startY                  float64
	startYSet               bool
	marginLeft              float64
	marginRight             float64
	marginTop               float64
	marginBottom            float64
	absoluteOriginX         float64
	absoluteOriginY         float64
	absoluteOriginSet       bool
	absoluteContainingWidth float64
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
	space.pageMarginSet = set
	return space
}

func (space layoutSpace) withAbsoluteContainingBlock(x, y, width float64) layoutSpace {
	space.absoluteOriginX = x
	space.absoluteOriginY = y
	space.absoluteOriginSet = true
	space.absoluteContainingWidth = width
	return space
}
