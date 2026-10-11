package layout

import (
	"strings"
	"testing"

	"github.com/robtrivi/pagecraft/boxes"
)

func TestLayout_FragmentsRetainBoxPathAndContinuationAcrossPages(t *testing.T) {
	html := `<div style="opacity:0.5;background:#ff0000"><p style="margin:0;line-height:20pt">filler1<br>filler2<br>filler3</p><p style="margin:0;line-height:20pt">keep1<br>keep2<br>keep3<br>keep4</p></div>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 200, 140, 20
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages=%d, want 2", len(pages))
	}
	wrapper, paragraph := root.Children[0], root.Children[0].Children[1]
	wrapperFragments := fragmentsForSource(pages, wrapper)
	paragraphFragments := fragmentsForSource(pages, paragraph)
	if len(wrapperFragments) != 2 || len(paragraphFragments) != 2 {
		t.Fatalf("fragments wrapper/paragraph=%d/%d, want 2/2", len(wrapperFragments), len(paragraphFragments))
	}
	assertFragmentChain(t, wrapperFragments)
	assertFragmentChain(t, paragraphFragments)
	for pageIndex, page := range pages {
		if len(page.Fragments) != len(page.Blocks) {
			t.Fatalf("page %d: fragments=%d blocks=%d", pageIndex+1, len(page.Fragments), len(page.Blocks))
		}
		for paintIndex, fragment := range page.Fragments {
			if fragment.PaintIndex != paintIndex || fragment.PageNumber != pageIndex+1 {
				t.Errorf("fragment %d on page %d lost its paint link: %+v", paintIndex, pageIndex+1, fragment)
			}
		}
	}
}

func TestLayout_TransferringOccurrenceMovesPaintOwnership(t *testing.T) {
	source := &boxes.Box{Tag: "p"}
	from, to := Page{Number: 1}, Page{Number: 2}
	from.emitBlock(PlacedBox{source: source, X: 10})
	to.emitBlock(*from.block(0))
	if from.blockCount() != 0 {
		t.Fatalf("source page retains %d occurrences after transfer", from.blockCount())
	}
	if to.blockCount() != 1 || to.block(0).X != 10 {
		t.Fatalf("destination page lost the transferred geometry: count=%d", to.blockCount())
	}
}

func TestLayout_FragmentsMapEveryPaginatingStructureToSourceBoxes(t *testing.T) {
	cases := []struct {
		name string
		html string
	}{
		{name: "block", html: `<p>BLOCK SOURCE</p>`},
		{name: "flex", html: `<div style="display:flex"><span>FLEX SOURCE</span><span>FLEX SIBLING</span></div>`},
		{name: "grid", html: `<div style="display:grid;grid-template-columns:1fr 1fr"><span>GRID SOURCE</span><span>GRID SIBLING</span></div>`},
		{name: "table", html: `<table><tr><td>TABLE SOURCE</td><td>TABLE SIBLING</td></tr></table>`},
		{name: "columns", html: `<div style="column-count:2"><p>COLUMN SOURCE</p><p>COLUMN SIBLING</p></div>`},
		{name: "nested", html: `<section style="padding:2pt;background:#eee"><p>NESTED SOURCE</p></section>`},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			root, err := boxes.Build(test.html)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			pages := Flow(root, testOptions())
			found := false
			for pageIndex, page := range pages {
				if len(page.Fragments) != len(page.Blocks) {
					t.Fatalf("page %d: fragments=%d blocks=%d", pageIndex+1, len(page.Fragments), len(page.Blocks))
				}
				for paintIndex, fragment := range page.Fragments {
					if fragment.PaintIndex != paintIndex || fragment.PageNumber != page.Number {
						t.Errorf("fragment has not paint link: %+v", fragment)
					}
					if fragment.Source == nil {
						continue
					}
					found = true
					if !equalFragmentPath(fragment.Path, fragment.Source.SourcePath()) || len(fragment.Path) == 0 {
						t.Errorf("incorrect source path: path=%v source=%v", fragment.Path, fragment.Source.SourcePath())
					}
				}
			}
			if !found {
				t.Fatal("did not generate fragments linked to source boxes")
			}
		})
	}
}

func TestLayout_FragmentsAssignPathsToHandBuiltBoxTrees(t *testing.T) {
	root := &boxes.Box{Tag: "html", Children: []*boxes.Box{{Tag: "p", Style: boxes.ComputedStyle{Display: boxes.DisplayBlock}, Text: "HAND BUILT"}}}
	pages := Flow(root, testOptions())
	if len(pages) != 1 || len(pages[0].Fragments) == 0 {
		t.Fatalf("pages/fragments=%d/%+v", len(pages), pages)
	}
	fragment := pages[0].Fragments[0]
	if fragment.Source != root.Children[0] || !equalFragmentPath(fragment.Path, []int{0}) {
		t.Fatalf("fragment without source path for manually built tree: %+v", fragment)
	}
	if len(pages[0].FragmentTree) != 1 || pages[0].FragmentTree[0].Fragment.Source != fragment.Source {
		t.Errorf("tree of fragments must preserve the root node without a parent: %+v", pages[0].FragmentTree)
	}
	if fragment.FragmentCount != 1 || !fragment.IsFirst || !fragment.IsLast {
		t.Errorf("fragment chain is incorrect: %+v", fragment)
	}
}

func TestLayout_FragmentTreeUsesEmittingOccurrenceInsteadOfBoundsCenters(t *testing.T) {
	root := &boxes.Box{Tag: "html"}
	parent := &boxes.Box{Tag: "section"}
	child := &boxes.Box{Tag: "p"}
	root.Children = []*boxes.Box{parent}
	parent.Children = []*boxes.Box{child}
	page := Page{Number: 1}
	first, second := newFragmentOccurrence(parent), newFragmentOccurrence(parent)
	page.emitBlock(PlacedBox{source: parent, fragmentRef: first, X: 10, Y: 10, Width: 80, Height: 10})
	page.emitBlock(PlacedBox{source: parent, fragmentRef: second, X: 10, Y: 30, Width: 80, Height: 10})
	page.emitBlock(PlacedBox{source: child, sourceParent: parent, parentFragmentRef: second, X: 500, Y: 500, Width: 20, Height: 10})
	pages := []Page{page}
	finalizePageFragments(root, pages)

	var parentFragments []*FragmentNode
	var childNode *FragmentNode
	var visit func([]*FragmentNode)
	visit = func(nodes []*FragmentNode) {
		for _, node := range nodes {
			switch node.Fragment.Source {
			case parent:
				parentFragments = append(parentFragments, node)
			case child:
				childNode = node
			}
			visit(node.Children)
		}
	}
	visit(pages[0].FragmentTree)
	if len(parentFragments) != 2 || childNode == nil {
		t.Fatalf("fragments parent/child=%d/%v", len(parentFragments), childNode != nil)
	}
	if !containsFragmentNode(parentFragments[1].Children, childNode) {
		t.Errorf("child must link to the emitting occurrence: parents=%+v child=%+v", parentFragments, childNode.Fragment)
	}
	if childNode.Parent != parentFragments[1] {
		t.Errorf("child parent reference=%p, want concrete instance %p", childNode.Parent, parentFragments[1])
	}
}

func TestLayout_NestedTableFragmentsExposeParentChildRelations(t *testing.T) {
	html := `<style>body{margin:0}table{margin:0;border-collapse:collapse}#outer{width:220pt}td{font:10pt/12pt Helvetica;padding:2pt;border:1pt solid #111}</style><table id="outer"><tr><td>OUTER <table><tr><td>LEFT</td><td>RIGHT</td></tr></table></td><td>SUMMARY</td></tr></table>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	page := Flow(root, testOptions())[0]
	parentSource := findTableCellContaining(root, "OUTER")
	leftSource := findTableCellContaining(root, "LEFT")
	rightSource := findTableCellContaining(root, "RIGHT")
	parent := fragmentNodeForSource(page.FragmentTree, parentSource)
	left, right := fragmentNodeForSource(page.FragmentTree, leftSource), fragmentNodeForSource(page.FragmentTree, rightSource)
	if parent == nil || left == nil || right == nil {
		t.Fatalf("missing nodes: parent=%v left=%v right=%v", parent != nil, left != nil, right != nil)
	}
	if len(parent.Children) != 2 || parent.Children[0] != left || parent.Children[1] != right {
		t.Errorf("outer td children=%v, want LEFT and RIGHT in paint order", parent.Children)
	}
	if parent.Fragment.PaintIndex >= left.Fragment.PaintIndex || left.Fragment.PaintIndex >= right.Fragment.PaintIndex {
		t.Errorf("parent/row paint order is incorrect: %d→%d→%d", parent.Fragment.PaintIndex, left.Fragment.PaintIndex, right.Fragment.PaintIndex)
	}
	seen := make(map[*FragmentNode]bool)
	var visit func([]*FragmentNode)
	visit = func(nodes []*FragmentNode) {
		for _, node := range nodes {
			if seen[node] {
				t.Errorf("repeated node or cycle between nodes: %+v", node.Fragment)
				continue
			}
			seen[node] = true
			visit(node.Children)
		}
	}
	visit(page.FragmentTree)
}

func TestLayout_FragmentTreeIncludesCellDecorations(t *testing.T) {
	html := `<style>table{border-collapse:collapse}td{padding:2pt}div{background:#eef;border:1pt solid #00f;padding:2pt}p{margin:0;background:#fee;border:1pt solid #f00}</style><table><tr><td><div><p>DECORATED</p></div></td></tr></table>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	page := Flow(root, testOptions())[0]
	cell := findTableCellContaining(root, "DECORATED")
	div := cell.Children[0]
	paragraph := findSourceBox(root, "DECORATED", "p")
	var cellNode, divNode, paragraphNode *FragmentNode
	var visit func([]*FragmentNode)
	visit = func(nodes []*FragmentNode) {
		for _, node := range nodes {
			switch node.Fragment.Source {
			case cell:
				cellNode = node
			case div:
				divNode = node
			case paragraph:
				paragraphNode = node
			}
			visit(node.Children)
		}
	}
	visit(page.FragmentTree)
	if cellNode == nil || divNode == nil || paragraphNode == nil {
		t.Fatalf("tree without fragments: cell=%v div=%v p=%v", cellNode != nil, divNode != nil, paragraphNode != nil)
	}
	if divNode.Kind != FragmentDecoration || paragraphNode.Kind != FragmentDecoration {
		t.Errorf("decorated fragments have types %q/%q", divNode.Kind, paragraphNode.Kind)
	}
	if !containsFragmentNode(cellNode.Children, divNode) || !containsFragmentNode(divNode.Children, paragraphNode) {
		t.Errorf("incorrect decoration relation: td→div→p is not in the tree")
	}
	if divNode.Fragment.Bounds.Width <= 0 || paragraphNode.Fragment.Bounds.Height <= 0 {
		t.Errorf("geometry decorativa ausente: div=%+v p=%+v", divNode.Fragment.Bounds, paragraphNode.Fragment.Bounds)
	}
	pageBlocks, decorations := make(map[int]bool), 0
	var countNodes func([]*FragmentNode)
	countNodes = func(nodes []*FragmentNode) {
		for _, node := range nodes {
			if node.Kind == FragmentPageBlock {
				if node.Fragment.PaintIndex < 0 || node.Fragment.PaintIndex >= len(page.Blocks) || pageBlocks[node.Fragment.PaintIndex] {
					t.Errorf("invalid or duplicate page-block node: %+v", node.Fragment)
				}
				pageBlocks[node.Fragment.PaintIndex] = true
			} else {
				decorations++
				if node.Fragment.PaintIndex != -1 {
					t.Errorf("decoration must use PaintIndex -1: %+v", node.Fragment)
				}
			}
			countNodes(node.Children)
		}
	}
	countNodes(page.FragmentTree)
	if len(pageBlocks) != len(page.Blocks) || decorations != 2 {
		t.Errorf("fragment tree does not cover the page: page-blocks=%d/%d decorations=%d, want 2", len(pageBlocks), len(page.Blocks), decorations)
	}
}

func findSourceBox(root *boxes.Box, text, tag string) *boxes.Box {
	if root.Tag == tag && strings.Contains(root.Text, text) {
		return root
	}
	for _, child := range root.Children {
		if found := findSourceBox(child, text, tag); found != nil {
			return found
		}
	}
	return nil
}

func fragmentNodeForSource(nodes []*FragmentNode, source *boxes.Box) *FragmentNode {
	for _, node := range nodes {
		if node.Fragment.Source == source {
			return node
		}
		if found := fragmentNodeForSource(node.Children, source); found != nil {
			return found
		}
	}
	return nil
}

func containsFragmentNode(nodes []*FragmentNode, target *FragmentNode) bool {
	for _, node := range nodes {
		if node == target {
			return true
		}
	}
	return false
}

func placedFragmentsForSource(pages []Page, source *boxes.Box) []PlacedBox {
	var placed []PlacedBox
	var visit func([]*FragmentNode)
	visit = func(nodes []*FragmentNode) {
		for _, node := range nodes {
			if node.Fragment.Source == source && node.Kind == FragmentDecoration {
				placed = append(placed, node.Placed)
			}
			visit(node.Children)
		}
	}
	for _, page := range pages {
		visit(page.FragmentTree)
	}
	return placed
}

func fragmentsForSource(pages []Page, source *boxes.Box) []BoxFragment {
	var fragments []BoxFragment
	for _, page := range pages {
		for _, fragment := range page.Fragments {
			if equalFragmentPath(fragment.Path, source.SourcePath()) {
				fragments = append(fragments, fragment)
			}
		}
	}
	return fragments
}

func equalFragmentPath(left, right []int) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func assertFragmentChain(t *testing.T, fragments []BoxFragment) {
	t.Helper()
	for i, fragment := range fragments {
		if fragment.FragmentIndex != i || fragment.FragmentCount != len(fragments) || fragment.IsFirst != (i == 0) || fragment.IsLast != (i == len(fragments)-1) {
			t.Errorf("fragment chain %d/%d is incorrect: %+v", i, len(fragments), fragment)
		}
		if len(fragment.Path) == 0 {
			t.Errorf("fragment of %q without box path", strings.TrimSpace(fragment.Source.Tag))
		}
	}
}

func TestLayout_PlacedFragmentsCarryTheirSourceParent(t *testing.T) {
	root, err := boxes.Build(`<section style="padding:1pt"><p style="background:#eee">child</p></section>`)
	if err != nil {
		t.Fatal(err)
	}
	pages := Flow(root, testOptions())
	section, paragraph := root.Children[0], root.Children[0].Children[0]
	found := false
	for _, page := range pages {
		for _, block := range page.Blocks {
			if block.source == paragraph {
				found = true
				if block.sourceParent != section {
					t.Fatalf("placed block parent=%p, want source parent %p", block.sourceParent, section)
				}
			}
		}
	}
	if !found {
		t.Fatal("paragraph fragment was not emitted")
	}
}

func TestLayout_FragmentTreeRetainsConcreteParentFragmentReference(t *testing.T) {
	root := &boxes.Box{Tag: "html"}
	parent := &boxes.Box{Tag: "section"}
	child := &boxes.Box{Tag: "p"}
	root.Children = []*boxes.Box{parent}
	parent.Children = []*boxes.Box{child}
	pages := []Page{{Number: 1}}
	first, second := newFragmentOccurrence(parent), newFragmentOccurrence(parent)
	pages[0].emitBlock(PlacedBox{source: parent, fragmentRef: first, X: 10, Y: 200, Width: 80, Height: 20, FragmentainerIndex: 1})
	pages[0].emitBlock(PlacedBox{source: parent, fragmentRef: second, X: 10, Y: 480, Width: 80, Height: 20, FragmentainerIndex: 0})
	pages[0].emitBlock(PlacedBox{source: child, sourceParent: parent, parentFragmentRef: first, X: 10, Y: 480, Width: 40, Height: 10, FragmentainerIndex: 1})
	finalizePageFragments(root, pages)

	var parentInTargetFragmentainer, childNode *FragmentNode
	var visit func([]*FragmentNode)
	visit = func(nodes []*FragmentNode) {
		for _, node := range nodes {
			if node.Fragment.Source == parent && node.Fragment.FragmentainerIndex == 1 {
				parentInTargetFragmentainer = node
			}
			if node.Fragment.Source == child {
				childNode = node
			}
			visit(node.Children)
		}
	}
	visit(pages[0].FragmentTree)
	if parentInTargetFragmentainer == nil || childNode == nil {
		t.Fatal("expected parent and child fragments were not constructed")
	}
	if childNode.Parent != parentInTargetFragmentainer {
		t.Fatalf("parent reference=%p, want concrete fragment=%p (fragmentainer=%d)", childNode.Parent, parentInTargetFragmentainer, childNode.Fragment.FragmentainerIndex)
	}
	if !containsFragmentNode(parentInTargetFragmentainer.Children, childNode) {
		t.Fatal("child list does not reflect the direct parent reference")
	}
	assertFragmentParentLinks(t, pages[0].FragmentTree)
}

func assertFragmentParentLinks(t *testing.T, roots []*FragmentNode) {
	t.Helper()
	var visit func(*FragmentNode)
	visit = func(parent *FragmentNode) {
		for _, child := range parent.Children {
			if child.Parent != parent {
				t.Errorf("source node=%v parent=%p, want %p", child.Fragment.Source, child.Parent, parent)
			}
			visit(child)
		}
	}
	for _, root := range roots {
		if root.Parent != nil {
			t.Errorf("root source=%v has Parent=%p", root.Fragment.Source, root.Parent)
		}
		visit(root)
	}
}

func TestLayout_NestedPlacementCarriesConcreteParentOccurrence(t *testing.T) {
	cases := []struct {
		name string
		html string
	}{
		{name: "nested block", html: `<section style="padding:2pt;border:1pt solid #000"><p>child content</p></section>`},
		{name: "flex container", html: `<section style="display:flex;padding:2pt;border:1pt solid #000"><p>child content</p></section>`},
		{name: "grid container", html: `<section style="display:grid;grid-template-columns:1fr;padding:2pt;border:1pt solid #000"><p>child content</p></section>`},
		{name: "table cell", html: `<table><tr><td style="padding:2pt;border:1pt solid #000"><p>child content</p></td></tr></table>`},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			root, err := boxes.Build(test.html)
			if err != nil {
				t.Fatal(err)
			}
			pages := Flow(root, testOptions())
			var parent, child *FragmentNode
			var visit func([]*FragmentNode)
			visit = func(nodes []*FragmentNode) {
				for _, node := range nodes {
					if node.Fragment.Source != nil {
						switch node.Fragment.Source.Tag {
						case "section", "td":
							parent = node
						case "p":
							child = node
						}
					}
					visit(node.Children)
				}
			}
			for _, page := range pages {
				visit(page.FragmentTree)
			}
			if parent == nil || child == nil {
				t.Fatal("decorated and descendant fragments were not emitted")
			}
			if child.Placed.parentFragmentRef == nil || child.Placed.parentFragmentRef != parent.Placed.fragmentRef {
				t.Fatalf("placement parent token=%p, want the identity token carried by parent fragment %p", child.Placed.parentFragmentRef, parent.Placed.fragmentRef)
			}
			if child.Parent != parent {
				t.Fatalf("resolved parent=%p, want direct parent fragment %p", child.Parent, parent)
			}
		})
	}
}

func TestLayout_FragmentTreeExplicitParentIgnoresGeometry(t *testing.T) {
	root := &boxes.Box{Tag: "html"}
	parent := &boxes.Box{Tag: "section"}
	child := &boxes.Box{Tag: "p"}
	root.Children = []*boxes.Box{parent}
	parent.Children = []*boxes.Box{child}
	pages := []Page{{Number: 1}}
	first, second := newFragmentOccurrence(parent), newFragmentOccurrence(parent)
	pages[0].emitBlock(PlacedBox{source: parent, fragmentRef: first, X: 10, Y: 480, Width: 80, Height: 20, FragmentainerIndex: 1})
	pages[0].emitBlock(PlacedBox{source: parent, fragmentRef: second, X: 10, Y: 10, Width: 80, Height: 20, FragmentainerIndex: 1})
	pages[0].emitBlock(PlacedBox{source: child, sourceParent: parent, parentFragmentRef: second, X: 10, Y: 480, Width: 40, Height: 10, FragmentainerIndex: 1})
	finalizePageFragments(root, pages)
	var parentFragments []*FragmentNode
	var childNode *FragmentNode
	var visit func([]*FragmentNode)
	visit = func(nodes []*FragmentNode) {
		for _, node := range nodes {
			if node.Fragment.Source == parent {
				parentFragments = append(parentFragments, node)
			}
			if node.Fragment.Source == child {
				childNode = node
			}
			visit(node.Children)
		}
	}
	visit(pages[0].FragmentTree)
	if childNode == nil || len(parentFragments) != 2 {
		t.Fatalf("fragments parent/child=%d/%v", len(parentFragments), childNode != nil)
	}
	if childNode.Parent != parentFragments[1] {
		t.Fatalf("ocurrencia explicit is ignoredda: got=%p, want=%p", childNode.Parent, parentFragments[1])
	}
}

func TestLayout_FragmentTreeDoesNotInventParentWithoutOccurrence(t *testing.T) {
	root := &boxes.Box{Tag: "html"}
	parent := &boxes.Box{Tag: "section"}
	child := &boxes.Box{Tag: "p"}
	root.Children = []*boxes.Box{parent}
	parent.Children = []*boxes.Box{child}
	pages := []Page{{Number: 1}}
	pages[0].emitBlock(PlacedBox{source: parent, fragmentRef: newFragmentOccurrence(parent)})
	pages[0].emitBlock(PlacedBox{source: child})
	finalizePageFragments(root, pages)
	if len(pages[0].FragmentTree) != 2 {
		t.Fatalf("unbound child must remain to root; roots=%d", len(pages[0].FragmentTree))
	}
	if pages[0].FragmentTree[1].Parent != nil {
		t.Fatal("a source ancestor cannot identify the emitting fragment")
	}
}

func TestLayout_FragmentTreeIsLinkedWhenChildIsEmitted(t *testing.T) {
	parent := &boxes.Box{Tag: "section"}
	child := &boxes.Box{Tag: "p"}
	parent.Children = []*boxes.Box{child}
	parentRef := newFragmentOccurrence(parent)
	page := Page{Number: 1}
	page.emitBlock(PlacedBox{source: parent, fragmentRef: parentRef})
	page.emitBlock(PlacedBox{source: child, sourceParent: parent, parentFragmentRef: parentRef})
	if len(page.FragmentTree) != 1 || len(page.FragmentTree[0].Children) != 1 {
		t.Fatalf("tree must exist at emission: roots=%d", len(page.FragmentTree))
	}
	if page.FragmentTree[0].Children[0].Parent != page.FragmentTree[0] {
		t.Fatal("child must hold the current emitted parent node")
	}
}

func TestLayout_PageGeometryLivesInEmittedOccurrenceUntilClose(t *testing.T) {
	source := &boxes.Box{Tag: "p"}
	page := Page{Number: 1}
	page.emitBlock(PlacedBox{source: source, X: 10, Width: 20})
	if len(page.Blocks) != 0 {
		t.Fatalf("flat blocks must be created only at page close; got %d", len(page.Blocks))
	}
	page.block(0).X = 42
	if page.FragmentTree[0].Placed.X != 42 {
		t.Fatal("bounds mutation did not reach the canonical occurrence")
	}
}

func TestLayout_AllFormattingContextsCloseBlocksFromOccurrences(t *testing.T) {
	cases := []struct {
		name, html string
	}{
		{"block", `<section style="background:#eee;padding:3pt"><p>BLOCK</p></section>`},
		{"flex", `<div style="display:flex"><p>FLEX ONE</p><p>FLEX TWO</p></div>`},
		{"grid", `<div style="display:grid;grid-template-columns:1fr 1fr"><p>GRID ONE</p><p>GRID TWO</p></div>`},
		{"table", `<table><tr><td>TABLE <table><tr><td>INNER</td></tr></table></td></tr></table>`},
		{"columns", `<div style="column-count:2"><p>COLUMN ONE</p><p>COLUMN TWO</p></div>`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, err := boxes.Build(tc.html)
			if err != nil {
				t.Fatal(err)
			}
			pages, _ := flowLayout(root, testOptions())
			for i := range pages {
				if len(pages[i].Blocks) != 0 {
					t.Fatalf("page %d published blocks before close", i+1)
				}
				if pages[i].blockCount() == 0 {
					t.Fatalf("page %d emitted no occurrences", i+1)
				}
			}
			finalizePageFragments(root, pages)
			for i := range pages {
				if len(pages[i].Blocks) != pages[i].blockCount() {
					t.Fatalf("page %d flat view has %d blocks for %d occurrences", i+1, len(pages[i].Blocks), pages[i].blockCount())
				}
				for j, node := range pages[i].paintNodes() {
					if pages[i].Blocks[j].X != node.Placed.X || pages[i].Blocks[j].Y != node.Placed.Y ||
						pages[i].Fragments[j].Bounds.X != node.Placed.X || pages[i].Fragments[j].Bounds.Y != node.Placed.Y {
						t.Fatalf("page %d occurrence %d has divergent geometry", i+1, j)
					}
				}
			}
		})
	}
}

func TestLayout_FragmentTreeRejectsUnresolvedParentOccurrence(t *testing.T) {
	root := &boxes.Box{Tag: "html"}
	parent := &boxes.Box{Tag: "section"}
	child := &boxes.Box{Tag: "p"}
	root.Children = []*boxes.Box{parent}
	parent.Children = []*boxes.Box{child}
	page := Page{Number: 1}
	page.emitBlock(PlacedBox{source: child, sourceParent: parent, parentFragmentRef: newFragmentOccurrence(parent)})
	defer func() {
		if recover() == nil {
			t.Fatal("unresolved explicit parent must fail instead of becoming a root")
		}
	}()
	finalizePageFragments(root, []Page{page})
}

func TestLayout_FragmentTreeKeepsDecorationsWhenNestedPageIsMerged(t *testing.T) {
	root, err := boxes.Build(`<section style="padding:2pt"><table><tr><td><div style="background:#eee"><p style="background:#fee">INNER</p></div></td></tr></table></section>`)
	if err != nil {
		t.Fatal(err)
	}
	page := Flow(root, testOptions())[0]
	cell := findTableCellContaining(root, "INNER")
	if cell == nil {
		t.Fatal("missing source cell")
	}
	cellNode := fragmentNodeForSource(page.FragmentTree, cell)
	if cellNode == nil {
		t.Fatal("missing cell occurrence")
	}
	decorations := 0
	var visit func(*FragmentNode)
	visit = func(node *FragmentNode) {
		for _, child := range node.Children {
			if child.Kind == FragmentDecoration {
				decorations++
				if child.Fragment.Source == nil || child.Fragment.Bounds.Width <= 0 || child.Fragment.PageNumber != page.Number {
					t.Errorf("decoration lost source or page geometry: %+v", child.Fragment)
				}
				if child.Fragment.Bounds.X < cellNode.Fragment.Bounds.X || child.Fragment.Bounds.Y < cellNode.Fragment.Bounds.Y {
					t.Errorf("decoration escaped its translated cell: cell=%+v decoration=%+v", cellNode.Fragment.Bounds, child.Fragment.Bounds)
				}
			}
			visit(child)
		}
	}
	visit(cellNode)
	if decorations != 2 {
		t.Fatalf("nested table decorations=%d, want 2", decorations)
	}
}
