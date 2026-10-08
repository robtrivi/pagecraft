package boxes

import (
	"strings"
	"unicode"
)

func validGridLineNumber(n int) bool {
	return n != 0 && n >= -maxGridTracks-1 && n <= maxGridTracks+1
}

func isGridAreaName(name string) bool {
	if name == "" || name == "auto" || name == "span" || strings.Contains(name, ".") {
		return false
	}
	for i, r := range name {
		if unicode.IsLetter(r) || r == '_' || r == '-' || (i > 0 && unicode.IsDigit(r)) {
			continue
		}
		return false
	}
	return true
}

func parseGridTemplateAreas(value string) ([][]string, bool) {
	var rows [][]string
	for value = strings.TrimSpace(value); value != ""; value = strings.TrimSpace(value) {
		quote := value[0]
		if quote != '\'' && quote != '"' {
			return nil, false
		}
		end := strings.IndexByte(value[1:], quote)
		if end < 0 {
			return nil, false
		}
		cells := strings.Fields(value[1 : end+1])
		if len(cells) == 0 || (len(rows) > 0 && len(cells) != len(rows[0])) || len(cells) > maxGridTracks {
			return nil, false
		}
		if len(rows)+1 > maxGridTracks/len(cells) {
			return nil, false
		}
		for _, cell := range cells {
			if strings.Trim(cell, ".") != "" && !isGridAreaName(cell) {
				return nil, false
			}
		}
		rows = append(rows, cells)
		value = value[end+2:]
	}
	if len(rows) == 0 {
		return nil, false
	}
	type bounds struct{ top, bottom, left, right, count int }
	areas := make(map[string]bounds)
	for row, cells := range rows {
		for col, name := range cells {
			if name == "." || strings.Trim(name, ".") == "" {
				continue
			}
			area, exists := areas[name]
			if !exists {
				area = bounds{top: row, bottom: row, left: col, right: col}
			} else {
				if row < area.top {
					area.top = row
				}
				if row > area.bottom {
					area.bottom = row
				}
				if col < area.left {
					area.left = col
				}
				if col > area.right {
					area.right = col
				}
			}
			area.count++
			areas[name] = area
		}
	}
	for name, area := range areas {
		if area.count != (area.bottom-area.top+1)*(area.right-area.left+1) {
			return nil, false
		}
		for row := area.top; row <= area.bottom; row++ {
			for col := area.left; col <= area.right; col++ {
				if rows[row][col] != name {
					return nil, false
				}
			}
		}
	}
	return rows, true
}

// collapseSingleChildMargins handles margin collapse through an empty block
// container with one block child. More complex boxes use the current flattened
// flow and are not handled here.
