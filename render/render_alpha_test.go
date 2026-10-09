package render

import (
	"bytes"
	"strings"
	"testing"
)

// Opacity step 2: ExtGState with /CA and isolated painting using q/Q.
func TestRender_Opacity_EmitsExtGState(t *testing.T) {
	got, err := RenderHTMLToPDF(`<p style="opacity: 0.5">x</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := streamText(t, got)
	for _, want := range []string{"/ExtGState", "/CA 0.5", "/GS", " gs", "\nQ\n"} {
		if !bytes.Contains(got, []byte(want)) && !strings.Contains(text, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestRender_OpacityOnAtomicBlockUsesTransparencyGroup(t *testing.T) {
	got, err := RenderHTMLToPDF(`<div style="opacity: 0.5; background-color: rgba(255, 0, 0, 0.5)">text</div>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Contains(got, []byte("/Subtype /Form")) || !bytes.Contains(got, []byte("/Group << /S /Transparency /CS /DeviceRGB /I true >>")) {
		t.Fatal("a block with opacity must be painted in a transparency Form XObject")
	}
	text := streamText(t, got)
	if !strings.Contains(text, "/GS") || !strings.Contains(text, " gs") || !strings.Contains(text, "/Fm1 Do") {
		t.Errorf("group opacity must be applied when its form is invoked: %q", text)
	}
}

func TestRender_OpacityWrapperGroupsAllChildrenOncePerPage(t *testing.T) {
	paragraph := `<p>` + strings.Repeat("word ", 35) + `</p>`
	html := `<div style="opacity: 0.5; background-color: #ff0000; padding: 5px">` + strings.Repeat(paragraph, 20) + `</div>`
	got, err := RenderHTMLToPDF(html)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	groups := strings.Count(string(got), "/Group << /S /Transparency /CS /DeviceRGB /I true >>")
	if groups < 2 {
		t.Fatalf("transparency groups=%d; the wrapper is fragmented and each fragment needs its own composition", groups)
	}
	content := streamText(t, got)
	if !strings.Contains(content, "/Fm1 Do") || !strings.Contains(content, "/Fm2 Do") {
		t.Errorf("each page must invoke its wrapper group: %q", content)
	}
	if got := strings.Count(content, " gs"); got < 2 {
		t.Errorf("alpha applications=%d; one or more fragments were not composed", got)
	}
	assertPDFHasWordCoordinates(t, got)
	if !strings.Contains(content, "(word)") {
		t.Error("wrapper text must appear inside the PDF content")
	}
}

func TestRender_NestedOpacityWrappersCreateNestedTransparencyGroups(t *testing.T) {
	html := `<div style="opacity: 0.5; background-color: #ff0000; padding: 5px"><div style="opacity: 0.25; background-color: #0000ff; padding: 2px"><p>interior</p></div><p>exterior</p></div>`
	got, err := RenderHTMLToPDF(html)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	groups := strings.Count(string(got), "/Group << /S /Transparency /CS /DeviceRGB /I true >>")
	if groups != 2 {
		t.Fatalf("nested transparency groups: got %d, want 2", groups)
	}
	content := streamText(t, got)
	if strings.Count(content, " gs") != 2 || !strings.Contains(content, "inter") || !strings.Contains(content, "xter") {
		t.Errorf("the two levels must preserve their alpha and text: %q", content)
	}
}

func TestRender_RGBA_TextKeepsColor(t *testing.T) {
	got, err := RenderHTMLToPDF(`<p style="color: rgba(255, 0, 0, 0.5)">x</p>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := streamText(t, got)
	if !strings.Contains(text, "1 0 0 rg") {
		t.Error("red must be preserved")
	}
	if !strings.Contains(text, "/GS") {
		t.Error("alpha must be set in ExtGState")
	}
}
