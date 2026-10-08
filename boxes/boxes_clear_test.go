package boxes

import "testing"

func TestBoxes_ParsesClearProperty(t *testing.T) {
	root, err := Build(`<p style="clear:left">left</p><p style="clear:both">both</p><p style="clear:right">right</p><p style="clear:none">none</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	want := []ClearSide{ClearLeft, ClearBoth, ClearRight, ClearNone}
	for index, value := range want {
		if got := root.Children[index].Style.Clear; got != value {
			t.Errorf("clear[%d]=%q, want %q", index, got, value)
		}
	}
}
