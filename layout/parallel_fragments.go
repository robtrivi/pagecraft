package layout

import (
	"math"

	"github.com/robtrivi/pagecraft-go/boxes"
)

type fragmentStream struct {
	pages     []Page
	endY      float64
	hasRoot   bool
	alignment boxes.FlexAlignment
}

type parallelFlowItem struct {
	x, width  float64
	alignment boxes.FlexAlignment
	layout    func(pages *[]Page, opts Options) float64
}

func alignFragmentStreams(streams []fragmentStream, firstPageY, pageTop float64) {
	pageCount := 0
	for _, stream := range streams {
		pageCount = max(pageCount, len(stream.pages))
	}
	for offset := 0; offset < pageCount; offset++ {
		rowTop := pageTop
		if offset == 0 {
			rowTop = firstPageY
		}
		rowBottom := math.Inf(-1)
		for _, stream := range streams {
			if offset >= len(stream.pages) {
				continue
			}
			for _, node := range stream.pages[offset].paintNodes() {
				block := &node.Placed
				rowBottom = math.Max(rowBottom, block.Y+block.Height)
			}
		}
		if math.IsInf(rowBottom, -1) {
			continue
		}
		rowHeight := math.Max(0, rowBottom-rowTop)
		for streamIndex := range streams {
			stream := &streams[streamIndex]
			if offset >= len(stream.pages) || stream.pages[offset].blockCount() == 0 {
				continue
			}
			itemTop, itemBottom := math.Inf(1), math.Inf(-1)
			rootIndex := -1
			for blockIndex := 0; blockIndex < stream.pages[offset].blockCount(); blockIndex++ {
				block := stream.pages[offset].block(blockIndex)
				itemTop = math.Min(itemTop, block.Y)
				itemBottom = math.Max(itemBottom, block.Y+block.Height)
				if blockIndex == 0 && stream.hasRoot {
					rootIndex = blockIndex
				}
			}
			itemHeight := math.Max(0, itemBottom-itemTop)
			alignment := stream.alignment
			shift := 0.0
			switch alignment {
			case boxes.AlignCenter:
				shift = (rowHeight - itemHeight) / 2
			case boxes.AlignFlexEnd:
				shift = rowHeight - itemHeight
			case boxes.AlignStretch:
				if rootIndex >= 0 {
					root := stream.pages[offset].block(rootIndex)
					root.Height = math.Max(root.Height, rowBottom-root.Y)
					itemBottom = math.Max(itemBottom, root.Y+root.Height)
				}
			}
			if shift != 0 {
				for blockIndex := 0; blockIndex < stream.pages[offset].blockCount(); blockIndex++ {
					shiftPlacedBox(stream.pages[offset].block(blockIndex), 0, shift)
				}
			}
			if offset == len(stream.pages)-1 {
				stream.endY = math.Max(stream.endY+shift, itemBottom+shift)
				if alignment == boxes.AlignStretch && rootIndex >= 0 {
					root := stream.pages[offset].block(rootIndex)
					stream.endY = math.Max(stream.endY, root.Y+root.Height)
				}
			}
		}
	}
}

// layoutParallelFlows gives each item an isolated page flow, then aligns and
// merges the resulting fragments into the parent flow.
func layoutParallelFlows(pages *[]Page, firstPageY, pageTop float64, opts Options, items []parallelFlowItem) (int, float64) {
	firstPage := len(*pages) - 1
	streams := make([]fragmentStream, 0, len(items))
	for _, item := range items {
		itemOpts := opts
		itemOpts.space.marginLeft = item.x - itemOpts.pageLeftMargin()
		itemOpts.space.marginRight = itemOpts.PageWidth - itemOpts.pageRightMargin() - item.x - item.width
		itemPages := []Page{{Number: 1}}
		endY := item.layout(&itemPages, itemOpts)
		streams = append(streams, fragmentStream{pages: itemPages, endY: endY, hasRoot: true, alignment: item.alignment})
	}
	alignFragmentStreams(streams, firstPageY, pageTop)
	return mergeFragmentStreams(pages, firstPage, streams)
}

// mergeFragmentStreams overlays independently continued layout contexts in
// the same fragmentainers while preserving each context's paint order.
func mergeFragmentStreams(pages *[]Page, firstPage int, streams []fragmentStream) (int, float64) {
	lastOffset, lastY := 0, math.Inf(-1)
	for _, stream := range streams {
		if len(stream.pages) == 0 {
			continue
		}
		offset := len(stream.pages) - 1
		if offset > lastOffset || offset == lastOffset && stream.endY > lastY {
			lastOffset, lastY = offset, stream.endY
		}
	}
	if math.IsInf(lastY, -1) {
		return 0, 0
	}
	for offset := 0; offset <= lastOffset; offset++ {
		ensurePageIndex(pages, firstPage+offset)
		page := &(*pages)[firstPage+offset]
		for _, stream := range streams {
			if offset < len(stream.pages) {
				for _, node := range append([]*FragmentNode(nil), stream.pages[offset].paintNodes()...) {
					page.emitBlock(node.Placed)
				}
			}
		}
	}
	return lastOffset, lastY
}
