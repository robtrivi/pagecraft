package boxes

import (
	"testing"
)

// List markers are text prefixes such as bullets, decimal numbers, letters, or none.
// Not hanging indent or nested markers in v1.
func TestBoxes_ListMarkers(t *testing.T) {
	root, err := Build(`<ul><li>a</li><li>b</li></ul><ol start="5"><li>c</li><li>d</li></ol><ol style="list-style-type: none"><li>e</li></ol>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"• a", "• b", "5. c", "6. d", "e"}
	var got []string
	var walk func(*Box)
	walk = func(b *Box) {
		if b.Tag == "li" {
			got = append(got, b.Text)
		}
		for _, c := range b.Children {
			walk(c)
		}
	}
	walk(root)
	if len(got) != len(want) {
		t.Fatalf("items: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("item %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestBoxes_ListAlphaRoman(t *testing.T) {
	root, err := Build(`<ol style="list-style-type: lower-alpha"><li>a</li><li>b</li></ol><ol style="list-style-type: upper-roman"><li>c</li><li>d</li></ol>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got []string
	var walk func(*Box)
	walk = func(b *Box) {
		if b.Tag == "li" {
			got = append(got, b.Text)
		}
		for _, c := range b.Children {
			walk(c)
		}
	}
	walk(root)
	want := []string{"a. a", "b. b", "I. c", "II. d"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("item %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestBoxes_NestedListTextStaysInChildListItem(t *testing.T) {
	root, err := Build(`<ol><li>Preparation<ul><li>create the request</li></ul></li></ol>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	outer := root.Children[0].Children[0]
	if got, want := outer.Text, "1. Preparation"; got != want {
		t.Errorf("text of the li parent: got %q, want %q", got, want)
	}
	inner := outer.Children[0].Children[0]
	if got, want := inner.Text, "o create the request"; got != want {
		t.Errorf("text of the li nested: got %q, want %q", got, want)
	}
}

func TestBoxes_NestedUnstyledULUsesCircleMarker(t *testing.T) {
	root, err := Build(`<ul><li>outer<ul><li>inner</li></ul></li></ul>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	inner := root.Children[0].Children[0].Children[0].Children[0]
	if got, want := inner.Text, "o inner"; got != want {
		t.Errorf("marker nested: got %q, want %q", got, want)
	}
}

func TestBoxes_ListMarkersApplyStylesheetType(t *testing.T) {
	root, err := Build(`<style>ol.roman { list-style-type: upper-roman }</style><ol class="roman"><li>first</li><li>second</li></ol>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	list := root.Children[0]
	if got, want := list.Children[0].Text, "I. first"; got != want {
		t.Errorf("first marker: got %q, want %q", got, want)
	}
	if got, want := list.Children[1].Text, "II. second"; got != want {
		t.Errorf("second marker: got %q, want %q", got, want)
	}
}

func TestBoxes_ListMarkerShapeTracksCircleAndSquareStyle(t *testing.T) {
	root, err := Build(`<ul style="list-style-type:circle"><li>round</li></ul><ul style="list-style-type:square"><li>square</li></ul>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if got := root.Children[0].Children[0].Style.ListMarkerShape; got != "circle" {
		t.Errorf("circle marker shape=%q", got)
	}
	if got := root.Children[1].Children[0].Style.ListMarkerShape; got != "square" {
		t.Errorf("square marker shape=%q", got)
	}
}
