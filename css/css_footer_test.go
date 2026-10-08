package css

import (
	"reflect"
	"testing"
)

// Footer step 1: @bottom-center content with literals and counter(page/pages).
func TestCSS_ParsesPageFooter(t *testing.T) {
	got, ok := ParsePageFooter(`@page { size: A4; @bottom-center { content: "Page " counter(page) " of " counter(pages); } }`)
	if !ok {
		t.Fatal("expected footer")
	}
	want := []ContentChunk{
		{Literal: "Page "}, {Counter: "page"}, {Literal: " of "}, {Counter: "pages"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestCSS_ParsesPageFooter_Absent(t *testing.T) {
	if _, ok := ParsePageFooter(`@page { size: A4; }`); ok {
		t.Error("there is no footer without bottom")
	}
	if _, ok := ParsePageFooter(`p { color: red; }`); ok {
		t.Error("there is no footer without @page")
	}
}
