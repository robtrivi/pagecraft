package boxes

import "testing"

func TestBoxes_ParsesColumnRule(t *testing.T) {
	root, err := Build(`<div style="column-count:3;column-rule:1pt solid #cbd5e1"></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	box := root.Children[0]
	if box.Style.ColumnRuleWidth != 1 || box.Style.ColumnRuleColor != [4]float64{203.0 / 255, 213.0 / 255, 225.0 / 255, 1} {
		t.Errorf("column-rule: width=%v color=%v", box.Style.ColumnRuleWidth, box.Style.ColumnRuleColor)
	}
}
