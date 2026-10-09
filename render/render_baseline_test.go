package render

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"testing"

	"github.com/robtrivi/pagecraft-go/fonts"
)

func TestRender_TTFBaselineCentersLeadingForNormalAndExplicitLineHeight(t *testing.T) {
	fontPath := "/usr/share/fonts/noto/NotoSerif-Regular.ttf"
	if _, err := os.Stat(fontPath); err != nil {
		t.Skip("Noto Serif is not installed")
	}
	data, err := os.ReadFile(fontPath)
	if err != nil {
		t.Fatalf("read Noto Serif: %v", err)
	}
	face, err := fonts.Load(data)
	if err != nil {
		t.Fatalf("load Noto Serif: %v", err)
	}
	upm, asc, _, _, _ := face.Head()
	if upm <= 0 {
		t.Fatal("Noto Serif has no units per em")
	}
	fontSize := 12.0 // 16 CSS px
	normalHeight := face.NormalLineHeight(fontSize)
	ascender := float64(asc) * fontSize / float64(upm)
	normalY := 842.0 - 84.0 - ascender // A4, 72pt margin + 12pt UA paragraph margin
	cases := []struct {
		name, lineHeight string
		wantY            float64
	}{
		{"normal", "", normalY},
		{"explicit_1_2", `line-height: 1.2;`, normalY + (normalHeight-14.4)/2},
	}
	position := regexp.MustCompile(`BT /F[0-9]+ 12 Tf\s+1 0 0 1 72 ([0-9.]+) Tm <[0-9A-F]+> Tj`)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			html := fmt.Sprintf(`<style>@font-face { font-family: Compare; src: url(%s); }</style><p style="font-family: Compare; font-size: 16px; margin: 1em 0; %s">baseline</p>`, fontPath, tc.lineHeight)
			pdf, err := RenderHTMLToPDF(html)
			if err != nil {
				t.Fatalf("RenderHTMLToPDF: %v", err)
			}
			match := position.FindStringSubmatch(streamText(t, pdf))
			if len(match) != 2 {
				t.Fatal("TTF baseline position was not found in the content stream")
			}
			gotY, err := strconv.ParseFloat(match[1], 64)
			if err != nil {
				t.Fatalf("parse baseline %q: %v", match[1], err)
			}
			if math.Abs(gotY-tc.wantY) > 0.001 {
				t.Errorf("baseline PDF y: got %g, want %g", gotY, tc.wantY)
			}
		})
	}
}
