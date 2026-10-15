package render

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
)

func TestRender_InlineSVG_EmbedsVectorForm(t *testing.T) {
	pdf, err := RenderHTMLToPDF(`<svg width="40" height="20" viewBox="0 0 40 20"><rect width="40" height="20" fill="#ff0000"/></svg>`)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	if !bytes.Contains(pdf, []byte("/Subtype /Form")) {
		t.Fatal("PDF does not contain an SVG Form XObject")
	}
	if !strings.Contains(streamText(t, pdf), "/Sv1 Do") {
		t.Fatal("content stream does not invoke the SVG Form XObject")
	}
}

func TestRender_InlineSVGInFlexItem_EmbedsBesideText(t *testing.T) {
	pdf, err := RenderHTMLToPDF(`<div style="display:flex"><span><svg style="width:5pt;height:5pt" viewBox="0 0 10 10"><rect width="10" height="10" fill="red"/></svg>BRAND</span><b>DOCUMENT</b></div>`)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	stream := streamText(t, pdf)
	if !bytes.Contains(pdf, []byte("/Subtype /Form")) || !strings.Contains(stream, "/Sv1 Do") || !strings.Contains(stream, "(BRAND) Tj") {
		t.Fatal("inline SVG or adjacent text was omitted from the PDF")
	}
}

func TestRender_InlineSVG_EmitsTextInVectorForm(t *testing.T) {
	pdf, err := RenderHTMLToPDF(`<svg width="100" height="30"><text x="4" y="20" font-size="12">JUL</text></svg>`)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	if !bytes.Contains(pdf, []byte("/Font << /F1 ")) || !bytes.Contains(pdf, []byte("1 0 0 -1 4 20 Tm")) || !bytes.Contains(pdf, []byte("(JUL) Tj")) {
		t.Fatal("SVG Form does not declare a source or emit node text")
	}
}

func TestRender_InlineSVG_AppliesShapeOpacity(t *testing.T) {
	pdf, err := RenderHTMLToPDF(`<svg width="20" height="20"><rect width="20" height="20" fill="red" opacity="0.5"/></svg>`)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	if !bytes.Contains(pdf, []byte("/ca 0.5")) || !bytes.Contains(pdf, []byte("/GS1 gs")) {
		t.Fatal("the PDF does not apply the opacity of the SVG form")
	}
}

func TestRender_ImgDataURI_SVGEmbedsVectorForm(t *testing.T) {
	data := base64.StdEncoding.EncodeToString([]byte(`<svg width="12" height="6"><circle cx="3" cy="3" r="2"/></svg>`))
	pdf, err := RenderHTMLToPDF(`<img src="data:image/svg+xml;base64,` + data + `">`)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	if !bytes.Contains(pdf, []byte("/Subtype /Form")) || !strings.Contains(streamText(t, pdf), "/Sv1 Do") {
		t.Fatal("the <img> SVG was not included as a vector in the PDF")
	}
}
