package render

import (
	"fmt"
	"sort"
	"strings"

	"github.com/robtrivi/pagecraft-go/boxes"
	"github.com/robtrivi/pagecraft-go/images"
	"github.com/robtrivi/pagecraft-go/layout"
	"github.com/robtrivi/pagecraft-go/svg"
)

func collectAlphas(pages []layout.Page) ([]float64, map[float64]int) {
	seen := map[float64]bool{}
	var alphas []float64
	add := func(a float64) {
		if a >= 1 || seen[a] {
			return
		}
		seen[a] = true
		alphas = append(alphas, a)
	}
	for _, pg := range pages {
		for _, b := range pg.Blocks {
			for _, group := range b.OpacityGroups {
				add(group.Opacity)
			}
			if b.SVG != nil {
				for _, shape := range b.SVG.Shapes {
					if shape.HasFill {
						add(shape.Fill[3])
					}
					if shape.HasStroke {
						add(shape.Stroke[3])
					}
				}
			}
			op := 1.0
			if b.HasOpacity {
				op = b.Opacity
				add(op)
			}
			if b.HasBg {
				add(b.Bg[3] * op)
			}
			if b.BorderWidth > 0 {
				add(b.Border[3] * op)
			}
			for _, s := range []boxes.BorderSide{b.BorderTop, b.BorderRight, b.BorderBottom, b.BorderLeft} {
				if s.Width > 0 {
					add(s.Color[3] * op)
				}
			}
			for _, ln := range b.Runs {
				for _, r := range ln {
					add(r.Alpha)
				}
			}
		}
	}
	sort.Float64s(alphas)
	index := map[float64]int{}
	for i, a := range alphas {
		index[a] = i + 1
	}
	return alphas, index
}

// extRefs builds " /ExtGState << /GS1 20 0 R ... >>" or an empty string.
func extRefs(alphas []float64, base int) string {
	if len(alphas) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(" /ExtGState <<")
	for i := range alphas {
		fmt.Fprintf(&sb, " /GS%d %d 0 R", i+1, base+i)
	}
	sb.WriteString(" >>")
	return sb.String()
}
func collectImages(pages []layout.Page) ([]*images.Raster, map[*images.Raster]int) {
	var imgs []*images.Raster
	index := map[*images.Raster]int{}
	for _, pg := range pages {
		for _, b := range pg.Blocks {
			if b.Image != nil {
				if _, ok := index[b.Image]; !ok {
					index[b.Image] = len(imgs) + 1
					imgs = append(imgs, b.Image)
				}
			}
			for _, line := range b.Runs {
				for _, run := range line {
					if run.Image == nil {
						continue
					}
					if _, ok := index[run.Image]; !ok {
						index[run.Image] = len(imgs) + 1
						imgs = append(imgs, run.Image)
					}
				}
			}
		}
	}
	return imgs, index
}

func collectSVGs(pages []layout.Page) ([]*svg.Document, map[*svg.Document]int) {
	var docs []*svg.Document
	index := map[*svg.Document]int{}
	for _, pg := range pages {
		for _, b := range pg.Blocks {
			if b.SVG != nil {
				if _, ok := index[b.SVG]; !ok {
					index[b.SVG] = len(docs) + 1
					docs = append(docs, b.SVG)
				}
			}
		}
	}
	return docs, index
}

// xobjectRefs builds " /XObject << /Im1 8 0 R ... >>" or an empty string.
func xobjectRefs(imgs []*images.Raster, svgs []*svg.Document, firstID int, forms []opacityFormSpec, formBase, page int) string {
	formCount := 0
	for _, f := range forms {
		if f.page == page {
			formCount++
		}
	}
	if len(imgs) == 0 && len(svgs) == 0 && formCount == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(" /XObject <<")
	for i := range imgs {
		fmt.Fprintf(&sb, " /Im%d %d 0 R", i+1, firstID+i)
	}
	for i := range svgs {
		fmt.Fprintf(&sb, " /Sv%d %d 0 R", i+1, firstID+len(imgs)+i)
	}
	for _, f := range forms {
		if f.page == page {
			fmt.Fprintf(&sb, " /Fm%d %d 0 R", f.id, formBase+f.id-1)
		}
	}
	sb.WriteString(" >>")
	return sb.String()
}

func svgForm(doc *svg.Document, alphas []float64, gs map[float64]int, gsBase, fontBase int) string {
	var content strings.Builder
	view := doc.ViewBoxTransform(doc.Width, doc.Height)
	fmt.Fprintf(&content, "q %g 0 0 %g %g %g cm\n", view.A, -view.D, view.E, doc.Height-view.F)
	for _, shape := range doc.Shapes {
		if shape.Text != "" {
			if shape.HasFill {
				if shape.Fill[3] < 1 {
					fmt.Fprintf(&content, "q /GS%d gs\n", gs[shape.Fill[3]])
				}
				fmt.Fprintf(&content, "%g %g %g rg\n", shape.Fill[0], shape.Fill[1], shape.Fill[2])
			}
			fmt.Fprintf(&content, "BT /F1 %g Tf 1 0 0 -1 %g %g Tm %s Tj ET\n", shape.TextSize, shape.TextX, shape.TextY, encodePDFString(shape.Text))
			if shape.HasFill && shape.Fill[3] < 1 {
				content.WriteString("Q\n")
			}
			continue
		}
		ops := svg.TransformOps(shape.Ops, svg.IdentityTransform())
		paint := func(color [4]float64, fill, evenOdd bool) {
			isolated := color[3] < 1
			if isolated {
				fmt.Fprintf(&content, "q /GS%d gs\n", gs[color[3]])
			}
			if fill {
				fmt.Fprintf(&content, "%g %g %g rg\n", color[0], color[1], color[2])
			} else {
				fmt.Fprintf(&content, "%g %g %g RG %g w\n", color[0], color[1], color[2], shape.Width)
			}
			emitSVGPath(&content, ops)
			if fill {
				if evenOdd {
					content.WriteString("f*\n")
				} else {
					content.WriteString("f\n")
				}
			} else {
				content.WriteString("S\n")
			}
			if isolated {
				content.WriteString("Q\n")
			}
		}
		if shape.HasFill {
			paint(shape.Fill, true, shape.EvenOdd)
		}
		if shape.HasStroke {
			paint(shape.Stroke, false, false)
		}
	}
	content.WriteString("Q")
	raw := content.String()
	return fmt.Sprintf("<< /Type /XObject /Subtype /Form /FormType 1 /BBox [0 0 %g %g] /Resources << /Font <<%s >>%s >> /Length %d >>\nstream\n%s\nendstream", doc.Width, doc.Height, base14Refs(fontBase), extRefs(alphas, gsBase), len(raw), raw)
}

func emitSVGPath(sb *strings.Builder, ops []svg.Op) {
	var x, y float64
	for _, op := range ops {
		p := op.Pts
		switch op.Kind {
		case svg.OpMove:
			fmt.Fprintf(sb, "%g %g m\n", p[0], p[1])
			x, y = p[0], p[1]
		case svg.OpLine:
			fmt.Fprintf(sb, "%g %g l\n", p[0], p[1])
			x, y = p[0], p[1]
		case svg.OpCurve:
			fmt.Fprintf(sb, "%g %g %g %g %g %g c\n", p[0], p[1], p[2], p[3], p[4], p[5])
			x, y = p[4], p[5]
		case svg.OpQuad:
			cx := x + 2.0/3*(p[0]-x)
			cy := y + 2.0/3*(p[1]-y)
			c2x := p[2] + 2.0/3*(p[0]-p[2])
			c2y := p[3] + 2.0/3*(p[1]-p[3])
			fmt.Fprintf(sb, "%g %g %g %g %g %g c\n", cx, cy, c2x, c2y, p[2], p[3])
			x, y = p[2], p[3]
		case svg.OpClose:
			sb.WriteString("h\n")
		}
	}
}

// imageStream embeds JPEG data with a known color space as-is (DCT);
// other images are converted to RGB and Flate-compressed.
func imageStream(r *images.Raster) string {
	if r.Format == "jpeg" && r.Space != "" {
		return fmt.Sprintf(
			"<< /Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /%s /BitsPerComponent 8 /Filter /DCTDecode /Length %d >>\nstream\n%s\nendstream",
			r.Width, r.Height, r.Space, len(r.Raw), r.Raw)
	}
	comp := flateBytes(r.Pixels)
	if r.Format == "png" && len(r.FlateData) > 0 && len(r.FlateData) < len(comp) {
		return fmt.Sprintf(
			"<< /Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /FlateDecode /DecodeParms << /Predictor 15 /Colors 3 /BitsPerComponent 8 /Columns %d >> /Length %d >>\nstream\n%s\nendstream",
			r.Width, r.Height, r.Width, len(r.FlateData), r.FlateData)
	}
	return fmt.Sprintf(
		"<< /Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceRGB /BitsPerComponent 8 /Filter /FlateDecode /Length %d >>\nstream\n%s\nendstream",
		r.Width, r.Height, len(comp), comp)
}
