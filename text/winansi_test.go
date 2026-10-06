package text

import (
	"testing"
)

// WinAnsiByte maps a rune to its WinAnsi (cp1252) byte. v0.5 regression coverage:
// without /Encoding, readers use StandardEncoding and accented characters are
// wrong; WinAnsiEncoding with literal bytes renders them correctly.
func TestText_WinAnsiByte_Maps(t *testing.T) {
	cases := []struct {
		r    rune
		want byte
	}{
		{'é', 0xE9}, {'ñ', 0xF1}, {'¿', 0xBF}, {'—', 0x97}, {'“', 0x93},
	}
	for _, tc := range cases {
		if got, ok := WinAnsiByte(tc.r); !ok || got != tc.want {
			t.Errorf("WinAnsiByte(%q): got %02X,%v want %02X,true", tc.r, got, ok, tc.want)
		}
	}
	for _, r := range []rune{'€', '文', 0x80} {
		if _, ok := WinAnsiByte(r); ok {
			t.Errorf("WinAnsiByte(%q): expected !ok", r)
		}
	}
}
