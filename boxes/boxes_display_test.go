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

func TestBoxes_ParsesCSSDisplayTableRoles(t *testing.T) {
	root, err := Build(`<div style="display:table"><div style="display:table-row"><span style="display:table-cell">A</span></div></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	table := root.Children[0]
	row := table.Children[0]
	cell := row.Children[0]
	if table.Style.Display != DisplayTable || row.Style.Display != DisplayTableRow || cell.Style.Display != DisplayTableCell {
		t.Fatalf("display roles = %q/%q/%q, want table/table-row/table-cell", table.Style.Display, row.Style.Display, cell.Style.Display)
	}
}
