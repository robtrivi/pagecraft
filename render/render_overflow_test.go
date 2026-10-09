package render

import (
	"strings"
	"testing"
)

func TestRender_OverflowHiddenEmitsScopedPDFClip(t *testing.T) {
	pdf, err := RenderHTMLToPDF(`<div style="width:100pt;height:40pt;overflow:hidden"><p style="white-space:nowrap">horizontally overflowing content</p><p>second block</p></div><p>visible</p>`)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	content := streamText(t, pdf)
	if !strings.Contains(content, " re W n\n") {
		t.Fatalf("missing PDF rectangular clipping operator: %s", content)
	}
	clipAt := strings.Index(content, " re W n\n")
	clipEnd := strings.Index(content[clipAt:], "\nQ\n")
	visibleAt := strings.Index(content, "(visib)")
	if clipEnd < 0 || visibleAt < 0 || clipAt+clipEnd >= visibleAt {
		t.Fatalf("the clip must be restored after the clipped block and before the following text")
	}
}

func TestRender_OverflowHiddenInsideOpacityRestoresNestedClip(t *testing.T) {
	pdf, err := RenderHTMLToPDF(`<div style="opacity:.8"><div style="width:100pt;height:30pt;overflow:hidden"><p>clipped</p></div></div><p>visible</p>`)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	content := streamText(t, pdf)
	if !strings.Contains(content, " re W n\n") {
		t.Fatal("the inner clip must be emitted in the opacity group")
	}
	if !strings.Contains(content, " re W n\n") || !strings.Contains(content, "Q\n") {
		t.Fatal("clip graphics state must be restored")
	}
}
