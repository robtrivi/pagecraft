package layout

import (
	"testing"

	"github.com/robtrivi/pagecraft-go/boxes"
)

func TestLayout_BoxInsetsCombinePaddingAndEachBorderSide(t *testing.T) {
	box := &boxes.Box{
		Style: boxes.ComputedStyle{
			PaddingTop: 1, PaddingRight: 2, PaddingBottom: 3, PaddingLeft: 4,
			BorderTop: boxes.BorderSide{Width: 5}, BorderRight: boxes.BorderSide{Width: 6},
			BorderBottom: boxes.BorderSide{Width: 7}, BorderLeft: boxes.BorderSide{Width: 8},
		},
	}
	want := containerInsets{top: 6, right: 8, bottom: 10, left: 12}
	if got := boxInsets(box); got != want {
		t.Errorf("box insets=%+v, want %+v", got, want)
	}
}

func TestLayout_FragmentBoxSlicesOnlyContinuationEdges(t *testing.T) {
	source := &boxes.Box{
		Style: boxes.ComputedStyle{
			PaddingTop: 2, PaddingRight: 3, PaddingBottom: 4, PaddingLeft: 5,
			BorderTop: boxes.BorderSide{Width: 6}, BorderRight: boxes.BorderSide{Width: 7},
			BorderBottom: boxes.BorderSide{Width: 8}, BorderLeft: boxes.BorderSide{Width: 9},
		},
	}
	for _, scenario := range []struct {
		name                            string
		first, last                     bool
		wantPaddingTop                  float64
		wantPaddingBottom               float64
		wantTopBorder, wantBottomBorder bool
	}{
		{name: "single", first: true, last: true, wantPaddingTop: 2, wantPaddingBottom: 4, wantTopBorder: true, wantBottomBorder: true},
		{name: "first", first: true, wantPaddingTop: 2, wantTopBorder: true},
		{name: "middle"},
		{name: "last", last: true, wantPaddingBottom: 4, wantBottomBorder: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			got := fragmentBox(source, scenario.first, scenario.last)
			if got.Style.PaddingTop != scenario.wantPaddingTop || (got.Style.BorderTop.Width > 0) != scenario.wantTopBorder {
				t.Errorf("decorationtop=%v/%v, want %v/present=%v", got.Style.PaddingTop, got.Style.BorderTop.Width, scenario.wantPaddingTop, scenario.wantTopBorder)
			}
			if got.Style.PaddingBottom != scenario.wantPaddingBottom || (got.Style.BorderBottom.Width > 0) != scenario.wantBottomBorder {
				t.Errorf("decorationbottom=%v/%v, want %v/present=%v", got.Style.PaddingBottom, got.Style.BorderBottom.Width, scenario.wantPaddingBottom, scenario.wantBottomBorder)
			}
			if got.Style.PaddingLeft != source.Style.PaddingLeft || got.Style.PaddingRight != source.Style.PaddingRight || got.Style.BorderLeft != source.Style.BorderLeft || got.Style.BorderRight != source.Style.BorderRight {
				t.Errorf("the fragment must preserve the decoration lateral: %+v", got)
			}
		})
	}
	if source.Style.PaddingTop != 2 || source.Style.BorderTop.Width != 6 || source.Style.PaddingBottom != 4 || source.Style.BorderBottom.Width != 8 {
		t.Errorf("fragmentation mutated the source box: %+v", source)
	}
}

func TestLayout_FragmentPlacedBoxDrawsBordersOnlyAtOuterEdges(t *testing.T) {
	source := PlacedBox{
		BorderWidth: 3,
		BorderTop:   boxes.BorderSide{Width: 1}, BorderRight: boxes.BorderSide{Width: 2},
		BorderBottom: boxes.BorderSide{Width: 3}, BorderLeft: boxes.BorderSide{Width: 4},
	}
	for _, scenario := range []struct {
		name                string
		first, last         bool
		wantTop, wantBottom bool
	}{
		{name: "single", first: true, last: true, wantTop: true, wantBottom: true},
		{name: "first", first: true, wantTop: true},
		{name: "middle"},
		{name: "last", last: true, wantBottom: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			got := fragmentPlacedBox(source, scenario.first, scenario.last)
			if got.BorderWidth != 0 {
				t.Errorf("remaining uniform border=%v, want 0", got.BorderWidth)
			}
			if (got.BorderTop.Width != 0) != scenario.wantTop || (got.BorderBottom.Width != 0) != scenario.wantBottom {
				t.Errorf("vertical borders=%v/%v, want present=%v/%v", got.BorderTop.Width, got.BorderBottom.Width, scenario.wantTop, scenario.wantBottom)
			}
			if got.BorderLeft != source.BorderLeft || got.BorderRight != source.BorderRight {
				t.Errorf("side borders changed: %+v", got)
			}
		})
	}
}
