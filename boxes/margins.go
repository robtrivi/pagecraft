package boxes

import "strings"

func collapseSingleChildMargins(parent *Box) {
	if parent.Style.Display != DisplayBlock || len(parent.Children) != 1 || parent.Children[0].Style.Display != DisplayBlock {
		return
	}
	for _, part := range parent.Parts {
		if part.Child == nil && strings.TrimSpace(part.Text) != "" {
			return
		}
	}
	child := parent.Children[0]
	if parent.Style.PaddingTop == 0 && parent.Style.BorderTop.Width == 0 {
		parent.Style.MarginTop = collapseMarginPair(parent.Style.MarginTop, child.Style.MarginTop)
	}
	if parent.Style.PaddingBottom == 0 && parent.Style.BorderBottom.Width == 0 {
		parent.Style.MarginBottom = collapseMarginPair(parent.Style.MarginBottom, child.Style.MarginBottom)
	}
}

func collapseMarginPair(a, b float64) float64 {
	positive := 0.0
	if a > positive {
		positive = a
	}
	if b > positive {
		positive = b
	}
	negative := 0.0
	if a < negative {
		negative = a
	}
	if b < negative {
		negative = b
	}
	return positive + negative
}

// resolveImage resolves <img> sources: data URI, local or relative file, or
// remote (downloaded). Without src, alt is used; a failed remote load returns
// alt text and a warning.
