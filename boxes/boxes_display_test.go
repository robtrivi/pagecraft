package boxes

import "testing"

func TestBoxes_DisplayNone_RemovesElementFromBoxTree(t *testing.T) {
	root, err := Build(`<style>.demo { display: none }</style><p>before</p><div class="demo">hidden text</div><p>after</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(root.Children) != 2 {
		t.Fatalf("root children: got %d, want 2 visible elements", len(root.Children))
	}
	if root.Children[0].Text != "before" || root.Children[1].Text != "after" {
		t.Errorf("visible text = %q, %q; want before, after", root.Children[0].Text, root.Children[1].Text)
	}
}
