package boxes

import (
	"github.com/robtrivi/pagecraft/css"
)

func applyGeometryStyle(b *Box, decls css.Declarations) {
	if t, ok := decls["top"]; ok {
		b.Style.Top, b.Style.HasTop = parsePx(t), true
	}
	if l, ok := decls["left"]; ok {
		b.Style.Left = parsePx(l)
	}
	if r, ok := decls["right"]; ok {
		if value, valid := tryLengthPx(r); valid {
			b.Style.Right, b.Style.HasRight = value, true
		}
	}
	if bottom, ok := decls["bottom"]; ok {
		if value, valid := tryLengthPx(bottom); valid {
			b.Style.Bottom, b.Style.HasBottom = value, true
		}
	}
	if value, ok := decls["width"]; ok {
		if dimension, valid := parseCSSDimension(value); valid {
			b.Style.Sizing.Width = dimension
		}
	}
	if value, ok := decls["min-width"]; ok {
		if dimension, valid := parseCSSDimension(value); valid {
			b.Style.Sizing.MinWidth = dimension
		}
	}
	if value, ok := decls["max-width"]; ok {
		if dimension, valid := parseCSSDimension(value); valid {
			b.Style.Sizing.MaxWidth = dimension
		}
	}
	if value, ok := decls["height"]; ok {
		if dimension, valid := parseCSSDimension(value); valid {
			b.Style.Sizing.Height = dimension
		}
	}
	if value, ok := decls["min-height"]; ok {
		if dimension, valid := parseCSSDimension(value); valid {
			b.Style.Sizing.MinHeight = dimension
		}
	}
	if value, ok := decls["max-height"]; ok {
		if dimension, valid := parseCSSDimension(value); valid {
			b.Style.Sizing.MaxHeight = dimension
		}
	}
}
