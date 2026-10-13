package layout

import (
	"fmt"
	"os"
	"testing"

	"github.com/robtrivi/pagecraft/boxes"
)

// A line-height factor of 2 doubles the height from 12 pt to 24 pt.
// Normal flow only in v1.
func TestLayout_LineHeight_DoublesSpacing(t *testing.T) {
	root, err := boxes.Build(`<p style="line-height: 2">Hi</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("blocks: got %d, want 1", len(blocks))
	}
	if !approx(blocks[0].Height, 24) {
		t.Errorf("height: got %v, want 24", blocks[0].Height)
	}
}

func TestLayout_LineHeightNormalHelveticaTracksChromiumMetrics(t *testing.T) {
	root, err := boxes.Build(`<p style="font: 10pt Helvetica; margin: 0">a<br>b</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := DefaultOptions()
	opts.Measure = func(s string, fs float64) float64 { return float64(len([]rune(s))) * fs / 2 }
	blocks := Flow(root, opts)[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("blocks: got %d, want 1", len(blocks))
	}
	if !approx(blocks[0].Height, 22.24) {
		t.Errorf("height normal Helvetica: got %v pt, want 22.24 (2 × 11.12 ptmeasureds in Chromium)", blocks[0].Height)
	}
}

func TestLayout_LineHeightNormalUsesSelectedFontMetrics(t *testing.T) {
	fontPath := "/usr/share/fonts/noto/NotoSerif-Regular.ttf"
	if _, err := os.Stat(fontPath); err != nil {
		t.Skip("Noto Serif is not instalada")
	}
	html := fmt.Sprintf(`<style>@font-face { font-family: Compare; src: url(%s); }</style><p style="font-family: Compare; font-size: 16px; margin: 0">normal</p><p style="font-family: Compare; font-size: 16px; line-height: 1.2; margin: 0">explicit</p>`, fontPath)
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := DefaultOptions()
	opts.Measure = func(s string, fs float64) float64 { return float64(len([]rune(s))) * fs / 2 }
	blocks := Flow(root, opts)[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("blocks: got %d, want 2", len(blocks))
	}
	if !approx(blocks[0].Height, 16.5) {
		t.Errorf("normal: got %v pt, want 16.5 (22 CSS px using Chromium's rounded Noto Serif metrics)", blocks[0].Height)
	}
	if !approx(blocks[1].Height, 14.4) {
		t.Errorf("explicit 1.2: got %v pt, want 14.4", blocks[1].Height)
	}
}
