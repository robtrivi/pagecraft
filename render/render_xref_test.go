package render

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

// XRef consistency: every "N 0 R" reference exists and startxref points to xref.
// Poppler found "xref num 11 not found" when imgBase was offset by +1.
func TestRender_XRef_Consistent(t *testing.T) {
	got, err := RenderHTMLToPDF(`<b>x</b><img src="` + pngURI(t, 4, 2) + `"><a href="https://x.test">y</a>`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := string(got)
	sx := strings.LastIndex(s, "startxref\n")
	var pos int
	if _, err := fmt.Sscanf(s[sx:], "startxref\n%d", &pos); err != nil {
		t.Fatalf("startxref ilegible: %v", err)
	}
	if pos < 0 || pos+4 > len(s) || s[pos:pos+4] != "xref" {
		t.Fatalf("startxref=%d does not point to xref", pos)
	}
	var maxObj int
	if _, err := fmt.Sscanf(s[pos:], "xref\n0 %d", &maxObj); err != nil {
		t.Fatalf("xref ilegible: %v", err)
	}
	for _, m := range regexp.MustCompile(`(\d+) 0 R`).FindAllStringSubmatch(s, -1) {
		var k int
		fmt.Sscanf(m[1], "%d", &k)
		if k < 1 || k > maxObj-1 {
			t.Errorf("reference outside range: %d 0 R (objects: %d)", k, maxObj-1)
		}
		if !strings.Contains(s, fmt.Sprintf("\n%d 0 obj\n", k)) {
			t.Errorf("missing object %d 0 obj", k)
		}
	}
}
