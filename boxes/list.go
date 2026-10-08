package boxes

import (
	"strconv"
	"strings"

	"github.com/robtrivi/pagecraft-go/css"
	"golang.org/x/net/html"
)

func applyMarkers(list *Box, decls css.Declarations, startAttr, defaultKind string) {
	kind, _ := decls["list-style-type"]
	if kind == "" {
		if defaultKind != "" {
			kind = defaultKind
		} else if list.Tag == "ol" {
			kind = "decimal"
		} else {
			kind = "disc"
		}
	}
	n := 1
	if list.Tag == "ol" {
		if s, err := strconv.Atoi(startAttr); err == nil && s > 0 {
			n = s
		}
	}
	for _, c := range list.Children {
		if c.Tag != "li" {
			continue
		}
		if m := listMarker(kind, n); m != "" {
			c.Text = m + " " + c.Text
			shape := ""
			if kind == "circle" || kind == "square" {
				shape = kind
				c.Style.ListMarkerShape = shape
			}
			c.Parts = append([]Part{{Text: m + " ", ListMarkerShape: shape}}, c.Parts...)
		}
		n++
	}
}

func defaultListStyle(element *html.Node) string {
	if element.Data != "ul" {
		return ""
	}
	depth := 0
	for parent := element.Parent; parent != nil; parent = parent.Parent {
		if parent.Type == html.ElementNode && (parent.Data == "ul" || parent.Data == "ol") {
			depth++
		}
	}
	if depth == 0 {
		return ""
	}
	switch (depth - 1) % 3 {
	case 0:
		return "circle"
	case 1:
		return "square"
	default:
		return "disc"
	}
}

// listMarker returns the marker (empty means none). Circle and square use
// ASCII fallbacks because WinAnsi does not include those glyphs.
func listMarker(kind string, n int) string {
	switch kind {
	case "none":
		return ""
	case "circle":
		return "o"
	case "square":
		return "-"
	case "decimal":
		return strconv.Itoa(n) + "."
	case "lower-alpha":
		return alphaNum(n, 'a') + "."
	case "upper-alpha":
		return alphaNum(n, 'A') + "."
	case "lower-roman":
		return strings.ToLower(romanNum(n)) + "."
	case "upper-roman":
		return romanNum(n) + "."
	default:
		return "•"
	}
}

// alphaNum maps 1 to a, 26 to z, and 27 to aa.
func alphaNum(n int, base rune) string {
	s := ""
	for n > 0 {
		n--
		s = string(rune(int(base)+n%26)) + s
		n /= 26
	}
	return s
}

// romanNum maps values from 1 to I through 3999 to MMMCMXCIX.
func romanNum(n int) string {
	if n < 1 {
		return strconv.Itoa(n)
	}
	var sb strings.Builder
	for _, p := range []struct {
		v int
		s string
	}{{1000, "M"}, {900, "CM"}, {500, "D"}, {400, "CD"}, {100, "C"}, {90, "XC"},
		{50, "L"}, {40, "XL"}, {10, "X"}, {9, "IX"}, {5, "V"}, {4, "IV"}, {1, "I"}} {
		for n >= p.v {
			sb.WriteString(p.s)
			n -= p.v
		}
	}
	return sb.String()
}

// parseLineHeight accepts a number or percentage; lengths are handled by tryLengthPx.
