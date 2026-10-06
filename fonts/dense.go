package fonts

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
)

type glyphComponent struct {
	offset int
	id     uint16
}

type unsupportedDenseSubsetError string

func (e unsupportedDenseSubsetError) Error() string {
	return "fonts: dense subsetting does not remap " + string(e)
}

var unsupportedDenseTables = map[string]bool{
	"COLR": true, "CPAL": true, "SVG ": true, "sbix": true,
	"CBDT": true, "CBLC": true, "EBDT": true, "EBLC": true, "EBSC": true,
	"MATH": true, "VORG": true, "vhea": true, "vmtx": true,
	"gvar": true, "HVAR": true, "VVAR": true, "VARC": true,
	"fvar": true, "avar": true, "cvar": true, "MVAR": true,
	"LTSH": true, "hdmx": true, "VDMX": true, "JSTF": true,
	"morx": true, "mort": true, "kerx": true, "ankr": true,
	"lcar": true, "opbd": true, "trak": true, "prop": true,
	"just": true, "bsln": true, "Silf": true, "Glat": true,
	"Gloc": true, "Feat": true, "Sill": true,
}

func denseGlyphIDs(data []byte, used []uint16, codepoints map[rune]uint16) ([]byte, map[uint16]uint16, error) {
	tables, err := readSFNTTables(data)
	if err != nil {
		return nil, nil, err
	}
	byTag := make(map[string]int, len(tables))
	for i := range tables {
		byTag[tables[i].tag] = i
		if unsupportedDenseTables[tables[i].tag] {
			return nil, nil, unsupportedDenseSubsetError(tables[i].tag)
		}
	}
	get := func(tag string, min int) (int, error) {
		i, ok := byTag[tag]
		if !ok || len(tables[i].data) < min {
			return 0, fmt.Errorf("fonts: dense subset requires %s table", tag)
		}
		return i, nil
	}
	headIdx, err := get(tagHead, 52)
	if err != nil {
		return nil, nil, err
	}
	maxpIdx, err := get(tagMaxp, 6)
	if err != nil {
		return nil, nil, err
	}
	hheaIdx, err := get("hhea", 36)
	if err != nil {
		return nil, nil, err
	}
	hmtxIdx, err := get("hmtx", 4)
	if err != nil {
		return nil, nil, err
	}
	glyfIdx, err := get(tagGlyf, 0)
	if err != nil {
		return nil, nil, err
	}
	locaIdx, err := get(tagLoca, 0)
	if err != nil {
		return nil, nil, err
	}
	oldNumGlyphs := int(binary.BigEndian.Uint16(tables[maxpIdx].data[4:6]))
	format := int16(binary.BigEndian.Uint16(tables[headIdx].data[50:52]))
	oldLoca := tables[locaIdx].data
	entrySize := 4
	if format == 0 {
		entrySize = 2
	} else if format != 1 {
		return nil, nil, fmt.Errorf("fonts: unsupported loca format %d", format)
	}
	if len(oldLoca) < (oldNumGlyphs+1)*entrySize {
		return nil, nil, errors.New("fonts: loca table shorter than maxp glyph count")
	}
	readOffset := func(gid int) uint32 {
		if format == 0 {
			return uint32(binary.BigEndian.Uint16(oldLoca[gid*2:gid*2+2])) * 2
		}
		return binary.BigEndian.Uint32(oldLoca[gid*4 : gid*4+4])
	}
	oldGlyf := tables[glyfIdx].data
	glyphBytes := func(gid uint16) ([]byte, error) {
		if int(gid) >= oldNumGlyphs {
			return nil, fmt.Errorf("fonts: GID %d exceeds maxp glyph count %d", gid, oldNumGlyphs)
		}
		start, end := readOffset(int(gid)), readOffset(int(gid)+1)
		if start > end || end > uint32(len(oldGlyf)) {
			return nil, fmt.Errorf("fonts: invalid offsets for GID %d", gid)
		}
		return oldGlyf[start:end], nil
	}

	keep := map[uint16]bool{0: true}
	queue := []uint16{0}
	for _, gid := range used {
		if !keep[gid] {
			keep[gid] = true
			queue = append(queue, gid)
		}
	}
	componentsByGID := make(map[uint16][]glyphComponent)
	for pos := 0; pos < len(queue); pos++ {
		gid := queue[pos]
		glyph, err := glyphBytes(gid)
		if err != nil {
			return nil, nil, err
		}
		components, err := componentsInGlyph(glyph)
		if err != nil {
			return nil, nil, fmt.Errorf("fonts: GID %d: %w", gid, err)
		}
		componentsByGID[gid] = components
		for _, component := range components {
			if !keep[component.id] {
				keep[component.id] = true
				queue = append(queue, component.id)
			}
		}
	}
	oldGIDs := make([]int, 0, len(keep))
	for gid := range keep {
		oldGIDs = append(oldGIDs, int(gid))
	}
	sort.Ints(oldGIDs)
	if len(oldGIDs) > 0xffff {
		return nil, nil, errors.New("fonts: dense subset exceeds TrueType glyph limit")
	}
	gidMap := make(map[uint16]uint16, len(oldGIDs))
	for newID, oldID := range oldGIDs {
		gidMap[uint16(oldID)] = uint16(newID)
	}

	newGlyf := make([]byte, 0)
	newOffsets := make([]uint32, len(oldGIDs)+1)
	for newID, rawOldID := range oldGIDs {
		oldID := uint16(rawOldID)
		newOffsets[newID] = uint32(len(newGlyf))
		glyph, err := glyphBytes(oldID)
		if err != nil {
			return nil, nil, err
		}
		glyph = append([]byte(nil), glyph...)
		for _, component := range componentsByGID[oldID] {
			newID, ok := gidMap[component.id]
			if !ok {
				return nil, nil, fmt.Errorf("fonts: missing composite GID mapping %d", component.id)
			}
			binary.BigEndian.PutUint16(glyph[component.offset:component.offset+2], newID)
		}
		newGlyf = append(newGlyf, glyph...)
		if len(newGlyf)%2 != 0 {
			newGlyf = append(newGlyf, 0)
		}
	}
	newOffsets[len(oldGIDs)] = uint32(len(newGlyf))
	shortLoca := len(newGlyf) <= 0x1fffe
	locaSize := 4
	if shortLoca {
		locaSize = 2
	}
	newLoca := make([]byte, len(newOffsets)*locaSize)
	for i, offset := range newOffsets {
		if shortLoca {
			binary.BigEndian.PutUint16(newLoca[i*2:i*2+2], uint16(offset/2))
		} else {
			binary.BigEndian.PutUint32(newLoca[i*4:i*4+4], offset)
		}
	}

	metrics, err := denseHorizontalMetrics(tables[hheaIdx].data, tables[hmtxIdx].data, oldNumGlyphs, oldGIDs)
	if err != nil {
		return nil, nil, err
	}
	newHhea := append([]byte(nil), tables[hheaIdx].data...)
	binary.BigEndian.PutUint16(newHhea[34:36], uint16(len(oldGIDs)))
	newMaxp := append([]byte(nil), tables[maxpIdx].data...)
	binary.BigEndian.PutUint16(newMaxp[4:6], uint16(len(oldGIDs)))
	newHead := append([]byte(nil), tables[headIdx].data...)
	if shortLoca {
		binary.BigEndian.PutUint16(newHead[50:52], 0)
	} else {
		binary.BigEndian.PutUint16(newHead[50:52], 1)
	}
	tables[glyfIdx].data, tables[locaIdx].data = newGlyf, newLoca
	tables[hheaIdx].data, tables[hmtxIdx].data = newHhea, metrics
	tables[maxpIdx].data, tables[headIdx].data = newMaxp, newHead
	cmap, err := buildDenseCmap(codepoints, gidMap)
	if err != nil {
		return nil, nil, err
	}
	cmapIdx, ok := byTag["cmap"]
	if !ok {
		return nil, nil, errors.New("fonts: dense subset requires cmap table")
	}
	tables[cmapIdx].data = cmap
	fontData, err := writeSFNT(data[:12], tables)
	if err != nil {
		return nil, nil, err
	}
	return fontData, gidMap, nil
}

func componentsInGlyph(glyph []byte) ([]glyphComponent, error) {
	if len(glyph) == 0 {
		return nil, nil
	}
	if len(glyph) < 10 {
		return nil, errors.New("truncated glyph header")
	}
	if int16(binary.BigEndian.Uint16(glyph[:2])) >= 0 {
		return nil, nil
	}
	var components []glyphComponent
	pos := 10
	var flags uint16
	for {
		if pos+4 > len(glyph) {
			return nil, errors.New("truncated composite glyph component")
		}
		flags = binary.BigEndian.Uint16(glyph[pos : pos+2])
		components = append(components, glyphComponent{offset: pos + 2, id: binary.BigEndian.Uint16(glyph[pos+2 : pos+4])})
		pos += 4
		if flags&0x0001 != 0 {
			pos += 4
		} else {
			pos += 2
		}
		switch {
		case flags&0x0008 != 0:
			pos += 2
		case flags&0x0040 != 0:
			pos += 4
		case flags&0x0080 != 0:
			pos += 8
		}
		if pos > len(glyph) {
			return nil, errors.New("truncated composite glyph transform")
		}
		if flags&0x0020 == 0 {
			break
		}
	}
	if flags&0x0100 != 0 {
		if pos+2 > len(glyph) {
			return nil, errors.New("truncated composite glyph instructions")
		}
		instructionLength := int(binary.BigEndian.Uint16(glyph[pos : pos+2]))
		if pos+2+instructionLength > len(glyph) {
			return nil, errors.New("truncated composite glyph instruction bytes")
		}
	}
	return components, nil
}

func denseHorizontalMetrics(hhea, hmtx []byte, numGlyphs int, glyphs []int) ([]byte, error) {
	numMetrics := int(binary.BigEndian.Uint16(hhea[34:36]))
	if numMetrics < 1 || numMetrics > numGlyphs || len(hmtx) < numMetrics*4+(numGlyphs-numMetrics)*2 {
		return nil, errors.New("fonts: malformed horizontal metrics tables")
	}
	newHmtx := make([]byte, len(glyphs)*4)
	for newID, oldID := range glyphs {
		metricID := oldID
		if metricID >= numMetrics {
			metricID = numMetrics - 1
		}
		advance := binary.BigEndian.Uint16(hmtx[metricID*4 : metricID*4+2])
		var lsb int16
		if oldID < numMetrics {
			lsb = int16(binary.BigEndian.Uint16(hmtx[oldID*4+2 : oldID*4+4]))
		} else {
			offset := numMetrics*4 + (oldID-numMetrics)*2
			lsb = int16(binary.BigEndian.Uint16(hmtx[offset : offset+2]))
		}
		binary.BigEndian.PutUint16(newHmtx[newID*4:newID*4+2], advance)
		binary.BigEndian.PutUint16(newHmtx[newID*4+2:newID*4+4], uint16(lsb))
	}
	return newHmtx, nil
}

type denseCmapEntry struct {
	codepoint uint32
	gid       uint16
}

func buildDenseCmap(codepoints map[rune]uint16, gidMap map[uint16]uint16) ([]byte, error) {
	entries := make([]denseCmapEntry, 0, len(codepoints))
	for r, oldGID := range codepoints {
		gid, ok := gidMap[oldGID]
		if ok && r >= 0 && r <= 0x10ffff {
			entries = append(entries, denseCmapEntry{uint32(r), gid})
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].codepoint < entries[j].codepoint })
	var bmp []denseCmapEntry
	for _, entry := range entries {
		if entry.codepoint < 0xffff {
			bmp = append(bmp, entry)
		}
	}
	format4, err := buildCmapFormat4(bmp)
	if err != nil {
		return nil, err
	}
	format12 := buildCmapFormat12(entries)
	const headerLength = 20
	cmap := make([]byte, headerLength+len(format4)+len(format12))
	binary.BigEndian.PutUint16(cmap[2:4], 2)
	// Windows BMP and Unicode full-repertoire cmap records.
	binary.BigEndian.PutUint16(cmap[4:6], 3)
	binary.BigEndian.PutUint16(cmap[6:8], 1)
	binary.BigEndian.PutUint32(cmap[8:12], headerLength)
	binary.BigEndian.PutUint16(cmap[12:14], 3)
	binary.BigEndian.PutUint16(cmap[14:16], 10)
	binary.BigEndian.PutUint32(cmap[16:20], uint32(headerLength+len(format4)))
	copy(cmap[headerLength:], format4)
	copy(cmap[headerLength+len(format4):], format12)
	return cmap, nil
}

func buildCmapFormat4(entries []denseCmapEntry) ([]byte, error) {
	type segment struct {
		start, end uint16
		delta      uint16
	}
	segments := make([]segment, 0, len(entries)+1)
	for _, entry := range entries {
		cp := uint16(entry.codepoint)
		delta := entry.gid - cp
		if len(segments) > 0 && uint32(segments[len(segments)-1].end)+1 == entry.codepoint && segments[len(segments)-1].delta == delta {
			segments[len(segments)-1].end = cp
		} else {
			segments = append(segments, segment{cp, cp, delta})
		}
	}
	segments = append(segments, segment{0xffff, 0xffff, 1})
	length := 16 + 8*len(segments)
	if length > 0xffff {
		return nil, errors.New("fonts: BMP cmap exceeds format 4 size limit")
	}
	out := make([]byte, length)
	binary.BigEndian.PutUint16(out[0:2], 4)
	binary.BigEndian.PutUint16(out[2:4], uint16(length))
	n := len(segments)
	binary.BigEndian.PutUint16(out[6:8], uint16(n*2))
	power, selector := 1, 0
	for power*2 <= n {
		power *= 2
		selector++
	}
	binary.BigEndian.PutUint16(out[8:10], uint16(power*2))
	binary.BigEndian.PutUint16(out[10:12], uint16(selector))
	binary.BigEndian.PutUint16(out[12:14], uint16(n*2-power*2))
	off := 14
	for i, seg := range segments {
		binary.BigEndian.PutUint16(out[off+2*i:off+2*i+2], seg.end)
	}
	off += 2 * n
	off += 2 // reservedPad
	for i, seg := range segments {
		binary.BigEndian.PutUint16(out[off+2*i:off+2*i+2], seg.start)
	}
	off += 2 * n
	for i, seg := range segments {
		binary.BigEndian.PutUint16(out[off+2*i:off+2*i+2], seg.delta)
	}
	off += 2 * n
	return out, nil
}

func buildCmapFormat12(entries []denseCmapEntry) []byte {
	type group struct{ start, end, gid uint32 }
	groups := make([]group, 0, len(entries))
	for _, entry := range entries {
		cp, gid := entry.codepoint, uint32(entry.gid)
		if len(groups) > 0 && groups[len(groups)-1].end+1 == cp && groups[len(groups)-1].gid+(groups[len(groups)-1].end-groups[len(groups)-1].start)+1 == gid {
			groups[len(groups)-1].end = cp
		} else {
			groups = append(groups, group{cp, cp, gid})
		}
	}
	out := make([]byte, 16+12*len(groups))
	binary.BigEndian.PutUint16(out[0:2], 12)
	binary.BigEndian.PutUint32(out[4:8], uint32(len(out)))
	binary.BigEndian.PutUint32(out[12:16], uint32(len(groups)))
	for i, g := range groups {
		off := 16 + i*12
		binary.BigEndian.PutUint32(out[off:off+4], g.start)
		binary.BigEndian.PutUint32(out[off+4:off+8], g.end)
		binary.BigEndian.PutUint32(out[off+8:off+12], g.gid)
	}
	return out
}
