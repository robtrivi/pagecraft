package css

import (
	"strings"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"
)

// ContentChunk is a literal or counter in a page footer's content.
type ContentChunk struct {
	Literal string
	Counter string // "page" | "pages" ("" = literal)
}

// PageFooterStyle contains the supported typography and color for @bottom-center.
type PageFooterStyle struct {
	FontSize float64
	Color    [4]float64
	HasColor bool
}

// ParsePageFooterStyle reads the @bottom-center content and supported style.
func ParsePageFooterStyle(src string) ([]ContentChunk, PageFooterStyle, bool) {
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
		if chunks, style, ok := pageFooterDetails(body); ok {
			return chunks, style, true
		}
	}
	return nil, PageFooterStyle{}, false
}

// ParsePageFooter reads @bottom-center content from the first @page rule that defines it.
func ParsePageFooter(src string) ([]ContentChunk, bool) {
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
		if chunks, ok := pageFooterIn(body); ok {
			return chunks, true
		}
	}
	return nil, false
}

func pageFooterIn(pageBody string) ([]ContentChunk, bool) {
	chunks, _, ok := pageFooterDetails(pageBody)
	return chunks, ok
}

func pageFooterDetails(pageBody string) ([]ContentChunk, PageFooterStyle, bool) {
	l := css.NewLexer(parse.NewInputString(pageBody))
	for {
		tt, data := l.Next()
		if tt == css.ErrorToken {
			break
		}
		if tt != css.AtKeywordToken || !strings.EqualFold(strings.TrimPrefix(string(data), "@"), "bottom-center") {
			continue
		}
		body, ok := atBody(l)
		if !ok {
			continue
		}
		decls, _ := ParseInlineStyle(body)
		content, ok := decls["content"]
		if !ok || content == "" || content == "none" {
			continue
		}
		if chunks := parseContent(content); len(chunks) > 0 {
			style := PageFooterStyle{FontSize: 10}
			if color, valid := ParseColor(decls["color"]); valid {
				style.Color, style.HasColor = color, true
			}
			font := decls["font"]
			if font == "" {
				font = decls["font-size"]
			}
			for _, token := range strings.Fields(font) {
				if size, valid := parseLength(token); valid {
					style.FontSize = size
					break
				}
			}
			return chunks, style, true
		}
	}
	return nil, PageFooterStyle{}, false
}

// parseContent splits quoted literals and counter(name) expressions.
func parseContent(s string) []ContentChunk {
	var out []ContentChunk
	l := css.NewLexer(parse.NewInputString(s))
	for {
		tt, data := l.Next()
		if tt == css.ErrorToken {
			break
		}
		switch tt {
		case css.StringToken:
			if lit := unquoteCSS(string(data)); lit != "" {
				out = append(out, ContentChunk{Literal: lit})
			}
		case css.FunctionToken:
			if !strings.EqualFold(strings.TrimSuffix(string(data), "("), "counter") {
				break
			}
			name := ""
			for {
				tt2, data2 := l.Next()
				if tt2 == css.ErrorToken {
					break
				}
				if tt2 == css.IdentToken && name == "" {
					name = strings.ToLower(string(data2))
				}
				if tt2 == css.RightParenthesisToken {
					break
				}
			}
			if name == "page" || name == "pages" {
				out = append(out, ContentChunk{Counter: name})
			}
		}
	}
	return out
}

// unquoteCSS removes quotes and resolves escaped quotes and backslashes.
func unquoteCSS(s string) string {
	if len(s) >= 2 && ((s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'')) {
		s = s[1 : len(s)-1]
	}
	s = strings.ReplaceAll(s, `\\`, "\x00")
	s = strings.ReplaceAll(s, `\"`, `"`)
	s = strings.ReplaceAll(s, `\'`, `'`)
	return strings.ReplaceAll(s, "\x00", `\`)
}
