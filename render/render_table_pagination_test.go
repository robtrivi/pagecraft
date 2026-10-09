package render

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestRender_OversizedTableRowFragmentsAcrossPages(t *testing.T) {
	lines := make([]string, 17)
	for index := range lines {
		lines[index] = fmt.Sprintf("ROW-LINE-%02d", index+1)
	}
	html := `<style>@page{size:220pt 120pt;margin:0}body{margin:0}table{margin:0;width:220pt;border-collapse:collapse}th,td{padding:0;font:10pt/12pt Helvetica}</style><table><thead><tr><th>HEADER</th><th>STATUS</th></tr></thead><tbody><tr><td>` + strings.Join(lines, "<br>") + `</td><td>SUMMARY</td></tr></tbody></table>`
	pdf, err := RenderHTMLToPDF(html)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	assertWordYAtOrAfter(t, pdf, "ROW-LINE-17", 120)
	content := streamText(t, pdf)
	if strings.Count(content, "(HEADER)") != 2 {
		t.Errorf("header appears %d times in the streams; want one original and one repeated", strings.Count(content, "(HEADER)"))
	}
	for index := range lines {
		marker := fmt.Sprintf("-LINE-%02d", index+1)
		if count := strings.Count(content, marker); count != 1 {
			t.Errorf("%s appears %d times in the streams; want 1", marker, count)
		}
	}
	textPositions := regexp.MustCompile(`Tf [^ ]+ [-0-9.]+ Td`).FindAllString(content, -1)
	for _, position := range textPositions {
		fields := strings.Fields(position)
		y, err := strconv.ParseFloat(fields[2], 64)
		if err != nil {
			t.Fatalf("coordenada PDF invalid in %q: %v", position, err)
		}
		if y < 0 {
			t.Errorf("text remains outside the page in %q", position)
		}
	}
}

func TestRender_NestedTableInCellPreservesItsRowsAndContent(t *testing.T) {
	html := `<style>@page{size:220pt 180pt;margin:0}body{margin:0}table{margin:0;border-collapse:collapse}#outer{width:220pt}td{font:10pt/12pt Helvetica;padding:2pt;border:1pt solid #111}#inner td:first-child{width:50%;background:#cceeff}#inner td:last-child{background:#fff0cc}</style><table id="outer"><tr><td>OUTER <table id="inner"><tr><td>LEFT</td><td>RIGHT</td></tr></table> AFTER</td><td>SUMMARY</td></tr></table>`
	pdf, err := RenderHTMLToPDF(html)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	content := streamText(t, pdf)
	for _, text := range []string{"OUTER", "LEFT", "RIGHT", "AFTER", "SUMMAR"} {
		if !strings.Contains(content, text) {
			t.Errorf("the text %q does not appear in the stream PDF", text)
		}
	}
}

func TestRender_NestedTableContinuesAcrossPages(t *testing.T) {
	html := `<style>@page{size:220pt 45pt;margin:0}body{margin:0}table{margin:0;border-collapse:collapse}#outer{width:220pt}td{font:10pt/12pt Helvetica;padding:1pt;border:1pt solid #111}#inner td{background:#e0f2fe}</style><table id="outer"><tr><td><table id="inner"><tr><td>INNER-1</td></tr><tr><td>INNER-2</td></tr><tr><td>INNER-3</td></tr><tr><td>INNER-4</td></tr></table></td><td>SUMMARY</td></tr></table>`
	pdf, err := RenderHTMLToPDF(html)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	assertWordYAtOrAfter(t, pdf, "INNER-4", 45)
	content := streamText(t, pdf)
	for index := 1; index <= 4; index++ {
		marker := fmt.Sprintf("INNER-%d", index)
		if count := strings.Count(content, marker); count != 1 {
			t.Errorf("%s appears %d times in PDF streams, want 1", marker, count)
		}
	}
}

func TestRender_TableRowspanContentContinuesAcrossPages(t *testing.T) {
	lines := make([]string, 17)
	for index := range lines {
		lines[index] = fmt.Sprintf("ACTIVE-%02d", index+1)
	}
	html := `<style>@page{size:220pt 120pt;margin:0}body{margin:0}table{margin:0;width:220pt;border-collapse:collapse}td{padding:0;vertical-align:top;border:1pt solid #111;font:10pt/12pt Helvetica}</style><table><tbody><tr><td rowspan="2" style="background:#fee">SPAN</td><td>START</td></tr><tr><td style="background:#eef">` + strings.Join(lines, "<br>") + `</td></tr></tbody></table>`
	pdf, err := RenderHTMLToPDF(html)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	assertWordYAtOrAfter(t, pdf, "ACTIVE-17", 120)
	content := streamText(t, pdf)
	if strings.Count(content, "(SP)") != 1 || strings.Count(content, "(AN)") != 1 {
		t.Errorf("SPAN appears %d/%d times in its kerned text runs, want once", strings.Count(content, "(SP)"), strings.Count(content, "(AN)"))
	}
	for _, line := range lines {
		marker := line[strings.LastIndex(line, "-"):]
		if count := strings.Count(content, marker); count != 1 {
			t.Errorf("%s appears %d times, want one occurrence", marker, count)
		}
	}
	for _, position := range regexp.MustCompile(`Tf [^ ]+ [-0-9.]+ Td`).FindAllString(content, -1) {
		fields := strings.Fields(position)
		y, err := strconv.ParseFloat(fields[2], 64)
		if err != nil {
			t.Fatalf("coordenada PDF invalid in %q: %v", position, err)
		}
		if y < 0 {
			t.Errorf("rowspan content is outside the page in %q", position)
		}
	}
}

func TestRender_OversizedRowStartingRowspanKeepsContentAcrossPages(t *testing.T) {
	lines := make([]string, 17)
	for index := range lines {
		lines[index] = fmt.Sprintf("ROWSPAN-%02d", index+1)
	}
	html := `<style>@page{size:220pt 120pt;margin:0}body{margin:0}table{margin:0;width:220pt;border-collapse:collapse}td{padding:0;border:1pt solid #111;font:10pt/12pt Helvetica}</style><table><tbody><tr><td rowspan="2" style="background:#fee;width:66.6667%">` + strings.Join(lines, "<br>") + `</td><td>START</td></tr><tr><td>NEXT</td></tr></tbody></table>`
	pdf, err := RenderHTMLToPDF(html)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	assertWordYAtOrAfter(t, pdf, "ROWSPAN-17", 120)
	content := streamText(t, pdf)
	for index := range lines {
		marker := fmt.Sprintf("-%02d", index+1)
		if count := strings.Count(content, marker); count != 1 {
			t.Errorf("%s appears %d times, want 2", marker, count)
		}
	}
	if strings.Count(content, "(ST)") != 1 || strings.Count(content, "(AR)") != 1 || strings.Count(content, "(NEXT)") != 1 {
		t.Errorf("START/NEXT is lost or duplicated in the PDF")
	}
	for _, position := range regexp.MustCompile(`Tf [^ ]+ [-0-9.]+ Td`).FindAllString(content, -1) {
		fields := strings.Fields(position)
		y, err := strconv.ParseFloat(fields[2], 64)
		if err != nil {
			t.Fatalf("coordenada PDF invalid in %q: %v", position, err)
		}
		if y < 0 || y > 120 {
			t.Errorf("text outside the page area in %q", position)
		}
	}
}

func TestRender_TableCellParagraphsKeepMarginsAndTextAcrossPages(t *testing.T) {
	lines := make([]string, 17)
	for index := range lines {
		lines[index] = fmt.Sprintf("FIRST-%02d", index+1)
	}
	html := `<style>@page{size:220pt 150pt;margin:0}body{margin:0}table{margin:0;width:220pt;border-collapse:collapse}td{padding:2pt;border:1pt solid #111;font:10pt/12pt Helvetica}td:first-child{width:66.6667%;background:#fee}p{margin:0 0 6pt}</style><table><tbody><tr><td><div><p>` + strings.Join(lines, "<br>") + `</p><p>FINAL-PARAGRAPH</p></div></td><td>SUMMARY</td></tr></tbody></table>`
	pdf, err := RenderHTMLToPDF(html)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	assertWordYAtOrAfter(t, pdf, "FINAL-PARAGRAPH", 150)
	content := streamText(t, pdf)
	for index := range lines {
		marker := fmt.Sprintf("-%02d", index+1)
		if count := strings.Count(content, marker); count != 1 {
			t.Errorf("%s appears %d times, want 2", marker, count)
		}
	}
	if strings.Count(content, "(FINAL-P)") != 1 || strings.Count(content, "(GRAPH)") != 1 {
		t.Error("second paragraph appears lost or duplicated in the PDF")
	}
}

func TestRender_TableCellBlockBackgroundContinuesAcrossFragments(t *testing.T) {
	lines := make([]string, 17)
	for index := range lines {
		lines[index] = fmt.Sprintf("PAINT-%02d", index+1)
	}
	html := `<style>@page{size:220pt 150pt;margin:0}body{margin:0}table{margin:0;width:220pt;border-collapse:collapse}td{padding:0;border:0;font:10pt/12pt Helvetica;vertical-align:top}td:first-child{width:66.6667%}div{background:#fff0cc;border:1pt solid #995500;padding:2pt}p{margin:0;background:#cceeff;border:2pt solid #123456;padding:3pt}</style><table><tbody><tr><td><div><p>` + strings.Join(lines, "<br>") + `</p></div></td><td>SUMMARY</td></tr></tbody></table>`
	pdf, err := RenderHTMLToPDF(html)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	assertWordYAtOrAfter(t, pdf, "PAINT-17", 150)
	content := streamText(t, pdf)
	for _, color := range []string{"0.8 0.9333333333333333 1 rg", "1 0.9411764705882353 0.8 rg"} {
		if count := strings.Count(content, color); count != 2 {
			t.Errorf("nested background %q must be painted in both fragments; occurrences=%d", color, count)
		}
	}
	for index := range lines {
		marker := fmt.Sprintf("-%02d", index+1)
		if count := strings.Count(content, marker); count != 1 {
			t.Errorf("%s appears %d times, want 1", marker, count)
		}
	}
}

func assertWordYAtOrAfter(t *testing.T, pdf []byte, marker string, minY float64) {
	t.Helper()
	words := assertPDFHasWordCoordinates(t, pdf)
	for _, word := range words {
		if strings.Contains(word.Text, marker) {
			if word.Y < minY {
				t.Errorf("%q appears in y=%.2fpt; expected y≥%.2fpt", marker, word.Y, minY)
			}
			return
		}
	}
	t.Errorf("could not find %q with text coordinates", marker)
}
