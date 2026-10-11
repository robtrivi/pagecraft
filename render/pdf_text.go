package render

import (
	"fmt"
	"strings"

	"github.com/robtrivi/pagecraft/text"
)

// drawText emits the PDF text operator, including AFM kerning where available.
func drawText(line string, bold bool) string {
	type run struct {
		s    string
		kern float64 // Adjustment applied after the run.
	}
	var runs []run
	var cur strings.Builder
	var prev rune
	hasKern := false
	for i, r := range line {
		if i > 0 && !bold {
			if k := text.Kern(prev, r); k != 0 {
				runs = append(runs, run{cur.String(), k})
				cur.Reset()
				hasKern = true
			}
		}
		cur.WriteRune(r)
		prev = r
	}
	if !hasKern {
		return encodePDFString(line) + " Tj"
	}
	runs = append(runs, run{cur.String(), 0})
	var tj strings.Builder
	tj.WriteByte('[')
	for _, rn := range runs {
		tj.WriteString(encodePDFString(rn.s))
		if rn.kern != 0 {
			fmt.Fprintf(&tj, "%g", -rn.kern)
		}
	}
	tj.WriteString("] TJ")
	return tj.String()
}

// encodePDFString encodes text for Tj as a WinAnsi literal: ASCII
// Printable bytes are emitted as-is (escaping backslashes and parentheses);
// other representable bytes are emitted in octal.
// Runes without a glyph become '?'.
func encodePDFString(s string) string {
	var sb strings.Builder
	sb.WriteByte('(')
	for _, r := range s {
		b, ok := text.WinAnsiByte(r)
		if !ok {
			b = '?'
		}
		switch {
		case b == '\\' || b == '(' || b == ')':
			sb.WriteByte('\\')
			sb.WriteByte(b)
		case 0x20 <= b && b <= 0x7E:
			sb.WriteByte(b)
		default:
			fmt.Fprintf(&sb, "\\%03o", b)
		}
	}
	sb.WriteByte(')')
	return sb.String()
}

// buildPDFFromPages builds a PDF 1.4 with a valid xref table, catalog, page
// tree, page objects, content streams, and the shared Helvetica font. PDF
// coordinates use a bottom-left origin.
