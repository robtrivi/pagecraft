package css

import (
	"strconv"
	"strings"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"
)

// FontFace represents an @font-face rule with a family and unresolved src value.
type FontFace struct {
	Family string
	Src    string
	Weight int
	Style  string
}

// ParseFontFaces returns @font-face rules in source order.
func ParseFontFaces(src string) ([]FontFace, error) {
	var out []FontFace
	l := css.NewLexer(parse.NewInputString(src))
	for {
		tt, data := l.Next()
		if tt == css.ErrorToken {
			break
		}
		if tt != css.AtKeywordToken || !strings.EqualFold(strings.TrimPrefix(string(data), "@"), "font-face") {
			continue
		}
		body, ok := atBody(l)
		if !ok {
			continue
		}
		decls, _ := ParseInlineStyle(body)
		fam, okF := decls["font-family"]
		src, okS := decls["src"]
		if !okF || !okS {
			continue
		}
		fam = unquote(strings.TrimSpace(fam))
		u, okU := extractURL(src)
		if fam == "" || !okU || u == "" {
			continue
		}
		weight := 400
		if raw := strings.TrimSpace(decls["font-weight"]); raw != "" {
			switch strings.ToLower(raw) {
			case "normal":
				weight = 400
			case "bold":
				weight = 700
			default:
				if n, err := strconv.Atoi(raw); err == nil && n >= 1 && n <= 1000 {
					weight = n
				}
			}
		}
		style := strings.ToLower(strings.TrimSpace(decls["font-style"]))
		if style == "" || (style != "italic" && style != "oblique") {
			style = "normal"
		}
		out = append(out, FontFace{Family: fam, Src: u, Weight: weight, Style: style})
	}
	return out, nil
}

// atBody consumes through { and returns the balanced rule body.
func atBody(l *css.Lexer) (string, bool) {
	for {
		tt, _ := l.Next()
		if tt == css.ErrorToken {
			return "", false
		}
		if tt == css.SemicolonToken {
			return "", false
		}
		if tt == css.LeftBraceToken {
			return ruleBody(l)
		}
	}
}

// unquote removes surrounding single or double quotes.
func unquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// extractURL returns the target of url(...), with or without quotes.
func extractURL(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if len(s) < 6 || !strings.EqualFold(s[:4], "url(") || !strings.HasSuffix(s, ")") {
		return "", false
	}
	return unquote(strings.TrimSpace(s[4 : len(s)-1])), true
}
