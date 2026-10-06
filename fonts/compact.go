package fonts

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
)

const (
	tagHead = "head"
	tagGlyf = "glyf"
	tagLoca = "loca"
	tagMaxp = "maxp"
	tagPost = "post"
)

var unusedEmbeddedTables = map[string]bool{
	"DSIG": true, "FFTM": true, "GDEF": true, "GPOS": true,
	"GSUB": true, "kern": true,
}

type sfntTable struct {
	tag  string
	data []byte
}

// compactGlyphTables removes the empty outline slots left by GID-preserving
// subsetting. GIDs remain stable; loca is rebuilt with empty entries for the
// unused glyphs, so PDF CIDs, shaped runs and ToUnicode need not be remapped.
func compactGlyphTables(data []byte) ([]byte, error) {
	tables, err := readSFNTTables(data)
	if err != nil {
		return nil, err
	}
	byTag := make(map[string]int, len(tables))
	for i := range tables {
		byTag[tables[i].tag] = i
	}
	idx, hasGlyf := byTag[tagGlyf]
	locaIdx, hasLoca := byTag[tagLoca]
	headIdx, hasHead := byTag[tagHead]
	maxpIdx, hasMaxp := byTag[tagMaxp]
	if !hasGlyf || !hasLoca || !hasHead || !hasMaxp {
		return data, nil
	}
	head := append([]byte(nil), tables[headIdx].data...)
	if len(head) < 52 || len(tables[maxpIdx].data) < 6 {
		return nil, errors.New("fonts: truncated head or maxp table")
	}
	format := int16(binary.BigEndian.Uint16(head[50:52]))
	if format != 0 && format != 1 {
		return nil, fmt.Errorf("fonts: unsupported loca format %d", format)
	}
	numGlyphs := int(binary.BigEndian.Uint16(tables[maxpIdx].data[4:6]))
	loca := tables[locaIdx].data
	entrySize := 4
	if format == 0 {
		entrySize = 2
	}
	if len(loca) < (numGlyphs+1)*entrySize {
		return nil, errors.New("fonts: loca table is shorter than maxp glyph count")
	}
	readOffset := func(i int) uint32 {
		if format == 0 {
			return uint32(binary.BigEndian.Uint16(loca[i*2:i*2+2])) * 2
		}
		return binary.BigEndian.Uint32(loca[i*4 : i*4+4])
	}
	oldGlyf := tables[idx].data
	newGlyf := make([]byte, 0, len(oldGlyf))
	newOffsets := make([]uint32, numGlyphs+1)
	for gid := 0; gid < numGlyphs; gid++ {
		newOffsets[gid] = uint32(len(newGlyf))
		start, end := readOffset(gid), readOffset(gid+1)
		if start > end || end > uint32(len(oldGlyf)) {
			return nil, fmt.Errorf("fonts: invalid glyph offsets for GID %d", gid)
		}
		glyph := oldGlyf[start:end]
		if len(glyph) > 0 && !allZero(glyph) {
			newGlyf = append(newGlyf, glyph...)
			if len(newGlyf)%2 != 0 {
				newGlyf = append(newGlyf, 0)
			}
		}
	}
	newOffsets[numGlyphs] = uint32(len(newGlyf))
	shortLoca := len(newGlyf) <= 0x1fffe
	locaEntrySize := 4
	if shortLoca {
		locaEntrySize = 2
	}
	newLoca := make([]byte, (numGlyphs+1)*locaEntrySize)
	for i, offset := range newOffsets {
		if shortLoca {
			binary.BigEndian.PutUint16(newLoca[i*2:i*2+2], uint16(offset/2))
		} else {
			binary.BigEndian.PutUint32(newLoca[i*4:i*4+4], offset)
		}
	}
	if shortLoca {
		binary.BigEndian.PutUint16(head[50:52], 0)
	} else {
		binary.BigEndian.PutUint16(head[50:52], 1)
	}
	tables[idx].data = newGlyf
	tables[locaIdx].data = newLoca
	tables[headIdx].data = head
	filtered := tables[:0]
	for _, table := range tables {
		if unusedEmbeddedTables[table.tag] {
			continue
		}
		if table.tag == tagPost {
			if len(table.data) < 32 {
				return nil, errors.New("fonts: truncated post table")
			}
			post := append([]byte(nil), table.data[:32]...)
			binary.BigEndian.PutUint32(post[:4], 0x00030000)
			table.data = post
		}
		filtered = append(filtered, table)
	}
	return writeSFNT(data[:12], filtered)
}

func allZero(data []byte) bool {
	for _, b := range data {
		if b != 0 {
			return false
		}
	}
	return true
}

func readSFNTTables(data []byte) ([]sfntTable, error) {
	if len(data) < 12 {
		return nil, errors.New("fonts: truncated SFNT header")
	}
	count := int(binary.BigEndian.Uint16(data[4:6]))
	if len(data) < 12+count*16 {
		return nil, errors.New("fonts: truncated SFNT table directory")
	}
	tables := make([]sfntTable, 0, count)
	for i := 0; i < count; i++ {
		entry := data[12+i*16 : 12+(i+1)*16]
		tag := string(entry[:4])
		offset := binary.BigEndian.Uint32(entry[8:12])
		length := binary.BigEndian.Uint32(entry[12:16])
		if uint64(offset)+uint64(length) > uint64(len(data)) {
			return nil, fmt.Errorf("fonts: SFNT table %q extends beyond input", tag)
		}
		tables = append(tables, sfntTable{tag: tag, data: append([]byte(nil), data[offset:offset+length]...)})
	}
	return tables, nil
}

func writeSFNT(header []byte, tables []sfntTable) ([]byte, error) {
	sort.Slice(tables, func(i, j int) bool { return tables[i].tag < tables[j].tag })
	count := len(tables)
	if len(header) != 12 || count > 0xffff {
		return nil, errors.New("fonts: invalid SFNT header or table count")
	}
	out := make([]byte, 12+count*16)
	copy(out, header)
	binary.BigEndian.PutUint16(out[4:6], uint16(count))
	searchRange, selector := sfntSearchParameters(count)
	binary.BigEndian.PutUint16(out[6:8], searchRange)
	binary.BigEndian.PutUint16(out[8:10], selector)
	binary.BigEndian.PutUint16(out[10:12], uint16(count*16)-searchRange)

	headRecord := -1
	for i, table := range tables {
		for len(out)%4 != 0 {
			out = append(out, 0)
		}
		offset := len(out)
		payload := append([]byte(nil), table.data...)
		if table.tag == tagHead {
			if len(payload) < 12 {
				return nil, errors.New("fonts: truncated head table")
			}
			binary.BigEndian.PutUint32(payload[8:12], 0)
			headRecord = i
		}
		entry := out[12+i*16 : 12+(i+1)*16]
		copy(entry[:4], table.tag)
		binary.BigEndian.PutUint32(entry[4:8], sfntChecksum(payload))
		binary.BigEndian.PutUint32(entry[8:12], uint32(offset))
		binary.BigEndian.PutUint32(entry[12:16], uint32(len(payload)))
		out = append(out, payload...)
	}
	if headRecord < 0 {
		return nil, errors.New("fonts: missing head table")
	}
	fontSum := sfntChecksum(out)
	adjustment := uint32(0xB1B0AFBA) - fontSum
	headOffset := binary.BigEndian.Uint32(out[12+headRecord*16+8 : 12+headRecord*16+12])
	binary.BigEndian.PutUint32(out[headOffset+8:headOffset+12], adjustment)
	return out, nil
}

func sfntChecksum(data []byte) uint32 {
	var sum uint32
	for i := 0; i < len(data); i += 4 {
		var word [4]byte
		copy(word[:], data[i:min(i+4, len(data))])
		sum += binary.BigEndian.Uint32(word[:])
	}
	return sum
}

func sfntSearchParameters(count int) (uint16, uint16) {
	if count == 0 {
		return 0, 0
	}
	power := 1
	selector := 0
	for power*2 <= count {
		power *= 2
		selector++
	}
	return uint16(power * 16), uint16(selector)
}
