package boxes

import (
	"math"
	"strconv"
	"strings"

	"github.com/robtrivi/pagecraft/css"
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
	if width, ok := decls["column-width"]; ok {
		if value, valid := parseColumnWidth(width); valid {
			b.Style.ColumnWidth = value
			b.Style.HasColumnWidth = true
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

func parseColumnWidth(value string) (float64, bool) {
	width, valid := tryLengthPx(strings.TrimSpace(value))
	return width, valid && width >= 0
}

func applyBoxSizingStyle(b *Box, decls css.Declarations) {
	if value, ok := decls["box-sizing"]; ok {
		if sizing, valid := parseBoxSizing(value); valid {
			b.Style.BoxSizing = sizing
		}
	}
	b.Style.BorderCollapse = decls["border-collapse"] == "collapse"
	b.Style.TableLayoutFixed = decls["table-layout"] == "fixed"
	if spacing, ok := decls["border-spacing"]; ok {
		if horizontal, vertical, valid := parseTableBorderSpacing(spacing); valid {
			b.Style.TableBorderSpacingX, b.Style.TableBorderSpacingY = horizontal, vertical
		}
	}
}

func applyTableCellStyle(b *Box, the *html.Node, decls css.Declarations) {
	b.Style.ColSpan, b.Style.RowSpan = 1, 1
	if the.Data == "table" {
		if width, valid := parseHTMLTableWidth(getAttr(the, "width"), true); valid {
			b.Style.Sizing.Width = width
		}
	}
	if the.Data == "col" || the.Data == "colgroup" {
		b.Style.TableColumnSpan = 1
		if span, err := strconv.Atoi(getAttr(the, "span")); err == nil && span > 1 {
			b.Style.TableColumnSpan = span
		}
		if width, valid := parseHTMLTableWidth(getAttr(the, "width"), true); valid {
			b.Style.Sizing.Width = width
		}
	}
	if the.Data == "td" || the.Data == "th" {
		if n, err := strconv.Atoi(getAttr(the, "colspan")); err == nil && n > 1 {
			b.Style.ColSpan = n
		}
		if n, err := strconv.Atoi(getAttr(the, "rowspan")); err == nil && n > 1 {
			b.Style.RowSpan = n
		}
		if width, ok := parseHTMLTableWidth(getAttr(the, "width"), false); ok {
			b.Style.Sizing.Width = width
		}
		if w, ok := decls["width"]; ok {
			if f, ok := parsePercent(w); ok {
				b.Style.Sizing.Width = CSSDimension{Kind: DimensionPercentage, Value: f}
			}
		}
	}
}

// parseHTMLTableWidth converts the legacy HTML width attribute into its CSS
// presentation-hint value. Unitless lengths in these attributes are CSS px;
// CSS declarations continue to use parseCSSDimension and its normal units.
func parseHTMLTableWidth(value string, allowZero bool) (CSSDimension, bool) {
	value = strings.TrimSpace(value)
	if percentage, ok := parsePercent(value); ok {
		return CSSDimension{Kind: DimensionPercentage, Value: percentage}, true
	}
	end := 0
	for end < len(value) {
		character := value[end]
		if character >= '0' && character <= '9' || character == '.' {
			end++
			continue
		}
		break
	}
	if end < len(value) && value[end] == '%' {
		percentage, ok := parsePercent(value[:end+1])
		if !ok {
			return CSSDimension{}, false
		}
		return CSSDimension{Kind: DimensionPercentage, Value: percentage}, true
	}
	length, err := strconv.ParseFloat(value[:end], 64)
	if err != nil || length < 0 || !allowZero && length == 0 || math.IsInf(length, 0) || math.IsNaN(length) {
		return CSSDimension{}, false
	}
	return CSSDimension{Kind: DimensionLength, Value: length * css.PxToPt}, true
}
