package boxes

import (
	"strconv"

	"github.com/robtrivi/pagecraft-go/css"
	"golang.org/x/net/html"
)

func applyLayoutStyle(b *Box, the *html.Node, decls css.Declarations) {
	applyFlexStyle(b, decls)
	applyFlowPositionStyle(b, decls)
	applyGridProperties(b, decls)
	applyColumnsStyle(b, decls)
	applyBoxSizingStyle(b, decls)
	applyTableCellStyle(b, the, decls)
	applyGridItemStyle(b, decls)
}

func applyColumnsStyle(b *Box, decls css.Declarations) {
	if cc, ok := decls["column-count"]; ok {
		if n, err := strconv.Atoi(cc); err == nil && n > 0 {
			b.Style.ColCount = n
		}
	}
	if gap, ok := decls["gap"]; ok {
		if row, column, valid := parseGap(gap); valid {
			b.Style.RowGap, b.Style.RowGapPercent = row.length, row.percent
			b.Style.ColumnGap, b.Style.ColumnGapPercent = column.length, column.percent
		}
	}
	for _, longhand := range []struct {
		name       string
		lengthDst  *float64
		percentDst *float64
	}{{"row-gap", &b.Style.RowGap, &b.Style.RowGapPercent}, {"column-gap", &b.Style.ColumnGap, &b.Style.ColumnGapPercent}} {
		if value, ok := decls[longhand.name]; ok {
			if gap, valid := parseGapComponent(value); valid {
				*longhand.lengthDst = gap.length
				*longhand.percentDst = gap.percent
			}
		}
	}
}

func applyBoxSizingStyle(b *Box, decls css.Declarations) {
	if value, ok := decls["box-sizing"]; ok {
		if sizing, valid := parseBoxSizing(value); valid {
			b.Style.BoxSizing = sizing
		}
	}
	b.Style.BorderCollapse = decls["border-collapse"] == "collapse"
}

func applyTableCellStyle(b *Box, the *html.Node, decls css.Declarations) {
	b.Style.ColSpan, b.Style.RowSpan = 1, 1
	if the.Data == "td" || the.Data == "th" {
		if n, err := strconv.Atoi(getAttr(the, "colspan")); err == nil && n > 1 {
			b.Style.ColSpan = n
		}
		if n, err := strconv.Atoi(getAttr(the, "rowspan")); err == nil && n > 1 {
			b.Style.RowSpan = n
		}
		if w, ok := parsePercent(getAttr(the, "width")); ok {
			b.Style.Sizing.Width = CSSDimension{Kind: DimensionPercentage, Value: w}
		}
		if w, ok := decls["width"]; ok {
			if f, ok := parsePercent(w); ok {
				b.Style.Sizing.Width = CSSDimension{Kind: DimensionPercentage, Value: f}
			}
		}
	}
}
