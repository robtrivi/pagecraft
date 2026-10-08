package boxes

import (
	"testing"
)

// Links step 1: <a href> stores the destination on the box (raw, without
// base resolution is not supported in v1. Without href, the value is empty.
func TestBoxes_Anchor_SetsLink(t *testing.T) {
	root, err := Build(`<a href="https://x.test/q">go</a>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := root.Children[0]
	if b.Link != "https://x.test/q" {
		t.Errorf("link: got %q, want %q", b.Link, "https://x.test/q")
	}
	if b.Text != "go" {
		t.Errorf("text: got %q, want %q", b.Text, "go")
	}
}

func TestBoxes_AnchorWithoutHref_Empty(t *testing.T) {
	root, err := Build(`<a name="x">y</a>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l := root.Children[0].Link; l != "" {
		t.Errorf("link: got %q, want empty", l)
	}
}
