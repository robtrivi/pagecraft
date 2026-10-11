package boxes

import (
	"github.com/robtrivi/pagecraft/css"
	"golang.org/x/net/html"
)

type textDecorations struct {
	underline bool
	strike    bool
}

func buildBox(the *html.Node, inherited css.Declarations, ss *styleSheet, ld *loader, ancestorDecorations textDecorations) (*Box, error) {
	b := newElementBox(the)
	decls := resolveBoxDeclarations(the, inherited, ss)
	applyLayoutStyle(b, the, decls)
	applyGeometryStyle(b, decls)
	applySpacingStyle(b, decls)
	applyPaintStyle(b, decls)
	applyTypographyStyle(b, the, decls)
	applyPaginationStyle(b, decls)
	applyTextStyle(b, decls, ancestorDecorations)
	if err := buildBoxChildren(the, b, decls, ss, ld); err != nil {
		return nil, err
	}
	if err := buildReplacedContent(the, b, decls, ld); err != nil {
		return nil, err
	}
	return b, nil
}
