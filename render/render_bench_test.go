package render

import (
	"strings"
	"testing"
)

// Typical document: headings, paragraphs, table, list, and link.
func typicalDoc() string {
	return `<h1>Title</h1>` +
		strings.Repeat("<p>Hello <b>world</b> with enough text for several lines of content.</p>", 20) +
		`<table><tr><td>A</td><td>B</td></tr><tr><td>C</td><td>D</td></tr></table>` +
		`<ul><li>one</li><li>two</li></ul>` +
		`<p>see <a href="https://x.test">here</a> end</p>`
}

func BenchmarkRender_TypicalDoc(b *testing.B) {
	html := typicalDoc()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := RenderHTMLToPDF(html); err != nil {
			b.Fatalf("error: %v", err)
		}
	}
}

func BenchmarkRender_LargeDoc(b *testing.B) {
	html := strings.Repeat("<p>Hello world with enough text.</p>", 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := RenderHTMLToPDF(html); err != nil {
			b.Fatalf("error: %v", err)
		}
	}
}
