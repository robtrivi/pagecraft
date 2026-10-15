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

func TestCSS_ParsesPageFooterStyle(t *testing.T) {
	chunks, style, ok := ParsePageFooterStyle(`@page { @bottom-center { content: "Page " counter(page); color: #53636b; font: 7pt Arial, sans-serif; } }`)
	if !ok || len(chunks) != 2 {
		t.Fatalf("footer: chunks=%+v ok=%v", chunks, ok)
	}
	if style.FontSize != 7 {
		t.Errorf("font size=%g, want 7pt", style.FontSize)
	}
	wantColor := [4]float64{83.0 / 255, 99.0 / 255, 107.0 / 255, 1}
	if !style.HasColor || style.Color != wantColor {
		t.Errorf("color=%v has=%v, want %v", style.Color, style.HasColor, wantColor)
	}
}
