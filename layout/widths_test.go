package layout

import (
	"testing"

	"github.com/robtrivi/pagecraft/boxes"
)

// Percent widths should partition the table's track area after separate-mode
// border spacing, preserving the requested ratio across every row.
func TestLayout_TablePercentWidthsPreserveRatiosAcrossRows(t *testing.T) {
	for _, tc := range []struct {
		name                          string
		widthCSS, firstCSS, secondCSS string
		width, firstPercent, second   float64
	}{
		{"30/70", "451pt", "30%", "70%", 451, 30, 70},
		{"quarter/three-quarters", "320pt", "25%", "75%", 320, 25, 75},
		{"equal columns", "200pt", "50%", "50%", 200, 50, 50},
	} {
		t.Run(tc.name, func(t *testing.T) {
			html := `<table style="width:` + tc.widthCSS + `"><tr><td style="width:` + tc.firstCSS + `">A</td><td style="width:` + tc.secondCSS + `">B</td></tr><tr><td>C</td><td>D</td></tr></table>`
			root, err := boxes.Build(html)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			blocks := Flow(root, testOptions())[0].Blocks
			if len(blocks) != 4 {
				t.Fatalf("blocks: got %d, want 4", len(blocks))
			}
			first, second := blocks[0].Width, blocks[1].Width
			trackWidth := tc.width - 4.5 // 2px spacing at both edges and between two columns.
			if !approx(first+second, trackWidth) {
				t.Errorf("columns occupy %gpt, want table track width %gpt after border spacing", first+second, trackWidth)
			}
			if !approx(first*tc.second, second*tc.firstPercent) {
				t.Errorf("column widths %g:%g do not match requested ratio %g:%g", first, second, tc.firstPercent, tc.second)
			}
			if !approx(blocks[2].Width, first) || !approx(blocks[3].Width, second) {
				t.Errorf("second row widths %g:%g differ from first row %g:%g", blocks[2].Width, blocks[3].Width, first, second)
			}
		})
	}
}

func TestLayout_TableMaxWidthClampsDeclaredWidth(t *testing.T) {
	root, err := boxes.Build(`<table style="width:120pt;max-width:100pt"></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	got := tableUsedWidth(root.Children[0], nil, 0, 200, testOptions())
	if got != 100 {
		t.Fatalf("table width=%v, want max-width 100pt", got)
	}
}
