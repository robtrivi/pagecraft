package boxes

import (
	"strconv"
	"strings"

	"github.com/robtrivi/pagecraft-go/css"
)

func applyPaginationStyle(b *Box, decls css.Declarations) {
	if op, ok := decls["opacity"]; ok {
		if f, err := strconv.ParseFloat(strings.TrimSpace(op), 64); err == nil && f >= 0 && f <= 1 {
			b.Style.Opacity, b.Style.HasOpacity = f, true
		}
	}
	breakAfter, valid := parseBreakAfter(decls["break-after"])
	if !valid {
		breakAfter, valid = parseLegacyBreakAfter(decls["page-break-after"])
	}
	if valid {
		b.Style.BreakAfter = breakAfter
	}
	if value, ok := decls["overflow"]; ok {
		if mode, valid := parseOverflowMode(value); valid {
			b.Style.Overflow = mode
			b.Style.OverflowX, b.Style.OverflowY = mode, mode
		}
	}
	if value, ok := decls["overflow-x"]; ok {
		if mode, valid := parseOverflowMode(value); valid {
			b.Style.OverflowX = mode
		}
	}
	if value, ok := decls["overflow-y"]; ok {
		if mode, valid := parseOverflowMode(value); valid {
			b.Style.OverflowY = mode
		}
	}
	if value, ok := decls["break-before"]; ok {
		if mode, valid := parseBreakBefore(value); valid {
			b.Style.BreakBefore = mode
		}
	} else if value, ok := decls["page-break-before"]; ok && value == "always" {
		b.Style.BreakBefore = BreakBeforePage
	}
	if value, ok := decls["widows"]; ok {
		if n, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && n > 0 {
			b.Style.Widows = n
		}
	}
	if value, ok := decls["orphans"]; ok {
		if n, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && n > 0 {
			b.Style.Orphans = n
		}
	}
	if value, ok := decls["break-inside"]; ok {
		if mode, valid := parseBreakInside(value); valid {
			b.Style.BreakInside = mode
		}
	} else if value, ok := decls["page-break-inside"]; ok {
		if mode, valid := parseBreakInside(value); valid {
			b.Style.BreakInside = mode
		}
	}
}

func parseLegacyBreakAfter(value string) (BreakAfterMode, bool) {
	switch value {
	case "avoid":
		return BreakAfterAvoid, true
	case "always":
		return BreakAfterPage, true
	default:
		return BreakAfterAuto, false
	}
}
