package render

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Custom fonts phase 2 final step: @font-face + font-family embeds a subset
// as CIDFontType2 with ToUnicode (2-byte glyph IDs, Identity-H).
func TestRender_CustomFont_EmbedsSubset(t *testing.T) {
	system := customSystemTTF(t)
	tmp := filepath.Join(t.TempDir(), "mi.ttf")
	data, err := os.ReadFile(system)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	html := fmt.Sprintf(`<style>@font-face { font-family: Mitipo; src: url(%s); }</style><p style="font-family: Mitipo">Hello</p>`, tmp)
	got, err := RenderHTMLToPDF(html)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{"/Subtype /CIDFontType2", "/FontFile2", "/ToUnicode", "/Identity-H"} {
		if !bytes.Contains(got, []byte(want)) {
			t.Errorf("the PDF does not contain %q", want)
		}
	}
}

func TestRender_InstalledFontFamily_UsesRequestedFace(t *testing.T) {
	fontPath := "/usr/share/fonts/liberation/LiberationSans-Bold.ttf"
	if _, err := os.Stat(fontPath); err != nil {
		t.Skip("Liberation Sans Bold is not installed")
	}

	pdf, err := RenderHTMLToPDF(`<p style="margin:0;font: bold 32pt/0.95 'Liberation Sans',sans-serif">Factura</p>`)
	if err != nil {
		t.Fatalf("RenderHTMLToPDF: %v", err)
	}
	if !bytes.Contains(pdf, []byte("/FontFile2")) {
		t.Fatal("the requested installed font was not embedded; rendering fell back to a base-14 font")
	}
	if !bytes.Contains(pdf, []byte("LiberationSansBold")) {
		t.Fatal("bold text did not select the installed bold face")
	}
}

func TestRender_CustomFontInNestedBlocksRegistersEveryRunFont(t *testing.T) {
	system := customSystemTTF(t)
	tmp := filepath.Join(t.TempDir(), "nested.ttf")
	data, err := os.ReadFile(system)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	html := []byte(`<style>
body { font: 8.5pt Arial; }
header { border-bottom: 1pt solid #000; }
.brand { font-size: 14pt; font-weight: bold; }
.service { font-size: 7pt; }
.recipient { font-size: 13pt; font-weight: bold; }
.postal { font-size: 23pt; font-weight: bold; }
footer { font-size: 7pt; font-style: italic; }
</style>
<div><header><div class="brand">NORTHSTAR LOGISTICS</div><div class="service">PRIORITY DELIVERY</div></header>
<section><small>DELIVER TO</small><div class="recipient">Brithany C.</div><p>Solano Avenue 4-81<br>Remigio Crespo, Azuay<br>Ecuador 010203</p></section>
<div class="postal">010203</div><footer>Example shipment · Replace the barcode before use.</footer></div>`)
	faces := fmt.Sprintf(`<style>@font-face{font-family:Compare;src:url(%s);font-weight:400}@font-face{font-family:Compare;src:url(%s);font-weight:700}@font-face{font-family:Compare;src:url(%s);font-weight:400;font-style:italic}@font-face{font-family:Compare;src:url(%s);font-weight:700;font-style:italic}</style>`, tmp, tmp, tmp, tmp)
	html = []byte(faces + strings.ReplaceAll(string(html), "Arial", "Compare"))
	pdf, err := RenderHTMLToPDF(string(html))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(streamText(t, pdf), "/F0 ") {
		t.Fatal("content references F0, but PDF resources start at F1")
	}
	if !bytes.Contains(pdf, []byte("/F5 ")) || !bytes.Contains(pdf, []byte("/Subtype /CIDFontType2")) {
		t.Fatal("the PDF does not register and embed the font used by nested runs")
	}
}

func TestRender_CustomFont_EmitsShapedGlyphAdvances(t *testing.T) {
	fontPath := "/usr/share/fonts/liberation/LiberationSans-Regular.ttf"
	if _, err := os.Stat(fontPath); err != nil {
		t.Skip("Liberation Sans is not installed")
	}
	tmp := filepath.Join(t.TempDir(), "liberation.ttf")
	data, err := os.ReadFile(fontPath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	html := fmt.Sprintf(`<style>@font-face { font-family: Compare; src: url(%s); }</style><p style="font-family: Compare; font-size: 12pt">AV</p>`, tmp)
	pdf, err := RenderHTMLToPDF(html)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	content := streamText(t, pdf)
	if got := strings.Count(content, " Tm <"); got != 2 {
		t.Errorf("the TTF run must emit a glyph positioned by GID; got %d", got)
	}
	if !strings.Contains(content, "1 0 0 1 79.11328125") {
		t.Error("V position must include the shaped advance of A")
	}
}

func TestRender_CustomFont_LigaturePreservesExtractedText(t *testing.T) {
	fontPath := "/usr/share/fonts/TTF/DejaVuSans.ttf"
	if _, err := os.Stat(fontPath); err != nil {
		t.Skip("DejaVu Sans is not installed")
	}
	tmp := filepath.Join(t.TempDir(), "dejavu.ttf")
	data, err := os.ReadFile(fontPath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	html := fmt.Sprintf(`<style>@font-face{font-family:Compare;src:url(%s)}</style><p style="font-family:Compare">office</p>`, tmp)
	pdf, err := RenderHTMLToPDF(html)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	content := streamText(t, pdf)
	if got := strings.Count(content, " Tm <"); got != 4 {
		t.Errorf("office must be emitted as 4 GIDs with ffi ligature; got %d", got)
	}
	if !bytes.Contains(pdf, []byte("<006600660069>")) {
		t.Error("ToUnicode does not assign the ffi sequence to the ligature GID")
	}
}

func TestRender_CustomFont_UsesDenseSubsetGlyphIDs(t *testing.T) {
	fontPath := "/usr/share/fonts/liberation/LiberationSans-Regular.ttf"
	if _, err := os.Stat(fontPath); err != nil {
		t.Skip("Liberation Sans is not installed")
	}
	tmp := filepath.Join(t.TempDir(), "liberation.ttf")
	data, err := os.ReadFile(fontPath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	html := fmt.Sprintf(`<style>@font-face{font-family:Compare;src:url(%s)}</style><p style="font-family:Compare">AV</p>`, tmp)
	pdf, err := RenderHTMLToPDF(html)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	content := streamText(t, pdf)
	for _, want := range []string{"<0001> Tj", "<0002> Tj"} {
		if !strings.Contains(content, want) {
			t.Errorf("content stream does not use compact GID %s: %s", want, content)
		}
	}
	for _, want := range []string{"/W [1 [", "<0001> <0041>", "<0002> <0056>"} {
		if !bytes.Contains(pdf, []byte(want)) {
			t.Errorf("CIDFont/ToUnicode was not remapped: missing %q", want)
		}
	}
}

func customSystemTTF(t *testing.T) string {
	t.Helper()
	cands, _ := filepath.Glob("/usr/share/fonts/*/*.ttf")
	for _, c := range cands {
		if st, err := os.Stat(c); err == nil && st.Size() > 50000 {
			return c
		}
	}
	t.Skip("without TTF of the system")
	return ""
}
