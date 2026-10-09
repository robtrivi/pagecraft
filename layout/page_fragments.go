package layout

import (
	"sort"
	"strconv"
	"strings"

	"github.com/robtrivi/pagecraft-go/boxes"
)

// pageFragmentState owns the occurrences emitted into one page.
type pageFragmentState struct {
	roots []*FragmentNode
	nodes []*FragmentNode
	paint []*FragmentNode
}

func (state *pageFragmentState) removeRoot(node *FragmentNode) {
	for index, root := range state.roots {
		if root == node {
			state.roots = append(state.roots[:index], state.roots[index+1:]...)
			return
		}
	}
}

func (state *pageFragmentState) removeNode(node *FragmentNode) {
	state.removeRoot(node)
	for index, candidate := range state.paint {
		if candidate == node {
			state.paint = append(state.paint[:index], state.paint[index+1:]...)
			break
		}
	}
	for index, candidate := range state.nodes {
		if candidate == node {
			state.nodes = append(state.nodes[:index], state.nodes[index+1:]...)
			return
		}
	}
}

func (state *pageFragmentState) contains(node *FragmentNode) bool {
	for _, candidate := range state.nodes {
		if candidate == node {
			return true
		}
	}
	return false
}

func (state *pageFragmentState) adopt(node *FragmentNode) {
	if node.state != state {
		if node.state != nil {
			node.state.removeNode(node)
		}
		node.state = state
	}
	if !state.contains(node) {
		state.nodes = append(state.nodes, node)
	}
	for _, child := range node.Children {
		state.adopt(child)
	}
}

func (page *Page) state() *pageFragmentState {
	if page.fragmentState == nil {
		page.fragmentState = &pageFragmentState{}
	}
	return page.fragmentState
}

func (page *Page) emitBlock(block PlacedBox) {
	page.insertBlock(page.blockCount(), block)
}

func (page *Page) blockCount() int {
	if page.fragmentState == nil {
		return 0
	}
	return len(page.fragmentState.paint)
}

func (page *Page) block(index int) *PlacedBox {
	return &page.state().paint[index].Placed
}

func (page *Page) paintNodes() []*FragmentNode {
	return page.state().paint
}

func (page *Page) insertBlock(index int, block PlacedBox) {
	if index < 0 || index > page.blockCount() {
		panic("layout: invalid fragment insertion index")
	}
	state := page.state()
	node := block.fragmentNode
	if node == nil {
		node = &FragmentNode{Kind: FragmentPageBlock, OwnerPaintIndex: -1, DecorationIndex: -1}
	}
	state.adopt(node)
	node.Placed = block
	node.Placed.fragmentNode = node
	if block.parentFragmentRef != nil {
		block.parentFragmentRef.attach(node)
	} else if node.Parent == nil {
		state.roots = append(state.roots, node)
	}
	if block.fragmentRef != nil {
		block.fragmentRef.node = node
		for _, child := range block.fragmentRef.pending {
			block.fragmentRef.attach(child)
		}
		block.fragmentRef.pending = nil
	}
	state.paint = append(state.paint, nil)
	copy(state.paint[index+1:], state.paint[index:])
	state.paint[index] = node
	page.FragmentTree = state.roots
}

func (page *Page) emitDecoration(ownerIndex, decorationIndex int, block PlacedBox) {
	if ownerIndex < 0 || ownerIndex >= page.blockCount() {
		panic("layout: decoration without emitting owner")
	}
	owner := page.state().paint[ownerIndex]
	if owner == nil {
		panic("layout: decoration owner has no fragment")
	}
	state := page.state()
	node := &FragmentNode{
		Placed: block, Kind: FragmentDecoration, OwnerPaintIndex: ownerIndex,
		DecorationIndex: decorationIndex, owner: owner,
	}
	node.state = state
	state.nodes = append(state.nodes, node)
	if block.parentFragmentRef != nil {
		block.parentFragmentRef.attach(node)
	} else {
		node.Parent = owner
		owner.Children = append(owner.Children, node)
	}
	if block.fragmentRef != nil {
		block.fragmentRef.node = node
		for _, child := range block.fragmentRef.pending {
			block.fragmentRef.attach(child)
		}
		block.fragmentRef.pending = nil
	}
}

func finalizePageFragments(root *boxes.Box, pages []Page) {
	if root == nil {
		return
	}
	boxes.AssignSourcePaths(root)
	occurrences := make(map[string][]*FragmentNode)
	for pageIndex := range pages {
		page := &pages[pageIndex]
		state := page.state()
		for paintIndex, node := range state.paint {
			node.Fragment.PaintIndex = paintIndex
		}
		for _, node := range state.nodes {
			if node.Placed.parentFragmentRef != nil && node.Parent == nil {
				panic("layout: unresolved parent fragment occurrence")
			}
			if node.Kind == FragmentDecoration {
				if node.owner == nil {
					panic("layout: decoration without owner occurrence")
				}
				node.OwnerPaintIndex = node.owner.Fragment.PaintIndex
			}
			if node.Parent == nil {
				state.roots = appendRootOnce(state.roots, node)
			} else {
				state.removeRoot(node)
			}
			block := node.Placed
			fragment := BoxFragment{
				Source: block.source, PageNumber: page.Number,
				FragmentainerIndex: block.FragmentainerIndex,
				PaintIndex:         node.Fragment.PaintIndex, Repeated: block.repeated,
				Bounds: FragmentBounds{X: block.X, Y: block.Y, Width: block.Width, Height: block.Height},
			}
			if node.Kind == FragmentDecoration {
				fragment.PaintIndex = -1
				fragment.FragmentainerIndex = node.owner.Placed.FragmentainerIndex
				fragment.Repeated = node.owner.Placed.repeated
			}
			if block.source != nil {
				fragment.Path = block.source.SourcePath()
				if block.source.Style.Position == boxes.PositionFixed || block.repeated {
					fragment.Repeated = true
				}
			}
			if fragment.Source == nil || fragment.Repeated {
				fragment.FragmentCount, fragment.IsFirst, fragment.IsLast = 1, true, true
			} else {
				key := sourcePathKey(fragment.Path)
				occurrences[key] = append(occurrences[key], node)
			}
			node.Fragment = fragment
		}
		page.FragmentTree = state.roots
		page.Fragments = make([]BoxFragment, len(state.paint))
	}
	for _, nodes := range occurrences {
		for index, node := range nodes {
			node.Fragment.FragmentIndex = index
			node.Fragment.FragmentCount = len(nodes)
			node.Fragment.IsFirst = index == 0
			node.Fragment.IsLast = index == len(nodes)-1
		}
	}
	for pageIndex := range pages {
		page := &pages[pageIndex]
		sortFragmentTree(page.FragmentTree)
		blocks := make([]PlacedBox, len(page.fragmentState.paint))
		for index, node := range page.fragmentState.paint {
			blocks[index] = node.Placed
			page.Fragments[index] = node.Fragment
		}
		page.Blocks = blocks
	}
}

func appendRootOnce(roots []*FragmentNode, node *FragmentNode) []*FragmentNode {
	for _, root := range roots {
		if root == node {
			return roots
		}
	}
	return append(roots, node)
}

func sortFragmentTree(nodes []*FragmentNode) {
	sort.SliceStable(nodes, func(i, j int) bool {
		return fragmentNodePaintOrder(nodes[i]) < fragmentNodePaintOrder(nodes[j])
	})
	for _, node := range nodes {
		sortFragmentTree(node.Children)
	}
}

func fragmentNodePaintOrder(node *FragmentNode) int {
	if node.Kind == FragmentDecoration {
		return node.OwnerPaintIndex*1_000_000 + node.DecorationIndex
	}
	return node.Fragment.PaintIndex*1_000_000 + 500_000
}

func sourcePathKey(path []int) string {
	parts := make([]string, len(path))
	for index, childIndex := range path {
		parts[index] = strconv.Itoa(childIndex)
	}
	return strings.Join(parts, "/")
}
