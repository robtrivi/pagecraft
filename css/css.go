// Package css parses the CSS properties supported by the engine.
package css

import (
	"strings"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"
)

// Declarations contains normalized CSS properties and their values.
type Declarations map[string]string

// ParseInlineStyle parses inline CSS declarations and ignores invalid ones.
func ParseInlineStyle(style string) (Declarations, error) {
	decls, _, err := ParseInlineStyleWithWarnings(style)
	return decls, err
}

// ParseInlineStyleWithWarnings parses inline declarations and reports those it discards.
func ParseInlineStyleWithWarnings(style string) (Declarations, []string, error) {
	decls := Declarations{}
	var warnings []string
	for _, seg := range splitDeclarations(style) {
		prop, val, ok := parseSegment(seg)
		if !ok {
			warnings = append(warnings, "css: ignored invalid declaration: "+seg)
			continue
		}
		decls[prop] = val
	}
	return decls, warnings, nil
}

// splitDeclarations splits on semicolons without breaking functions (url(...),
// rgb(...)) or strings containing ;.
func splitDeclarations(style string) []string {
	var segs []string
	var sb strings.Builder
	depth := 0
	flush := func() {
		if s := strings.TrimSpace(sb.String()); s != "" {
			segs = append(segs, s)
		}
		sb.Reset()
	}
	l := css.NewLexer(parse.NewInputString(style))
	for {
		tt, data := l.Next()
		if tt == css.ErrorToken {
			break
		}
		switch tt {
		case css.SemicolonToken:
			if depth == 0 {
				flush()
				continue
			}
		case css.FunctionToken, css.LeftParenthesisToken:
			depth++
		case css.RightParenthesisToken:
			if depth > 0 {
				depth--
			}
		case css.WhitespaceToken:
			sb.WriteByte(' ')
			continue
		}
		sb.Write(data)
	}
	flush()
	return segs
}

// parseSegment parses "property: value". ok is false if the property is missing,
// is incomplete when the colon is missing or the value is empty.
func parseSegment(seg string) (prop, val string, ok bool) {
	l := css.NewLexer(parse.NewInputString(seg))
	for {
		tt, data := l.Next()
		if tt == css.ErrorToken {
			return "", "", false
		}
		if tt == css.WhitespaceToken || tt == css.CommentToken {
			continue
		}
		if tt != css.IdentToken {
			return "", "", false
		}
		prop = strings.ToLower(string(data))
		break
	}
	for {
		tt, _ := l.Next()
		if tt == css.ErrorToken {
			return "", "", false
		}
		if tt == css.WhitespaceToken || tt == css.CommentToken {
			continue
		}
		if tt != css.ColonToken {
			return "", "", false
		}
		break
	}
	var sb strings.Builder
	for {
		tt, data := l.Next()
		if tt == css.ErrorToken {
			break
		}
		if tt == css.CommentToken {
			continue
		}
		if tt == css.WhitespaceToken {
			sb.WriteByte(' ')
			continue
		}
		sb.Write(data)
	}
	if val = normalizeValue(sb.String()); val == "" {
		return "", "", false
	}
	return prop, val, true
}

func normalizeValue(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	s = strings.ReplaceAll(s, "! important", "")
	s = strings.TrimSuffix(s, "!important")
	return strings.TrimSpace(s)
}
