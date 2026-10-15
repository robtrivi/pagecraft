package render

import (
	"testing"
)

func TestRender_FlexColumnItemKeepsMultilineContentInFlow(t *testing.T) {
	html := `<style>
@page{size:A4;margin:12mm 13mm 13mm}*{box-sizing:border-box}body{margin:0;font:7.2pt Arial,sans-serif}
.balance{display:flex;justify-content:space-between;align-items:stretch;gap:8mm;margin-top:3mm;border-top:1px solid #c9d0d8;padding-top:4mm}
.balance>div:first-child{background:#263c63;color:white;width:38%;padding:5mm;display:flex;flex-direction:column;justify-content:center}
.balance span{font:6pt Arial,sans-serif;letter-spacing:.8px}.balance strong{font:22pt Georgia,serif;margin:2mm 0}.balance small{font:7pt Georgia,serif}
.totals{width:58%;border-collapse:collapse;font:7.6pt Arial,sans-serif}.totals td{padding:2.1mm 1.2mm;border-bottom:1px solid #d8dfe2}.totals td:first-child{width:68%}.totals td:last-child{text-align:right;white-space:nowrap}
</style><div class="balance"><div><span>IMPORTE A PAGAR</span><strong>USD 4,104.64</strong><small>Incluye IVA de 535.39</small></div><table class="totals"><tr><td>Subtotal sin impuestos</td><td>3,618.75</td></tr><tr><td>IVA 15%</td><td>535.39</td></tr><tr><td>TOTAL USD</td><td>4,104.64</td></tr></table></div>`
	pdf, err := RenderHTMLToPDF(html)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	words := wordsFromPDF(t, pdf)
	word := func(text string) pdfWord {
		t.Helper()
		for _, word := range words {
			if word.Text == text {
				return word
			}
		}
		t.Fatalf("%q is missing from the PDF", text)
		return pdfWord{}
	}
	usd, amount, note := word("USD"), word("4,104.64"), word("Incluye")
	if amount.Y-usd.Y > 2 || usd.Y-amount.Y > 2 {
		t.Errorf("amount split across lines: USD top=%gpt, value top=%gpt", usd.Y, amount.Y)
	}
	if note.Y < amount.Y+amount.Height {
		t.Errorf("note overlaps the amount line: amount bounds=%g..%gpt, note top=%gpt", amount.Y, amount.Y+amount.Height, note.Y)
	}
}

func TestRender_FlexAlignItemsEndAlignsShortItemWithMultilineItemBottom(t *testing.T) {
	html := `<style>
@page{size:A4;margin:12mm 13mm 13mm}body{margin:0;font:6.5pt Arial,sans-serif}
.footer{display:flex;justify-content:space-between;align-items:end;gap:4mm;border-top:1px solid #c8d0d4;padding-top:2.5mm}
.footer>div:first-child{flex:1}.footer span,.footer strong{display:block}.footer span{font-size:5.7pt;letter-spacing:.4px}.footer code{font:6.5pt monospace;display:block;margin-top:1mm}.footer>div:nth-child(2){min-width:37mm}.footer>div:nth-child(2) strong{margin-top:1mm;font-size:6.5pt}.footer>b{font-size:6pt;white-space:nowrap}
</style><footer class="footer"><div><span>ACCESS KEY</span><code>1234567890123456789012345678901234567890123456789</code></div><div><span>AUTHORIZATION DATE AND TIME</span><strong>14/10/2026 · 11:28:04</strong></div><b>PAGE 2 OF 2</b></footer>`
	pdf, err := RenderHTMLToPDF(html)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	words := wordsFromPDF(t, pdf)
	wordY := func(text string) float64 {
		t.Helper()
		for _, word := range words {
			if word.Text == text {
				return word.Y
			}
		}
		t.Fatalf("%q is missing from the PDF", text)
		return 0
	}
	labelY, dateY, pageY := wordY("AUTHORIZATION"), wordY("14/10/2026"), wordY("PAGE")
	if dateY-labelY < 7 {
		t.Errorf("authorization date is not below its label: label top=%gpt, date top=%gpt", labelY, dateY)
	}
	if delta := pageY - dateY; delta < -1.5 || delta > 1.5 {
		t.Errorf("page count top=%gpt, want aligned with date top=%gpt (±1.5pt)", pageY, dateY)
	}
}
