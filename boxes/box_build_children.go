package boxes

import (
	"strings"

	"github.com/robtrivi/pagecraft/css"
	"golang.org/x/net/html"
)

func buildBoxChildren(the *html.Node, b *Box, decls css.Declarations, ss *styleSheet, ld *loader) error {
	var raw strings.Builder
	for c := the.FirstChild; c != nil; c = c.NextSibling {
		switch c.Type {
		case html.TextNode:
			b.Parts = append(b.Parts, Part{Text: transformText(c.Data, b.Style.Transform)})
			raw.WriteString(transformText(c.Data, b.Style.Transform))
		case html.ElementNode:
			cb, err := buildBox(c, decls, ss, ld, textDecorations{underline: b.Style.Underline, strike: b.Style.Strike})
			if err != nil {
				return err
			}
			if cb == nil {
				continue
			}
			cb.Parent = b
			b.Children = append(b.Children, cb)
			b.Parts = append(b.Parts, Part{Child: cb})
			if the.Data != "li" || c.Data != "ul" && c.Data != "ol" {
				raw.WriteString(transformText(textContent(c), cb.Style.Transform))
			}
		}
	}
	b.Text = strings.Join(strings.Fields(raw.String()), " ")
	collapseSingleChildMargins(b)
	if the.Data == "ul" || the.Data == "ol" {
		applyMarkers(b, decls, getAttr(the, "start"), defaultListStyle(the))
	}
	return nil
}
