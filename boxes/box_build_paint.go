package boxes

import (
	"strings"

	"github.com/robtrivi/pagecraft-go/css"
)

func applyPaintStyle(b *Box, decls css.Declarations) {
	if bg, ok := decls["background"]; ok {
		if c, ok := parseBackgroundColor(bg); ok {
			b.Style.HasBg = true
			b.Style.Bg = c
		}
	}
	if bg, ok := decls["background-color"]; ok {
		if c, ok := css.ParseColor(bg); ok {
			b.Style.HasBg = true
			b.Style.Bg = c
		}
	}
	if radius, ok := decls["border-radius"]; ok {
		if values, valid := parseBorderCornerRadii(radius); valid {
			b.Style.BorderCornerRadii = values
		}
	}
	if bd, ok := decls["border"]; ok {
		if w, c, ok := parseBorder(bd); ok {
			b.Style.BorderWidth = w
			b.Style.Border = c
			side := BorderSide{Width: w, Color: c}
			b.Style.BorderTop, b.Style.BorderRight, b.Style.BorderBottom, b.Style.BorderLeft = side, side, side, side
		}
	}
	if rule, ok := decls["column-rule"]; ok {
		if width, color, valid := parseBorder(rule); valid {
			b.Style.ColumnRuleWidth, b.Style.ColumnRuleColor = width, color
		}
	}
	for _, sb := range []struct {
		name string
		prop string
	}{{"Top", "border-top"}, {"Right", "border-right"}, {"Bottom", "border-bottom"}, {"Left", "border-left"}} {
		if bd, ok := decls[sb.prop]; ok {
			if t := strings.TrimSpace(bd); t == "none" || t == "hidden" {
				setBorderSide(b, sb.name, BorderSide{})
			} else if w, c, ok := parseBorder(bd); ok {
				setBorderSide(b, sb.name, BorderSide{Width: w, Color: c})
			}
		}
	}
}
