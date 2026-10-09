package render

import (
	"bytes"
	"compress/zlib"
	"encoding/hex"
	"fmt"
	"math"
	"strings"

	"github.com/robtrivi/pagecraft-go/css"
	"github.com/robtrivi/pagecraft-go/fonts"
	"github.com/robtrivi/pagecraft-go/images"
	"github.com/robtrivi/pagecraft-go/layout"
	"github.com/robtrivi/pagecraft-go/svg"
	"github.com/robtrivi/pagecraft-go/text"
)

func contentStream(pg layout.Page, opts layout.Options, imgIndex map[*images.Raster]int, svgIndex map[*svg.Document]int, fontIndex map[*fonts.Face]int, fontSubsets map[*fonts.Face]fonts.PDFSubset, footer string, gs map[float64]int, forms map[int]opacityFormRef) string {
	var sb strings.Builder
	fragmentNodes := fragmentNodesByPaintIndex(pg.FragmentTree)
	if opts.HasPageBackground {
		fmt.Fprintf(&sb, "%g %g %g rg\n0 0 %g %g re f\n", opts.PageBackground[0], opts.PageBackground[1], opts.PageBackground[2], opts.PageWidth, opts.PageHeight)
	}
	for bi := 0; bi < len(pg.Blocks); bi++ {
		b := pg.Blocks[bi]
		if form, ok := forms[bi]; ok {
			writeClipStart(&sb, b, opts.PageHeight)
			fmt.Fprintf(&sb, "q\n/GS%d gs\n/Fm%d Do\nQ\n", gs[form.opacity], form.id)
			writeClipEnd(&sb, b)
			bi = form.end
			continue
		}
		writeClipStart(&sb, b, opts.PageHeight)
		drawBackground(&sb, b, opts.PageHeight, gs)
		if node := fragmentNodes[bi]; node != nil {
			drawFragmentDecorations(&sb, node.Children, opts.PageHeight, gs)
		}
		if b.Image != nil {
			pdfY := opts.PageHeight - b.Y - b.Height
			fmt.Fprintf(&sb, "q %g 0 0 %g %g %g cm /Im%d Do Q\n",
				b.Width, b.Height, b.X, pdfY, imgIndex[b.Image])
		} else if b.SVG != nil {
			pdfY := opts.PageHeight - b.Y - b.Height
			fmt.Fprintf(&sb, "q %g 0 0 %g %g %g cm /Sv%d Do Q\n",
				b.Width/b.SVG.Width, b.Height/b.SVG.Height, b.X, pdfY, svgIndex[b.SVG])
		} else {
			drawRuns(&sb, b, opts.PageHeight, imgIndex, fontIndex, fontSubsets, gs)
		}
		writeClipEnd(&sb, b)
	}
	drawFooter(&sb, footer, opts)
	comp := flateBytes([]byte(sb.String()))
	return fmt.Sprintf("<< /Length %d /Filter /FlateDecode >>\nstream\n%s\nendstream", len(comp), comp)
}

func fragmentNodesByPaintIndex(roots []*layout.FragmentNode) map[int]*layout.FragmentNode {
	byPaintIndex := make(map[int]*layout.FragmentNode)
	var visit func([]*layout.FragmentNode)
	visit = func(nodes []*layout.FragmentNode) {
		for _, node := range nodes {
			if node.Kind == layout.FragmentPageBlock {
				byPaintIndex[node.Fragment.PaintIndex] = node
			}
			visit(node.Children)
		}
	}
	visit(roots)
	return byPaintIndex
}

func drawFragmentDecorations(sb *strings.Builder, children []*layout.FragmentNode, pageHeight float64, gs map[float64]int) {
	for _, child := range children {
		if child.Kind != layout.FragmentDecoration {
			continue
		}
		drawBackground(sb, child.Placed, pageHeight, gs)
		drawFragmentDecorations(sb, child.Children, pageHeight, gs)
	}
}

func writeClipStart(sb *strings.Builder, b layout.PlacedBox, pageH float64) {
	for _, clip := range b.ClipRects {
		pdfY := pageH - clip.Y - clip.Height
		fmt.Fprintf(sb, "q %g %g %g %g re W n\n", clip.X, pdfY, clip.Width, clip.Height)
	}
}

func writeClipEnd(sb *strings.Builder, b layout.PlacedBox) {
	for range b.ClipRects {
		sb.WriteString("Q\n")
	}
}

type opacityFormSpec struct {
	page, start, end, id, depth int
	opacity                     float64
}

func collectOpacityForms(pages []layout.Page) ([]opacityFormSpec, []map[int]map[int]opacityFormRef) {
	var forms []opacityFormSpec
	byPage := make([]map[int]map[int]opacityFormRef, len(pages))
	for pi, page := range pages {
		byPage[pi] = make(map[int]map[int]opacityFormRef)
		var collect func(start, end, depth int)
		collect = func(start, end, depth int) {
			for bi := start; bi <= end; bi++ {
				block := page.Blocks[bi]
				if depth < len(block.OpacityGroups) {
					layer := block.OpacityGroups[depth]
					groupEnd := bi
					for groupEnd+1 <= end && depth < len(page.Blocks[groupEnd+1].OpacityGroups) && page.Blocks[groupEnd+1].OpacityGroups[depth].ID == layer.ID {
						groupEnd++
					}
					id := len(forms) + 1
					forms = append(forms, opacityFormSpec{page: pi, start: bi, end: groupEnd, id: id, depth: depth, opacity: layer.Opacity})
					if byPage[pi][depth] == nil {
						byPage[pi][depth] = make(map[int]opacityFormRef)
					}
					byPage[pi][depth][bi] = opacityFormRef{id: id, end: groupEnd, opacity: layer.Opacity}
					collect(bi, groupEnd, depth+1)
					bi = groupEnd
					continue
				}
				if depth == 0 && block.HasOpacity && block.Opacity < 1 {
					id := len(forms) + 1
					forms = append(forms, opacityFormSpec{page: pi, start: bi, end: bi, id: id, opacity: block.Opacity})
					if byPage[pi][0] == nil {
						byPage[pi][0] = make(map[int]opacityFormRef)
					}
					byPage[pi][0][bi] = opacityFormRef{id: id, end: bi, opacity: block.Opacity}
				}
			}
		}
		if len(page.Blocks) > 0 {
			collect(0, len(page.Blocks)-1, 0)
		}
	}
	return forms, byPage
}

func opacityForm(blocks []layout.PlacedBox, form opacityFormSpec, forms map[int]map[int]opacityFormRef, fragmentNodes map[int]*layout.FragmentNode, opts layout.Options,
	imgIndex map[*images.Raster]int, svgIndex map[*svg.Document]int, fontIndex map[*fonts.Face]int, fontSubsets map[*fonts.Face]fonts.PDFSubset, gs map[float64]int, resources string) string {
	var content strings.Builder
	for bi := 0; bi < len(blocks); bi++ {
		block := blocks[bi]
		writeClipStart(&content, block, opts.PageHeight)
		if nested, ok := forms[form.depth+1][form.start+bi]; ok {
			fmt.Fprintf(&content, "q\n/GS%d gs\n/Fm%d Do\nQ\n", gs[nested.opacity], nested.id)
			writeClipEnd(&content, block)
			bi = nested.end - form.start
			continue
		}
		inner := block
		inner.HasOpacity = false
		for li := range inner.Runs {
			for ri := range inner.Runs[li] {
				run := &inner.Runs[li][ri]
				if run.GroupOpacity && form.opacity > 0 {
					run.Alpha /= form.opacity
				}
			}
		}
		drawBackground(&content, inner, opts.PageHeight, gs)
		if node := fragmentNodes[form.start+bi]; node != nil {
			drawFragmentDecorations(&content, node.Children, opts.PageHeight, gs)
		}
		if inner.Image != nil {
			pdfY := opts.PageHeight - inner.Y - inner.Height
			fmt.Fprintf(&content, "q %g 0 0 %g %g %g cm /Im%d Do Q\n", inner.Width, inner.Height, inner.X, pdfY, imgIndex[inner.Image])
		} else if inner.SVG != nil {
			pdfY := opts.PageHeight - inner.Y - inner.Height
			fmt.Fprintf(&content, "q %g 0 0 %g %g %g cm /Sv%d Do Q\n",
				inner.Width/inner.SVG.Width, inner.Height/inner.SVG.Height, inner.X, pdfY, svgIndex[inner.SVG])
		} else {
			drawRuns(&content, inner, opts.PageHeight, imgIndex, fontIndex, fontSubsets, gs)
		}
		writeClipEnd(&content, block)
	}
	compressed := flateBytes([]byte(content.String()))
	return fmt.Sprintf("<< /Type /XObject /Subtype /Form /FormType 1 /BBox [0 0 %g %g] /Group << /S /Transparency /CS /DeviceRGB /I true >> /Resources %s /Length %d /Filter /FlateDecode >>\nstream\n%s\nendstream",
		opts.PageWidth, opts.PageHeight, resources, len(compressed), compressed)
}

// flateBytes compresses data for PDF streams.
func flateBytes(raw []byte) []byte {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	w.Write(raw)
	w.Close()
	return buf.Bytes()
}

// footerSize is the fixed footer size (v1).
const footerSize = 10.0

// expandFooters evaluates footer chunks per page (page/pages are 1-based).
func expandFooters(footer []css.ContentChunk, total int) []string {
	if len(footer) == 0 {
		return make([]string, total)
	}
	out := make([]string, total)
	for i := 0; i < total; i++ {
		var sb strings.Builder
		for _, c := range footer {
			if c.Counter == "" {
				sb.WriteString(c.Literal)
				continue
			}
			if c.Counter == "page" {
				fmt.Fprintf(&sb, "%d", i+1)
			} else {
				fmt.Fprintf(&sb, "%d", total)
			}
		}
		out[i] = sb.String()
	}
	return out
}

// drawFooter centers footer text (halfway through the bottom margin).
func drawFooter(sb *strings.Builder, s string, opts layout.Options) {
	if s == "" {
		return
	}
	w := text.Measure(s, footerSize)
	x := (opts.PageWidth - w) / 2
	y := opts.Margin / 2
	fmt.Fprintf(sb, "BT /F1 %g Tf %g %g Td %s Tj ET\n",
		footerSize, x, y, encodePDFString(s))
}

// drawDecoration draws underlines (baseline-0.1em, AFM) and strike
// (x-height/2 is about 0.26em) using the given color.
func drawDecoration(sb *strings.Builder, w, r, g, bl float64, hasColor, underline, strike bool, fs, x, pdfY float64) {
	if !underline && !strike {
		return
	}
	if hasColor {
		fmt.Fprintf(sb, "%g %g %g RG\n", r, g, bl)
	} else {
		sb.WriteString("0 0 0 RG\n")
	}
	fmt.Fprintf(sb, "%g w\n", 0.05*fs)
	if underline {
		y := pdfY - 0.1*fs
		fmt.Fprintf(sb, "%g %g m %g %g l S\n", x, y, x+w, y)
	}
	if strike {
		y := pdfY + 0.26*fs
		fmt.Fprintf(sb, "%g %g m %g %g l S\n", x, y, x+w, y)
	}
}

func drawListMarker(sb *strings.Builder, shape string, cx, cy, fontSize, r, g, b float64) {
	size := fontSize * 0.28
	if size <= 0 {
		return
	}
	sb.WriteString("q\n")
	switch shape {
	case "circle":
		radius := size / 2
		k := 0.5522847498307936 * radius
		fmt.Fprintf(sb, "%g %g %g RG %g w\n", r, g, b, math.Max(0.5, fontSize*0.06))
		fmt.Fprintf(sb, "%g %g m\n", cx+radius, cy)
		fmt.Fprintf(sb, "%g %g %g %g %g %g c\n", cx+radius, cy+k, cx+k, cy+radius, cx, cy+radius)
		fmt.Fprintf(sb, "%g %g %g %g %g %g c\n", cx-k, cy+radius, cx-radius, cy+k, cx-radius, cy)
		fmt.Fprintf(sb, "%g %g %g %g %g %g c\n", cx-radius, cy-k, cx-k, cy-radius, cx, cy-radius)
		fmt.Fprintf(sb, "%g %g %g %g %g %g c S\n", cx+k, cy-radius, cx+radius, cy-k, cx+radius, cy)
	case "square":
		fmt.Fprintf(sb, "%g %g %g rg %g %g %g %g re f\n", r, g, b, cx-size/2, cy-size/2, size, size)
	}
	sb.WriteString("Q\n")
}

// drawRuns draws inline runs, each with its own style.
func drawRuns(sb *strings.Builder, b layout.PlacedBox, pageH float64, imgIndex map[*images.Raster]int, fontIndex map[*fonts.Face]int, fontSubsets map[*fonts.Face]fonts.PDFSubset, gs map[float64]int) {
	if b.LetterSpacing != 0 {
		fmt.Fprintf(sb, "%g Tc\n", b.LetterSpacing)
	}
	for _, ln := range b.Runs {
		for _, run := range ln {
			for _, inlineDecoration := range run.InlineDecorations {
				decoration := inlineDecoration.Box
				decoration.X = run.X + inlineDecoration.OffsetX
				decoration.Y = run.LineY + inlineDecoration.OffsetY
				drawBackground(sb, decoration, pageH, gs)
			}
		}
		for _, r := range ln {
			if r.Image != nil {
				pdfY := pageH - r.Y - r.ImageHeight
				withAlpha(sb, r.Alpha, gs, func() {
					fmt.Fprintf(sb, "q %g 0 0 %g %g %g cm /Im%d Do Q\n", r.Width, r.ImageHeight, r.X, pdfY, imgIndex[r.Image])
				})
				continue
			}
			baseline := r.BaselineOffset
			if baseline <= 0 {
				baseline = r.FontSize
			}
			pdfY := pageH - r.Y - baseline
			fname, draw := runFont(r, fontIndex)
			var cr, cg, cb float64
			if r.HasColor {
				cr, cg, cb = r.Color[0], r.Color[1], r.Color[2]
			}
			withAlpha(sb, r.Alpha, gs, func() {
				if r.HasColor {
					fmt.Fprintf(sb, "%g %g %g rg\n", cr, cg, cb)
				} else {
					sb.WriteString("0 0 0 rg\n")
				}
				if r.Font != nil && len(r.Shaped.Glyphs) > 0 {
					fmt.Fprintf(sb, "BT /%s %g Tf", fname, r.FontSize)
					if r.ListMarkerShape != "" {
						sb.WriteString(" 3 Tr")
					}
					sb.WriteByte('\n')
					pen := 0.0
					if r.Shaped.RightToLeft {
						pen = r.Width
						for _, glyph := range r.Shaped.Glyphs {
							pen -= glyph.Advance + r.LetterSpacing
							gid := glyph.ID
							if subset, ok := fontSubsets[r.Font]; ok {
								if compact, found := subset.GIDMap[gid]; found {
									gid = compact
								}
							}
							fmt.Fprintf(sb, "1 0 0 1 %g %g Tm <%04X> Tj\n", r.X+pen+glyph.XOffset, pdfY+glyph.YOffset, gid)
						}
					} else {
						for _, glyph := range r.Shaped.Glyphs {
							gid := glyph.ID
							if subset, ok := fontSubsets[r.Font]; ok {
								if compact, found := subset.GIDMap[gid]; found {
									gid = compact
								}
							}
							fmt.Fprintf(sb, "1 0 0 1 %g %g Tm <%04X> Tj\n", r.X+pen+glyph.XOffset, pdfY+glyph.YOffset, gid)
							pen += glyph.Advance + r.LetterSpacing
						}
					}
					if r.ListMarkerShape != "" {
						sb.WriteString("0 Tr\n")
					}
					sb.WriteString("ET\n")
				} else {
					fmt.Fprintf(sb, "BT /%s %g Tf", fname, r.FontSize)
					if r.ListMarkerShape != "" {
						sb.WriteString(" 3 Tr")
					}
					fmt.Fprintf(sb, " %g %g Td %s", r.X, pdfY, draw)
					if r.ListMarkerShape != "" {
						sb.WriteString(" 0 Tr")
					}
					sb.WriteString(" ET\n")
				}
				if r.ListMarkerShape != "" {
					drawListMarker(sb, r.ListMarkerShape, r.X+r.Width/2, pdfY+r.FontSize*0.32, r.FontSize, cr, cg, cb)
				} else {
					drawDecoration(sb, r.Width, cr, cg, cb, r.HasColor, r.Underline, r.Strike, r.FontSize, r.X, pdfY)
				}
			})
		}
	}
	if b.LetterSpacing != 0 {
		sb.WriteString("0 Tc\n")
	}
}

// runFont selects the font and encoding for a text run.
func runFont(r layout.TextRun, fontIndex map[*fonts.Face]int) (string, string) {
	if r.Font != nil {
		return fmt.Sprintf("F%d", fontIndex[r.Font]),
			"<" + strings.ToUpper(hex.EncodeToString(r.Font.Encode(r.Text))) + "> Tj"
	}
	switch {
	case r.Bold && r.Italic:
		return "F4", drawText(r.Text, r.Bold)
	case r.Bold:
		return "F2", drawText(r.Text, r.Bold)
	case r.Italic:
		return "F3", drawText(r.Text, r.Bold)
	default:
		return "F1", drawText(r.Text, r.Bold)
	}
}

// drawBackground paints a block's background and uniform border before its content.
func drawBackground(sb *strings.Builder, b layout.PlacedBox, pageH float64, gs map[float64]int) {
	if !b.HasBg && b.BorderWidth <= 0 && !hasSideBorders(b) {
		return
	}
	sb.WriteString("q\n")
	ry := pageH - b.Y - b.Height
	op := blockOpacity(b)
	radii := resolveBorderCornerRadii(b, b.Width, b.Height)
	if b.HasBg {
		withAlpha(sb, b.Bg[3]*op, gs, func() {
			fmt.Fprintf(sb, "%g %g %g rg\n", b.Bg[0], b.Bg[1], b.Bg[2])
			if hasEllipseRadius(radii) {
				drawRoundedEllipseRect(sb, b.X, ry, b.Width, b.Height, radii)
				sb.WriteString("f")
			} else {
				fmt.Fprintf(sb, "%g %g %g %g re f", b.X, ry, b.Width, b.Height)
			}
		})
	}
	if b.BorderWidth > 0 {
		withAlpha(sb, b.Border[3]*op, gs, func() {
			half := b.BorderWidth / 2
			fmt.Fprintf(sb, "%g w %g %g %g RG\n", b.BorderWidth, b.Border[0], b.Border[1], b.Border[2])
			if hasEllipseRadius(radii) {
				inner := radii
				for i := range inner {
					inner[i].X = math.Max(0, inner[i].X-half)
					inner[i].Y = math.Max(0, inner[i].Y-half)
				}
				drawRoundedEllipseRect(sb, b.X+half, ry+half, b.Width-b.BorderWidth, b.Height-b.BorderWidth, inner)
				sb.WriteString("S")
			} else {
				fmt.Fprintf(sb, "%g %g %g %g re S", b.X+half, ry+half, b.Width-b.BorderWidth, b.Height-b.BorderWidth)
			}
		})
	}
	drawSideBorders(sb, b, pageH, gs, op)
	sb.WriteString("Q\n")
}

type ellipseRadius struct {
	X, Y float64
}

func resolveBorderCornerRadii(b layout.PlacedBox, width, height float64) [4]ellipseRadius {
	var radii [4]ellipseRadius
	for i, corner := range b.BorderCornerRadii {
		radii[i].X = corner.X.Value
		if corner.X.Percent {
			radii[i].X = width * corner.X.Value / 100
		}
		radii[i].Y = corner.Y.Value
		if corner.Y.Percent {
			radii[i].Y = height * corner.Y.Value / 100
		}
	}
	return normalizeEllipticalRadii(width, height, radii)
}

func hasEllipseRadius(radii [4]ellipseRadius) bool {
	for _, radius := range radii {
		if radius.X > 0 || radius.Y > 0 {
			return true
		}
	}
	return false
}

func normalizeEllipticalRadii(width, height float64, radii [4]ellipseRadius) [4]ellipseRadius {
	for i := range radii {
		radii[i].X = math.Max(0, radii[i].X)
		radii[i].Y = math.Max(0, radii[i].Y)
	}
	scale := 1.0
	for _, pair := range []struct{ available, sum float64 }{
		{width, radii[0].X + radii[1].X},
		{width, radii[3].X + radii[2].X},
		{height, radii[0].Y + radii[3].Y},
		{height, radii[1].Y + radii[2].Y},
	} {
		if pair.sum > 0 && pair.available/pair.sum < scale {
			scale = pair.available / pair.sum
		}
	}
	if scale < 1 {
		for i := range radii {
			radii[i].X *= scale
			radii[i].Y *= scale
		}
	}
	return radii
}

// drawRoundedEllipseRect draws elliptical radii using the PDF bottom-left origin.
func drawRoundedEllipseRect(sb *strings.Builder, x, y, width, height float64, radii [4]ellipseRadius) {
	radii = normalizeEllipticalRadii(width, height, radii)
	tl, tr, br, bl := radii[0], radii[1], radii[2], radii[3]
	k := 0.5522847498307936
	fmt.Fprintf(sb, "%g %g m\n", x+tl.X, y)
	fmt.Fprintf(sb, "%g %g l\n", x+width-tr.X, y)
	fmt.Fprintf(sb, "%g %g %g %g %g %g c\n", x+width-tr.X+tr.X*k, y, x+width, y+tr.Y-tr.Y*k, x+width, y+tr.Y)
	fmt.Fprintf(sb, "%g %g l\n", x+width, y+height-br.Y)
	fmt.Fprintf(sb, "%g %g %g %g %g %g c\n", x+width, y+height-br.Y+br.Y*k, x+width-br.X+br.X*k, y+height, x+width-br.X, y+height)
	fmt.Fprintf(sb, "%g %g l\n", x+bl.X, y+height)
	fmt.Fprintf(sb, "%g %g %g %g %g %g c\n", x+bl.X-bl.X*k, y+height, x, y+height-bl.Y+bl.Y*k, x, y+height-bl.Y)
	fmt.Fprintf(sb, "%g %g l\n", x, y+tl.Y)
	fmt.Fprintf(sb, "%g %g %g %g %g %g c h\n", x, y+tl.Y-tl.Y*k, x+tl.X-tl.X*k, y, x+tl.X, y)
}

// withAlpha paints in isolation using alpha state (q/Q); it paints directly when opaque.
// Always appends a final newline to separate operators.
func withAlpha(sb *strings.Builder, alpha float64, gs map[float64]int, paint func()) {
	if alpha >= 1 {
		paint()
		sb.WriteByte('\n')
		return
	}
	fmt.Fprintf(sb, "q\n/GS%d gs\n", gs[alpha])
	paint()
	sb.WriteString("\nQ\n")
}

// blockOpacity returns the block opacity (1 when unset).
func blockOpacity(b layout.PlacedBox) float64 {
	if b.HasOpacity {
		return b.Opacity
	}
	return 1
}

// hasSideBorders reports whether any side has nonzero width.
func hasSideBorders(b layout.PlacedBox) bool {
	return b.BorderTop.Width > 0 || b.BorderRight.Width > 0 ||
		b.BorderBottom.Width > 0 || b.BorderLeft.Width > 0
}

// drawSideBorders paints sides that differ from the uniform border as filled rectangles.
// fill (the shorthand already sets all four sides, so `border:` alone does
// not duplicate it). Corners overlap cleanly.
func drawSideBorders(sb *strings.Builder, b layout.PlacedBox, pageH float64, gs map[float64]int, op float64) {
	yTop := pageH - b.Y
	yBot := yTop - b.Height
	x0 := b.X
	type seg struct {
		w, x, y, wdt, hgt float64
		c                 [4]float64
	}
	segs := []seg{
		{b.BorderTop.Width, x0, yTop - b.BorderTop.Width, b.Width, b.BorderTop.Width, b.BorderTop.Color},
		{b.BorderBottom.Width, x0, yBot, b.Width, b.BorderBottom.Width, b.BorderBottom.Color},
		{b.BorderLeft.Width, x0, yBot, b.BorderLeft.Width, b.Height, b.BorderLeft.Color},
		{b.BorderRight.Width, x0 + b.Width - b.BorderRight.Width, yBot, b.BorderRight.Width, b.Height, b.BorderRight.Color},
	}
	for _, s := range segs {
		if s.w <= 0 {
			continue
		}
		if b.BorderWidth == s.w && b.Border == s.c {
			continue
		}
		withAlpha(sb, s.c[3]*op, gs, func() {
			fmt.Fprintf(sb, "%g %g %g rg %g %g %g %g re f",
				s.c[0], s.c[1], s.c[2], s.x, s.y, s.wdt, s.hgt)
		})
	}
}

// linkAnnot is a clickable destination: PDF rectangle and URI.
