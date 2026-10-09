package css

import (
	"sort"
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
	HasRotation   bool
	Bleed         float64
	HasBleed      bool
	Marks         PageMarks
	HasMarks      bool
}

// PageMarks selects printer registration marks from the CSS marks descriptor.
type PageMarks uint8

const (
	PageMarkCrop PageMarks = 1 << iota
	PageMarkCross
)

// PageSelector identifies the named page and optional page pseudo-class.
type PageSelector struct {
	Name   string
	Pseudo string
}

// PageRule keeps each @page selector and its computed declarations in source order.
type PageRule struct {
	Selector     PageSelector
	Style        PageStyle
	Declarations Declarations
	MarginBoxes  map[string]Declarations
}

// ParsePageRules reads all @page rules, retaining selectors and source order.
func ParsePageRules(src string) []PageRule {
	l := css.NewLexer(parse.NewInputString(src))
	var rules []PageRule
	for {
		tt, data := l.Next()
		if tt == css.ErrorToken {
			break
		}
		if tt != css.AtKeywordToken || !strings.EqualFold(strings.TrimPrefix(string(data), "@"), "page") {
			continue
		}
		var prelude strings.Builder
		var body string
		for {
			tok, part := l.Next()
			if tok == css.ErrorToken || tok == css.SemicolonToken {
				body = ""
				break
			}
			if tok == css.LeftBraceToken {
				body, _ = ruleBody(l)
				break
			}
			prelude.Write(part)
		}
		if body == "" {
			continue
		}
		selector, ok := parsePageSelector(prelude.String())
		if !ok {
			continue
		}
		style, _ := ParsePageStyle("@page {" + body + "}")
		marginBoxes := parsePageMarginBoxes(body)
		declarations, _, _ := ParseInlineStyleWithWarnings(body)
		for name := range marginBoxes {
			delete(declarations, "@"+name)
		}
		rules = append(rules, PageRule{Selector: selector, Style: style, Declarations: declarations, MarginBoxes: marginBoxes})
	}
	return rules
}

var pageMarginBoxNames = map[string]bool{
	"top-left-corner": true, "top-left": true, "top-center": true, "top-right": true, "top-right-corner": true,
	"right-top": true, "right-middle": true, "right-bottom": true,
	"bottom-left-corner": true, "bottom-left": true, "bottom-center": true, "bottom-right": true, "bottom-right-corner": true,
	"left-top": true, "left-middle": true, "left-bottom": true,
}

func parsePageMarginBoxes(body string) map[string]Declarations {
	boxes := make(map[string]Declarations)
	l := css.NewLexer(parse.NewInputString(body))
	for {
		tt, data := l.Next()
		if tt == css.ErrorToken {
			break
		}
		if tt != css.AtKeywordToken {
			continue
		}
		name := strings.ToLower(strings.TrimPrefix(string(data), "@"))
		if !pageMarginBoxNames[name] {
			continue
		}
		boxBody, ok := atBody(l)
		if !ok {
			continue
		}
		decls, _, _ := ParseInlineStyleWithWarnings(boxBody)
		boxes[name] = decls
	}
	return boxes
}

// ResolvePageStyle applies matching @page rules in source order. Page numbers are one-based.
func ResolvePageStyle(rules []PageRule, name string, pageNumber int) PageStyle {
	return ResolvePageRule(rules, name, pageNumber).Style
}

// ResolvePageRule cascades descriptors and margin-box declarations for one page context.
func ResolvePageRule(rules []PageRule, name string, pageNumber int) PageRule {
	result := PageRule{Selector: PageSelector{Name: name}, Declarations: Declarations{}, MarginBoxes: map[string]Declarations{}}
	var matching []PageRule
	for _, rule := range rules {
		if pageSelectorMatches(rule.Selector, name, pageNumber) {
			matching = append(matching, rule)
		}
	}
	sort.SliceStable(matching, func(i, j int) bool {
		return pageSelectorSpecificity(matching[i].Selector) < pageSelectorSpecificity(matching[j].Selector)
	})
	for _, rule := range matching {
		mergePageStyle(&result.Style, rule.Style)
		for property, value := range rule.Declarations {
			result.Declarations[property] = value
		}
		for name, declarations := range rule.MarginBoxes {
			if result.MarginBoxes[name] == nil {
				result.MarginBoxes[name] = Declarations{}
			}
			for property, value := range declarations {
				result.MarginBoxes[name][property] = value
			}
		}
	}
	return result
}

func pageSelectorSpecificity(selector PageSelector) int {
	specificity := 0
	if selector.Name != "" {
		specificity++
	}
	if selector.Pseudo != "" {
		specificity++
	}
	return specificity
}

func pageSelectorMatches(selector PageSelector, name string, pageNumber int) bool {
	if selector.Name != "" && selector.Name != name {
		return false
	}
	switch selector.Pseudo {
	case "":
		return true
	case "first":
		return pageNumber == 1
	case "left":
		return pageNumber > 1 && pageNumber%2 == 0
	case "right":
		return pageNumber%2 == 1
	case "blank":
		return false // Blank pages are generated only after pagination knows their role.
	default:
		return false
	}
}

func mergePageStyle(dst *PageStyle, src PageStyle) {
	if src.HasSize {
		dst.Width, dst.Height, dst.HasSize = src.Width, src.Height, true
	}
	if src.Margins.HasTop {
		dst.Margins.Top, dst.Margins.TopPercent, dst.Margins.HasTop = src.Margins.Top, src.Margins.TopPercent, true
	}
	if src.Margins.HasRight {
		dst.Margins.Right, dst.Margins.RightPercent, dst.Margins.HasRight = src.Margins.Right, src.Margins.RightPercent, true
	}
	if src.Margins.HasBottom {
		dst.Margins.Bottom, dst.Margins.BottomPercent, dst.Margins.HasBottom = src.Margins.Bottom, src.Margins.BottomPercent, true
	}
	if src.Margins.HasLeft {
		dst.Margins.Left, dst.Margins.LeftPercent, dst.Margins.HasLeft = src.Margins.Left, src.Margins.LeftPercent, true
	}
	if src.HasBackground {
		dst.Background, dst.HasBackground = src.Background, true
	}
	if src.HasRotation {
		dst.Rotation, dst.HasRotation = src.Rotation, true
	}
	if src.HasBleed {
		dst.Bleed, dst.HasBleed = src.Bleed, true
	}
	if src.HasMarks {
		dst.Marks, dst.HasMarks = src.Marks, true
	}
}

func parsePageSelector(prelude string) (PageSelector, bool) {
	fields := strings.Fields(strings.TrimSpace(prelude))
	if len(fields) == 0 {
		return PageSelector{}, true
	}
	if len(fields) != 1 {
		return PageSelector{}, false
	}
	parts := strings.Split(fields[0], ":")
	selector := PageSelector{Name: parts[0]}
	if selector.Name != "" && !isCSSIdent(selector.Name) {
		return PageSelector{}, false
	}
	if len(parts) > 2 {
		return PageSelector{}, false
	}
	if len(parts) == 2 {
		switch strings.ToLower(parts[1]) {
		case "first", "left", "right", "blank":
			selector.Pseudo = strings.ToLower(parts[1])
		default:
			return PageSelector{}, false
		}
	}
	return selector, true
}

func isCSSIdent(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if !(r == '-' || r == '_' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9') {
			return false
		}
	}
	return true
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
					page.Rotation, page.HasRotation = 0, true
				case "rotate-right":
					page.Rotation, page.HasRotation = 90, true
				case "rotate-left":
					page.Rotation, page.HasRotation = 270, true
				}
			case "bleed":
				if bleed, valid := parsePageMarginLength(value); valid && !bleed.percent && bleed.value >= 0 {
					page.Bleed, page.HasBleed = bleed.value, true
				}
			case "marks":
				marks := PageMarks(0)
				valid := true
				for _, mark := range strings.Fields(strings.ToLower(value)) {
					switch mark {
					case "none":
					case "crop":
						marks |= PageMarkCrop
					case "cross":
						marks |= PageMarkCross
					default:
						valid = false
					}
				}
				if valid && strings.TrimSpace(value) != "" {
					page.Marks, page.HasMarks = marks, true
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
