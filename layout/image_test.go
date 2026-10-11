package layout

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"github.com/robtrivi/pagecraft/boxes"
)

// Image step 3: <img> in normal flow is sized by aspect ratio (1px = 1pt),
// capped at available width. Normal flow only in v1.
func imgURI(t *testing.T, w, h int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.RGBA{0, 128, 255, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func TestLayout_Image_SizedByAspect(t *testing.T) {
	root, err := boxes.Build(`<img src="` + imgURI(t, 200, 100) + `">`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pages := Flow(root, testOptions())
	blocks := pages[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("blocks: got %d, want 1", len(blocks))
	}
	b := blocks[0]
	if b.Image == nil {
		t.Fatal("Image nil in PlacedBox")
	}
	if b.Width != 150 || b.Height != 75 {
		t.Errorf("size: got %vx%v, want 150x75", b.Width, b.Height)
	}
}

func TestLayout_InlineImageInTableCellKeepsItsInlinePosition(t *testing.T) {
	root, err := boxes.Build(`<style>body{margin:0}table{margin:0;width:220pt}td{font:10pt/12pt Helvetica;padding:0}</style><table><tr><td>BEFORE <img style="width:24pt;height:20pt" src="` + imgURI(t, 4, 2) + `"> AFTER</td></tr></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	var cell *PlacedBox
	for index := range blocks {
		if len(blocks[index].Runs) > 0 {
			cell = &blocks[index]
			break
		}
	}
	if cell == nil {
		t.Fatal("cell with inline content was not found")
	}
	var before, imageRun, after *TextRun
	for _, line := range cell.Runs {
		for index := range line {
			run := &line[index]
			switch run.Text {
			case "BEFORE":
				before = run
			case "AFTER":
				after = run
			}
			if run.Image != nil {
				imageRun = run
			}
		}
	}
	if before == nil || after == nil || imageRun == nil {
		t.Fatalf("was lost content inline of cell: before=%v image=%v after=%v", before != nil, imageRun != nil, after != nil)
	}
	if imageRun.Width != 24 {
		t.Errorf("width CSS of image inline=%g, want 24 pt", imageRun.Width)
	}
	if cell.Height < 20 {
		t.Errorf("height of line/cell=%g, must include the 20pt of the image", cell.Height)
	}
	if !(before.X < imageRun.X && imageRun.X+imageRun.Width <= after.X) {
		t.Errorf("incorrect inline order: BEFORE x=%g, image x=%g w=%g, AFTER x=%g", before.X, imageRun.X, imageRun.Width, after.X)
	}
}

func TestLayout_InlineImageInBlockTextRemainsInInlineRuns(t *testing.T) {
	root, err := boxes.Build(`<p style="margin:0;font:10pt/12pt Helvetica">BEFORE <img style="width:24pt;height:20pt" src="` + imgURI(t, 4, 2) + `"> AFTER</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	var imageRun *TextRun
	for _, block := range blocks {
		for _, line := range block.Runs {
			for index := range line {
				if line[index].Image != nil {
					imageRun = &line[index]
				}
			}
		}
	}
	if imageRun == nil {
		t.Fatal("the replaced image was lost when flattening the block inline content")
	}
}

func TestLayout_InlineImageInTableCellStaysAtomicAcrossPages(t *testing.T) {
	lines := make([]string, 17)
	for index := range lines {
		lines[index] = "LINE"
	}
	cellHTML := strings.Join(lines[:11], "<br>") + `<br>BEFORE <img style="width:24pt;height:20pt" src="` + imgURI(t, 4, 2) + `"> AFTER<br>` + strings.Join(lines[12:], "<br>")
	root, err := boxes.Build(`<style>body{margin:0}table{margin:0;width:220pt}td{font:10pt/12pt Helvetica;padding:0;vertical-align:top}</style><table><tr><td>` + cellHTML + `</td></tr></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.PageHeight, opts.Margin = 220, 150, 0
	opts.FontSize, opts.LineHeight = 10, 1.2
	pages := Flow(root, opts)
	if len(pages) != 2 {
		t.Fatalf("pages=%d, want 2", len(pages))
	}
	imageCount := 0
	for pageIndex, page := range pages {
		for _, block := range page.Blocks {
			for _, line := range block.Runs {
				for _, run := range line {
					if run.Image == nil {
						continue
					}
					imageCount++
					if pageIndex != 1 {
						t.Errorf("the image indivisible should continue on page 2; pageIndex=%d", pageIndex)
					}
					if run.Y < 0 || run.Y+run.ImageHeight > opts.PageHeight {
						t.Errorf("image outside its page: y=%g height=%g", run.Y, run.ImageHeight)
					}
				}
			}
		}
	}
	if imageCount != 1 {
		t.Errorf("fragments of image=%d, want 1; the images inline not dividen ni duplican", imageCount)
	}
}

func TestLayout_InlineImageContributesToTableLineWrapping(t *testing.T) {
	root, err := boxes.Build(`<style>body{margin:0}table{margin:0;width:220pt}td{font:10pt/12pt Helvetica;padding:0}td:first-child{width:40%}</style><table><tr><td>BEFORE <img style="width:30pt;height:10pt" src="` + imgURI(t, 4, 2) + `"> AFTER</td><td></td></tr></table>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	opts := testOptions()
	opts.PageWidth, opts.Margin = 220, 0
	blocks := Flow(root, opts)[0].Blocks
	var cell *PlacedBox
	for index := range blocks {
		if len(blocks[index].Runs) > 0 {
			cell = &blocks[index]
			break
		}
	}
	if cell == nil || len(cell.Runs) != 2 {
		t.Fatalf("the image and following text must wrap to a second line; cell=%v", cell)
	}
	var before, imageRun, after *TextRun
	for _, line := range cell.Runs {
		for index := range line {
			run := &line[index]
			switch run.Text {
			case "BEFORE":
				before = run
			case "AFTER":
				after = run
			}
			if run.Image != nil {
				imageRun = run
			}
		}
	}
	if before == nil || imageRun == nil || after == nil {
		t.Fatalf("flow items were lost: before=%v image=%v after=%v", before != nil, imageRun != nil, after != nil)
	}
	if before.Y != imageRun.Y || after.Y <= imageRun.Y {
		t.Errorf("image width does not participate in wrapping: BEFORE y=%g image y=%g AFTER y=%g", before.Y, imageRun.Y, after.Y)
	}
}

func TestLayout_InlineWidthConstrainsImageInsideStyledRun(t *testing.T) {
	root, err := boxes.Build(`<div><span><img src="` + imgURI(t, 100, 50) + `"></span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	span := root.Children[0].Children[0]
	words := flattenInlineWidth(span, 40, testOptions())
	if len(words) != 1 || words[0].image == nil {
		t.Fatalf("inline words=%+v, want one image", words)
	}
	if words[0].w > 40 || words[0].imageH > 20 {
		t.Errorf("image did not preserve the available width or aspect ratio: %.2fx%.2fpt", words[0].w, words[0].imageH)
	}
}

func TestLayout_WideImage_ScalesToUsable(t *testing.T) {
	root, err := boxes.Build(`<img src="` + imgURI(t, 900, 100) + `">`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b := Flow(root, testOptions())[0].Blocks[0]
	// 451 * 100/900 = 50.111...
	if b.Width != 451 {
		t.Errorf("width: got %v, want 451", b.Width)
	}
	if !approx(b.Height, 451.0*100/900) {
		t.Errorf("height: got %v, want %v", b.Height, 451.0*100/900)
	}
}

func TestLayout_ImageInFlex_UsesIntrinsicSizeAndSetsRowHeight(t *testing.T) {
	root, err := boxes.Build(`<div style="display:flex"><img src="` + imgURI(t, 200, 100) + `"><span>A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 2 {
		t.Fatalf("blocks: got %d, want 2", len(blocks))
	}
	img := blocks[0]
	if img.Image == nil {
		t.Fatal("the image was not preserved in the block flex")
	}
	if !approx(img.Width, 150) || !approx(img.Height, 75) || !approx(blocks[1].Y, 72) || !approx(blocks[1].Height, 75) {
		t.Errorf("image flex=%gx%g; second box y=%g height=%g; want image 150x75 and box 72/75", img.Width, img.Height, blocks[1].Y, blocks[1].Height)
	}
}

func TestLayout_ImageInGrid_UsesIntrinsicSizeAndSizesAutomaticRow(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;grid-template-columns:100pt;grid-template-rows:auto"><img src="` + imgURI(t, 200, 100) + `"></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, testOptions())[0].Blocks
	if len(blocks) != 1 {
		t.Fatalf("blocks: got %d, want 1", len(blocks))
	}
	img := blocks[0]
	if img.Image == nil {
		t.Fatal("the image was not preserved in the block grid")
	}
	if !approx(img.Width, 150) || !approx(img.Height, 75) {
		t.Errorf("image grid=%gx%g, want intrinsic 150x75", img.Width, img.Height)
	}
}

func TestLayout_ImageInFlex_ExplicitWidthKeepsAspectRatio(t *testing.T) {
	root, err := boxes.Build(`<div style="display:flex"><img style="width:80px" src="` + imgURI(t, 200, 100) + `"><span>A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	img := Flow(root, testOptions())[0].Blocks[0]
	if !approx(img.Width, 60) || !approx(img.Height, 30) {
		t.Errorf("image flex with width: %gx%g, want 60x30", img.Width, img.Height)
	}
}

func TestLayout_ImageInGrid_ExplicitHeightKeepsAspectRatio(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;grid-template-columns:100pt;grid-template-rows:auto"><img style="height:40pt" src="` + imgURI(t, 200, 100) + `"></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	img := Flow(root, testOptions())[0].Blocks[0]
	if !approx(img.Width, 80) || !approx(img.Height, 40) {
		t.Errorf("image grid with height: %gx%g, want 80x40", img.Width, img.Height)
	}
}

func TestLayout_ImageInGrid_ExplicitWidthAndHeightAreUsed(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;grid-template-columns:100pt;grid-template-rows:auto"><img style="width:80px;height:40pt" src="` + imgURI(t, 200, 100) + `"></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	img := Flow(root, testOptions())[0].Blocks[0]
	if !approx(img.Width, 60) || !approx(img.Height, 40) {
		t.Errorf("image grid with width and height: %gx%g, want 60x40", img.Width, img.Height)
	}
}

func TestLayout_ImageInFlex_PercentWidthUsesItemTrack(t *testing.T) {
	root, err := boxes.Build(`<div style="display:flex"><img style="width:50%" src="` + imgURI(t, 200, 100) + `"><span>A</span></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	img := Flow(root, testOptions())[0].Blocks[0]
	if !approx(img.Width, 112.75) || !approx(img.Height, 56.375) {
		t.Errorf("image flex with percentage width: %gx%g, want 112.75x56.375", img.Width, img.Height)
	}
}

func TestLayout_ImageInGrid_PercentWidthUsesTrack(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;grid-template-columns:100pt;grid-template-rows:auto"><img style="width:50%" src="` + imgURI(t, 200, 100) + `"></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	img := Flow(root, testOptions())[0].Blocks[0]
	if !approx(img.Width, 50) || !approx(img.Height, 25) {
		t.Errorf("image grid with percentage width: %gx%g, want 50x25", img.Width, img.Height)
	}
}

func TestLayout_ImageInGrid_PercentHeightUsesFixedRow(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;grid-template-columns:100pt;grid-template-rows:80pt"><img style="height:50%" src="` + imgURI(t, 200, 100) + `"></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	img := Flow(root, testOptions())[0].Blocks[0]
	if !approx(img.Width, 80) || !approx(img.Height, 40) {
		t.Errorf("image grid with percentage height: %gx%g, want 80x40", img.Width, img.Height)
	}
}

func TestLayout_ImageInGrid_UsesIntrinsicSizeInFixedRow(t *testing.T) {
	root, err := boxes.Build(`<div style="display:grid;grid-template-columns:100pt;grid-template-rows:30pt"><img src="` + imgURI(t, 200, 100) + `"></div>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	img := Flow(root, testOptions())[0].Blocks[0]
	if !approx(img.Width, 150) || !approx(img.Height, 75) {
		t.Errorf("image grid in fixed row: %gx%g, want intrinsic 150x75", img.Width, img.Height)
	}
}

func TestLayout_InlineSVG_UsesCSSViewportSize(t *testing.T) {
	for _, tc := range []struct {
		name, style string
		w, h        float64
	}{
		{"width", "width: 80px", 60, 30},
		{"height", "height: 40px", 60, 30},
		{"both", "width: 80px; height: 60px", 60, 45},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, err := boxes.Build(`<svg width="40" height="20" style="` + tc.style + `"><rect width="40" height="20"/></svg>`)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			b := Flow(root, testOptions())[0].Blocks[0]
			if b.Width != tc.w || b.Height != tc.h {
				t.Errorf("SVG size = %gx%gpt, want %gx%gpt", b.Width, b.Height, tc.w, tc.h)
			}
		})
	}
}

func TestLayout_InlineSVG_PercentWidthUsesAvailableWidth(t *testing.T) {
	root, err := boxes.Build(`<svg width="40" height="20" style="width: 50%"><rect width="40" height="20"/></svg>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	b := Flow(root, testOptions())[0].Blocks[0]
	if b.Width != 225.5 || b.Height != 112.75 {
		t.Errorf("SVG size = %gx%gpt, want 225.5x112.75pt", b.Width, b.Height)
	}
}

func TestLayout_InlineSVG_PercentHeightOnAutoParentUsesIntrinsicRatio(t *testing.T) {
	root, err := boxes.Build(`<svg width="40" height="20" style="height: 10%"><rect width="40" height="20"/></svg>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	b := Flow(root, testOptions())[0].Blocks[0]
	if !approx(b.Width, 30) || !approx(b.Height, 15) {
		t.Errorf("SVG size = %gx%gpt, want intrinsic 30x15pt because the containing height is auto", b.Width, b.Height)
	}
}
