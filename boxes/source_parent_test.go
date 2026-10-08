package boxes

import "testing"

func TestBoxes_BuildStoresDirectSourceParents(t *testing.T) {
	root, err := Build(`<main><section><p>parent chain</p></section></main>`)
	if err != nil {
		t.Fatal(err)
	}
	main := root.Children[0]
	section := main.Children[0]
	paragraph := section.Children[0]
	if main.Parent != root || section.Parent != main || paragraph.Parent != section {
		t.Fatalf("parent pointers do not reflect the source tree: main=%p/%p section=%p/%p p=%p/%p", main.Parent, root, section.Parent, main, paragraph.Parent, section)
	}
}
