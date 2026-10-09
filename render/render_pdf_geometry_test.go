package render

import (
	"bytes"
	"encoding/xml"
	"io"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

type pdfWord struct {
	Text                string
	X, Y, Width, Height float64
}

type bboxWord struct {
	XMin float64 `xml:"xMin,attr"`
	YMin float64 `xml:"yMin,attr"`
	XMax float64 `xml:"xMax,attr"`
	YMax float64 `xml:"yMax,attr"`
	Text string  `xml:",chardata"`
}

func assertPDFHasWordCoordinates(t *testing.T, pdf []byte) []pdfWord {
	t.Helper()
	words := wordsFromPDF(t, pdf)
	if len(words) == 0 {
		t.Fatal("the PDF contains no words with coordinates")
	}
	for i, word := range words {
		if math.IsNaN(word.X) || math.IsNaN(word.Y) || math.IsInf(word.X, 0) || math.IsInf(word.Y, 0) || word.Width <= 0 || word.Height <= 0 {
			t.Errorf("word[%d] %q has invalid bounds: %+v", i, word.Text, word)
		}
	}
	return words
}

func hasPDFWord(words []pdfWord, phrase ...string) bool {
	for start := 0; start+len(phrase) <= len(words); start++ {
		match := true
		for offset, part := range phrase {
			if !strings.Contains(strings.ToLower(words[start+offset].Text), strings.ToLower(part)) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func wordsFromPDF(t *testing.T, data []byte) []pdfWord {
	t.Helper()
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Skip("pdftotext is required to inspect PDF word coordinates")
	}
	f, err := os.CreateTemp(t.TempDir(), "geometry-*.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("pdftotext", "-bbox-layout", f.Name(), "-").Output()
	if err != nil {
		t.Fatalf("pdftotext: %v", err)
	}
	var result []pdfWord
	pageOffset := 0.0
	decoder := xml.NewDecoder(bytes.NewReader(out))
	var pageHeight float64
	var current *bboxWord
	var wordText strings.Builder
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("parse bbox: %v", err)
		}
		switch value := token.(type) {
		case xml.StartElement:
			if value.Name.Local == "page" {
				for _, attr := range value.Attr {
					if attr.Name.Local == "height" {
						pageHeight, _ = strconv.ParseFloat(attr.Value, 64)
					}
				}
			}
			if value.Name.Local == "word" {
				current = &bboxWord{}
				for _, attr := range value.Attr {
					n, _ := strconv.ParseFloat(attr.Value, 64)
					switch attr.Name.Local {
					case "xMin":
						current.XMin = n
					case "yMin":
						current.YMin = n
					case "xMax":
						current.XMax = n
					case "yMax":
						current.YMax = n
					}
				}
				wordText.Reset()
			}
		case xml.CharData:
			if current != nil {
				wordText.Write([]byte(value))
			}
		case xml.EndElement:
			if value.Name.Local == "word" && current != nil {
				text := strings.Join(strings.Fields(wordText.String()), " ")
				if text != "" {
					result = append(result, pdfWord{Text: text, X: current.XMin, Y: pageOffset + current.YMin, Width: current.XMax - current.XMin, Height: current.YMax - current.YMin})
				}
				current = nil
			}
			if value.Name.Local == "page" {
				pageOffset += pageHeight
			}
		}
	}
	return result
}
