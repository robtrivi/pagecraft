package render

import (
	"bytes"
	"compress/zlib"
	"io"
	"strings"
	"testing"
)

// Flate step 1: content streams are compressed; content tests use streamText
// to decompress them, while structural tests continue to inspect raw streams.
func TestRender_ContentStreams_Compressed(t *testing.T) {
	got, err := RenderHTMLToPDF("<p>Hello</p>")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Contains(got, []byte("/Filter /FlateDecode")) {
		t.Fatal("without Flate in streams")
	}
	text := streamText(t, got)
	if !strings.Contains(text, "(Hello)") {
		t.Errorf("inflated stream does not contain the text: %q", text)
	}
}

func TestRender_FlateWriterReuse_PreservesCompressedBytes(t *testing.T) {
	w := takeFlateWriter()
	putFlateWriter(w)
	reused := takeFlateWriter()
	if reused != w {
		t.Fatal("compressed stream writer was not reused")
	}
	putFlateWriter(reused)

	for _, raw := range [][]byte{[]byte("first paragraph"), bytes.Repeat([]byte("Latin text "), 200)} {
		got := flateBytes(raw)
		var want bytes.Buffer
		standard := zlib.NewWriter(&want)
		if _, err := standard.Write(raw); err != nil {
			t.Fatal(err)
		}
		if err := standard.Close(); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want.Bytes()) {
			t.Fatal("reused writer changed the compressed stream bytes")
		}
	}
}

// streamText concatenates the decompressed content streams from the PDF.
func streamText(t *testing.T, pdf []byte) string {
	t.Helper()
	var sb strings.Builder
	rest := pdf
	for {
		i := bytes.Index(rest, []byte("stream\n"))
		if i < 0 {
			break
		}
		rest = rest[i+len("stream\n"):]
		j := bytes.Index(rest, []byte("\nendstream"))
		if j < 0 {
			break
		}
		zr, err := zlib.NewReader(bytes.NewReader(rest[:j]))
		if err != nil {
			rest = rest[j+len("\nendstream"):]
			continue
		}
		raw, err := io.ReadAll(zr)
		zr.Close()
		if err != nil {
			t.Fatalf("inflate: %v", err)
		}
		sb.Write(raw)
		rest = rest[j+len("\nendstream"):]
	}
	return sb.String()
}
