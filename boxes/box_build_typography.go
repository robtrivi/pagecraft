package boxes

import (
	"github.com/robtrivi/pagecraft-go/css"
	"golang.org/x/net/html"
)

func applyTypographyStyle(b *Box, the *html.Node, decls css.Declarations) {
	if c, ok := decls["color"]; ok {
		if c, ok := css.ParseColor(c); ok {
			b.Style.HasColor = true
			b.Style.Color = c
		}
	}
	if the.Data == "a" {
		b.Link = getAttr(the, "href")
	}
	if al, ok := decls["text-align"]; ok {
		switch al {
		case "center", "right", "justify":
			b.Style.Align = al
		default:
			b.Style.Align = "left"
		}
	} else {
		b.Style.Align = "left"
	}
	if ff, ok := decls["font-family"]; ok {
		b.Style.FontFamily = firstFamily(ff)
	}
	if lh, ok := decls["line-height"]; ok {
		if f, ok := parseLineHeight(lh); ok {
			b.Style.LineHeight = f
		} else if v, ok := tryLengthPx(lh); ok {
			b.Style.LineHeightPt = v
		}
	}
	if ls, ok := decls["letter-spacing"]; ok && ls != "normal" {
		if v, ok := tryLengthPx(ls); ok {
			b.Style.LetterSpacing = v
		}
	}
}
