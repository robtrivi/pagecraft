package boxes

import (
	"strings"

	"github.com/robtrivi/pagecraft/css"
)

func applySpacingStyle(b *Box, decls css.Declarations) {
	if fs, ok := decls["font-size"]; ok {
		if v, ok := parseFontSize(fs); ok {
			b.Style.FontSize = v
		}
	}
	marginFS := b.Style.FontSize
	if marginFS <= 0 {
		marginFS = 12
	}
	if b.Tag == "p" {
		b.Style.MarginTop, b.Style.MarginBottom = marginFS, marginFS
	}
	if m, ok := decls["margin"]; ok {
		b.Style.MarginTop, b.Style.MarginRight, b.Style.MarginBottom, b.Style.MarginLeft = expandSidesRelative(m, marginFS)
	}
	if pd, ok := decls["padding"]; ok {
		b.Style.PaddingTop, b.Style.PaddingRight, b.Style.PaddingBottom, b.Style.PaddingLeft = expandSides(pd)
	}
	// Longhands override the shorthand regardless of order (documented v1 behavior).
	for _, ml := range []struct {
		prop string
		dst  *float64
	}{
		{"margin-top", &b.Style.MarginTop}, {"margin-right", &b.Style.MarginRight},
		{"margin-bottom", &b.Style.MarginBottom}, {"margin-left", &b.Style.MarginLeft},
		{"padding-top", &b.Style.PaddingTop}, {"padding-right", &b.Style.PaddingRight},
		{"padding-bottom", &b.Style.PaddingBottom}, {"padding-left", &b.Style.PaddingLeft},
	} {
		if v, ok := decls[ml.prop]; ok {
			var f float64
			var valid bool
			if strings.HasPrefix(ml.prop, "margin-") {
				f, valid = tryLengthRelative(v, marginFS)
			} else {
				f, valid = tryLengthPx(v)
			}
			if valid {
				*ml.dst = f
			}
		}
	}
	if margin, ok := decls["margin"]; ok {
		if values, valid := expandMarginTokens(margin); valid {
			b.Style.MarginAutoTop, b.Style.MarginAutoRight = values[0] == "auto", values[1] == "auto"
			b.Style.MarginAutoBottom, b.Style.MarginAutoLeft = values[2] == "auto", values[3] == "auto"
		}
	}
	for _, side := range []struct {
		property string
		auto     *bool
		margin   *float64
	}{
		{"margin-top", &b.Style.MarginAutoTop, &b.Style.MarginTop},
		{"margin-right", &b.Style.MarginAutoRight, &b.Style.MarginRight},
		{"margin-bottom", &b.Style.MarginAutoBottom, &b.Style.MarginBottom},
		{"margin-left", &b.Style.MarginAutoLeft, &b.Style.MarginLeft},
	} {
		if value, ok := decls[side.property]; ok {
			*side.auto = strings.TrimSpace(value) == "auto"
			if *side.auto {
				*side.margin = 0
			}
		}
	}
}
