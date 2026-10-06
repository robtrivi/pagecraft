// Package fonts loads TrueType fonts and provides metrics, shaping, and PDF subsetting.
package fonts

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"sync"
	"unicode"
	"unicode/utf16"

	"github.com/go-text/typesetting/di"
	goFont "github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/shaping"
	"github.com/gpdf-dev/gpdf/pdf/font"
	"golang.org/x/image/math/fixed"
)

// Face is a loaded font that can be measured and embedded.
type Face struct {
	ttf       *font.TrueTypeFont
	shapeFace *goFont.Face
	shaper    shaping.HarfbuzzShaper
	shapeMu   sync.Mutex
	name      string
	data      []byte
}

// Variants stores the available CSS faces for a font family.
type Variants struct {
	Regular, Bold, Italic, BoldItalic *Face
}

// Load parses a TTF or OTF font.
func Load(data []byte) (*Face, error) {
	t, err := font.ParseTrueType(data)
	if err != nil {
		return nil, err
	}
	sf, err := goFont.ParseTTF(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return &Face{ttf: t, shapeFace: sf, name: t.Name(), data: append([]byte(nil), data...)}, nil
}

// Name returns the font's internal name.
func (f *Face) Name() string { return f.name }

// Measure uses the font'scurrent glyph advances.
func (f *Face) Measure(s string, size float64) float64 {
	return font.MeasureString(f.ttf, s, size)
}

// Glyph is a glyph selected by the shaper, with metrics in points.
type Glyph struct {
	ID               uint16
	Cluster          int
	Text             string
	Advance          float64
	XOffset, YOffset float64
}

// ShapedText stores logical text and the glyphs and advances shared per line
// breaking and the PDF writer.
type ShapedText struct {
	Text        string
	Glyphs      []Glyph
	Width       float64
	RightToLeft bool
}

// SplitAtRuneOffsets splits shaped text at rune boundaries without reshaping.
// It returns false if the split would divide a glyph cluster.
func (s ShapedText) SplitAtRuneOffsets(offsets []int) ([]ShapedText, bool) {
	runes := []rune(s.Text)
	if len(offsets) == 0 {
		return []ShapedText{s}, true
	}
	clusterStarts := make(map[int]struct{}, len(s.Glyphs)+1)
	clusterStarts[len(runes)] = struct{}{}
	for _, glyph := range s.Glyphs {
		clusterStarts[glyph.Cluster] = struct{}{}
	}
	boundaries := append([]int{0}, offsets...)
	boundaries = append(boundaries, len(runes))
	previous := -1
	for _, offset := range offsets {
		if offset <= previous || offset <= 0 || offset >= len(runes) {
			return nil, false
		}
		if _, ok := clusterStarts[offset]; !ok {
			return nil, false
		}
		previous = offset
	}
	parts := make([]ShapedText, 0, len(boundaries)-1)
	for index := 0; index+1 < len(boundaries); index++ {
		start, end := boundaries[index], boundaries[index+1]
		part := ShapedText{Text: string(runes[start:end]), RightToLeft: s.RightToLeft}
		for _, glyph := range s.Glyphs {
			if glyph.Cluster < start || glyph.Cluster >= end {
				continue
			}
			glyph.Cluster -= start
			part.Glyphs = append(part.Glyphs, glyph)
			part.Width += glyph.Advance
		}
		parts = append(parts, part)
	}
	return parts, true
}

// PDFSubset contains one dense TrueType subset and the mappings needed to
// keep PDF CIDs, widths and Unicode extraction aligned with its new GIDs.
type PDFSubset struct {
	Data    []byte
	GIDMap  map[uint16]uint16
	Widths  map[uint16]int
	Unicode map[uint16]string
}

// ToUnicodeCMap serializes source text by the dense subset CID/GID.
func (s PDFSubset) ToUnicodeCMap() []byte {
	if len(s.Unicode) == 0 {
		return nil
	}
	gids := make([]int, 0, len(s.Unicode))
	for gid := range s.Unicode {
		gids = append(gids, int(gid))
	}
	sort.Ints(gids)
	var b strings.Builder
	b.WriteString("/CIDInit /ProcSet findresource begin\n12 dict begin\nbegincmap\n")
	b.WriteString("/CIDSystemInfo << /Registry (Adobe) /Ordering (UCS) /Supplement 0 >> def\n")
	b.WriteString("/CMapName /Adobe-Identity-UCS def\n/CMapType 2 def\n1 begincodespacerange\n<0000> <FFFF>\nendcodespacerange\n")
	for start := 0; start < len(gids); start += 100 {
		end := min(start+100, len(gids))
		fmt.Fprintf(&b, "%d beginbfchar\n", end-start)
		for _, raw := range gids[start:end] {
			var encoded strings.Builder
			for _, unit := range utf16.Encode([]rune(s.Unicode[uint16(raw)])) {
				fmt.Fprintf(&encoded, "%04X", unit)
			}
			fmt.Fprintf(&b, "<%04X> <%s>\n", raw, encoded.String())
		}
		b.WriteString("endbfchar\n")
	}
	b.WriteString("endcmap\nCMapName currentdict /CMap defineresource pop\nend\nend")
	return []byte(b.String())
}

// Shape applies OpenType tables to a run at a high reference scale, then
// converts the results to CSS size while preserving fractional metrics.
func (f *Face) Shape(s string, size float64) ShapedText {
	result := ShapedText{Text: s}
	if s == "" || size <= 0 {
		return result
	}
	const referenceSize = 10000.0
	sizeScale := size / referenceSize
	f.shapeMu.Lock()
	defer f.shapeMu.Unlock()
	runes := []rune(s)
	scriptRune := firstShapingRune(runes)
	direction := di.DirectionLTR
	if isRightToLeftRune(scriptRune) {
		direction = di.DirectionRTL
		result.RightToLeft = true
	}
	input := shaping.Input{
		Text: runes, RunEnd: len(runes), Direction: direction, Script: language.LookupScript(scriptRune),
		Face: f.shapeFace, Size: fixed.Int26_6(referenceSize * 64),
		FontFeatures: []shaping.FontFeature{
			{Tag: opentype.MustNewTag("liga"), Value: 1},
			{Tag: opentype.MustNewTag("clig"), Value: 1},
		},
	}
	out := f.shaper.Shape(input)
	result.Glyphs = make([]Glyph, len(out.Glyphs))
	clusters := make([]int, 0, len(out.Glyphs))
	seen := make(map[int]bool, len(out.Glyphs))
	for _, glyph := range out.Glyphs {
		if !seen[glyph.ClusterIndex] {
			seen[glyph.ClusterIndex] = true
			clusters = append(clusters, glyph.ClusterIndex)
		}
	}
	sort.Ints(clusters)
	clusterText := make(map[int]string, len(clusters))
	for i, start := range clusters {
		end := len(runes)
		if i+1 < len(clusters) {
			end = clusters[i+1]
		}
		if start >= 0 && start < end && end <= len(runes) {
			clusterText[start] = string(runes[start:end])
		}
	}
	assigned := make(map[int]bool, len(clusters))
	for i, glyph := range out.Glyphs {
		result.Glyphs[i] = Glyph{
			ID: uint16(glyph.GlyphID), Cluster: glyph.ClusterIndex,
			Text:    clusterText[glyph.ClusterIndex],
			Advance: float64(glyph.Advance) / 64 * sizeScale,
			XOffset: float64(glyph.XOffset) / 64 * sizeScale,
			YOffset: float64(glyph.YOffset) / 64 * sizeScale,
		}
		if assigned[glyph.ClusterIndex] {
			result.Glyphs[i].Text = ""
		}
		assigned[glyph.ClusterIndex] = true
		result.Width += result.Glyphs[i].Advance
	}
	return result
}

func firstShapingRune(runes []rune) rune {
	for _, r := range runes {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return r
		}
	}
	return runes[0]
}

func isRightToLeftRune(r rune) bool {
	return (r >= 0x0590 && r <= 0x08FF) ||
		(r >= 0xFB1D && r <= 0xFDFF) ||
		(r >= 0xFE70 && r <= 0xFEFF) ||
		(r >= 0x10800 && r <= 0x10FFF) ||
		(r >= 0x1E800 && r <= 0x1EEFF)
}

// NormalLineHeight returns the font's normal line height at the given size.
func (f *Face) NormalLineHeight(size float64) float64 {
	m := f.ttf.Metrics()
	if m.UnitsPerEm <= 0 {
		return 0
	}
	height := m.Ascender - m.Descender + m.LineGap
	if height <= 0 {
		return 0
	}
	return float64(height) * size / float64(m.UnitsPerEm)
}

// BaselineOffset positions the baseline in the line box, centering extra
// leading around the font's typographic metrics.
func (f *Face) BaselineOffset(size, lineHeight float64) float64 {
	m := f.ttf.Metrics()
	if m.UnitsPerEm <= 0 {
		return size
	}
	normal := f.NormalLineHeight(size)
	if normal <= 0 {
		return float64(m.Ascender) * size / float64(m.UnitsPerEm)
	}
	ascender := float64(m.Ascender) * size / float64(m.UnitsPerEm)
	return ascender + (lineHeight-normal)/2
}

// GlyphID returns the glyph ID for a rune, or 0 if the font does not contain it.
func (f *Face) GlyphID(r rune) uint16 { return f.ttf.GlyphID(r) }

// Encode returns two-byte big-endian glyph codes for the PDF Tj operator.
func (f *Face) Encode(s string) []byte { return f.ttf.Encode(s) }

// Head returns units per em and the font's vertical metrics.
func (f *Face) Head() (upm, asc, desc, cap int, italic float64) {
	m := f.ttf.Metrics()
	return m.UnitsPerEm, m.Ascender, m.Descender, m.CapHeight, m.ItalicAngle
}

// GlyphMap returns sorted glyph IDs, widths in thousandths, and the rune-to-GID
// mapping used by ToUnicode. Missing glyphs (GID 0) are omitted.
func (f *Face) GlyphMap(texts []string) (gids []uint16, widths map[uint16]int, uni map[rune]uint16) {
	upm, _, _, _, _ := f.Head()
	seen := map[uint16]bool{}
	widths = map[uint16]int{}
	uni = map[rune]uint16{}
	for _, t := range texts {
		for _, r := range t {
			gid := f.ttf.GlyphID(r)
			if gid == 0 {
				continue
			}
			uni[r] = gid
			if seen[gid] {
				continue
			}
			seen[gid] = true
			gids = append(gids, gid)
			if w, ok := f.ttf.GlyphWidth(r); ok {
				widths[gid] = w * 1000 / upm
			}
		}
	}
	sort.Slice(gids, func(i, j int) bool { return gids[i] < gids[j] })
	return gids, widths, uni
}

// ToUnicode generates a rune-to-GID CMap for text extraction.
func (f *Face) ToUnicode(texts []string) []byte {
	_, _, uni := f.GlyphMap(texts)
	return font.GenerateToUnicodeCMap(uni)
}

// GlyphMapShaped returns the glyph IDs, widths, and Unicode text needed to
// create a CID font and ToUnicode map for shaped text.
func (f *Face) GlyphMapShaped(texts []ShapedText) (gids []uint16, widths map[uint16]int, uni map[uint16]string) {
	seen := map[uint16]bool{0: true}
	widths = make(map[uint16]int)
	uni = make(map[uint16]string)
	upm, _, _, _, _ := f.Head()
	for _, shaped := range texts {
		for _, glyph := range shaped.Glyphs {
			if glyph.ID == 0 {
				continue
			}
			if glyph.Text != "" {
				if previous, exists := uni[glyph.ID]; !exists || previous == glyph.Text {
					uni[glyph.ID] = glyph.Text
				}
			}
			if seen[glyph.ID] {
				continue
			}
			seen[glyph.ID] = true
			gids = append(gids, glyph.ID)
			advance := f.shapeFace.HorizontalAdvance(goFont.GID(glyph.ID))
			if upm > 0 {
				widths[glyph.ID] = int(float64(advance)*1000/float64(upm) + 0.5)
			}
		}
	}
	sort.Slice(gids, func(i, j int) bool { return gids[i] < gids[j] })
	return gids, widths, uni
}

// ToUnicodeShaped maps each CID/GID to the Unicode text of its cluster.
// Remaining glyphs in the cluster use an empty string to avoid duplicates.
func (f *Face) ToUnicodeShaped(texts []ShapedText) []byte {
	_, _, mapping := f.GlyphMapShaped(texts)
	if len(mapping) == 0 {
		return nil
	}
	gids := make([]int, 0, len(mapping))
	for gid := range mapping {
		gids = append(gids, int(gid))
	}
	sort.Ints(gids)
	var b strings.Builder
	b.WriteString("/CIDInit /ProcSet findresource begin\n12 dict begin\nbegincmap\n")
	b.WriteString("/CIDSystemInfo << /Registry (Adobe) /Ordering (UCS) /Supplement 0 >> def\n")
	b.WriteString("/CMapName /Adobe-Identity-UCS def\n/CMapType 2 def\n1 begincodespacerange\n<0000> <FFFF>\nendcodespacerange\n")
	for start := 0; start < len(gids); start += 100 {
		end := start + 100
		if end > len(gids) {
			end = len(gids)
		}
		fmt.Fprintf(&b, "%d beginbfchar\n", end-start)
		for _, raw := range gids[start:end] {
			gid := uint16(raw)
			var encoded strings.Builder
			for _, unit := range utf16.Encode([]rune(mapping[gid])) {
				fmt.Fprintf(&encoded, "%04X", unit)
			}
			fmt.Fprintf(&b, "<%04X> <%s>\n", gid, encoded.String())
		}
		b.WriteString("endbfchar\n")
	}
	b.WriteString("endcmap\nCMapName currentdict /CMap defineresource pop\nend\nend")
	return []byte(b.String())
}

// Subset creates a font subset for the given text and returns its bytes.
func (f *Face) Subset(texts ...string) ([]byte, error) {
	shaped := make([]ShapedText, 0, len(texts))
	for _, text := range texts {
		shaped = append(shaped, f.Shape(text, 1000))
	}
	return f.SubsetShaped(shaped)
}

// SubsetShaped preserves the GIDs selected by OpenType substitutions.
func (f *Face) SubsetShaped(texts []ShapedText) ([]byte, error) {
	gids, _, _ := f.GlyphMapShaped(texts)
	gids = append(gids, 0)
	subset, err := font.SubsetTrueType(f.data, gids)
	if err != nil {
		return nil, err
	}
	return compactGlyphTables(subset)
}

// PDFSubsetShaped creates an embedded font with densely renumbered GIDs.
// The writer must use GIDMap, Widths, and Unicode to keep content codes,
// CID widths, and the ToUnicode map aligned.
func (f *Face) PDFSubsetShaped(texts []ShapedText) (PDFSubset, error) {
	gids, widths, unicodeMap := f.GlyphMapShaped(texts)
	gids = append(gids, 0)
	subset, err := font.SubsetTrueType(f.data, gids)
	if err != nil {
		return PDFSubset{}, err
	}
	subset, err = compactGlyphTables(subset)
	if err != nil {
		return PDFSubset{}, err
	}
	codepoints := make(map[rune]uint16)
	for _, shaped := range texts {
		for _, r := range shaped.Text {
			if gid := f.GlyphID(r); gid != 0 {
				codepoints[r] = gid
			}
		}
	}
	data, gidMap, err := denseGlyphIDs(subset, gids, codepoints)
	if err != nil {
		if _, supported := err.(unsupportedDenseSubsetError); !supported {
			return PDFSubset{}, err
		}
		gidMap = make(map[uint16]uint16, len(gids))
		for _, gid := range gids {
			gidMap[gid] = gid
		}
		data = subset
	}
	newWidths := make(map[uint16]int, len(widths))
	newUnicode := make(map[uint16]string, len(unicodeMap))
	for oldGID, width := range widths {
		if newGID, ok := gidMap[oldGID]; ok {
			newWidths[newGID] = width
		}
	}
	for oldGID, value := range unicodeMap {
		if newGID, ok := gidMap[oldGID]; ok {
			newUnicode[newGID] = value
		}
	}
	return PDFSubset{Data: data, GIDMap: gidMap, Widths: newWidths, Unicode: newUnicode}, nil
}
