package layout

import (
	"testing"

	"github.com/robtrivi/pagecraft/boxes"
	"github.com/robtrivi/pagecraft/text"
)

func TestLayout_TableCellTextDoesNotExceedContentEdge(t *testing.T) {
	html := `<style>
@page{size:A4;margin:0}*{box-sizing:border-box}body{margin:0;color:#16212b;font:9pt "Liberation Sans",sans-serif}
.sheet{width:auto;min-height:267mm;padding:0 12mm;position:relative}.d{color:#263044;padding-left:5mm;border-left:7mm solid #344b91}
table{width:100%;border-collapse:collapse}td{width:50%;vertical-align:top}
.d .top{display:table;width:100%;padding-bottom:6mm;border-bottom:1px solid #d8dde5}
.d .top td:last-child{text-align:right}.doc-title{color:#344b91;font-size:26pt;letter-spacing:-1px;font-weight:bold}
.doc-number{margin-top:2mm;padding:2mm 3mm;background:#eff2fa;font-weight:bold}
</style><section class="sheet"><section class="d"><table class="top"><tr>
<td><div>RUCALIA / PACÍFICO</div><div>Servicios Andinos del Pacífico S.A.</div><div>Matriz · República E7-123 y Diego de Almagro, Quito</div></td>
<td><div class="doc-title">Factura</div><div class="doc-number">001-001-000000128</div></td>
</tr></table></section></section>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := DefaultOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 595, 842, 0
	opts.Measure, opts.MeasureBold = text.Measure, text.MeasureBold
	opts.Faces, opts.FontVariants = root.Fonts, root.FontVariants
	opts.PageMargins, opts.PageRules = root.Page.Margins, root.PageRules
	pages := Flow(root, opts)
	if len(pages) != 1 {
		t.Fatalf("pages: got %d, want 1", len(pages))
	}
	for _, block := range pages[0].Blocks {
		for _, line := range block.Runs {
			for _, run := range line {
				if run.Text != "Factura" {
					continue
				}
				const tableRight = 595 - 12*72/25.4
				if got := run.X + run.Width; got > tableRight+0.01 {
					t.Errorf("Factura right edge = %.2fpt, table content edge = %.2fpt", got, tableRight)
				}
				return
			}
		}
	}
	t.Fatal("Factura text run not found")
}
