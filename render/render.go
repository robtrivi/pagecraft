// Package render converts HTML documents to PDF files.
package render

import (
	"errors"
	"strings"

	"github.com/robtrivi/pagecraft-go/boxes"
	"github.com/robtrivi/pagecraft-go/layout"
	"github.com/robtrivi/pagecraft-go/text"
)

// RenderHTMLToPDF converts HTML to PDF, with one PDF page per layout page.
func RenderHTMLToPDF(htmlStr string) ([]byte, error) {
	if strings.TrimSpace(htmlStr) == "" {
		return nil, errors.New("render: empty HTML")
	}
	root, err := boxes.Build(htmlStr)
	if err != nil {
		return nil, err
	}
	opts := layout.DefaultOptions()
	opts.Measure = text.Measure
	opts.MeasureBold = text.MeasureBold
	opts.Faces = root.Fonts
	opts.FontVariants = root.FontVariants
	if root.Page.Width > 0 && root.Page.Height > 0 {
		opts.PageWidth, opts.PageHeight = root.Page.Width, root.Page.Height
	}
	opts.PageMargins = root.Page.Margins.Resolve(opts.PageWidth, opts.PageHeight)
	opts.PageRotation = root.Page.Rotation
	opts.PageBleed, opts.PageMarks = root.Page.Bleed, root.Page.Marks
	if root.Page.HasMargin {
		opts.Margin = root.Page.Margin
	}
	if root.Page.HasBackground {
		opts.HasPageBackground, opts.PageBackground = true, root.Page.Background
	} else {
		opts.HasPageBackground, opts.PageBackground = root.Style.HasBg, root.Style.Bg
	}
	return buildPDFFromPages(layout.Flow(root, opts), opts, root.Meta, root.PageFooter), nil
}
