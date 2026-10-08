package boxes

import (
	"strconv"
	"strings"

	"github.com/robtrivi/pagecraft-go/css"
	"golang.org/x/net/html"
)

func resolveBoxDeclarations(the *html.Node, inherited css.Declarations, ss *styleSheet) css.Declarations {
	own, _ := css.ParseInlineStyle(getAttr(the, "style"))
	expandFontShorthand(own)
	decls := make(css.Declarations)
	for k, v := range inherited {
		if inheritable[k] {
			decls[k] = v
		}
	}
	hasCascadedFontWeight := false
	if ss != nil {
		cascaded := ss.cascaded(the)
		_, hasCascadedFontWeight = cascaded["font-weight"]
		for k, v := range cascaded {
			decls[k] = v
		}
	}
	_, hasOwnFontWeight := own["font-weight"]
	if (the.Data == "b" || the.Data == "strong") && !hasCascadedFontWeight && !hasOwnFontWeight {
		decls["font-weight"] = "bold"
	}
	for k, v := range own {
		decls[k] = v
	}
	if fontSize, ok := decls["font-size"]; ok {
		decls["font-size"] = resolveRelativeFontSize(fontSize, inherited)
	}
	return decls
}

func resolveRelativeFontSize(value string, inherited css.Declarations) string {
	parentSize := 12.0
	if inheritedSize, ok := inherited["font-size"]; ok {
		if size, valid := parseFontSize(inheritedSize); valid {
			parentSize = size
		}
	}
	value = strings.TrimSpace(strings.ToLower(value))
	var size float64
	switch value {
	case "smaller":
		size = parentSize * 5 / 6
	case "larger":
		size = parentSize * 6 / 5
	default:
		if strings.HasSuffix(value, "%") {
			percentage, err := strconv.ParseFloat(strings.TrimSuffix(value, "%"), 64)
			if err == nil && percentage > 0 {
				size = parentSize * percentage / 100
			}
		} else if strings.HasSuffix(value, "em") {
			multiple, err := strconv.ParseFloat(strings.TrimSuffix(value, "em"), 64)
			if err == nil && multiple > 0 {
				size = parentSize * multiple
			}
		}
	}
	if size > 0 {
		return strconv.FormatFloat(size, 'f', -1, 64) + "pt"
	}
	return value
}

func newElementBox(the *html.Node) *Box {
	b := &Box{Tag: the.Data, Style: ComputedStyle{Display: defaultDisplay(the.Data), Position: PositionStatic, Float: FloatNone}}
	// HTML presentational defaults; CSS declarations are applied afterward.
	switch the.Data {
	case "b", "strong":
		b.Style.Bold = true
	case "i", "em":
		b.Style.Italic = true
	case "u":
		b.Style.Underline = true
	case "s", "strike", "of-the":
		b.Style.Strike = true
	case "sub":
		b.Style.VAlign = "sub"
	case "sup":
		b.Style.VAlign = "super"
	}
	return b
}

func elementAttribute(the *html.Node, name string) string {
	for _, attr := range the.Attr {
		if strings.EqualFold(attr.Key, name) {
			return attr.Val
		}
	}
	return ""
}

var inheritable = map[string]bool{"color": true, "font-size": true, "font-weight": true, "font-style": true, "font-family": true, "text-align": true, "line-height": true, "letter-spacing": true, "white-space": true, "text-transform": true, "list-style-position": true, "widows": true, "orphans": true}

func expandFontShorthand(decls css.Declarations) {
	value, ok := decls["font"]
	if !ok {
		return
	}
	fields := strings.Fields(value)
	for sizeIndex, field := range fields {
		sizePart, lineHeight, hasLineHeight := strings.Cut(field, "/")
		if _, valid := parseFontSize(sizePart); !valid {
			continue
		}
		if _, explicit := decls["font-size"]; !explicit {
			decls["font-size"] = sizePart
		}
		if _, explicit := decls["font-weight"]; !explicit {
			weight := "normal"
			for _, token := range fields[:sizeIndex] {
				if token == "bold" || token == "bolder" || isNumericFontWeight(token) {
					weight = token
				}
			}
			decls["font-weight"] = weight
		}
		if _, explicit := decls["font-style"]; !explicit {
			style := "normal"
			for _, token := range fields[:sizeIndex] {
				if token == "italic" || token == "oblique" {
					style = token
				}
			}
			decls["font-style"] = style
		}
		if _, explicit := decls["line-height"]; !explicit {
			if hasLineHeight {
				decls["line-height"] = lineHeight
			} else {
				decls["line-height"] = "normal"
			}
		}
		if sizeIndex+1 < len(fields) {
			if _, explicit := decls["font-family"]; !explicit {
				decls["font-family"] = strings.Join(fields[sizeIndex+1:], " ")
			}
		}
		return
	}
}

func isNumericFontWeight(value string) bool {
	weight, err := strconv.Atoi(value)
	return err == nil && weight >= 100 && weight <= 900
}
