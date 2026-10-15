package boxes

import (
	"bytes"
	"fmt"

	"github.com/robtrivi/pagecraft/css"
	"github.com/robtrivi/pagecraft/svg"
	"golang.org/x/net/html"
)

func buildReplacedContent(the *html.Node, b *Box, decls css.Declarations, ld *loader) error {
	if the.Data == "img" {
		if value, ok := decls["width"]; ok {
			if width, valid := parseFontSize(value); valid {
				b.Style.ImageWidth = width
			} else if percent, valid := parsePercent(value); valid {
				b.Style.ImageWidthPct = percent
			}
		}
		if value, ok := decls["height"]; ok {
			if height, valid := parseFontSize(value); valid {
				b.Style.ImageHeight = height
			} else if percent, valid := parsePercent(value); valid {
				b.Style.ImageHeightPct = percent
			}
		}
		if err := resolveImage(the, b, ld); err != nil {
			return err
		}
	}
	if the.Data == "svg" {
		var data bytes.Buffer
		if err := html.Render(&data, the); err != nil {
			return err
		}
		parsed, err := svg.Parse(data.Bytes())
		if err != nil {
			return fmt.Errorf("boxes: <svg>: %w", err)
		}
		b.SVG = parsed
		if value, ok := decls["width"]; ok {
			if size, ok := tryLengthPx(value); ok && size > 0 {
				b.Style.SVGWidth = size
			} else if pct, ok := parsePercent(value); ok {
				b.Style.SVGWidthPct = pct
			}
		}
		if value, ok := decls["height"]; ok {
			if size, ok := tryLengthPx(value); ok && size > 0 {
				b.Style.SVGHeight = size
			} else if pct, ok := parsePercent(value); ok {
				b.Style.SVGHeightPct = pct
			}
		}
	}
	return nil
}
