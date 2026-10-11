package layout

import (
	"math"
	"strings"

	"github.com/robtrivi/pagecraft/boxes"
)

func unboundedFragmentainerPageHeight(pageHeight float64) float64 {
	return math.Max(pageHeight*1000, 100000)
}

func nestedContainerInsets(b *boxes.Box) (containerInsets, bool) {
	decorated := b.Style.HasOpacity || b.Style.HasBg || b.Style.BorderWidth > 0 || b.Style.PaddingTop != 0 || b.Style.PaddingRight != 0 || b.Style.PaddingBottom != 0 || b.Style.PaddingLeft != 0 ||
		b.Style.BorderTop.Width > 0 || b.Style.BorderRight.Width > 0 || b.Style.BorderBottom.Width > 0 || b.Style.BorderLeft.Width > 0
	decorated = decorated || clipsOverflow(b)
	decorated = decorated || hasHeightSizing(b)
	decorated = decorated || needsHeightContainingBlock(b)
	decorated = decorated || b.Style.Position == boxes.PositionRelative && hasAbsoluteDescendant(b)
	if b.Tag == "anonymous" || b.Style.Display != boxes.DisplayBlock || len(b.Children) == 0 {
		return containerInsets{}, false
	}
	hasFloat := false
	for _, child := range b.Children {
		if child.Style.Float != boxes.FloatNone {
			hasFloat = true
			continue
		}
		specialLayout := child.Style.Display == boxes.DisplayFlex || child.Style.Display == boxes.DisplayGrid || child.Style.ColCount > 1 || child.Tag == "table"
		if child.Style.Display != boxes.DisplayBlock && child.Style.Display != boxes.DisplayInline && child.Tag != "br" && !specialLayout || child.Style.Position == boxes.PositionFixed {
			return containerInsets{}, false
		}
	}
	if !decorated && !hasFloat {
		return containerInsets{}, false
	}
	return boxInsets(b), true
}

func hasHeightSizing(b *boxes.Box) bool {
	if b.Image != nil || b.SVG != nil {
		return false
	}
	return !b.Style.Sizing.Height.IsAuto() || !b.Style.Sizing.MinHeight.IsAuto() || !b.Style.Sizing.MaxHeight.IsAuto()
}

func needsHeightContainingBlock(b *boxes.Box) bool {
	if b.Style.Sizing.Height.IsAuto() || b.Image != nil || b.SVG != nil {
		return false
	}
	for _, child := range b.Children {
		if child.Style.Sizing.Height.IsPercentage() || child.Style.SVGHeightPct > 0 {
			return true
		}
	}
	return false
}

func clipsOverflow(b *boxes.Box) bool {
	return clipsOverflowX(b) || clipsOverflowY(b)
}

func clipsOverflowX(b *boxes.Box) bool {
	return overflowAxisIsScrollable(b, true)
}

func clipsOverflowY(b *boxes.Box) bool {
	return overflowAxisIsScrollable(b, false)
}

func overflowAxisIsScrollable(box *boxes.Box, horizontal bool) bool {
	x, y := box.Style.OverflowX, box.Style.OverflowY
	if x == boxes.OverflowUnset {
		x = box.Style.Overflow
	}
	if y == boxes.OverflowUnset {
		y = box.Style.Overflow
	}
	if x == boxes.OverflowUnset {
		x = boxes.OverflowVisible
	}
	if y == boxes.OverflowUnset {
		y = boxes.OverflowVisible
	}
	axis, cross := x, y
	if !horizontal {
		axis, cross = y, x
	}
	return axis == boxes.OverflowHidden || axis == boxes.OverflowVisible && cross == boxes.OverflowHidden
}

// nestedContainerItems splits mixed content into anonymous inline boxes and
// explicit blocks while preserving Parts order.
func nestedContainerItems(b *boxes.Box) []*boxes.Box {
	var items []*boxes.Box
	var parts []boxes.Part
	var text strings.Builder
	flush := func() {
		if len(parts) == 0 || strings.TrimSpace(text.String()) == "" && !hasInlinePart(parts) {
			parts = nil
			text.Reset()
			return
		}
		anonymous := *b
		anonymous.Tag = "anonymous"
		anonymous.Style.BreakBefore = boxes.BreakBeforeAuto
		anonymous.Style.BreakAfter = boxes.BreakAfterAuto
		anonymous.Style.BreakInside = boxes.BreakInsideAuto
		anonymous.Text = text.String()
		anonymous.Parts = append([]boxes.Part(nil), parts...)
		anonymous.Children = nil
		anonymous.Style.HasBg = false
		anonymous.Style.BorderWidth = 0
		anonymous.Style.Border = [4]float64{}
		anonymous.Style.BorderTop, anonymous.Style.BorderRight = boxes.BorderSide{}, boxes.BorderSide{}
		anonymous.Style.BorderBottom, anonymous.Style.BorderLeft = boxes.BorderSide{}, boxes.BorderSide{}
		anonymous.Style.PaddingTop, anonymous.Style.PaddingRight = 0, 0
		anonymous.Style.PaddingBottom, anonymous.Style.PaddingLeft = 0, 0
		anonymous.Style.MarginTop, anonymous.Style.MarginRight = 0, 0
		anonymous.Style.MarginBottom, anonymous.Style.MarginLeft = 0, 0
		anonymous.Style.HasOpacity = false
		items = append(items, &anonymous)
		parts = nil
		text.Reset()
	}
	for _, part := range b.Parts {
		if part.Child == nil {
			parts = append(parts, part)
			text.WriteString(part.Text)
			continue
		}
		child := part.Child
		if inlinePartChild(child) {
			parts = append(parts, part)
			continue
		}
		flush()
		items = append(items, child)
	}
	flush()
	return items
}

// nestedContainerFlowMargins collapses margins with the outer block child when
// neither a border nor padding separates them on that side.
func nestedContainerFlowMargins(b *boxes.Box, inset containerInsets) ([]*boxes.Box, float64, float64, bool) {
	items := nestedContainerItems(b)
	topMargin, bottomMargin := b.Style.MarginTop, b.Style.MarginBottom
	if inset.top == 0 && len(items) > 0 && items[0].Style.Display == boxes.DisplayBlock {
		first := *items[0]
		topMargin = collapseMargins(topMargin, first.Style.MarginTop)
		first.Style.MarginTop = 0
		items[0] = &first
	}
	collapseBottom := inset.bottom == 0 && len(items) > 0 && items[len(items)-1].Style.Display == boxes.DisplayBlock
	if collapseBottom {
		lastIndex := len(items) - 1
		last := *items[lastIndex]
		bottomMargin = collapseMargins(bottomMargin, last.Style.MarginBottom)
		last.Style.MarginBottom = bottomMargin
		items[lastIndex] = &last
	}
	return items, topMargin, bottomMargin, collapseBottom
}

func hasInlinePart(parts []boxes.Part) bool {
	for _, part := range parts {
		if part.Child != nil && inlinePartChild(part.Child) {
			return true
		}
	}
	return false
}

func hasInlineContent(b *boxes.Box) bool {
	for _, part := range b.Parts {
		if part.Child == nil && strings.TrimSpace(part.Text) != "" || part.Child != nil && inlinePartChild(part.Child) {
			return true
		}
	}
	return false
}

func shouldMoveDecoratedBlockWithFollowingLines(b, next *boxes.Box, y, available, bottom float64, opts Options) bool {
	if b == nil || next == nil || b.Style.BreakAfter != boxes.BreakAfterAvoid || !hasInlineContent(b) || hasBlockChild(b) || !isKeepable(next) || hasDeclaredHeight(next, opts) {
		return false
	}
	currentWidth := usedWidth(b, math.Max(0, available-b.Style.MarginLeft-b.Style.MarginRight))
	currentLines := breakInlineStream(buildInlineStream(b, opts), math.Max(0, currentWidth-horizontalInsets(b)), wraps(b))
	currentHeight := usedHeight(b, inlineHeight(currentLines, b, opts), opts)
	nextWidth := usedWidth(next, math.Max(0, available-next.Style.MarginLeft-next.Style.MarginRight))
	nextLines := breakInlineStream(buildInlineStream(next, opts), math.Max(0, nextWidth-horizontalInsets(next)), wraps(next))
	orphans := next.Style.Orphans
	if orphans <= 0 {
		orphans = 2
	}
	keepLines := min(len(nextLines), orphans)
	if keepLines == 0 {
		keepLines = 1
	}
	firstFragmentHeight := inlineHeight(nextLines[:keepLines], next, opts) + next.Style.BorderTop.Width + next.Style.PaddingTop
	if keepLines == len(nextLines) {
		firstFragmentHeight += next.Style.BorderBottom.Width + next.Style.PaddingBottom
	}
	betweenMargins := collapseMargins(b.Style.MarginBottom, next.Style.MarginTop)
	pairHeight := currentHeight + betweenMargins + firstFragmentHeight
	pageCapacity := bottom - opts.top()
	return y+pairHeight > bottom+fragmentEpsilon && pairHeight <= pageCapacity+fragmentEpsilon && y+currentHeight <= bottom+fragmentEpsilon
}

type containerFragmentPlan struct {
	pageCount              int
	clipsAcrossPages       bool
	constrainDescendants   bool
	isolateDescendants     bool
	remainingOuterHeight   float64
	remainingContentHeight float64
}

func planContainerFragments(b *boxes.Box, outerTop float64, inset containerInsets, childPageCount int, opts Options) containerFragmentPlan {
	plan := containerFragmentPlan{pageCount: childPageCount}
	if !clipsOverflowY(b) || !hasDeclaredHeight(b, opts) {
		return plan
	}
	plan.constrainDescendants = true
	plan.remainingOuterHeight = usedDeclaredHeight(b, opts)
	if outerTop+plan.remainingOuterHeight <= opts.pageBottom()+fragmentEpsilon {
		plan.pageCount = min(childPageCount, 1)
		plan.isolateDescendants = true
		return plan
	}
	plan.clipsAcrossPages = true
	continuationHeight := plan.remainingOuterHeight - (opts.pageBottom() - outerTop)
	pageCapacity := opts.pageBottom() - opts.top()
	fragmentCount := 1
	if continuationHeight > 0 && pageCapacity > fragmentEpsilon {
		fragmentCount += int(math.Ceil(continuationHeight / pageCapacity))
	}
	plan.pageCount = min(childPageCount, fragmentCount)
	firstContentCapacity := math.Max(0, opts.pageBottom()-(outerTop+inset.top))
	plan.remainingContentHeight = math.Max(0, plan.remainingOuterHeight-inset.top-inset.bottom-firstContentCapacity)
	return plan
}

// flowNestedContainer retains the outer box for backgrounds/borders and lays out
// its children in an inset content area, including subtree pagination.
func flowNestedContainer(b *boxes.Box, outerTop float64, inset containerInsets, items []*boxes.Box, collapseBottom bool, pages *[]Page, opts Options, usable float64) float64 {
	outerX := opts.left() + b.Style.MarginLeft
	outerWidth := usedWidth(b, usable-b.Style.MarginLeft-b.Style.MarginRight)
	childContentWidth := outerWidth - inset.left - inset.right
	currentChildWidth := usable - b.Style.MarginLeft - b.Style.MarginRight - inset.left - inset.right

	childRoot := &boxes.Box{Tag: "root", Style: boxes.ComputedStyle{Display: boxes.DisplayBlock}, Children: items, Fonts: opts.Faces}
	childOpts := opts
	childSize := childOpts.space.containingSize
	childSize.Block = sizeBasis{}
	if hasDeclaredHeight(b, opts) {
		childSize.Block = definiteSize(math.Max(0, usedDeclaredHeight(b, opts)-verticalInsets(b)))
	}
	childOpts.space = childOpts.space.withContainingBlock(childSize)
	groupStack := append([]OpacityGroup(nil), opts.groupStack...)
	groupID := 0
	if b.Style.HasOpacity {
		if opts.groupSequence == nil {
			groupSequence := 0
			opts.groupSequence = &groupSequence
		}
		(*opts.groupSequence)++
		groupID = *opts.groupSequence
		groupStack = append(groupStack, OpacityGroup{ID: groupID, Opacity: b.Style.Opacity})
	}
	childOpts.groupStack = groupStack
	space := childOpts.space
	marginBottom := space.marginBottom
	if !clipsOverflowY(b) || !hasDeclaredHeight(b, opts) {
		marginBottom += inset.bottom
	}
	childOpts.space = space.withMargins(
		space.marginLeft+inset.left+b.Style.MarginLeft,
		space.marginRight+inset.right+b.Style.MarginRight+currentChildWidth-childContentWidth,
		space.marginTop,
		marginBottom,
	).withStartY(outerTop + inset.top)
	if b.Style.Position == boxes.PositionRelative {
		childOpts.space = childOpts.space.withAbsoluteContainingBlock(
			opts.left()+b.Style.MarginLeft+inset.left,
			outerTop+inset.top,
			childContentWidth,
		)
	}
	childPages, childEnd := flowLayout(childRoot, childOpts)
	if len(childPages) == 0 {
		if collapseBottom {
			return outerTop + inset.top + inset.bottom
		}
		return outerTop + inset.top + inset.bottom + b.Style.MarginBottom
	}
	startPage := len(*pages) - 1
	for pi := range childPages {
		for bi := 0; bi < childPages[pi].blockCount(); bi++ {
			ensureOpacityGroups(childPages[pi].block(bi), groupStack, opts.groupSequence)
		}
	}
	fs := resolveSize(b, opts)
	_, font := measureFor(opts, b)
	fragmentPlan := planContainerFragments(b, outerTop, inset, len(childPages), opts)
	paintedChildPageCount := fragmentPlan.pageCount
	fragmentedClip := fragmentPlan.clipsAcrossPages
	remainingOuterHeight := fragmentPlan.remainingOuterHeight
	fragmentedFlowEnd := 0.0
	if paintedChildPageCount == 1 && hasHeightSizing(b) && !fragmentedClip {
		contentEnd := childEnd
		if collapseBottom && len(items) > 0 {
			contentEnd -= items[len(items)-1].Style.MarginBottom
		}
		contentHeight := math.Max(0, contentEnd-outerTop-inset.top)
		outerHeight := usedHeight(b, contentHeight, opts)
		if clipsOverflowY(b) && hasDeclaredHeight(b, opts) {
			outerHeight = usedDeclaredHeight(b, opts)
		}
		childEnd = outerTop + outerHeight - inset.bottom
	}
	remainingContentHeight := fragmentPlan.remainingContentHeight
	for pi := 0; pi < paintedChildPageCount; pi++ {
		childPage := childPages[pi]
		pageIndex := startPage + pi
		for len(*pages) <= pageIndex {
			*pages = append(*pages, Page{Number: len(*pages) + 1})
		}
		segmentTop := opts.top()
		if pi == 0 {
			segmentTop = outerTop
		}
		visualTop := segmentTop
		visualX := outerX
		if b.Style.Position == boxes.PositionRelative {
			visualTop += b.Style.Top
			visualX += b.Style.Left
		}
		segmentBottom := opts.pageBottom()
		if pi == len(childPages)-1 {
			segmentBottom = childEnd + inset.bottom
			if collapseBottom && len(items) > 0 {
				segmentBottom -= items[len(items)-1].Style.MarginBottom
			}
		}
		if pi == 0 && clipsOverflowY(b) && hasDeclaredHeight(b, opts) {
			segmentBottom = segmentTop + usedDeclaredHeight(b, opts)
		}
		if fragmentedClip {
			fragmentHeight := math.Min(remainingOuterHeight, math.Max(0, opts.pageBottom()-segmentTop))
			segmentBottom = segmentTop + fragmentHeight
			remainingOuterHeight -= fragmentHeight
			fragmentedFlowEnd = segmentBottom
		}
		if pi == 0 && hasDeclaredHeight(b, opts) && !fragmentedClip {
			minimumBottom := outerTop + usedDeclaredHeight(b, opts)
			if segmentBottom < minimumBottom {
				segmentBottom = minimumBottom
			}
		}
		height := segmentBottom - segmentTop
		if height < 0 {
			height = 0
		}
		outer := placed(b, "", nil, visualX, visualTop, outerWidth, height, fs, font)
		outer.fragmentRef = newFragmentOccurrence(b)
		for blockIndex := 0; blockIndex < childPage.blockCount(); blockIndex++ {
			bindDirectFragmentChild(outer.fragmentRef, childPage.block(blockIndex))
		}
		if fragmentedClip {
			outer = fragmentPlacedBox(outer, pi == 0, remainingOuterHeight <= 0)
		} else if paintedChildPageCount > 1 {
			outer = fragmentPlacedBox(outer, pi == 0, pi == paintedChildPageCount-1)
		}
		outer.HasOpacity = false
		outer.OpacityGroups = append([]OpacityGroup(nil), groupStack...)
		page := &(*pages)[pageIndex]
		if b.Style.Position == boxes.PositionRelative {
			for i := 0; i < childPage.blockCount(); i++ {
				shiftPlacedBox(childPage.block(i), b.Style.Left, b.Style.Top)
			}
		}
		if clip := overflowClipForBox(b, visualX, visualTop, outerWidth, height, opts, true); clip != nil {
			if clipsOverflowY(b) {
				if hasDeclaredHeight(b, opts) {
					clip.Height = usedDeclaredHeight(b, opts) - b.Style.BorderTop.Width - b.Style.BorderBottom.Width
				}
				if fragmentedClip && pi > 0 {
					clip.Y = opts.top() + inset.top
					clip.Height = remainingContentHeight
					remainingContentHeight = math.Max(0, remainingContentHeight-(opts.pageBottom()-(opts.top()+inset.top)))
				}
			}
			clip.Width = math.Max(0, clip.Width)
			clip.Height = math.Max(0, clip.Height)
			for i := 0; i < childPage.blockCount(); i++ {
				childPage.block(i).ClipRects = append(childPage.block(i).ClipRects, *clip)
			}
		}
		page.emitBlock(outer)
		for _, node := range append([]*FragmentNode(nil), childPage.paintNodes()...) {
			page.emitBlock(node.Placed)
		}
	}
	if hasDeclaredHeight(b, opts) {
		minimumContentEnd := outerTop + usedDeclaredHeight(b, opts) - inset.bottom
		if childEnd < minimumContentEnd {
			childEnd = minimumContentEnd
		}
	}
	if clipsOverflowY(b) && hasDeclaredHeight(b, opts) {
		childEnd = outerTop + usedDeclaredHeight(b, opts) - inset.bottom
	}
	if fragmentedClip && fragmentedFlowEnd > 0 {
		return fragmentedFlowEnd + b.Style.MarginBottom
	}
	if collapseBottom {
		return childEnd + inset.bottom
	}
	return childEnd + inset.bottom + b.Style.MarginBottom
}
