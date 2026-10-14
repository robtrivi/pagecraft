package render

import (
	"regexp"
	"strings"
	"testing"
)

func TestRender_TableFrameStaysInsideAsymmetricPageMargins(t *testing.T) {
	html := `<style>@page { size: 200pt 200pt; margin: 20pt 20pt 20pt 36pt } body { margin: 0 } table { border-collapse: separate; border: 2pt solid red; margin: 0; width: 100% } td { padding: 2pt }</style><table><tr><td>LEFT</td><td>RIGHT</td></tr></table>`
	pdf, err := RenderHTMLToPDF(html)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	content := streamText(t, pdf)
	for name, pattern := range map[string]string{
		"left border":  `1 0 0 rg 36 [0-9.]+ 2 [0-9.]+ re f`,
		"right border": `1 0 0 rg 178 [0-9.]+ 2 [0-9.]+ re f`,
	} {
		if !regexp.MustCompile(pattern).MatchString(content) {
			t.Errorf("table %s is missing at the resolved page edge; content stream:\n%s", name, content)
		}
	}
}

func TestRender_TableBackgroundLayersPaintInCSSOrder(t *testing.T) {
	html := `<table style="background:#fff2cc"><thead style="background:#163b64;color:white"><tr style="background:#008000"><th>Measure</th></tr></thead><tbody style="background:#e6e6e6"><tr style="background:#0000ff"><td>Reading</td></tr></tbody></table>`
	pdf, err := RenderHTMLToPDF(html)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	content := streamText(t, pdf)
	colors := []string{
		"1 0.9490196078431372 0.8 rg",                                    // table
		"0.08627450980392157 0.23137254901960785 0.39215686274509803 rg", // thead
		"0 0.5019607843137255 0 rg",                                      // header row
		"0.9019607843137255 0.9019607843137255 0.9019607843137255 rg",    // tbody
		"0 0 1 rg", // body row
	}
	previous := -1
	for _, color := range colors {
		index := strings.Index(content, color)
		if index < 0 {
			t.Fatalf("table paint color %q was not emitted", color)
		}
		if index <= previous {
			t.Fatalf("table paint layers are out of order at %q", color)
		}
		previous = index
	}
	if !strings.Contains(content, "(Measure)") || !strings.Contains(content, "(Reading)") {
		t.Fatal("table text is missing from the content stream")
	}
}

func TestRender_TableGroupOpacityCompositesItsCellsAndBackground(t *testing.T) {
	pdf, err := RenderHTMLToPDF(`<table><thead style="opacity:.5;background:#163b64;color:white"><tr><th>HEAD</th><th>SECOND</th></tr></thead><tbody><tr><td>BODY</td><td>ROW</td></tr></tbody></table>`)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	if !strings.Contains(string(pdf), "/Subtype /Form") || !strings.Contains(string(pdf), "/CA 0.5 /ca 0.5") {
		t.Fatal("row-group opacity must composite the complete table group")
	}
	content := streamText(t, pdf)
	if !strings.Contains(content, "0.08627450980392157 0.23137254901960785 0.39215686274509803 rg") || !strings.Contains(content, "(HEAD)") {
		t.Fatal("the opacity form must retain both the group background and its cell text")
	}
}

func TestRender_TableStructuralBordersPaintInSeparateMode(t *testing.T) {
	pdf, err := RenderHTMLToPDF(`<table style="border-collapse:separate;border:2px solid red"><thead style="border:1px solid blue"><tr style="border-bottom:1px solid green"><th>HEAD</th></tr></thead><tbody><tr><td>BODY</td></tr></tbody></table>`)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	content := streamText(t, pdf)
	for _, color := range []string{"1 0 0 rg", "0 0 1 rg", "0 0.5 0 rg"} {
		if !strings.Contains(content, color) {
			t.Errorf("structural border color %q is missing", color)
		}
	}
}

func TestRender_CollapsedTableRowBorderPaintsAcrossCells(t *testing.T) {
	pdf, err := RenderHTMLToPDF(`<style>table{border-collapse:collapse} .grand{border-top:2px solid #b6cf58}</style><table><tr><td>LEFT</td><td>RIGHT</td></tr><tr class="grand"><td>TOTAL</td><td>333.50</td></tr></table>`)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	content := streamText(t, pdf)
	green := "0.7137254901960784 0.8117647058823529 0.34509803921568627 rg"
	if count := strings.Count(content, green); count != 2 {
		t.Fatalf("collapsed row border segments=%d, want one painted segment for each cell; content stream:\n%s", count, content)
	}
}

func TestRender_CollapsedTableOuterBorderKeepsFullThickness(t *testing.T) {
	html := `<style>@page{size:200pt 100pt;margin:0}body{margin:0}table{width:100pt;border-collapse:collapse;border-bottom:1pt solid red}td{padding:0}</style><table><tr><td>LEFT</td><td>RIGHT</td></tr></table>`
	pdf, err := RenderHTMLToPDF(html)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	content := streamText(t, pdf)
	if !regexp.MustCompile(`1 0 0 rg [0-9.]+ [0-9.]+ [0-9.]+ 1 re f`).MatchString(content) {
		t.Fatalf("collapsed outer border is not painted at its full 1pt thickness; content stream:\n%s", content)
	}
}

func TestRender_CollapsedTableBordersAndCellPaddingInsideFlexItem(t *testing.T) {
	html := `<style>@page{size:240pt 160pt;margin:20pt}body{margin:0}.summary{display:flex;gap:8pt}.notes{width:120pt}.totals{width:100pt;border-collapse:collapse;font:8pt Helvetica}.totals td{padding:4pt 3pt;border-bottom:1pt solid #e1e5e6}.totals tr:last-child td{border-top:2pt solid #173b45}</style><section class="summary"><div class="notes">NOTES</div><table class="totals"><tr><td>Subtotal</td><td>100.00</td></tr><tr><td>Discount</td><td>5.00</td></tr><tr><td>Total</td><td>95.00</td></tr></table></section>`
	pdf, err := RenderHTMLToPDF(html)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	content := streamText(t, pdf)
	gray := "0.8823529411764706 0.8980392156862745 0.9019607843137255 rg"
	if count := strings.Count(content, gray); count < 2 {
		t.Errorf("expected collapsed row separators to paint across the total rows; gray border segments=%d, content stream:\n%s", count, content)
	}
	if !strings.Contains(content, "(Subtotal)") || !strings.Contains(content, "(95.00)") {
		t.Fatal("table content is missing")
	}
	if !regexp.MustCompile(`0\.8823529411764706 0\.8980392156862745 0\.9019607843137255 rg [0-9.]+ [0-9.]+ [0-9.]+ 1 re f`).MatchString(content) {
		t.Errorf("row separator must retain its CSS 1pt thickness; content stream:\n%s", content)
	}
}

func TestRender_RepeatedHeaderKeepsItsPaintLayers(t *testing.T) {
	var body strings.Builder
	body.WriteString(`<table style="margin:0"><thead style="background:#163b64;color:white"><tr><th>REPEATED HEADER</th></tr></thead><tbody>`)
	for i := 0; i < 90; i++ {
		body.WriteString(`<tr><td>row data that continues the table across physical pages</td></tr>`)
	}
	body.WriteString(`</tbody></table>`)
	pdf, err := RenderHTMLToPDF(body.String())
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	content := streamText(t, pdf)
	blue := "0.08627450980392157 0.23137254901960785 0.39215686274509803 rg"
	if count := strings.Count(content, blue); count < 2 {
		t.Fatalf("painted header background count=%d, want one for each page fragment", count)
	}
	if count := strings.Count(content, "(REPEATED)"); count < 2 {
		t.Fatalf("repeated header text count=%d, want at least 2", count)
	}
}
