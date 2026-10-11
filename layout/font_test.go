package layout

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robtrivi/pagecraft/boxes"
	"github.com/robtrivi/pagecraft/fonts"
)

func TestLayout_FontForSelectsWeightVariant(t *testing.T) {
	regular, bold := &fonts.Face{}, &fonts.Face{}
	opts := Options{Faces: map[string]*fonts.Face{"Compare": regular}, FontVariants: map[string]fonts.Variants{"Compare": {Regular: regular, Bold: bold}}}
	box := &boxes.Box{Style: boxes.ComputedStyle{FontFamily: "Compare"}}
	if got := fontFor(opts, box); got != regular {
		t.Fatal("normal text does not use the regular face")
	}
	box.Style.Bold = true
	if got := fontFor(opts, box); got != bold {
		t.Fatal("bold text does not use the bold face")
	}
}

func TestLayout_IsSimpleLTRText_SkipsOnlySafeScripts(t *testing.T) {
	tests := []struct {
		name string
		text string
		want bool
	}{
		{name: "ASCII prose", text: "Northstar Studio 2026", want: true},
		{name: "Latin accents and punctuation", text: "Bogotá — Perú", want: true},
		{name: "mixed Latin and Greek", text: "Northstar Ελληνικά", want: false},
		{name: "Arabic", text: "مرحبا", want: false},
		{name: "bidi control", text: "abc\u202e", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSimpleLTRText(tt.text); got != tt.want {
				t.Errorf("isSimpleLTRText(%q) = %v, want %v", tt.text, got, tt.want)
			}
		})
	}
}

// Per-family measurement step 1: the block carries its Face, and Flow wraps
// lines using that face rather than the global measurer.
// Skips when not system TTF is available (as in fonts/).
func loadFace(t *testing.T) *fonts.Face {
	t.Helper()
	cands, _ := filepath.Glob("/usr/share/fonts/*/*.ttf")
	for _, c := range cands {
		if st, err := os.Stat(c); err == nil && st.Size() > 50000 {
			data, err := os.ReadFile(c)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			f, err := fonts.Load(data)
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			return f
		}
	}
	t.Skip("not system TTF available")
	return nil
}

func TestLayout_CustomFont_MeasuresDifferently(t *testing.T) {
	face := loadFace(t)
	// Text whose wrapping differs between the fake measurer (6 pt/rune) and the font.
	text := ""
	for n := 10; n < 400; n += 10 {
		cand := strings.Repeat("ab ", n)
		fake := len(BreakLines(cand, 451, func(s string, _ float64) float64 {
			return float64(len([]rune(s)) * 6)
		}, 12))
		real := len(BreakLines(cand, 451, face.Measure, 12))
		if fake != real {
			text = cand
			break
		}
	}
	if text == "" {
		t.Skip("the source matches the fake measurer in these cases")
	}
	root, err := boxes.Build(`<p>` + text + `</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	root.Fonts = map[string]*fonts.Face{"Mi": face}
	root.Children[0].Style.FontFamily = "Mi"
	opts := testOptions()
	opts.Faces = root.Fonts
	b := Flow(root, opts)[0].Blocks[0]
	if b.Font == nil {
		t.Fatal("Font nil in PlacedBox")
	}
	want := float64(len(BreakLines(text, 451, face.Measure, 12))) * 12 * 1.2
	if diff := b.Height - want; diff < -1e-9 || diff > 1e-9 {
		t.Errorf("height: got %v, want %v (measured with the source)", b.Height, want)
	}
}

func TestLayout_CustomFont_UsesShapedAdvance(t *testing.T) {
	path := "/usr/share/fonts/liberation/LiberationSans-Regular.ttf"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip("Liberation Sans is not instalada")
	}
	face, err := fonts.Load(data)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	root, err := boxes.Build(`<p>AV</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	root.Fonts = map[string]*fonts.Face{"Mi": face}
	root.Children[0].Style.FontFamily = "Mi"
	root.Children[0].Style.FontSize = 12
	blocks := Flow(root, testOptionsWithFace(face))[0].Blocks
	if len(blocks) != 1 || len(blocks[0].Runs) != 1 || len(blocks[0].Runs[0]) != 1 {
		t.Fatalf("unexpected run structure: %+v", blocks)
	}
	got, want := blocks[0].Runs[0][0].Width, face.Shape("AV", 12).Width
	if got != want {
		t.Errorf("width layout=%v, want shaped advance=%v", got, want)
	}
}

func TestLayout_CustomFont_WrapsWithShapedAdvances(t *testing.T) {
	path := "/usr/share/fonts/liberation/LiberationSans-Regular.ttf"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip("Liberation Sans is not instalada")
	}
	face, err := fonts.Load(data)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	root, err := boxes.Build(`<p style="margin: 0">AV AV AV</p>`)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	root.Fonts = map[string]*fonts.Face{"Mi": face}
	root.Children[0].Style.FontFamily = "Mi"
	root.Children[0].Style.FontSize = 12
	opts := testOptionsWithFace(face)
	opts.PageWidth = 178 // 34 pt usable: two words fit only with TTF kerning.
	blocks := Flow(root, opts)[0].Blocks
	if got := len(blocks[0].Runs); got != 2 {
		t.Fatalf("lines=%d, want 2; content=%q", got, runLines(blocks[0]))
	}
}

func testOptionsWithFace(face *fonts.Face) Options {
	opts := testOptions()
	opts.Faces = map[string]*fonts.Face{"Mi": face}
	return opts
}

func TestLayout_FontFamilyWrapperPreservesMixedInlineAndBlockFlow(t *testing.T) {
	fontPath := "/usr/share/fonts/noto/NotoSerif-Regular.ttf"
	if _, err := os.Stat(fontPath); err != nil {
		t.Skip("Noto Serif is not instalada")
	}
	html := `<style>@font-face { font-family: Compare; src: url(` + fontPath + `); }</style><div style="font-family: Compare; font-size: 16px">before <p style="margin: 0">middle</p>after</div>`
	root, err := boxes.Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	blocks := Flow(root, DefaultOptions())[0].Blocks
	if len(blocks) != 3 {
		t.Fatalf("boxes: got %d, want inline, paragraph e inline", len(blocks))
	}
	want := []string{"before", "middle", "after"}
	for i, block := range blocks {
		got := strings.TrimSpace(strings.Join(runLines(block), " "))
		if got != want[i] {
			t.Errorf("text box %d=%q, want %q", i, got, want[i])
		}
		if block.Font != root.Fonts["Compare"] {
			t.Errorf("box %d does not use the font inherited from the wrapper", i)
		}
	}
}

func TestLayout_RTLLinePlacesLogicalWordsFromRightToLeft(t *testing.T) {
	path := "/usr/share/fonts/noto/NotoSansArabic-Regular.ttf"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip("Noto Sans Arabic is not instalada")
	}
	face, err := fonts.Load(data)
	if err != nil {
		t.Fatal(err)
	}
	root, err := boxes.Build(`<p style="margin:0">سلام عليكم</p>`)
	if err != nil {
		t.Fatal(err)
	}
	root.Fonts = map[string]*fonts.Face{"Arabic": face}
	root.Children[0].Style.FontFamily = "Arabic"
	root.Children[0].Style.FontSize = 12
	opts := testOptionsWithFace(face)
	opts.Faces["Arabic"] = face
	pages := Flow(root, opts)
	if len(pages) != 1 || len(pages[0].Blocks) != 1 || len(pages[0].Blocks[0].Runs) != 1 {
		t.Fatalf("layout unexpected: %+v", pages)
	}
	runs := pages[0].Blocks[0].Runs[0]
	if len(runs) != 2 {
		t.Fatalf("runs=%d, want 2: %+v", len(runs), runs)
	}
	if !runs[0].Shaped.RightToLeft || !runs[1].Shaped.RightToLeft {
		t.Fatal("RTL direction was lost")
	}
	if runs[0].X <= runs[1].X {
		t.Errorf("logical words do not advance from right to left: X=%v,%v", runs[0].X, runs[1].X)
	}
}

func TestLayout_RTLShapingKeepsContextAcrossColorSpan(t *testing.T) {
	path := "/usr/share/fonts/noto/NotoSansArabic-Regular.ttf"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip("Noto Sans Arabic is not instalada")
	}
	face, err := fonts.Load(data)
	if err != nil {
		t.Fatal(err)
	}
	root, err := boxes.Build(`<p style="margin:0"><span>سلا</span><span style="color:#c00">م</span></p>`)
	if err != nil {
		t.Fatal(err)
	}
	root.Fonts = map[string]*fonts.Face{"Arabic": face}
	root.Children[0].Style.FontFamily = "Arabic"
	root.Children[0].Style.FontSize = 12
	for _, child := range root.Children[0].Children {
		child.Style.FontFamily = "Arabic"
		child.Style.FontSize = 12
	}
	opts := testOptionsWithFace(face)
	opts.Faces["Arabic"] = face
	pages := Flow(root, opts)
	runs := pages[0].Blocks[0].Runs[0]
	if len(runs) != 2 {
		t.Fatalf("runs=%d, want 2 by style: %+v", len(runs), runs)
	}
	want := face.Shape("سلام", 12).Width
	got := runs[0].Width + runs[1].Width
	if diff := got - want; diff < -0.01 || diff > 0.01 {
		t.Errorf("span color interrupted the shaping: width=%v, paragraph=%v, delta=%v", got, want, diff)
	}
	if !runs[0].Shaped.RightToLeft || !runs[1].Shaped.RightToLeft {
		t.Fatal("SplitAtRuneOffsets lost the direction")
	}
	if runs[0].X <= runs[1].X {
		t.Errorf("incorrect RTL order: X=%v,%v", runs[0].X, runs[1].X)
	}
}

func TestLayout_MixedBidiLineReordersDirectionalRunsVisually(t *testing.T) {
	path := "/usr/share/fonts/noto/NotoSansArabic-Regular.ttf"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip("Noto Sans Arabic is not instalada")
	}
	face, err := fonts.Load(data)
	if err != nil {
		t.Fatal(err)
	}
	root, err := boxes.Build(`<p style="margin:0">سلام abc دنيا</p>`)
	if err != nil {
		t.Fatal(err)
	}
	root.Fonts = map[string]*fonts.Face{"Arabic": face}
	root.Children[0].Style.FontFamily = "Arabic"
	root.Children[0].Style.FontSize = 12
	opts := testOptionsWithFace(face)
	opts.Faces["Arabic"] = face
	pages := Flow(root, opts)
	runs := pages[0].Blocks[0].Runs[0]
	var firstArabic, latin, lastArabic *TextRun
	for i := range runs {
		switch runs[i].Text {
		case "سلام":
			firstArabic = &runs[i]
		case "abc":
			latin = &runs[i]
		case "دنيا":
			lastArabic = &runs[i]
		}
	}
	if firstArabic == nil || latin == nil || lastArabic == nil {
		t.Fatalf("runs inesperatwo: %+v", runs)
	}
	if !(lastArabic.X < latin.X && latin.X < firstArabic.X) {
		t.Errorf("incorrect mixed RTL visual order: دنيا=%g abc=%g سلام=%g", lastArabic.X, latin.X, firstArabic.X)
	}
}

func TestLayout_BidiItemizesMixedDirectionsInsideWord(t *testing.T) {
	path := "/usr/share/fonts/noto/NotoSansArabic-Regular.ttf"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip("Noto Sans Arabic is not instalada")
	}
	face, err := fonts.Load(data)
	if err != nil {
		t.Fatal(err)
	}
	root, err := boxes.Build(`<p style="margin:0">سلامabc</p>`)
	if err != nil {
		t.Fatal(err)
	}
	root.Fonts = map[string]*fonts.Face{"Arabic": face}
	root.Children[0].Style.FontFamily = "Arabic"
	root.Children[0].Style.FontSize = 12
	opts := testOptionsWithFace(face)
	opts.Faces["Arabic"] = face
	pages := Flow(root, opts)
	runs := pages[0].Blocks[0].Runs[0]
	if len(runs) != 2 {
		t.Fatalf("directions within a word were not itemized: %+v", runs)
	}
	var rtl, ltr *TextRun
	for i := range runs {
		if runs[i].Text == "سلام" {
			rtl = &runs[i]
		}
		if runs[i].Text == "abc" {
			ltr = &runs[i]
		}
	}
	if rtl == nil || ltr == nil {
		t.Fatalf("unexpected bidi segments: %+v", runs)
	}
	if ltr.X >= rtl.X {
		t.Errorf("in an RTL paragraph, the Latin sequence must remain on the left: LTR=%g RTL=%g", ltr.X, rtl.X)
	}
}

func TestLayout_ItemizesScriptChangesInsideRTLWord(t *testing.T) {
	path := "/usr/share/fonts/TTF/DejaVuSans.ttf"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip("DejaVu Sans is not instalada")
	}
	face, err := fonts.Load(data)
	if err != nil {
		t.Fatal(err)
	}
	root, err := boxes.Build(`<p style="margin:0">שלוםسلام</p>`)
	if err != nil {
		t.Fatal(err)
	}
	root.Fonts = map[string]*fonts.Face{"Mixed": face}
	root.Children[0].Style.FontFamily = "Mixed"
	root.Children[0].Style.FontSize = 12
	opts := testOptionsWithFace(face)
	opts.Faces["Mixed"] = face
	pages := Flow(root, opts)
	runs := pages[0].Blocks[0].Runs[0]
	if len(runs) != 2 {
		t.Fatalf("script change was not segmented: %+v", runs)
	}
	if runs[0].Text != "שלום" || runs[1].Text != "سلام" {
		t.Fatalf("segments are out of logical order: %+v", runs)
	}
	if !runs[0].Shaped.RightToLeft || !runs[1].Shaped.RightToLeft {
		t.Fatal("RTL direction was lost from the segment")
	}
}

func TestLayout_BidiParagraphBaseSurvivesExplicitLineBreak(t *testing.T) {
	path := "/usr/share/fonts/noto/NotoSansArabic-Regular.ttf"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip("Noto Sans Arabic is not instalada")
	}
	face, err := fonts.Load(data)
	if err != nil {
		t.Fatal(err)
	}
	root, err := boxes.Build(`<p style="margin:0">سلام<br>abc دنيا</p>`)
	if err != nil {
		t.Fatal(err)
	}
	root.Fonts = map[string]*fonts.Face{"Arabic": face}
	root.Children[0].Style.FontFamily = "Arabic"
	root.Children[0].Style.FontSize = 12
	opts := testOptionsWithFace(face)
	opts.Faces["Arabic"] = face
	pages := Flow(root, opts)
	lines := pages[0].Blocks[0].Runs
	if len(lines) != 2 {
		t.Fatalf("lines=%d, want 2", len(lines))
	}
	var latin, arabic *TextRun
	for i := range lines[1] {
		if lines[1][i].Text == "abc" {
			latin = &lines[1][i]
		}
		if lines[1][i].Text == "دنيا" {
			arabic = &lines[1][i]
		}
	}
	if latin == nil || arabic == nil {
		t.Fatalf("runs inesperatwo in second line: %+v", lines[1])
	}
	if latin.X <= arabic.X {
		t.Errorf("the second line lost its RTL base level: abc=%g دنيا=%g", latin.X, arabic.X)
	}
}
