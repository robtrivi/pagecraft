package layout

import (
	"strings"
	"testing"
)

// runLines joins run text per line with spaces.
// It replaces the removed Lines field.
func runLines(b PlacedBox) []string {
	var out []string
	for _, ln := range b.Runs {
		var parts []string
		for _, r := range ln {
			parts = append(parts, r.Text)
		}
		out = append(out, strings.Join(parts, " "))
	}
	return out
}

func TestHelper_Sanity(t *testing.T) {
	if runLines(PlacedBox{}) != nil {
		t.Error("empty input must return nil")
	}
}
