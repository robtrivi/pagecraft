package render

import (
	"fmt"
	"strings"

	"github.com/robtrivi/pagecraft-go/boxes"
	"github.com/robtrivi/pagecraft-go/css"
	"github.com/robtrivi/pagecraft-go/fonts"
	"github.com/robtrivi/pagecraft-go/layout"
)

func buildPDFFromPages(pages []layout.Page, opts layout.Options, meta boxes.Metadata, footer []css.ContentChunk) []byte {
	imgs, imgIndex := collectImages(pages)
	svgs, svgIndex := collectSVGs(pages)
	annots, annotPages := collectLinks(pages, opts.PageHeight)
	faces, fontIndex := collectFonts(pages)
	fontSubsets := make(map[*fonts.Face]fonts.PDFSubset, len(faces))
	for _, face := range faces {
		subset, err := face.PDFSubsetShaped(fontTexts(pages, face))
		if err == nil {
			fontSubsets[face] = subset
		}
	}
	footers := expandFooters(footer, len(pages))
	alphas, gsIndex := collectAlphas(pages, opts)
	forms, formIndex := collectOpacityForms(pages)
	fontID := 3 + 2*len(pages)
	base14 := []string{"Helvetica", "Helvetica-Bold", "Helvetica-Oblique", "Helvetica-BoldOblique"}
	imgBase := fontID + len(base14)
	svgBase := imgBase + len(imgs)
	annotBase := svgBase + len(svgs)
	fontObjBase := annotBase + len(annots)
	infoStr, hasInfo := infoDict(meta)
	infoID := fontObjBase + 5*len(faces)
	gsBase := infoID
	if hasInfo {
		gsBase++
	}
	formBase := gsBase + len(alphas)
	bodies := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", kidsRefs(len(pages)), len(pages)),
	}
	for i, pg := range pages {
		pageOpts := optionsForPageGeometry(opts, pg.Geometry)
		pageObj := 3 + 2*i
		rotation := ""
		if pageOpts.PageRotation != 0 {
			rotation = fmt.Sprintf(" /Rotate %d", pageOpts.PageRotation)
		}
		pageBoxes := fmt.Sprintf("/MediaBox [0 0 %g %g]", pageOpts.PageWidth, pageOpts.PageHeight)
		if pageOpts.PageBleed > 0 || pageOpts.PageMarks != 0 {
			offset := pageOpts.PageBleed
			if pageOpts.PageMarks != 0 {
				offset += 18
			}
			pageBoxes = fmt.Sprintf("/MediaBox [%g %g %g %g] /CropBox [%g %g %g %g] /BleedBox [%g %g %g %g] /TrimBox [0 0 %g %g]",
				-offset, -offset, pageOpts.PageWidth+offset, pageOpts.PageHeight+offset,
				-offset, -offset, pageOpts.PageWidth+offset, pageOpts.PageHeight+offset,
				-pageOpts.PageBleed, -pageOpts.PageBleed, pageOpts.PageWidth+pageOpts.PageBleed, pageOpts.PageHeight+pageOpts.PageBleed,
				pageOpts.PageWidth, pageOpts.PageHeight)
		}
		bodies = append(bodies, fmt.Sprintf(
			"<< /Type /Page /Parent 2 0 R %s%s /Resources << /Font <<%s%s >>%s%s >>%s /Contents %d 0 R >>",
			pageBoxes, rotation, base14Refs(fontID), fontRefs(faces, fontObjBase),
			xobjectRefs(imgs, svgs, imgBase, forms, formBase, i), extRefs(alphas, gsBase),
			annotsRefs(annotPages[i], annotBase), pageObj+1))
		bodies = append(bodies, contentStream(pg, pageOpts, imgIndex, svgIndex, fontIndex, fontSubsets, footers[i], gsIndex, formIndex[i][0]))
	}
	bodies = append(bodies, base14Bodies(base14)...)
	for _, r := range imgs {
		bodies = append(bodies, imageStream(r))
	}
	for _, doc := range svgs {
		bodies = append(bodies, svgForm(doc, alphas, gsIndex, gsBase, fontID))
	}
	for _, a := range annots {
		bodies = append(bodies, fmt.Sprintf(
			"<< /Type /Annot /Subtype /Link /Rect [%g %g %g %g] /Border [0 0 0] /A << /S /URI /URI %s >> >>",
			a.rect[0], a.rect[1], a.rect[2], a.rect[3], encodePDFString(a.uri)))
	}
	for fi, face := range faces {
		subset := fontSubsets[face]
		base := fontObjBase + 5*fi
		bodies = append(bodies, fontType0(face, fi, base+1, base+4))
		bodies = append(bodies, fontCID(face, subset, base+2, fi))
		bodies = append(bodies, fontDescriptor(face, fi, base+3))
		bodies = append(bodies, fontFile(subset))
		bodies = append(bodies, fontToUnicode(subset))
	}
	infoRef := ""
	if hasInfo {
		bodies = append(bodies, infoStr)
		infoRef = fmt.Sprintf(" /Info %d 0 R", len(bodies))
	}
	for _, a := range alphas {
		bodies = append(bodies, fmt.Sprintf("<< /Type /ExtGState /CA %g /ca %g >>", a, a))
	}
	for _, form := range forms {
		resources := fmt.Sprintf("<< /Font <<%s%s >>%s%s >>", base14Refs(fontID), fontRefs(faces, fontObjBase),
			xobjectRefs(imgs, svgs, imgBase, forms, formBase, form.page), extRefs(alphas, gsBase))
		blocks := pages[form.page].Blocks[form.start : form.end+1]
		fragmentNodes := fragmentNodesByPaintIndex(pages[form.page].FragmentTree)
		pageOpts := optionsForPageGeometry(opts, pages[form.page].Geometry)
		bodies = append(bodies, opacityForm(blocks, form, formIndex[form.page], fragmentNodes, pageOpts, imgIndex, svgIndex, fontIndex, fontSubsets, gsIndex, resources))
	}

	var sb strings.Builder
	sb.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(bodies))
	for i, body := range bodies {
		offsets[i] = len(sb.String())
		fmt.Fprintf(&sb, "%d 0 obj\n%s\nendobj\n", i+1, body)
	}
	xrefPos := len(sb.String())
	fmt.Fprintf(&sb, "xref\n0 %d\n", len(bodies)+1)
	sb.WriteString("0000000000 65535 f \n")
	for _, off := range offsets {
		fmt.Fprintf(&sb, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&sb, "trailer\n<< /Size %d /Root 1 0 R%s >>\nstartxref\n%d\n%%%%EOF",
		len(bodies)+1, infoRef, xrefPos)
	return []byte(sb.String())
}

func optionsForPageGeometry(opts layout.Options, geometry layout.PageGeometry) layout.Options {
	if geometry.Width == 0 || geometry.Height == 0 {
		return opts
	}
	opts.PageWidth, opts.PageHeight = geometry.Width, geometry.Height
	opts.PageMargins = geometry.Margins
	if geometry.Margins.HasBottom {
		opts.Margin = geometry.Margins.Bottom
	}
	opts.PageBackground, opts.HasPageBackground = geometry.Background, geometry.HasBackground
	opts.PageRotation, opts.PageBleed, opts.PageMarks = geometry.Rotation, geometry.Bleed, geometry.Marks
	return opts
}

// infoDict builds the Info dictionary (Title/Author/Subject); ok is false when empty.
func infoDict(meta boxes.Metadata) (string, bool) {
	var sb strings.Builder
	sb.WriteString("<<")
	if meta.Title != "" {
		fmt.Fprintf(&sb, " /Title %s", encodePDFString(meta.Title))
	}
	if meta.Author != "" {
		fmt.Fprintf(&sb, " /Author %s", encodePDFString(meta.Author))
	}
	if meta.Description != "" {
		fmt.Fprintf(&sb, " /Subject %s", encodePDFString(meta.Description))
	}
	sb.WriteString(" >>")
	if sb.Len() <= len("<< >>") {
		return "", false
	}
	return sb.String(), true
}

func kidsRefs(n int) string {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&sb, "%d 0 R ", 3+2*i)
	}
	return strings.TrimSpace(sb.String())
}

type opacityFormRef struct {
	id, end int
	opacity float64
}
