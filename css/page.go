package css

import (
	"strconv"
	"strings"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"
)

// pageSizes stores portrait page dimensions in points (width by height).
var pageSizes = map[string][2]float64{
	"a6":     {105 * 72 / 25.4, 148 * 72 / 25.4},
	"a5":     {420, 595},
	"a4":     {595, 842},
	"a3":     {842, 1190},
	"letter": {612, 792},
	"legal":  {612, 1008},
}

// ParsePageRule reads the first @page rule and returns its width, height, and margin in points.
// A missing size returns 0,0; a missing margin returns -1. It returns false only when no @page rule exists.
func ParsePageRule(src string) (w, h, margin float64, ok bool) {
	l := css.NewLexer(parse.NewInputString(src))
	for {
		tt, data := l.Next()
		if tt == css.ErrorToken {
			break
		}
		if tt != css.AtKeywordToken || !strings.EqualFold(strings.TrimPrefix(string(data), "@"), "page") {
			continue
		}
		body, good := atBody(l)
		if !good {
			continue
		}
		decls, _ := ParseInlineStyle(body)
		if sz, ok := decls["size"]; ok {
			if pw, ph, good := parsePageSize(sz); good {
				w, h = pw, ph
			}
		}
		margin = -1
		if m, ok := decls["margin"]; ok && len(strings.Fields(m)) == 1 {
			if v, ok := parseLength(m); ok {
				margin = v
			}
		}
		return w, h, margin, true
	}
	return 0, 0, -1, false
}

// parsePageSize resolves "A4 landscape", "Letter", or "800px 600px".
func parsePageSize(s string) (float64, float64, bool) {
	fields := strings.Fields(strings.ToLower(s))
	landscape := false
	var lens []float64
	for _, f := range fields {
		switch f {
		case "landscape":
			landscape = true
		case "portrait":
		default:
			if sz, known := pageSizes[f]; known {
				lens = append(lens, sz[0], sz[1])
			} else if v, ok := parseLength(f); ok {
				lens = append(lens, v)
			} else {
				return 0, 0, false
			}
		}
	}
	if len(lens) != 2 {
		return 0, 0, false
	}
	if landscape {
		return lens[1], lens[0], true
	}
	return lens[0], lens[1], true
}

// parseLength converts a number and unit to points (px is 1:1 in v1).
func parseLength(s string) (float64, bool) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "0" {
		return 0, true
	}
	mult := 1.0
	num := ""
	switch {
	case strings.HasSuffix(s, "pt"):
		num = strings.TrimSuffix(s, "pt")
	case strings.HasSuffix(s, "px"):
		num, mult = strings.TrimSuffix(s, "px"), PxToPt
	case strings.HasSuffix(s, "pc"):
		num, mult = strings.TrimSuffix(s, "pc"), 12
	case strings.HasSuffix(s, "cm"):
		num, mult = strings.TrimSuffix(s, "cm"), 72/2.54
	case strings.HasSuffix(s, "mm"):
		num, mult = strings.TrimSuffix(s, "mm"), 72/25.4
	case strings.HasSuffix(s, "in"):
		num, mult = strings.TrimSuffix(s, "in"), 72
	default:
		return 0, false
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(num), 64)
	if err != nil || f < 0 {
		return 0, false
	}
	return f * mult, true
}
