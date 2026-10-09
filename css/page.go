package css

import (
	"strconv"
	"strings"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"
)

// pageSizes stores portrait page dimensions in points (width by height).
var pageSizes = map[string][2]float64{
	"a0":      {2384, 3370},
	"a1":      {1684, 2384},
	"a2":      {1190, 1684},
	"a6":      {105 * 72 / 25.4, 148 * 72 / 25.4},
	"a5":      {420, 595},
	"a4":      {595, 842},
	"a3":      {842, 1190},
	"b5":      {498, 709},
	"b4":      {709, 1001},
	"letter":  {612, 792},
	"legal":   {612, 1008},
	"ledger":  {1224, 792},
	"tabloid": {792, 1224},
}

// PageMargins stores absolute values in points and percentage values in percent.
// Presence bits let layout inherit its default for omitted sides.
type PageMargins struct {
	Top, Right, Bottom, Left             float64
	TopPercent, RightPercent             bool
	BottomPercent, LeftPercent           bool
	HasTop, HasRight, HasBottom, HasLeft bool
}

// Resolve converts percentage margins to points using the page-box axes.
func (m PageMargins) Resolve(width, height float64) PageMargins {
	if m.TopPercent {
		m.Top, m.TopPercent = height*m.Top/100, false
	}
	if m.BottomPercent {
		m.Bottom, m.BottomPercent = height*m.Bottom/100, false
	}
	if m.LeftPercent {
		m.Left, m.LeftPercent = width*m.Left/100, false
	}
	if m.RightPercent {
		m.Right, m.RightPercent = width*m.Right/100, false
	}
	return m
}

// PageStyle is the page-box geometry parsed from one @page rule.
type PageStyle struct {
	Width, Height float64
	HasSize       bool
	Margins       PageMargins
	Background    [4]float64
	HasBackground bool
	Rotation      int // PDF page rotation in clockwise degrees.
}

// ParsePageStyle reads page-box size and margins from the first @page rule.
func ParsePageStyle(src string) (PageStyle, bool) {
	l := css.NewLexer(parse.NewInputString(src))
	for {
		tt, data := l.Next()
		if tt == css.ErrorToken {
			break
		}
		if tt != css.AtKeywordToken || !strings.EqualFold(strings.TrimPrefix(string(data), "@"), "page") {
			continue
		}
		body, ok := atBody(l)
		if !ok {
			continue
		}
		var page PageStyle
		for _, declaration := range splitDeclarations(body) {
			property, value, valid := parseSegment(declaration)
			if !valid {
				continue
			}
			switch property {
			case "size":
				if width, height, valid := parsePageSize(value); valid {
					page.Width, page.Height, page.HasSize = width, height, width > 0 && height > 0
				}
			case "background", "background-color":
				if color, valid := ParseColor(value); valid {
					page.Background, page.HasBackground = color, true
				}
			case "page-orientation":
				switch strings.ToLower(strings.TrimSpace(value)) {
				case "upright":
					page.Rotation = 0
				case "rotate-right":
					page.Rotation = 90
				case "rotate-left":
					page.Rotation = 270
				}
			case "margin":
				if values, valid := parsePageMarginShorthand(value); valid {
					page.Margins.set(values[0], values[1], values[2], values[3])
				}
			case "margin-top", "margin-right", "margin-bottom", "margin-left":
				if value, valid := parsePageMarginLength(value); valid {
					switch property {
					case "margin-top":
						page.Margins.Top, page.Margins.TopPercent, page.Margins.HasTop = value.value, value.percent, true
					case "margin-right":
						page.Margins.Right, page.Margins.RightPercent, page.Margins.HasRight = value.value, value.percent, true
					case "margin-bottom":
						page.Margins.Bottom, page.Margins.BottomPercent, page.Margins.HasBottom = value.value, value.percent, true
					case "margin-left":
						page.Margins.Left, page.Margins.LeftPercent, page.Margins.HasLeft = value.value, value.percent, true
					}
				}
			}
		}
		return page, true
	}
	return PageStyle{}, false
}

type pageMarginValue struct {
	value   float64
	percent bool
}

func (m *PageMargins) set(top, right, bottom, left pageMarginValue) {
	m.Top, m.Right, m.Bottom, m.Left = top.value, right.value, bottom.value, left.value
	m.TopPercent, m.RightPercent = top.percent, right.percent
	m.BottomPercent, m.LeftPercent = bottom.percent, left.percent
	m.HasTop, m.HasRight, m.HasBottom, m.HasLeft = true, true, true, true
}

func parsePageMarginShorthand(value string) ([4]pageMarginValue, bool) {
	fields := strings.Fields(value)
	var values [4]pageMarginValue
	if len(fields) < 1 || len(fields) > 4 {
		return values, false
	}
	for i, field := range fields {
		v, ok := parsePageMarginLength(field)
		if !ok {
			return values, false
		}
		values[i] = v
	}
	switch len(fields) {
	case 1:
		values[1], values[2], values[3] = values[0], values[0], values[0]
	case 2:
		values[2], values[3] = values[0], values[1]
	case 3:
		values[3] = values[1]
	}
	return values, true
}

func parsePageMarginLength(value string) (pageMarginValue, bool) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "0" {
		return pageMarginValue{}, true
	}
	negative := strings.HasPrefix(value, "-")
	if negative || strings.HasPrefix(value, "+") {
		value = value[1:]
	}
	percent := strings.HasSuffix(value, "%")
	if percent {
		value = strings.TrimSuffix(value, "%")
		v, err := strconv.ParseFloat(value, 64)
		if err != nil || v < 0 {
			return pageMarginValue{}, false
		}
		if negative {
			v = -v
		}
		return pageMarginValue{value: v, percent: true}, true
	}
	v, ok := parseLength(value)
	if negative {
		v = -v
	}
	return pageMarginValue{value: v}, ok
}

// ParsePageRule reads the first @page rule and returns its width, height, and margin in points.
// A missing size returns 0,0; a missing margin returns -1. It returns false only when no @page rule exists.
func ParsePageRule(src string) (w, h, margin float64, ok bool) {
	page, ok := ParsePageStyle(src)
	if !ok {
		return 0, 0, -1, false
	}
	w, h, margin = page.Width, page.Height, -1
	if page.Margins.HasTop && !page.Margins.TopPercent && !page.Margins.RightPercent && !page.Margins.BottomPercent && !page.Margins.LeftPercent && page.Margins.Top == page.Margins.Right &&
		page.Margins.Top == page.Margins.Bottom && page.Margins.Top == page.Margins.Left {
		margin = page.Margins.Top
	}
	return w, h, margin, true
}

// parsePageSize resolves "A4 landscape", "Letter", or "800px 600px".
func parsePageSize(s string) (float64, float64, bool) {
	fields := strings.Fields(strings.ToLower(s))
	landscape := false
	portrait := false
	var lens []float64
	for _, f := range fields {
		switch f {
		case "auto":
			if len(fields) != 1 {
				return 0, 0, false
			}
			return 0, 0, true
		case "landscape":
			landscape = true
		case "portrait":
			portrait = true
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
	if landscape && portrait {
		return 0, 0, false
	}
	if len(lens) == 0 && (landscape || portrait) {
		lens = []float64{pageSizes["a4"][0], pageSizes["a4"][1]}
	}
	if len(lens) == 1 {
		lens = append(lens, lens[0])
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
