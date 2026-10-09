package render

import (
	"bytes"
	"fmt"
	"math"
	"strings"
	"testing"
)

// @page dimensions define the PDF MediaBox; the resolved margin offsets the
// content origin on both axes, and the top margin is converted to PDF baseline.
func TestRender_PageRuleComposesPhysicalSizeAndMargins(t *testing.T) {
	for _, tc := range []struct {
		name, pageCSS string
		width, height float64
		margin        float64
	}{
		{"named paper in inches", "size: Letter; margin: 1in", 612, 792, 72},
		{"explicit dimensions in points", "size: 400pt 500pt; margin: 20px", 400, 500, 15},
		{"landscape paper in centimeters", "size: A4 landscape; margin: 2cm", 842, 595, 2 * 72 / 2.54},
	} {
		t.Run(tc.name, func(t *testing.T) {
			html := `<style>@page { ` + tc.pageCSS + ` }</style><p style="margin: 0">Hello</p>`
			pdf, err := RenderHTMLToPDF(html)
			if err != nil {
				t.Fatalf("RenderHTMLToPDF: %v", err)
			}
			if box := fmt.Sprintf("/MediaBox [0 0 %g %g]", tc.width, tc.height); !bytes.Contains(pdf, []byte(box)) {
				t.Errorf("PDF MediaBox does not match @page size %q; want %s", tc.pageCSS, box)
			}
			positions := tdAll([]byte(streamText(t, pdf)), "Hello")
			if len(positions) != 1 {
				t.Fatalf("text positions: got %v, want one rendered line", positions)
			}
			wantY := tc.height - tc.margin - 12 // page height - top margin - font baseline offset
			if math.Abs(positions[0][0]-tc.margin) > 0.001 || math.Abs(positions[0][1]-wantY) > 0.001 {
				t.Errorf("text position=(%g,%g), want (%g,%g) from page margins and size", positions[0][0], positions[0][1], tc.margin, wantY)
			}
		})
	}
}

func TestRender_PageRuleAppliesIndependentMargins(t *testing.T) {
	pdf, err := RenderHTMLToPDF(`<style>@page { size: 400pt 500pt; margin: 10pt 20pt 30pt 40pt } body { margin: 0 }</style><p style="margin:0">Hello</p>`)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	positions := tdAll([]byte(streamText(t, pdf)), "Hello")
	if len(positions) != 1 {
		t.Fatalf("text positions: got %v, want one rendered line", positions)
	}
	if math.Abs(positions[0][0]-40) > 0.001 || math.Abs(positions[0][1]-478) > 0.001 {
		t.Errorf("text position=(%g,%g), want (40,478) from left and top @page margins", positions[0][0], positions[0][1])
	}
}

func TestRender_PageRuleThreeValueMarginsMatchCSSExpansion(t *testing.T) {
	pdf, err := RenderHTMLToPDF(`<!doctype html><style>@page { size: A4; margin: 18mm 16mm 20mm } p { margin: 0 }</style><p>Hello</p>`)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	positions := tdAll([]byte(streamText(t, pdf)), "Hello")
	if len(positions) != 1 {
		t.Fatalf("text positions: got %v, want one rendered line", positions)
	}
	wantX := 16*72/25.4 + 6 // horizontal page margin plus the UA body's 8px margin
	wantY := 842 - 18*72/25.4 - 6 - 12
	if math.Abs(positions[0][0]-wantX) > 0.001 || math.Abs(positions[0][1]-wantY) > 0.001 {
		t.Errorf("text position=(%g,%g), want (%g,%g) from three-value page margins", positions[0][0], positions[0][1], wantX, wantY)
	}
}

func TestRender_PageRuleResolvesPercentageMarginsAgainstPageAxes(t *testing.T) {
	pdf, err := RenderHTMLToPDF(`<style>@page { size: 400pt 600pt; margin: 10% 5% } body { margin: 0 }</style><p style="margin:0">Hello</p>`)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	positions := tdAll([]byte(streamText(t, pdf)), "Hello")
	if len(positions) != 1 {
		t.Fatalf("text positions: got %v, want one rendered line", positions)
	}
	if math.Abs(positions[0][0]-20) > 0.001 || math.Abs(positions[0][1]-528) > 0.001 {
		t.Errorf("text position=(%g,%g), want (20,528) from 5%% horizontal and 10%% vertical margins", positions[0][0], positions[0][1])
	}
}

func TestRender_PageRulePaintsPageBackgroundColor(t *testing.T) {
	pdf, err := RenderHTMLToPDF(`<style>@page { size: 400pt 500pt; background-color: #123456 } body { margin: 0 }</style><p>Page</p>`)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	content := streamText(t, pdf)
	if !strings.Contains(content, "0.07058823529411765 0.20392156862745098 0.33725490196078434 rg\n0 0 400 500 re f") {
		t.Errorf("page background was not painted across the page; stream:\n%s", content)
	}
}

func TestRender_PageRuleCompositesTranslucentBackgroundColor(t *testing.T) {
	pdf, err := RenderHTMLToPDF(`<style>@page { background-color: rgba(255,0,0,0.5) } body { margin: 0 }</style><p>Page</p>`)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	if !bytes.Contains(pdf, []byte("/ca 0.5")) || !strings.Contains(streamText(t, pdf), "/GS1 gs") {
		t.Error("page background alpha was not represented by a PDF graphics state")
	}
}

func TestRender_PageRuleAppliesOutputRotation(t *testing.T) {
	pdf, err := RenderHTMLToPDF(`<style>@page { size: 400pt 500pt; page-orientation: rotate-left }</style><p>Page</p>`)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	if !bytes.Contains(pdf, []byte("/Rotate 270")) {
		t.Error("PDF page dictionary does not contain the counter-clockwise page rotation")
	}
}

func TestRender_PageRuleAddsBleedBoxesAndCropMarks(t *testing.T) {
	pdf, err := RenderHTMLToPDF(`<style>@page { size: 400pt 500pt; margin: 20pt; bleed: 3mm; marks: crop }</style><p>Page</p>`)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	for _, want := range []string{
		"/MediaBox [-26.503937007874015 -26.503937007874015 426.503937007874 526.503937007874]",
		"/CropBox [-26.503937007874015 -26.503937007874015 426.503937007874 526.503937007874]",
		"/BleedBox [-8.503937007874015 -8.503937007874015 408.503937007874 508.503937007874]",
		"/TrimBox [0 0 400 500]",
	} {
		if !bytes.Contains(pdf, []byte(want)) {
			t.Errorf("missing PDF page box %q", want)
		}
	}
	if !strings.Contains(streamText(t, pdf), " S\n") {
		t.Error("crop marks were not painted")
	}
}
