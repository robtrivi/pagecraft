package text

import (
	"testing"
)

// WinAnsi is supported; controls, CJK, emoji, and missing euro glyphs are not.
func TestText_Supports_Coverage(t *testing.T) {
	for _, r := range []rune{'A', 'é', 'ñ', '–', '“', ' '} {
		if !Supports(r) {
			t.Errorf("Supports(%q): got false, want true", r)
		}
	}
	for _, r := range []rune{'€', '文', 0x80, 0x9F, 0x1F600} {
		if Supports(r) {
			t.Errorf("Supports(%q U+%04X): got true, want false", r, r)
		}
	}
}
