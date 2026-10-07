package css

import (
	"strings"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"
)

// UAStylesheet contains the default styles for HTML elements.
const UAStylesheet = `
h1 { font-size: 24px; font-weight: bold; margin: .67em 0; }
h2 { font-size: 18px; font-weight: bold; margin: .83em 0; }
h3 { font-size: 14px; font-weight: bold; margin: 1em 0; }
h4 { font-size: 12px; font-weight: bold; margin: 1.33em 0; }
h5 { font-size: 10px; font-weight: bold; margin: 1.67em 0; }
h6 { font-size: 8px; font-weight: bold; margin: 2.33em 0; }
pre { white-space: pre; }
body { margin: 8px; }
th { font-weight: bold; }
td, th { vertical-align: middle; }
small { font-size: smaller; }
`

// PxToPt converts CSS pixels to PDF points at 96 DPI.
const PxToPt = 0.75

// Rule contains a selector, its declarations, and its order in the stylesheet.
type Rule struct {
	Selector string
	Decls    Declarations
	Order    int
}

// ParseStylesheet extracts selector rules in source order and ignores at-rules and invalid input.
func ParseStylesheet(src string) ([]Rule, error) {
	var rules []Rule
	l := css.NewLexer(parse.NewInputString(src))
	order := 0
	for {
		tt, data := l.Next()
		if tt == css.ErrorToken {
			break
		}
		switch tt {
		case css.WhitespaceToken, css.CommentToken, css.SemicolonToken:
			continue
		case css.LeftBraceToken, css.RightBraceToken:
			continue // Unmatched closing brace.
		case css.AtKeywordToken:
			skipAtRule(l)
			continue
		}
		// Read the possible rule prelude through the next top-level {.
		pre, ok := rulePrelude(l, tt, data)
		if !ok {
			continue
		}
		body, ok := ruleBody(l)
		if !ok {
			continue
		}
		decls, _ := ParseInlineStyle(body)
		rules = append(rules, Rule{Selector: pre, Decls: decls, Order: order})
		order++
	}
	return rules, nil
}

// rulePrelude reads the selector; the first token has already been consumed.
// ok is false if ; or EOF appears before {.
func rulePrelude(l *css.Lexer, first css.TokenType, data []byte) (string, bool) {
	var sb strings.Builder
	writeTok(&sb, first, data)
	depth := 0
	for {
		tt, data := l.Next()
		switch tt {
		case css.ErrorToken:
			return "", false
		case css.SemicolonToken:
			if depth == 0 {
				return "", false
			}
		case css.LeftBraceToken:
			if depth == 0 {
				if s := normalizeSelector(sb.String()); s != "" {
					return s, true
				}
				return "", false
			}
			depth++
		case css.RightBraceToken:
			if depth == 0 {
				return "", false
			}
			depth--
		case css.FunctionToken, css.LeftParenthesisToken:
			depth++
		case css.RightParenthesisToken:
			if depth > 0 {
				depth--
			}
		}
		writeTok(&sb, tt, data)
	}
}

// ruleBody reads declarations through the matching closing brace.
func ruleBody(l *css.Lexer) (string, bool) {
	var sb strings.Builder
	depth := 1
	for depth > 0 {
		tt, data := l.Next()
		if tt == css.ErrorToken {
			return "", false
		}
		switch tt {
		case css.LeftBraceToken:
			depth++
		case css.RightBraceToken:
			depth--
			if depth == 0 {
				continue
			}
		case css.WhitespaceToken:
			sb.WriteByte(' ')
			continue
		case css.CommentToken:
			continue
		}
		sb.Write(data)
	}
	return sb.String(), true
}

// skipAtRule skips an at-rule through its semicolon or matching closing brace.
func skipAtRule(l *css.Lexer) {
	depth := 0
	opened := false
	for {
		tt, _ := l.Next()
		if tt == css.ErrorToken {
			return
		}
		switch tt {
		case css.SemicolonToken:
			if depth == 0 {
				return
			}
		case css.LeftBraceToken:
			depth++
			opened = true
		case css.RightBraceToken:
			if depth == 0 {
				return
			}
			depth--
			if depth == 0 && opened {
				return
			}
		}
	}
}

func writeTok(sb *strings.Builder, tt css.TokenType, data []byte) {
	if tt == css.WhitespaceToken {
		sb.WriteByte(' ')
		return
	}
	if tt == css.CommentToken {
		return
	}
	sb.Write(data)
}

func normalizeSelector(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
