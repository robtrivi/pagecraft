package boxes

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/robtrivi/pagecraft-go/css"
	"github.com/robtrivi/pagecraft-go/fonts"
	"github.com/robtrivi/pagecraft-go/images"
	"github.com/robtrivi/pagecraft-go/svg"
	"golang.org/x/net/html"
)

func resolveImage(the *html.Node, b *Box, ld *loader) error {
	src := getAttr(the, "src")
	if src == "" {
		b.Text = getAttr(the, "alt")
		return nil
	}
	alt := func(err error) error {
		if isRemote(src) {
			ld.warn("img %s: %v (using alt text)", src, err)
			b.Text = getAttr(the, "alt")
			return nil
		}
		return err
	}
	data, err := loadImageBytes(src, ld)
	if err != nil {
		return alt(err)
	}
	if doc, svgErr := svg.Parse(data); svgErr == nil {
		b.SVG = doc
		return nil
	}
	r, err := images.Decode(data)
	if err != nil {
		return alt(fmt.Errorf("boxes: <img>: %w", err))
	}
	b.Image = r
	return nil
}

// isRemote reports whether src uses HTTP(S), rather than a data URI or local path.
func isRemote(src string) bool {
	return strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://")
}

// readMetadata extracts the title and author/description metadata from <head>.
func readMetadata(doc *html.Node) Metadata {
	var m Metadata
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "title":
				if m.Title == "" {
					m.Title = strings.Join(strings.Fields(rawText(n)), " ")
				}
			case "meta":
				name := strings.ToLower(getAttr(n, "name"))
				content := getAttr(n, "content")
				switch name {
				case "author":
					if m.Author == "" {
						m.Author = content
					}
				case "description":
					if m.Description == "" {
						m.Description = content
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return m
}

// loadPage reads the first valid @page rule and its footer from style elements.
func (b *Box) loadPage(texts []string) {
	for _, t := range texts {
		if w, h, m, ok := css.ParsePageRule(t); ok {
			if w > 0 && h > 0 {
				b.Page.Width, b.Page.Height = w, h
			}
			if m >= 0 {
				b.Page.Margin = m
				b.Page.HasMargin = true
			}
			break
		}
	}
	for _, t := range texts {
		if chunks, ok := css.ParsePageFooter(t); ok {
			b.PageFooter = chunks
			return
		}
	}
}

// loadFonts registers @font-face rules from style elements (local file or data URI).
// Part is an ordered segment of anonymous text or a child element.
type Part struct {
	Text            string
	Child           *Box
	ListMarkerShape string
}

// Metadata contains title, author, and description values from <head>.
type Metadata struct {
	Title       string
	Author      string
	Description string
}

// PageConfig stores the @page size and margin in points. Zero values mean no rule was set.
type PageConfig struct {
	Width, Height float64
	Margin        float64
	HasMargin     bool
}

func (b *Box) loadFonts(texts []string, ld *loader) error {
	for _, t := range texts {
		faces, _ := css.ParseFontFaces(t)
		for _, ff := range faces {
			data, err := loadFontBytes(ff.Src, ld)
			if err != nil {
				if isRemote(ff.Src) {
					ld.warn("@font-face %s: %v (skipped)", ff.Family, err)
					continue
				}
				return fmt.Errorf("boxes: @font-face %s: %w", ff.Family, err)
			}
			face, err := fonts.Load(data)
			if err != nil {
				if isRemote(ff.Src) {
					ld.warn("@font-face %s: %v (skipped)", ff.Family, err)
					continue
				}
				return fmt.Errorf("boxes: @font-face %s: %w", ff.Family, err)
			}
			variants := b.FontVariants[ff.Family]
			bold := ff.Weight >= 700
			italic := ff.Style == "italic" || ff.Style == "oblique"
			switch {
			case bold && italic:
				variants.BoldItalic = face
			case bold:
				variants.Bold = face
			case italic:
				variants.Italic = face
			default:
				variants.Regular = face
			}
			b.FontVariants[ff.Family] = variants
			if variants.Regular != nil {
				b.Fonts[ff.Family] = variants.Regular
			} else if b.Fonts[ff.Family] == nil {
				b.Fonts[ff.Family] = face
			}
		}
	}
	return nil
}

// loadFontBytes accepts data URIs, local/relative paths (resolved against base), and remote URLs.
func loadFontBytes(src string, ld *loader) ([]byte, error) {
	if strings.HasPrefix(src, "data:") {
		_, payload, ok := splitDataURI(src)
		if !ok {
			return nil, errors.New("boxes: invalid data URI")
		}
		data, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return nil, fmt.Errorf("boxes: base64: %w", err)
		}
		return data, nil
	}
	if isRemote(src) {
		return ld.fetch(src)
	}
	data, err := os.ReadFile(resolveURL(ld.base, src))
	if err != nil {
		return nil, fmt.Errorf("boxes: file: %w", err)
	}
	return data, nil
}

// isBold reports whether the value is bold, bolder, or a number >= 700.
func isBold(s string) bool {
	if s == "bold" || s == "bolder" {
		return true
	}
	if n, err := strconv.Atoi(s); err == nil && n >= 700 {
		return true
	}
	return false
}

// transformText applies uppercase/lowercase/capitalize.
func transformText(s, mode string) string {
	switch mode {
	case "uppercase":
		return strings.ToUpper(s)
	case "lowercase":
		return strings.ToLower(s)
	case "capitalize":
		out := make([]rune, 0, len(s))
		upperNext := true
		for _, r := range s {
			if unicode.IsSpace(r) {
				upperNext = true
			} else if upperNext {
				r = unicode.ToUpper(r)
				upperNext = false
			}
			out = append(out, r)
		}
		return string(out)
	default:
		return s
	}
}

// firstFamily returns the first family name, without quotes or surrounding spaces.
func firstFamily(s string) string {
	if i := strings.Index(s, ","); i >= 0 {
		s = s[:i]
	}
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			s = s[1 : len(s)-1]
		}
	}
	return strings.TrimSpace(s)
}

// splitDataURI splits "data:<meta>,<payload>".
func splitDataURI(src string) (meta, payload string, ok bool) {
	comma := strings.Index(src, ",")
	if comma < 0 {
		return "", "", false
	}
	return src[len("data:"):comma], src[comma+1:], true
}
func loadImageBytes(src string, ld *loader) ([]byte, error) {
	if strings.HasPrefix(src, "data:") {
		comma := strings.Index(src, ",")
		if comma < 0 {
			return nil, errors.New("boxes: data URI is missing a comma")
		}
		meta, payload := src[len("data:"):comma], src[comma+1:]
		mediatype := meta
		if i := strings.Index(meta, ";"); i >= 0 {
			mediatype = meta[:i]
		}
		if mediatype != "" && mediatype != "image/png" && mediatype != "image/jpeg" && mediatype != "image/svg+xml" {
			return nil, fmt.Errorf("boxes: unsupported MIME type: %s", mediatype)
		}
		if !strings.Contains(meta, ";base64") {
			return nil, errors.New("boxes: non-base64 data URI is not supported")
		}
		data, err := base64.StdEncoding.DecodeString(payload)
		if err != nil {
			return nil, fmt.Errorf("boxes: base64: %w", err)
		}
		return data, nil
	}
	if isRemote(src) {
		return ld.fetch(src)
	}
	data, err := os.ReadFile(resolveURL(ld.base, src))
	if err != nil {
		return nil, fmt.Errorf("boxes: file: %w", err)
	}
	return data, nil
}

// defaultDisplay returns the subset's minimum user-agent display style.
