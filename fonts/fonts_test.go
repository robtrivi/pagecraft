package fonts

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// Custom fonts step 1: load a real TTF, measure with its advances, and subset it.
// Uses a system TTF (skips when unavailable; font binaries are not vendored).
func systemTTF(t *testing.T) string {
	t.Helper()
	cands, _ := filepath.Glob("/usr/share/fonts/*/*.ttf")
	for _, c := range cands {
		if st, err := os.Stat(c); err == nil && st.Size() > 50000 {
			return c
		}
	}
	t.Skip("no system TTF available")
	return ""
}

func TestFonts_Load_MeasuresWithOwnAdvances(t *testing.T) {
	data, err := os.ReadFile(systemTTF(t))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	f, err := Load(data)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if f.Name() == "" {
		t.Error("missing source name")
	}
	w := f.Measure("Hello", 12)
	if w <= 0 || w > 451 {
		t.Errorf("unexpected measurement: %v", w)
	}
}

func TestFonts_Shape_KerningChangesAdvance(t *testing.T) {
	path := "/usr/share/fonts/liberation/LiberationSans-Regular.ttf"
	if _, err := os.Stat(path); err != nil {
		path = systemTTF(t)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	f, err := Load(data)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	got := f.Shape("AV", 12)
	if got.Text != "AV" || len(got.Glyphs) != 2 {
		t.Fatalf("shape lost the text/GIDs: %+v", got)
	}
	if got.Width >= f.Measure("AV", 12) {
		t.Errorf("shaping did not apply typographic adjustment: shaped=%v unshaped=%v", got.Width, f.Measure("AV", 12))
	}
	for i, r := range "AV" {
		if got.Glyphs[i].ID != f.GlyphID(r) {
			t.Errorf("GID[%d]=%d, cmap maps to %d", i, got.Glyphs[i].ID, f.GlyphID(r))
		}
	}
}

func TestFonts_Shape_LigatureMapsBackToSourceText(t *testing.T) {
	path := "/usr/share/fonts/TTF/DejaVuSans.ttf"
	if _, err := os.Stat(path); err != nil {
		t.Skip("DejaVu Sans is not installed")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	f, err := Load(data)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	got := f.Shape("office", 12)
	if len(got.Glyphs) != 4 {
		t.Fatalf("office must use the ffi glyph: got %d GIDs (%+v)", len(got.Glyphs), got.Glyphs)
	}
	var source string
	for _, glyph := range got.Glyphs {
		source += glyph.Text
	}
	if source != "office" {
		t.Errorf("source text by clusters=%q, want %q", source, "office")
	}
}

func TestFonts_ShapedTextSplitPreservesSourceGlyphsAndAdvances(t *testing.T) {
	path := "/usr/share/fonts/liberation/LiberationSans-Regular.ttf"
	if _, err := os.Stat(path); err != nil {
		path = systemTTF(t)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	face, err := Load(data)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	whole := face.Shape("AV", 12)
	parts, ok := whole.SplitAtRuneOffsets([]int{1})
	if !ok || len(parts) != 2 {
		t.Fatalf("shaped text split=%+v, ok=%v; want 2 fragments", parts, ok)
	}
	if parts[0].Text != "A" || parts[1].Text != "V" {
		t.Fatalf("split text=%q/%q, want A/V", parts[0].Text, parts[1].Text)
	}
	if math.Abs(parts[0].Width+parts[1].Width-whole.Width) > 0.001 {
		t.Errorf("fragmented advances=%g, original shape=%g; splitting must preserve already-shaped glyphs", parts[0].Width+parts[1].Width, whole.Width)
	}
	if len(parts[0].Glyphs) != 1 || len(parts[1].Glyphs) != 1 || parts[0].Glyphs[0].ID != whole.Glyphs[0].ID || parts[1].Glyphs[0].ID != whole.Glyphs[1].ID {
		t.Errorf("source GIDs were not preserved: whole=%+v parts=%+v", whole.Glyphs, parts)
	}
}

func TestFonts_ShapedTextSplitRejectsLigatureClusterInterior(t *testing.T) {
	path := "/usr/share/fonts/TTF/DejaVuSans.ttf"
	if _, err := os.Stat(path); err != nil {
		t.Skip("DejaVu Sans is not installed")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	face, err := Load(data)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	whole := face.Shape("office", 12)
	if _, ok := whole.SplitAtRuneOffsets([]int{2}); ok {
		t.Fatal("allowed split inside the cluster ffi")
	}
}

func TestFonts_Shape_FractionalSizeScalesAdvancesLinearly(t *testing.T) {
	path := "/usr/share/fonts/liberation/LiberationSans-Regular.ttf"
	if _, err := os.Stat(path); err != nil {
		t.Skip("Liberation Sans is not installed")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	f, err := Load(data)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	whole, fractional := f.Shape("AVATAR", 10), f.Shape("AVATAR", 10.5)
	want := whole.Width * 1.05
	if math.Abs(fractional.Width-want) > 0.02 {
		t.Errorf("advance at 10.5pt=%v, want linear scaling to %v from 10pt", fractional.Width, want)
	}
}

func TestFonts_Subset_OnlyUsedGlyphs(t *testing.T) {
	data, err := os.ReadFile(systemTTF(t))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	f, err := Load(data)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	sub, err := f.Subset("Hello")
	if err != nil {
		t.Fatalf("subset: %v", err)
	}
	// The subset preserves structure (unused glyphs are zeroed); the embedded
	// Flate-compressed form should be smaller than the full font.
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(sub); err != nil {
		t.Fatalf("flate: %v", err)
	}
	w.Close()
	if buf.Len() == 0 || buf.Len() >= len(data) {
		t.Errorf("subset+flate: got %d bytes of %d", buf.Len(), len(data))
	}
	t.Logf("full=%d subset=%d flate=%d", len(data), len(sub), buf.Len())
}

func TestFonts_Subset_CompactsUnusedGlyphOutlines(t *testing.T) {
	path := "/usr/share/fonts/liberation/LiberationSans-Regular.ttf"
	if _, err := os.Stat(path); err != nil {
		path = systemTTF(t)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	f, err := Load(data)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	sub, err := f.Subset("coöperation AV office")
	if err != nil {
		t.Fatalf("subset: %v", err)
	}
	if len(sub) >= len(data)/2 {
		t.Errorf("the subset preserves too many bytes from unused sources: subset=%d original=%d", len(sub), len(data))
	}
	compacted, err := Load(sub)
	if err != nil {
		t.Fatalf("the subset font must remain valid: %v", err)
	}
	originalShape := f.Shape("coöperation AV office", 12)
	compactedShape := compacted.Shape("coöperation AV office", 12)
	if len(compactedShape.Glyphs) != len(originalShape.Glyphs) {
		t.Fatalf("the subset changed the shaped sequence: original=%d subset=%d", len(originalShape.Glyphs), len(compactedShape.Glyphs))
	}
	for i := range originalShape.Glyphs {
		if compactedShape.Glyphs[i].ID != originalShape.Glyphs[i].ID {
			t.Fatalf("GID remapped for glyph %d: original=%d subset=%d", i, originalShape.Glyphs[i].ID, compactedShape.Glyphs[i].ID)
		}
	}
}

func TestFonts_Subset_DropsShapingTablesFromEmbeddedFont(t *testing.T) {
	path := "/usr/share/fonts/liberation/LiberationSans-Regular.ttf"
	if _, err := os.Stat(path); err != nil {
		path = systemTTF(t)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	f, err := Load(data)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	sub, err := f.Subset("AV office")
	if err != nil {
		t.Fatalf("subset: %v", err)
	}
	tables, err := readSFNTTables(sub)
	if err != nil {
		t.Fatalf("read subset tables: %v", err)
	}
	for _, table := range tables {
		switch table.tag {
		case "GDEF", "GPOS", "GSUB", "kern":
			t.Errorf("the PDF subset must preserve shaping table %q already applied before embedding", table.tag)
		case "FFTM", "DSIG":
			t.Errorf("PDF subset contains unnecessary metadata %q", table.tag)
		case "post":
			if len(table.data) < 32 || binary.BigEndian.Uint32(table.data[:4]) != 0x00030000 {
				t.Errorf("post must use format 3.0 without GID names; got %x (%d bytes)", table.data[:min(4, len(table.data))], len(table.data))
			}
		}
	}
	if _, err := Load(sub); err != nil {
		t.Errorf("the PDF subset must remain a valid TTF: %v", err)
	}
}

func TestFonts_PDFSubsetDenseRemapsGlyphsAndUnicode(t *testing.T) {
	path := "/usr/share/fonts/liberation/LiberationSans-Regular.ttf"
	if _, err := os.Stat(path); err != nil {
		path = systemTTF(t)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	f, err := Load(data)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	shaped := f.Shape("Café AV", 12)
	subset, err := f.PDFSubsetShaped([]ShapedText{shaped})
	if err != nil {
		t.Fatalf("pdf subset: %v", err)
	}
	tables, err := readSFNTTables(subset.Data)
	if err != nil {
		t.Fatalf("read dense subset: %v", err)
	}
	var numGlyphs int
	for _, table := range tables {
		if table.tag == "maxp" {
			numGlyphs = int(binary.BigEndian.Uint16(table.data[4:6]))
		}
	}
	if numGlyphs != len(subset.GIDMap) {
		t.Fatalf("subset GID space is not dense: maxp=%d map=%d", numGlyphs, len(subset.GIDMap))
	}
	if len(subset.Data) >= len(data)/2 {
		t.Errorf("dense subset did not reduce the TTF program enough: subset=%d original=%d", len(subset.Data), len(data))
	}
	seen := make(map[uint16]bool, len(subset.GIDMap))
	for original, compact := range subset.GIDMap {
		if compact >= uint16(numGlyphs) || seen[compact] {
			t.Fatalf("GID mapping is not bijective/dense: %d→%d with %d glyphs", original, compact, numGlyphs)
		}
		seen[compact] = true
	}
	for compact := 0; compact < numGlyphs; compact++ {
		if !seen[uint16(compact)] {
			t.Fatalf("missing compact GID %d", compact)
		}
	}
	for _, glyph := range shaped.Glyphs {
		if _, ok := subset.GIDMap[glyph.ID]; !ok {
			t.Fatalf("missing mapping for shaped GID %d", glyph.ID)
		}
	}
	compacted, err := Load(subset.Data)
	if err != nil {
		t.Fatalf("subset TTF is invalid: %v", err)
	}
	for _, r := range "Café AV" {
		oldGID := f.GlyphID(r)
		newGID, included := subset.GIDMap[oldGID]
		if included && compacted.GlyphID(r) != newGID {
			t.Errorf("cmap(%q)=%d, want compact GID %d", r, compacted.GlyphID(r), newGID)
		}
	}
	for _, glyph := range shaped.Glyphs {
		compact := subset.GIDMap[glyph.ID]
		if _, ok := subset.Widths[compact]; !ok {
			t.Errorf("missing PDF width for compact GID %d", compact)
		}
		if glyph.Text != "" && subset.Unicode[compact] != glyph.Text {
			t.Errorf("ToUnicode GID %d=%q, want %q", compact, subset.Unicode[compact], glyph.Text)
		}
	}
	originalTables, err := readSFNTTables(f.data)
	if err != nil {
		t.Fatal(err)
	}
	denseTables, err := readSFNTTables(subset.Data)
	if err != nil {
		t.Fatal(err)
	}
	originalGlyphData := tableByTag(t, originalTables, tagGlyf, tagLoca, tagHead, tagMaxp)
	denseGlyphData := tableByTag(t, denseTables, tagGlyf, tagLoca, tagHead, tagMaxp)
	compositeCount := 0
	for oldGID, newGID := range subset.GIDMap {
		originalGlyph := testGlyphAt(t, originalGlyphData, oldGID)
		denseGlyph := testGlyphAt(t, denseGlyphData, newGID)
		originalComponents, err := componentsInGlyph(originalGlyph)
		if err != nil {
			t.Fatalf("source composite %d: %v", oldGID, err)
		}
		denseComponents, err := componentsInGlyph(denseGlyph)
		if err != nil {
			t.Fatalf("dense composite %d: %v", newGID, err)
		}
		if len(originalComponents) != len(denseComponents) {
			t.Fatalf("composite %d changed component count", oldGID)
		}
		for i, component := range originalComponents {
			want, ok := subset.GIDMap[component.id]
			if !ok || denseComponents[i].id != want {
				t.Fatalf("composite %d component GID %d→%d, want %d", oldGID, component.id, denseComponents[i].id, want)
			}
		}
		if len(originalComponents) > 0 {
			compositeCount++
		}
	}
	if compositeCount == 0 {
		t.Fatal("fixture text/font did not exercise composite glyph remapping")
	}
}

type testGlyphTables struct{ glyf, loca, head, maxp []byte }

func tableByTag(t *testing.T, tables []sfntTable, tags ...string) testGlyphTables {
	t.Helper()
	found := make(map[string][]byte, len(tags))
	for _, table := range tables {
		found[table.tag] = table.data
	}
	for _, tag := range tags {
		if len(found[tag]) == 0 {
			t.Fatalf("missing %s table", tag)
		}
	}
	return testGlyphTables{found[tagGlyf], found[tagLoca], found[tagHead], found[tagMaxp]}
}

func testGlyphAt(t *testing.T, tables testGlyphTables, gid uint16) []byte {
	t.Helper()
	format := int16(binary.BigEndian.Uint16(tables.head[50:52]))
	var start, end uint32
	if format == 0 {
		start = uint32(binary.BigEndian.Uint16(tables.loca[int(gid)*2:int(gid)*2+2])) * 2
		end = uint32(binary.BigEndian.Uint16(tables.loca[(int(gid)+1)*2:(int(gid)+1)*2+2])) * 2
	} else {
		start = binary.BigEndian.Uint32(tables.loca[int(gid)*4 : int(gid)*4+4])
		end = binary.BigEndian.Uint32(tables.loca[(int(gid)+1)*4 : (int(gid)+1)*4+4])
	}
	if start > end || end > uint32(len(tables.glyf)) {
		t.Fatalf("invalid glyph %d offsets %d..%d", gid, start, end)
	}
	return tables.glyf[start:end]
}

func TestFonts_PDFSubsetKeepsGIDsForUnmappedColorTables(t *testing.T) {
	data, err := os.ReadFile("/usr/share/fonts/liberation/LiberationSans-Regular.ttf")
	if err != nil {
		t.Skip("Liberation Sans is not installed")
	}
	face, err := Load(data)
	if err != nil {
		t.Fatalf("load font: %v", err)
	}
	tables, err := readSFNTTables(data)
	if err != nil {
		t.Fatal(err)
	}
	colr := make([]byte, 14)
	binary.BigEndian.PutUint32(colr[4:8], 14)
	binary.BigEndian.PutUint32(colr[8:12], 14)
	tables = append(tables, sfntTable{tag: "COLR", data: colr})
	data, err = writeSFNT(data[:12], tables)
	if err != nil {
		t.Fatal(err)
	}
	// The shaping face was parsed before injecting a synthetic table; the
	// subsetter sees the additional SFNT table and must select its safe path.
	face.data = data
	shaped := face.Shape("AV", 12)
	subset, err := face.PDFSubsetShaped([]ShapedText{shaped})
	if err != nil {
		t.Fatalf("PDF subset: %v", err)
	}
	for _, glyph := range shaped.Glyphs {
		if mapped := subset.GIDMap[glyph.ID]; mapped != glyph.ID {
			t.Errorf("GID %d with unsupported color table must stay stable, got %d", glyph.ID, mapped)
		}
	}
}

func TestFonts_Shape_ArabicRunUsesRTLClusters(t *testing.T) {
	path := "/usr/share/fonts/noto/NotoSansArabic-Regular.ttf"
	if _, err := os.Stat(path); err != nil {
		t.Skip("Noto Sans Arabic is not installed")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	face, err := Load(data)
	if err != nil {
		t.Fatal(err)
	}
	shaped := face.Shape("سلام", 12)
	if len(shaped.Glyphs) == 0 {
		t.Fatal("Arabic shaping produced no glyphs")
	}
	if !shaped.RightToLeft {
		t.Fatal("the shaped run must preserve its direction for glyph positioning")
	}
	if shaped.Glyphs[0].Cluster <= shaped.Glyphs[len(shaped.Glyphs)-1].Cluster {
		t.Fatalf("cluster order=%v, expected RTL output", glyphClusters(shaped.Glyphs))
	}
}

func glyphClusters(glyphs []Glyph) []int {
	clusters := make([]int, len(glyphs))
	for i, glyph := range glyphs {
		clusters[i] = glyph.Cluster
	}
	return clusters
}
