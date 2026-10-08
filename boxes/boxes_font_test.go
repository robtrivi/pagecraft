package boxes

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Font-face step 2b: @font-face registers the family on the root (from
// local file), and font-family selects it per box.
func TestBoxes_FontFace_RegistersFamily(t *testing.T) {
	system := systemTTF(t)
	tmp := filepath.Join(t.TempDir(), "mi.ttf")
	data, err := os.ReadFile(system)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	html := fmt.Sprintf(`<style>@font-face { font-family: Mitipo; src: url(%s); }</style><p style="font-family: Mitipo">x</p>`, tmp)
	root, err := Build(html)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root.Fonts["Mitipo"] == nil {
		t.Fatal("font family not registered at the root")
	}
	if root.Children[0].Style.FontFamily != "Mitipo" {
		t.Errorf("family: got %q, want Mitipo", root.Children[0].Style.FontFamily)
	}
}

func TestBoxes_FontFace_MissingFile_ReturnsError(t *testing.T) {
	_, err := Build(`<style>@font-face { font-family: X; src: url(/no/such-font.ttf); }</style><p>x</p>`)
	if err == nil {
		t.Error("expected an error for a nonexistent source")
	}
}

func TestBoxes_FontFace_PreservesWeightVariants(t *testing.T) {
	path := systemTTF(t)
	html := fmt.Sprintf(`<style>
		@font-face { font-family: Compare; src: url(%s); font-weight: 400; }
		@font-face { font-family: Compare; src: url(%s); font-weight: 700; }
		</style><p style="font-family: Compare">x</p>`, path, path)
	root, err := Build(html)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	v := root.FontVariants["Compare"]
	if v.Regular == nil || v.Bold == nil || v.Regular == v.Bold {
		t.Fatalf("variable fonts were registered incorrectly: %+v", v)
	}
	if root.Fonts["Compare"] != v.Regular {
		t.Fatal("compatible family registry must point to the regular face")
	}
}

// systemTTF locates a system TTF (skips if unavailable).
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

func TestBoxes_FontShorthandOverridesUAScaledHeading(t *testing.T) {
	cases := []struct {
		name, html string
	}{
		{"stylesheet", `<style>h1{font:bold 28pt Arial}</style><h1>x</h1>`},
		{"inline", `<h1 style="font:bold 28pt Arial">x</h1>`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, err := Build(tc.html)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			h1 := root.Children[0]
			if h1.Style.FontSize != 28 || h1.Style.FontFamily != "Arial" || !h1.Style.Bold {
				t.Errorf("font shorthand=%v/%q/bold=%v, want 28pt/Arial/bold", h1.Style.FontSize, h1.Style.FontFamily, h1.Style.Bold)
			}
		})
	}
}
