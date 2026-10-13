package boxes

import (
	"strconv"
	"strings"

	"github.com/robtrivi/pagecraft/css"
)

func defaultDisplay(tag string) Display {
	switch tag {
	case "table":
		return DisplayTable
	case "thead":
		return DisplayTableHeaderGroup
	case "tbody":
		return DisplayTableRowGroup
	case "tfoot":
		return DisplayTableFooterGroup
	case "tr":
		return DisplayTableRow
	case "colgroup":
		return DisplayTableColumnGroup
	case "col":
		return DisplayTableColumn
	case "td", "th":
		return DisplayTableCell
	case "caption":
		return DisplayTableCaption
	case "span", "a", "b", "i", "em", "strong", "small", "code", "img", "of-the", "s", "strike", "sub", "sup", "u":
		return DisplayInline
	default:
		return DisplayBlock
	}
}

// expandSides expands the CSS shorthand's one to four values.
func expandSides(s string) (top, right, bottom, left float64) {
	var vals []float64
	for _, p := range strings.Fields(s) {
		vals = append(vals, parsePx(p))
	}
	switch len(vals) {
	case 1:
		top, right, bottom, left = vals[0], vals[0], vals[0], vals[0]
	case 2:
		top, right, bottom, left = vals[0], vals[1], vals[0], vals[1]
	case 3:
		top, right, bottom, left = vals[0], vals[1], vals[2], vals[1]
	case 4:
		top, right, bottom, left = vals[0], vals[1], vals[2], vals[3]
	}
	return top, right, bottom, left
}

// RadiusLength stores an absolute or percentage CSS radius until the box size is known.
type RadiusLength struct {
	Value   float64
	Percent bool
}

// CornerRadius contains the horizontal and vertical radii of a corner.
type CornerRadius struct {
	X, Y RadiusLength
}

// BorderSide describes a border side by width and color. A zero width means no border.
type BorderSide struct {
	Width float64
	Color [4]float64
	Style string
}

// parsePercent parses a nonnegative "N%" value.
func parsePercent(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if !strings.HasSuffix(s, "%") {
		return 0, false
	}
	f, err := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 64)
	if err != nil || f <= 0 {
		return 0, false
	}
	return f, true
}

func parseTableBorderSpacing(value string) (horizontal, vertical float64, valid bool) {
	parts := strings.Fields(value)
	if len(parts) < 1 || len(parts) > 2 {
		return 0, 0, false
	}
	first, ok := parseCSSDimension(parts[0])
	if !ok || !first.IsLength() {
		return 0, 0, false
	}
	second := first
	if len(parts) == 2 {
		second, ok = parseCSSDimension(parts[1])
		if !ok || !second.IsLength() {
			return 0, 0, false
		}
	}
	return first.Value, second.Value, true
}

// countTracks counts tracks of grid-template-columns: list or repeat(n, ...).
func setBorderSide(b *Box, side string, v BorderSide) {
	switch side {
	case "Top":
		b.Style.BorderTop = v
	case "Right":
		b.Style.BorderRight = v
	case "Bottom":
		b.Style.BorderBottom = v
	case "Left":
		b.Style.BorderLeft = v
	}
}

// applyMarkers prefixes direct <li> text with its marker.
// The tag selects disc/decimal; start sets the initial number; none omits it.
func parseLineHeight(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "%") {
		f, err := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 64)
		if err != nil || f <= 0 {
			return 0, false
		}
		return f / 100, true
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f <= 0 {
		return 0, false
	}
	return f, true
}

// parseFontSize accepts px (converted to pt) and absolute pt values.
func parseFontSize(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "px") {
		f, err := strconv.ParseFloat(strings.TrimSuffix(s, "px"), 64)
		if err != nil || f <= 0 {
			return 0, false
		}
		return f * css.PxToPt, true
	}
	if strings.HasSuffix(s, "pt") {
		f, err := strconv.ParseFloat(strings.TrimSuffix(s, "pt"), 64)
		if err != nil || f <= 0 {
			return 0, false
		}
		return f, true
	}
	return 0, false
}

func tryLengthRelative(s string, fontSizePt float64) (float64, bool) {
	s = strings.TrimSpace(strings.ToLower(s))
	if strings.HasSuffix(s, "em") {
		v, err := strconv.ParseFloat(strings.TrimSuffix(s, "em"), 64)
		if err != nil {
			return 0, false
		}
		return v * fontSizePt, true
	}
	return tryLengthPx(s)
}

func expandSidesRelative(s string, fontSizePt float64) (top, right, bottom, left float64) {
	var vals []float64
	for _, part := range strings.Fields(s) {
		v, _ := tryLengthRelative(part, fontSizePt)
		vals = append(vals, v)
	}
	switch len(vals) {
	case 1:
		top, right, bottom, left = vals[0], vals[0], vals[0], vals[0]
	case 2:
		top, right, bottom, left = vals[0], vals[1], vals[0], vals[1]
	case 3:
		top, right, bottom, left = vals[0], vals[1], vals[2], vals[1]
	case 4:
		top, right, bottom, left = vals[0], vals[1], vals[2], vals[3]
	}
	return
}

func parsePx(s string) float64 {
	v, _ := tryLengthPx(s)
	return v
}

type gapComponent struct {
	length  float64
	percent float64
}

func parseGap(s string) (row, column gapComponent, ok bool) {
	values := strings.Fields(s)
	if len(values) < 1 || len(values) > 2 {
		return gapComponent{}, gapComponent{}, false
	}
	row, ok = parseGapComponent(values[0])
	if !ok {
		return gapComponent{}, gapComponent{}, false
	}
	column = row
	if len(values) == 2 {
		column, ok = parseGapComponent(values[1])
	}
	return row, column, ok
}

func parseGapComponent(s string) (gapComponent, bool) {
	if strings.HasSuffix(s, "%") {
		percent, err := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 64)
		return gapComponent{percent: percent}, err == nil && percent >= 0
	}
	length, ok := parseGapLength(s)
	return gapComponent{length: length}, ok
}

func parseBorderCornerRadii(value string) ([4]CornerRadius, bool) {
	var result [4]CornerRadius
	parts := strings.Split(value, "/")
	if len(parts) > 2 {
		return result, false
	}
	horizontal, ok := parseRadiusList(parts[0])
	if !ok {
		return result, false
	}
	vertical := horizontal
	if len(parts) == 2 {
		vertical, ok = parseRadiusList(parts[1])
		if !ok {
			return result, false
		}
	}
	x := expandRadiusList(horizontal)
	y := expandRadiusList(vertical)
	for i := range result {
		result[i] = CornerRadius{X: x[i], Y: y[i]}
	}
	return result, true
}

func parseRadiusList(value string) ([]RadiusLength, bool) {
	parts := strings.Fields(value)
	if len(parts) == 0 || len(parts) > 4 {
		return nil, false
	}
	values := make([]RadiusLength, len(parts))
	for i, part := range parts {
		if strings.HasSuffix(part, "%") {
			v, err := strconv.ParseFloat(strings.TrimSuffix(part, "%"), 64)
			if err != nil || v < 0 {
				return nil, false
			}
			values[i] = RadiusLength{Value: v, Percent: true}
			continue
		}
		v, ok := tryLengthPx(part)
		if !ok || v < 0 {
			return nil, false
		}
		values[i] = RadiusLength{Value: v}
	}
	return values, true
}

func expandRadiusList(values []RadiusLength) [4]RadiusLength {
	var result [4]RadiusLength
	switch len(values) {
	case 1:
		result = [4]RadiusLength{values[0], values[0], values[0], values[0]}
	case 2:
		result = [4]RadiusLength{values[0], values[1], values[0], values[1]}
	case 3:
		result = [4]RadiusLength{values[0], values[1], values[2], values[1]}
	case 4:
		copy(result[:], values)
	}
	return result
}

func parseGapLength(s string) (float64, bool) {
	v, ok := tryLengthPx(s)
	return v, ok && v >= 0
}

// tryLengthPx converts absolute CSS lengths to points.
func tryLengthPx(s string) (float64, bool) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "0" {
		return 0, true
	}
	unit, factor := "", 0.0
	switch {
	case strings.HasSuffix(s, "px"):
		unit, factor = "px", css.PxToPt
	case strings.HasSuffix(s, "pt"):
		unit, factor = "pt", 1
	case strings.HasSuffix(s, "pc"):
		unit, factor = "pc", 12
	case strings.HasSuffix(s, "in"):
		unit, factor = "in", 72
	case strings.HasSuffix(s, "cm"):
		unit, factor = "cm", 72/2.54
	case strings.HasSuffix(s, "mm"):
		unit, factor = "mm", 72/25.4
	case strings.HasSuffix(s, "q"):
		unit, factor = "q", 72/101.6
	default:
		return 0, false
	}
	f, err := strconv.ParseFloat(strings.TrimSuffix(s, unit), 64)
	if err != nil {
		return 0, false
	}
	return f * factor, true
}

func expandMarginTokens(value string) ([4]string, bool) {
	parts := strings.Fields(value)
	switch len(parts) {
	case 1:
		return [4]string{parts[0], parts[0], parts[0], parts[0]}, true
	case 2:
		return [4]string{parts[0], parts[1], parts[0], parts[1]}, true
	case 3:
		return [4]string{parts[0], parts[1], parts[2], parts[1]}, true
	case 4:
		return [4]string{parts[0], parts[1], parts[2], parts[3]}, true
	default:
		return [4]string{}, false
	}
}

// parseBorder parses "Npx style color" in any order. It is visible
// only with an explicit style and positive width; CSS draws no border without a style.
func parseBackgroundColor(value string) ([4]float64, bool) {
	if c, ok := css.ParseColor(value); ok {
		return c, true
	}
	lower := strings.ToLower(value)
	for _, fn := range []string{"rgba(", "rgb("} {
		if start := strings.Index(lower, fn); start >= 0 {
			if end := strings.IndexByte(value[start:], ')'); end >= 0 {
				if c, ok := css.ParseColor(value[start : start+end+1]); ok {
					return c, true
				}
			}
		}
	}
	for _, token := range strings.Fields(value) {
		if c, ok := css.ParseColor(strings.Trim(token, ",")); ok {
			return c, true
		}
	}
	return [4]float64{}, false
}

func parseBorder(s string) (float64, [4]float64, bool) {
	width := 0.0
	col := [4]float64{0, 0, 0, 1}
	styled, hasWidth := false, false
	for _, tok := range strings.Fields(s) {
		switch tok {
		case "none", "hidden":
			return 0, [4]float64{}, false
		case "solid", "dotted", "dashed", "double", "groove", "ridge", "inset", "outset":
			styled = true
			continue
		}
		if v, ok := tryLengthPx(tok); ok {
			width, hasWidth = v, true
			continue
		}
		if c, ok := css.ParseColor(tok); ok {
			col = c
		}
	}
	if !styled || !hasWidth || width <= 0 {
		return 0, [4]float64{}, false
	}
	return width, col, true
}

func borderStyleToken(value string) string {
	for _, token := range strings.Fields(strings.ToLower(value)) {
		switch token {
		case "none", "hidden", "solid", "dotted", "dashed", "double", "groove", "ridge", "inset", "outset":
			return token
		}
	}
	return "none"
}
