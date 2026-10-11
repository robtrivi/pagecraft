package render

import (
	"fmt"
	"sort"
	"strings"

	"github.com/robtrivi/pagecraft/fonts"
	"github.com/robtrivi/pagecraft/layout"
)

func base14Refs(firstID int) string {
	var sb strings.Builder
	for i := range 4 {
		fmt.Fprintf(&sb, " /F%d %d 0 R", i+1, firstID+i)
	}
	return sb.String()
}

// base14Bodies emits the four Type1 dictionaries using WinAnsi.
func base14Bodies(names []string) []string {
	bodies := make([]string, 0, len(names))
	for _, n := range names {
		bodies = append(bodies, fmt.Sprintf(
			"<< /Type /Font /Subtype /Type1 /BaseFont /%s /Encoding /WinAnsiEncoding >>", n))
	}
	return bodies
}

// collectFonts collects fonts in first-use order and assigns font numbers (F5+; F1-F4 are base-14).
func collectFonts(pages []layout.Page) ([]*fonts.Face, map[*fonts.Face]int) {
	var faces []*fonts.Face
	index := map[*fonts.Face]int{}
	add := func(face *fonts.Face) {
		if face == nil {
			return
		}
		if _, ok := index[face]; ok {
			return
		}
		index[face] = len(faces) + 5
		faces = append(faces, face)
	}
	for _, pg := range pages {
		for _, b := range pg.Blocks {
			for _, line := range b.Runs {
				for _, run := range line {
					add(run.Font)
				}
			}
		}
	}
	return faces, index
}

// fontRefs builds " /F5 20 0 R ..." for each custom Type0 font.
func fontRefs(faces []*fonts.Face, base int) string {
	var sb strings.Builder
	for i := range faces {
		fmt.Fprintf(&sb, " /F%d %d 0 R", i+5, base+5*i)
	}
	return sb.String()
}

// fontTexts collects all text using that font for subsetting.
func fontTexts(pages []layout.Page, face *fonts.Face) []fonts.ShapedText {
	var out []fonts.ShapedText
	for _, pg := range pages {
		for _, b := range pg.Blocks {
			for _, ln := range b.Runs {
				for _, r := range ln {
					if r.Font == face {
						out = append(out, r.Shaped)
					}
				}
			}
		}
	}
	return out
}

// subsetTag generates a subset tag from the index using six uppercase letters.
func subsetTag(i int) string {
	b := make([]byte, 6)
	for k := 5; k >= 0; k-- {
		b[k] = 'A' + byte(i%26)
		i /= 26
	}
	return string(b)
}

// psName sanitizes a PostScript name to alphanumeric characters.
func psName(face *fonts.Face) string {
	var sb strings.Builder
	for _, r := range face.Name() {
		if 'A' <= r && r <= 'Z' || 'a' <= r && r <= 'z' || '0' <= r && r <= '9' {
			sb.WriteRune(r)
		}
	}
	if sb.Len() == 0 {
		return "Custom"
	}
	return sb.String()
}

func baseName(face *fonts.Face, i int) string {
	return subsetTag(i) + "+" + psName(face)
}

func fontType0(face *fonts.Face, i, cid, tu int) string {
	return fmt.Sprintf(
		"<< /Type /Font /Subtype /Type0 /BaseFont /%s /Encoding /Identity-H /DescendantFonts [%d 0 R] /ToUnicode %d 0 R >>",
		baseName(face, i), cid, tu)
}

func fontCID(face *fonts.Face, subset fonts.PDFSubset, desc, i int) string {
	gids := make([]uint16, 0, len(subset.Widths))
	for gid := range subset.Widths {
		gids = append(gids, gid)
	}
	sort.Slice(gids, func(i, j int) bool { return gids[i] < gids[j] })
	return fmt.Sprintf(
		"<< /Type /Font /Subtype /CIDFontType2 /BaseFont /%s /CIDSystemInfo << /Registry (Adobe) /Ordering (Identity) /Supplement 0 >> /FontDescriptor %d 0 R /DW 1000 /W [%s] >>",
		baseName(face, i), desc, widthsArray(gids, subset.Widths))
}

// widthsArray groups consecutive GIDs: "3 [500 600] 10 [400]".
func widthsArray(gids []uint16, widths map[uint16]int) string {
	var parts []string
	for i := 0; i < len(gids); {
		j := i
		for j+1 < len(gids) && gids[j+1] == gids[j]+1 {
			j++
		}
		var ws []string
		for k := i; k <= j; k++ {
			ws = append(ws, fmt.Sprintf("%d", widths[gids[k]]))
		}
		parts = append(parts, fmt.Sprintf("%d [%s]", gids[i], strings.Join(ws, " ")))
		i = j + 1
	}
	return strings.Join(parts, " ")
}

func fontDescriptor(face *fonts.Face, i, file int) string {
	upm, asc, desc, cap, italic := face.Head()
	scale := func(v int) int { return v * 1000 / upm }
	return fmt.Sprintf(
		"<< /Type /FontDescriptor /FontName /%s /Flags 32 /ItalicAngle %g /Ascent %d /Descent %d /CapHeight %d /StemV 80 /FontFile2 %d 0 R >>",
		baseName(face, i), italic, scale(asc), scale(desc), scale(cap), file)
}

func fontFile(subset fonts.PDFSubset) string {
	comp := flateBytes(subset.Data)
	return fmt.Sprintf(
		"<< /Length1 %d /Length %d /Filter /FlateDecode >>\nstream\n%s\nendstream",
		len(subset.Data), len(comp), comp)
}

func fontToUnicode(subset fonts.PDFSubset) string {
	cmap := subset.ToUnicodeCMap()
	return fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(cmap), cmap)
}

// collectAlphas collects alpha values below 1 in order and assigns graphics-state indices (1-based).
