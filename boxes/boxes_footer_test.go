package boxes

import (
	"reflect"
	"testing"

	"github.com/robtrivi/pagecraft/css"
)

// Footer step 2: @bottom-center reaches the root as chunks.
func TestBoxes_PageFooter_ReachesRoot(t *testing.T) {
	root, err := Build(`<style>@page { @bottom-center { content: "N " counter(page); } }</style><p>x</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []css.ContentChunk{{Literal: "N "}, {Counter: "page"}}
	if !reflect.DeepEqual(root.PageFooter, want) {
		t.Errorf("got %+v, want %+v", root.PageFooter, want)
	}
}

func TestBoxes_PageFooter_Absent(t *testing.T) {
	root, err := Build(`<p>x</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(root.PageFooter) != 0 {
		t.Errorf("got %+v, want empty", root.PageFooter)
	}
}
