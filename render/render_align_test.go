package render

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/robtrivi/pagecraft/text"
)

// Text-align step 3: center/right shift X per line; justify uses word
// spacing (Tw) except on the final line.
func TestRender_TextAlignmentPositionsMeasuredLine(t *testing.T) {
	const pageLeft, contentWidth, fontSize, baseline = 72.0, 451.0, 12.0, 758.0
	lineWidth := text.Measure("Hi", fontSize)
	for _, tc := range []struct {
		align string
		wantX float64
	}{
		{"left", pageLeft},
		{"center", pageLeft + (contentWidth-lineWidth)/2},
		{"right", pageLeft + contentWidth - lineWidth},
	} {
		t.Run(tc.align, func(t *testing.T) {
			html := `<p style="margin: 0; text-align: ` + tc.align + `">Hi</p>`
			got, err := RenderHTMLToPDF(html)
			if err != nil {
				t.Fatalf("RenderHTMLToPDF: %v", err)
			}
			want := fmt.Sprintf("%g %g Td (Hi) Tj ET", tc.wantX, baseline)
			if !strings.Contains(streamText(t, got), want) {
				t.Errorf("PDF stream does not place the measured line at %q", want)
			}
		})
	}
}

func TestRender_Justify_SpacesWords(t *testing.T) {
	long := strings.Repeat("ab ", 40)
	left, err := RenderHTMLToPDF(`<p style="margin: 0">` + long + `</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	just, err := RenderHTMLToPDF(`<p style="margin: 0; text-align: justify">` + long + `</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// For the same word and line position, justification expands the gaps.
	found := false
	for _, pl := range tdAll([]byte(streamText(t, left)), "ab") {
		for _, pj := range tdAll([]byte(streamText(t, just)), "ab") {
			if pl[1] == pj[1] && pl[0] != pj[0] {
				found = true
			}
		}
	}
	if !found {
		t.Error("justified text did not expand positions relative to left-aligned text")
	}
	if strings.Contains(streamText(t, just), " Tw\n") {
		t.Error("the Tw mechanism is obsolete for runs")
	}
}

// tdAll extracts (X,y) pairs from "Td (word)".
func tdAll(pdf []byte, word string) [][2]float64 {
	var out [][2]float64
	marker := " Td (" + word + ")"
	for i := 0; i+len(marker) < len(pdf); {
		j := bytes.Index(pdf[i:], []byte(marker))
		if j < 0 {
			break
		}
		// Move back to "BT /F? .. Tf X y"
		start := i + j
		line0 := bytes.LastIndex(pdf[:start], []byte("\n"))
		var fsize, x, y float64
		var fname string
		if _, err := fmt.Sscanf(string(pdf[line0+1:start+len(marker)]), "BT /%s %g Tf %g %g Td ("+word+")", &fname, &fsize, &x, &y); err == nil {
			out = append(out, [2]float64{x, y})
		}
		i = start + len(marker)
	}
	return out
}
