// Package text provides advance and kerning metrics for the base-14 fonts.
package text

// advances: rune positions in thousandths of an em. Subset: ASCII, Latin-1, and extras
// cp1252 glyphs. Unknown glyphs use an advance of 500.
var advances = map[rune]float64{
	' ': 278, '!': 278, '"': 355, '#': 556, '$': 556, '%': 889, '&': 667, '\'': 222,
	'(': 333, ')': 333, '*': 389, '+': 584, ',': 278, '-': 333, '.': 278, '/': 278,
	'0': 556, '1': 556, '2': 556, '3': 556, '4': 556, '5': 556, '6': 556, '7': 556, '8': 556, '9': 556,
	':': 278, ';': 278, '<': 584, '=': 584, '>': 584, '?': 556, '@': 1015,
	'A': 667, 'B': 667, 'C': 722, 'D': 722, 'E': 667, 'F': 611, 'G': 778, 'H': 722,
	'I': 278, 'J': 500, 'K': 667, 'L': 556, 'M': 833, 'N': 722, 'O': 778, 'P': 667,
	'Q': 778, 'R': 722, 'S': 667, 'T': 611, 'U': 722, 'V': 667, 'W': 944, 'X': 667,
	'Y': 667, 'Z': 611,
	'[': 278, '\\': 278, ']': 278, '^': 469, '_': 556, '`': 222,
	'a': 556, 'b': 556, 'c': 500, 'd': 556, 'e': 556, 'f': 278, 'g': 556, 'h': 556,
	'i': 222, 'j': 222, 'k': 500, 'l': 222, 'm': 833, 'n': 556, 'o': 556, 'p': 556,
	'q': 556, 'r': 333, 's': 500, 't': 278, 'u': 556, 'v': 500, 'w': 722, 'x': 500,
	'y': 500, 'z': 500,
	'{': 334, '|': 260, '}': 334, '~': 584,
	// Latin-1.
	'¡': 333, '¢': 556, '£': 556, '¤': 556, '¥': 556, '¦': 260, '§': 556, '¨': 333,
	'©': 737, 'ª': 370, '«': 556, '¬': 584, '®': 737, '¯': 333,
	'°': 400, '±': 584, '²': 333, '³': 333, '´': 333, 'µ': 556, '¶': 537, '·': 278,
	'¸': 333, '¹': 333, 'º': 365, '»': 556, '¼': 834, '½': 834, '¾': 834, '¿': 611,
	'À': 667, 'Á': 667, 'Â': 667, 'Ã': 667, 'Ä': 667, 'Å': 667, 'Æ': 1000, 'Ç': 722,
	'È': 667, 'É': 667, 'Ê': 667, 'Ë': 667, 'Ì': 278, 'Í': 278, 'Î': 278, 'Ï': 278,
	'Ð': 722, 'Ñ': 722, 'Ò': 778, 'Ó': 778, 'Ô': 778, 'Õ': 778, 'Ö': 778, '×': 584,
	'Ø': 778, 'Ù': 722, 'Ú': 722, 'Û': 722, 'Ü': 722, 'Ý': 667, 'Þ': 667, 'ß': 611,
	'à': 556, 'á': 556, 'â': 556, 'ã': 556, 'ä': 556, 'å': 556, 'æ': 889, 'ç': 500,
	'è': 556, 'é': 556, 'ê': 556, 'ë': 556, 'ì': 278, 'í': 278, 'î': 278, 'ï': 278,
	'ð': 556, 'ñ': 556, 'ò': 556, 'ó': 556, 'ô': 556, 'õ': 556, 'ö': 556, '÷': 584,
	'ø': 611, 'ù': 556, 'ú': 556, 'û': 556, 'ü': 556, 'ý': 500, 'þ': 556, 'ÿ': 500,
	// Additional cp1252 glyphs available in Helvetica.
	'Œ': 1000, 'œ': 944, 'Š': 667, 'š': 500, 'Ÿ': 667, 'Ž': 611, 'ž': 500, 'ƒ': 556,
	'–': 556, '—': 1000, '‘': 222, '’': 222, '‚': 222, '“': 333, '”': 333, '„': 333,
	'†': 556, '‡': 556, '•': 350, '…': 1000, '‰': 1000, '™': 1000,
}

// cp1252Bytes maps extra WinAnsi characters above U+00FF to bytes.
var cp1252Bytes = map[rune]byte{
	'‚': 0x82, 'ƒ': 0x83, '„': 0x84, '…': 0x85, '†': 0x86, '‡': 0x87,
	'•': 0x95, '–': 0x96, '—': 0x97, '™': 0x99,
	'‘': 0x91, '’': 0x92, '“': 0x93, '”': 0x94,
	'Œ': 0x8C, 'œ': 0x9C, 'Š': 0x8A, 'š': 0x9A, 'Ž': 0x8E, 'ž': 0x9E, 'Ÿ': 0x9F,
}

// WinAnsiByte maps a rune to its WinAnsi byte and reports whether it is representable.
func WinAnsiByte(r rune) (byte, bool) {
	if r < 0x80 {
		return byte(r), true
	}
	if 0xA0 <= r && r <= 0xFF {
		if _, ok := advances[r]; ok {
			return byte(r), true
		}
		return 0, false
	}
	if b, ok := cp1252Bytes[r]; ok {
		return b, true
	}
	return 0, false
}

// Advance returns the glyph advance in thousandths of an em, or 500 if it is unknown.
func Advance(r rune) float64 {
	if w, ok := advances[r]; ok {
		return w
	}
	return 500
}

// Measure sums glyph advances and kerning, then scales the result to points.
func Measure(s string, size float64) float64 {
	total := 0.0
	var prev rune
	for i, r := range s {
		if i > 0 {
			total += Kern(prev, r)
		}
		total += Advance(r)
		prev = r
	}
	return total / 1000 * size
}

// Supports reports whether Helvetica/WinAnsi can represent the rune.
func Supports(r rune) bool {
	_, ok := WinAnsiByte(r)
	return ok
}
