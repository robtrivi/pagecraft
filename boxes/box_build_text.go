package boxes

import (
	"strings"

	"github.com/robtrivi/pagecraft/css"
)

func applyTextStyle(b *Box, decls css.Declarations, ancestorDecorations textDecorations) {
	if lp, ok := decls["list-style-position"]; ok && lp == "inside" {
		b.Style.ListInside = true
	}
	if ws, ok := decls["white-space"]; ok {
		switch ws {
		case "pre", "pre-wrap", "pre-line", "nowrap":
			b.Style.WhiteSpace = ws
		}
	}
	if va, ok := decls["vertical-align"]; ok {
		switch va {
		case "sub", "super", "top", "middle", "bottom":
			b.Style.VAlign = va
		default:
			b.Style.VAlign = ""
		}
	}
	if tt, ok := decls["text-transform"]; ok {
		switch tt {
		case "uppercase", "lowercase", "capitalize":
			b.Style.Transform = tt
		default:
			b.Style.Transform = ""
		}
	}
	if fw, ok := decls["font-weight"]; ok {
		b.Style.Bold = isBold(fw)
	}
	if fs, ok := decls["font-style"]; ok {
		b.Style.Italic = fs == "italic" || fs == "oblique"
	}
	if td, ok := decls["text-decoration"]; ok {
		for _, tok := range strings.Fields(td) {
			switch tok {
			case "underline":
				b.Style.Underline = true
			case "line-through":
				b.Style.Strike = true
			case "none":
				b.Style.Underline, b.Style.Strike = false, false
			}
		}
	}
	b.Style.Underline = b.Style.Underline || ancestorDecorations.underline
	b.Style.Strike = b.Style.Strike || ancestorDecorations.strike
}
