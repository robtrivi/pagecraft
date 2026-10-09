package render

import (
	"bytes"
	"strings"
	"testing"
)

// Inline step 2b: runs have their own style (sample text in F2 bold), and
// links use precise rectangles rather than the whole block.
func TestRender_InlineRuns_DrawsStyles(t *testing.T) {
	got, err := RenderHTMLToPDF(`<p>Hello <b>world</b> end</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := streamText(t, got)
	if !strings.Contains(text, "(world) Tj") {
		t.Error("bold run is missing")
	}
	if !strings.Contains(text, "BT /F2") {
		t.Error("the run bold must use F2")
	}
	if !strings.Contains(text, "BT /F1") {
		t.Error("the remaining run must use F1")
	}
}

func TestRender_InlineLink_PreciseRect(t *testing.T) {
	got, err := RenderHTMLToPDF(`<p>see <a href="https://x.test/q">here</a> end</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Contains(got, []byte("/URI (https://x.test/q)")) {
		t.Fatal("annotation is missing")
	}
	// The rectangle starts where "here" begins (X>72), not at the block edge.
	idx := bytes.Index(got, []byte("/Rect ["))
	if idx < 0 {
		t.Fatal("without rect")
	}
	rect := string(got[idx : idx+12])
	if len(rect) > 8 && rect[:10] == "/Rect [72 " {
		t.Errorf("rectangle is imprecise (covers the whole block): %q", rect)
	}
}
