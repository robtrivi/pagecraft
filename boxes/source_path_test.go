package boxes

import "testing"

func TestBoxes_BuildAssignsStableSourcePaths(t *testing.T) {
	root, err := Build(`<section><p>A</p><p>B</p></section>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	section := root.Children[0]
	first, second := section.Children[0], section.Children[1]
	if !equalPath(section.SourcePath(), []int{0}) || !equalPath(first.SourcePath(), []int{0, 0}) || !equalPath(second.SourcePath(), []int{0, 1}) {
		t.Fatalf("source paths: section=%v first=%v second=%v", section.SourcePath(), first.SourcePath(), second.SourcePath())
	}
	copy := *second
	if !equalPath(copy.SourcePath(), second.SourcePath()) {
		t.Errorf("copied box lost source path: copy=%v original=%v", copy.SourcePath(), second.SourcePath())
	}
}

func equalPath(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
