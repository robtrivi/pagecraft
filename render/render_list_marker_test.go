package render

import (
	"strings"
	"testing"
)

func TestRender_ListCircleAndSquareUseVectorMarkers(t *testing.T) {
	pdf, err := RenderHTMLToPDF(`<ul style="list-style-type:circle"><li>round</li></ul><ul style="list-style-type:square"><li>square</li></ul>`)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	content := streamText(t, pdf)
	if !strings.Contains(content, "3 Tr") {
		t.Fatal("marker alternative text must be preserved without being painted below the vector")
	}
	if !strings.Contains(content, " c S\n") {
		t.Fatal("circle must be drawn as an unfilled vector path")
	}
	if !strings.Contains(content, " re f\n") {
		t.Fatal("square must be drawn as a filled vector rectangle")
	}
}
