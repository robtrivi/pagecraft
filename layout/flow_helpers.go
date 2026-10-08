package layout

import (
	"strings"

	"github.com/robtrivi/pagecraft-go/boxes"
)

func shiftPlacedBox(block *PlacedBox, dx, dy float64) {
	block.X += dx
	block.Y += dy
	for line := range block.Runs {
		for run := range block.Runs[line] {
			block.Runs[line][run].X += dx
			block.Runs[line][run].Y += dy
		}
	}
	shiftChildDecorations(block.fragmentNode, dx, dy)
}

func shiftChildDecorations(node *FragmentNode, dx, dy float64) {
	if node == nil {
		return
	}
	for _, child := range node.Children {
		if child.Kind != FragmentDecoration {
			continue
		}
		child.Placed.X += dx
		child.Placed.Y += dy
		for line := range child.Placed.Runs {
			for run := range child.Placed.Runs[line] {
				child.Placed.Runs[line][run].X += dx
				child.Placed.Runs[line][run].Y += dy
			}
		}
		shiftChildDecorations(child, dx, dy)
	}
}

func hasAbsoluteDescendant(b *boxes.Box) bool {
	for _, child := range b.Children {
		if child.Style.Position == boxes.PositionAbsolute || hasAbsoluteDescendant(child) {
			return true
		}
	}
	return false
}

func ensureOpacityGroups(block *PlacedBox, groups []OpacityGroup, sequence *int) {
	if block.HasOpacity {
		if sequence == nil {
			localSequence := 0
			sequence = &localSequence
		}
		(*sequence)++
		groups = append(append([]OpacityGroup(nil), groups...), OpacityGroup{ID: *sequence, Opacity: block.Opacity})
		block.HasOpacity = false
	}
	prefixMatches := len(block.OpacityGroups) >= len(groups)
	for i := range groups {
		if !prefixMatches || block.OpacityGroups[i].ID != groups[i].ID {
			prefixMatches = false
			break
		}
	}
	if !prefixMatches {
		block.OpacityGroups = append(append([]OpacityGroup(nil), groups...), block.OpacityGroups...)
	}
}

// normalFlowItems keeps nested blocks as separate boxes in transparent
// containers; decorated containers are handled by flowNestedContainer.
func normalFlowItems(b *boxes.Box) []*boxes.Box {
	if b.Style.Display != boxes.DisplayBlock || b.Style.HasBg || b.Image != nil || b.SVG != nil ||
		b.Style.HasOpacity || clipsOverflow(b) ||
		b.Style.PaddingTop != 0 || b.Style.PaddingRight != 0 || b.Style.PaddingBottom != 0 || b.Style.PaddingLeft != 0 ||
		b.Style.BorderTop.Width != 0 || b.Style.BorderRight.Width != 0 || b.Style.BorderBottom.Width != 0 || b.Style.BorderLeft.Width != 0 ||
		b.Tag == "table" || b.Tag == "ul" || b.Tag == "ol" || len(b.Children) == 0 {
		return []*boxes.Box{b}
	}
	if hasInlineContent(b) && hasBlockChild(b) {
		if b.Style.MarginLeft != 0 || b.Style.MarginRight != 0 {
			return []*boxes.Box{b}
		}
		return collapseWrapperMargins(b, nestedContainerItems(b))
	}
	for _, part := range b.Parts {
		if part.Child == nil && strings.TrimSpace(part.Text) != "" {
			return []*boxes.Box{b}
		}
	}
	var items []*boxes.Box
	for _, child := range b.Children {
		if child.Style.Float != boxes.FloatNone || child.Style.Position != boxes.PositionStatic {
			return []*boxes.Box{b}
		}
		if child.Style.Display == boxes.DisplayFlex || child.Style.Display == boxes.DisplayGrid || child.Style.ColCount > 1 ||
			child.Tag == "table" || child.Tag == "ul" || child.Tag == "ol" {
			items = append(items, child)
			continue
		}
		if child.Style.Display != boxes.DisplayBlock {
			return []*boxes.Box{b}
		}
		items = append(items, normalFlowItems(child)...)
	}
	if len(items) == 0 {
		return []*boxes.Box{b}
	}
	return collapseWrapperMargins(b, items)
}

func hasBlockChild(b *boxes.Box) bool {
	for _, child := range b.Children {
		if child.Style.Display == boxes.DisplayBlock && !inlinePartChild(child) {
			return true
		}
	}
	return false
}

func inlinePartChild(child *boxes.Box) bool {
	if child.Tag == "br" {
		return true
	}
	if child.Style.Display == boxes.DisplayBlock || child.Style.Display == boxes.DisplayFlex || child.Style.Display == boxes.DisplayGrid {
		return false
	}
	if child.Style.Display == boxes.DisplayInline {
		return true
	}
	switch child.Tag {
	case "a", "b", "code", "of-the", "em", "i", "s", "small", "strong", "sub", "sup", "strike", "u":
		return true
	default:
		return false
	}
}

func collapseWrapperMargins(b *boxes.Box, items []*boxes.Box) []*boxes.Box {
	if len(items) == 0 {
		return []*boxes.Box{b}
	}
	first := *items[0]
	first.Style.MarginTop = collapseMargins(b.Style.MarginTop, first.Style.MarginTop)
	items[0] = &first
	lastIndex := len(items) - 1
	if lastIndex == 0 {
		items[0].Style.MarginBottom = collapseMargins(b.Style.MarginBottom, items[0].Style.MarginBottom)
	} else {
		last := *items[lastIndex]
		last.Style.MarginBottom = collapseMargins(b.Style.MarginBottom, last.Style.MarginBottom)
		items[lastIndex] = &last
	}
	return items
}

// collapseMargins combines adjacent CSS margins: the largest positive plus
// the smallest negative; if all are negative, the most negative wins.
func collapseMargins(a, b float64) float64 {
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

func blockTopMarginGap(previousBottom float64, hasPrevious bool, bodyMargin, currentTop float64) float64 {
	if hasPrevious {
		return collapseMargins(previousBottom, currentTop) - previousBottom
	}
	if bodyMargin != 0 {
		return collapseMargins(bodyMargin, currentTop) - bodyMargin
	}
	return currentTop
}

// placeFixed places a fixed box without changing the cursor (text or image).
