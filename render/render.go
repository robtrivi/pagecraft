// Package render converts HTML documents to PDF files.
package render

import (
	"errors"
	"strings"

	"github.com/robtrivi/pagecraft/boxes"
	"github.com/robtrivi/pagecraft/css"
	"github.com/robtrivi/pagecraft/layout"
	"github.com/robtrivi/pagecraft/text"
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
	basePage := css.ResolvePageDefaults(root.PageRules)
	if basePage.HasSize {
		opts.PageWidth, opts.PageHeight = basePage.Width, basePage.Height
	}
	opts.PageRules = root.PageRules
	opts.PageMargins = basePage.Margins.Resolve(opts.PageWidth, opts.PageHeight)
	if basePage.HasRotation {
		opts.PageRotation = basePage.Rotation
	}
	if basePage.HasBleed {
		opts.PageBleed = basePage.Bleed
	}
	if basePage.HasMarks {
		opts.PageMarks = basePage.Marks
	}
	if basePage.HasBackground {
		opts.HasPageBackground, opts.PageBackground = true, basePage.Background
	} else {
		opts.HasPageBackground, opts.PageBackground = root.Style.HasBg, root.Style.Bg
	}
	return buildPDFFromPages(layout.Flow(root, opts), opts, root.Meta, root.PageFooter), nil
}
