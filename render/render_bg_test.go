package render

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/robtrivi/pagecraft/boxes"
	"github.com/robtrivi/pagecraft/layout"
)

func TestRender_InlineBlockBackgroundUsesIntrinsicWidth(t *testing.T) {
	html := `<style>@page{size:240pt 100pt;margin:20pt}body{margin:0}div{width:200pt;margin:0}p{display:inline-block;background:#dcebe8;padding:2pt 4pt;margin:0;font:10pt Helvetica}</style><div><p>NOTICE</p></div>`
	pdf, err := RenderHTMLToPDF(html)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	content := streamText(t, pdf)
	background := regexp.MustCompile(`0\.8627450980392157 0\.9215686274509803 0\.9098039215686274 rg\s+([0-9.]+) ([0-9.]+) ([0-9.]+) ([0-9.]+) re f`)
	match := background.FindStringSubmatch(content)
	if len(match) != 5 {
		t.Fatalf("inline-block background rectangle not found; content stream:\n%s", content)
	}
	width, err := strconv.ParseFloat(match[3], 64)
	if err != nil {
		t.Fatalf("parse inline-block background width: %v", err)
	}
	if width >= 100 {
		t.Errorf("inline-block background width=%gpt, want intrinsic text width well below its 200pt containing block", width)
	}
}

func TestRender_InlineBlockTopMarginMovesItsBaselineContent(t *testing.T) {
	render := func(marginTop string) []pdfWord {
		html := `<style>@page{size:240pt 100pt;margin:20pt}body{margin:0}div{width:200pt;margin:0}p{display:inline-block;background:#dcebe8;padding:2pt 4pt;margin:0;font:10pt Helvetica}</style><div><p style="margin-top:` + marginTop + `">NOTICE</p></div>`
		pdf, err := RenderHTMLToPDF(html)
		if err != nil {
			t.Fatalf("RenderHTMLToPDF: %v", err)
		}
		return wordsFromPDF(t, pdf)
	}
	wordY := func(words []pdfWord) float64 {
		for _, word := range words {
			if word.Text == "NOTICE" {
				return word.Y
			}
		}
		t.Fatal("NOTICE is missing from the PDF")
		return 0
	}
	withoutMargin := wordY(render("0pt"))
	withMargin := wordY(render("10pt"))
	if got := withMargin - withoutMargin; got < 9.5 || got > 10.5 {
		t.Errorf("inline-block margin-top moved its text by %gpt, want 10pt as Chromium does", got)
	}
}

func TestRender_InlineBlockBackgroundUsesItsOwnLineHeight(t *testing.T) {
	html := `<style>@page{size:240pt 100pt;margin:20pt}body{margin:0}div{width:200pt;margin:0;font:9pt Helvetica;line-height:1}p{display:inline-block;margin:0;background:#dcebe8;padding:1.5mm 2mm;font:7pt Helvetica;line-height:1.45}</style><div><p>NOTICE</p></div>`
	pdf, err := RenderHTMLToPDF(html)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	content := streamText(t, pdf)
	background := regexp.MustCompile(`0\.8627450980392157 0\.9215686274509803 0\.9098039215686274 rg\s+([0-9.]+) ([0-9.]+) ([0-9.]+) ([0-9.]+) re f`)
	match := background.FindStringSubmatch(content)
	if len(match) != 5 {
		t.Fatalf("inline-block background rectangle not found; content stream:\n%s", content)
	}
	height, err := strconv.ParseFloat(match[4], 64)
	if err != nil {
		t.Fatalf("parse inline-block background height: %v", err)
	}
	if height < 18.5 || height > 18.8 {
		t.Errorf("inline-block background height=%gpt, want 18.65pt from 7pt × 1.45 line-height and 1.5mm vertical padding", height)
	}
}

func TestRender_FlexHeaderCompositionMatchesChromium(t *testing.T) {
	html := `<style>@page{size:A4;margin:13mm 14mm 16mm}*{box-sizing:border-box}body{margin:0;color:#202a31;font:9pt Arial,sans-serif}.mast{display:flex;justify-content:space-between;gap:10mm;padding:7mm 0 5mm;border-bottom:1px solid #cbd3d5}.issuer{width:62%}.brand{font-size:17pt;letter-spacing:.3px;font-weight:800;color:#173b45}.legal{font-weight:bold;margin:3mm 0 1mm}p{margin:1mm 0;line-height:1.45}.ribbon{background:#dcebe8;color:#173b45;display:inline-block;padding:1.5mm 2mm;font-size:7pt;font-weight:bold;margin-top:2mm}.doc{width:34%;border-left:1px solid #ccd4d7;padding-left:5mm}.doc-label,label{color:#66737a;font-size:6.5pt;font-weight:bold;letter-spacing:.7px;display:block;margin-bottom:1.5mm}h1{color:#173b45;font-size:22pt;margin:1mm 0 2mm;letter-spacing:1px}.doc p{margin:1mm 0}.buyer{display:flex;gap:8mm;padding:5mm 0;border-bottom:1px solid #d5dcde}.buyer>div{flex:1}</style><header class="mast"><div class="issuer"><div class="brand">NORTHSTAR SUPPLY <small>Co.</small></div><p class="legal">NORTHSTAR SUPPLY CO. · REGISTRATION 1792145876001</p><p>Head office: 123 Harbor Avenue and Pine Street, Northport<br>Branch 001: 451 Market Street, Northport</p><p class="ribbon">NOTICE: REGISTERED BUSINESS · GENERAL TAX REGIME</p></div><div class="doc"><div class="doc-label">ELECTRONIC DOCUMENT</div><h1>INVOICE</h1><strong>001-001-000004286</strong><p>Issued · 14/10/2026</p><p>Environment · TEST</p></div></header><div class="buyer"><div><label>BUYER</label><strong>Customer</strong></div><div>TRANSACTION DETAILS</div></div>`
	pdf, err := RenderHTMLToPDF(html)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	words := wordsFromPDF(t, pdf)
	for _, tc := range []struct {
		text string
		want float64
	}{
		{text: "NOTICE:", want: 140.665343},
		{text: "BUYER", want: 186.120241},
	} {
		var got float64
		found := false
		for _, word := range words {
			if word.Text == tc.text {
				got, found = word.Y, true
				break
			}
		}
		if !found {
			t.Fatalf("%q is missing from the first page", tc.text)
		}
		if math.Abs(got-tc.want) > 1.0 {
			t.Errorf("%s top=%gpt, Chromium composition is %gpt (±1pt)", tc.text, got, tc.want)
		}
	}
}

// Background/borders step 3: a box with background and border emits a filled
// and stroked rectangle in the correct colors before its text.
func TestRender_DrawsBackgroundAndBorder(t *testing.T) {
	got, err := RenderHTMLToPDF(`<p style="background-color: #ff0000; border: 2px solid black">x</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := streamText(t, got)
	for _, want := range []string{"1 0 0 rg", " re f", "1.5 w", "0 0 0 RG", " re S"} {
		if !strings.Contains(text, want) {
			t.Errorf("the PDF does not contain %q", want)
		}
	}
}

func TestRender_BorderRadiusUsesCurvedBackgroundAndBorder(t *testing.T) {
	pdf, err := RenderHTMLToPDF(`<div style="margin:0;background-color:#dcfce7;border:2pt solid #16a34a;border-radius:12pt;padding:14pt">Rounded background</div>`)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	content := streamText(t, pdf)
	if !strings.Contains(content, "84 724.656 m") || !strings.Contains(content, " c\n") {
		t.Error("the background and border should use a curved path")
	}
	if strings.Contains(content, "re f") {
		t.Error("the rounded background must not be painted as a rectangle")
	}
}

func TestRender_BackgroundColorDoesNotLeakIntoText(t *testing.T) {
	pdf, err := RenderHTMLToPDF(`<p style="background-color: #ddd">Gray</p><p>Black</p>`)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	content := streamText(t, pdf)
	if !strings.Contains(content, "0.8666666666666667 0.8666666666666667 0.8666666666666667 rg") {
		t.Fatal("expected gray fill was not found")
	}
	if !strings.Contains(content, " re f\nQ\n0 0 0 rg\nBT /F1") {
		t.Fatal("background graphics state was not restored before drawing text")
	}
}

func TestRender_NormalizesAsymmetricBorderRadiiToFitBox(t *testing.T) {
	tests := []struct {
		name     string
		w, h     float64
		in, want [4]ellipseRadius
	}{
		{name: "fits", w: 100, h: 60, in: [4]ellipseRadius{{10, 10}, {20, 20}, {15, 15}, {5, 5}}, want: [4]ellipseRadius{{10, 10}, {20, 20}, {15, 15}, {5, 5}}},
		{name: "horizontal", w: 40, h: 60, in: [4]ellipseRadius{{30, 30}, {30, 30}, {10, 10}, {10, 10}}, want: [4]ellipseRadius{{20, 20}, {20, 20}, {6.6666666667, 6.6666666667}, {6.6666666667, 6.6666666667}}},
		{name: "vertical", w: 100, h: 30, in: [4]ellipseRadius{{40, 40}, {20, 20}, {10, 10}, {30, 30}}, want: [4]ellipseRadius{{17.1428571429, 17.1428571429}, {8.5714285714, 8.5714285714}, {4.2857142857, 4.2857142857}, {12.8571428571, 12.8571428571}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeEllipticalRadii(tc.w, tc.h, tc.in)
			for i := range got {
				if math.Abs(got[i].X-tc.want[i].X) > 0.000001 || math.Abs(got[i].Y-tc.want[i].Y) > 0.000001 {
					t.Errorf("radius[%d]=%+v, want %+v; all=%v", i, got[i], tc.want[i], got)
				}
			}
		})
	}
}

func TestRender_DrawsEachAsymmetricCornerRadius(t *testing.T) {
	var path strings.Builder
	drawRoundedEllipseRect(&path, 0, 0, 100, 50, [4]ellipseRadius{{20, 20}, {2, 2}, {10, 10}, {4, 4}})
	got := path.String()
	if !strings.HasPrefix(got, "20 0 m\n98 0 l\n") {
		t.Fatalf("left/right corner radii must differ: %s", got)
	}
	for _, want := range []string{"100 2 c\n", "90 50 c\n", "4 50 l\n", "0 20 l\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("path does not contain the expected segment %q: %s", want, got)
		}
	}
}

func TestRender_NormalizesEllipticalRadiiTogether(t *testing.T) {
	input := [4]ellipseRadius{{X: 80, Y: 30}, {X: 80, Y: 30}, {X: 10, Y: 10}, {X: 10, Y: 10}}
	got := normalizeEllipticalRadii(100, 50, input)
	want := [4]ellipseRadius{{X: 50, Y: 18.75}, {X: 50, Y: 18.75}, {X: 6.25, Y: 6.25}, {X: 6.25, Y: 6.25}}
	for i := range got {
		if math.Abs(got[i].X-want[i].X) > 0.000001 || math.Abs(got[i].Y-want[i].Y) > 0.000001 {
			t.Errorf("radii[%d]=%+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestRender_DrawsEllipticalCornerPath(t *testing.T) {
	var path strings.Builder
	drawRoundedEllipseRect(&path, 0, 0, 100, 50, [4]ellipseRadius{{X: 50, Y: 25}, {X: 50, Y: 25}, {X: 50, Y: 25}, {X: 50, Y: 25}})
	got := path.String()
	if !strings.HasPrefix(got, "50 0 m\n50 0 l\n") {
		t.Fatalf("curve must preserve different X/y radii: %s", got)
	}
	if !strings.Contains(got, "100 25 c\n") {
		t.Fatalf("vertical arc must end at 25pt: %s", got)
	}
}

func TestRender_PercentageRadiiResolveAgainstEachBoxAxis(t *testing.T) {
	percent := boxes.RadiusLength{Value: 50, Percent: true}
	box := layout.PlacedBox{BorderCornerRadii: [4]boxes.CornerRadius{
		{X: percent, Y: percent}, {X: percent, Y: percent}, {X: percent, Y: percent}, {X: percent, Y: percent},
	}}
	got := resolveBorderCornerRadii(box, 100, 50)
	for i, radius := range got {
		if radius.X != 50 || radius.Y != 25 {
			t.Errorf("corner[%d]=%+v, want X=50 y=25", i, radius)
		}
	}
}
