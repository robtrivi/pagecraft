// Package boxes builds the box tree and resolves the CSS subset used by layout.
package boxes

import (
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/robtrivi/pagecraft/css"
	"github.com/robtrivi/pagecraft/fonts"
	"github.com/robtrivi/pagecraft/images"
	"github.com/robtrivi/pagecraft/svg"
)

// Box stores a node's content, resources, and metadata.
// Resolved CSS and user-agent values are stored in ComputedStyle.
type Box struct {
	Tag          string
	sourcePath   []int
	Style        ComputedStyle
	Text         string
	Image        *images.Raster            // resolved <img> only
	SVG          *svg.Document             // <svg> inline vectorial
	Parts        []Part                    // ordered segments (anonymous text or children)
	Link         string                    // Link is the raw href value from <a>.
	Fonts        map[string]*fonts.Face    // document font registry (root only)
	FontVariants map[string]fonts.Variants // faces by weight and style
	Page         PageConfig                // @page from the first valid rule (root only)
	PageRules    []css.PageRule            // Ordered @page rules for page-context resolution.
	PageFooter   []css.ContentChunk        // @bottom-center (root only)
	Meta         Metadata                  // <title>/meta (root only)
	Children     []*Box
	Parent       *Box // direct structural parent in the source tree
}

// SourcePath returns the box's stable child-index path under the document root.
func (b *Box) SourcePath() []int {
	return append([]int(nil), b.sourcePath...)
}

// GridTrack describes a fixed or fractional grid track.
type GridTrack struct {
	Fixed      float64 // fixed width in points
	Percent    float64 // percentage of the available width
	Fraction   float64 // unidad fr
	Min        float64 // fixed minimum for minmax(..., fr)
	MinPercent float64 // percentage minimum for minmax(..., fr)
	MinAuto    bool    // automatic minimum for fr
}

// DefaultMaxBytes limits each external resource to 10 MiB.
const DefaultMaxBytes = 10 << 20

const maxGridTracks = 10000

// defaultHTTPTimeout is used for downloads.
const defaultHTTPTimeout = 10 * time.Second

// Options configures Build with a base URL, an optional fetch function, and a byte limit.
type Options struct {
	Base     string
	Fetch    func(url string) ([]byte, error)
	MaxBytes int64
}

// Build constructs a box tree from an HTML fragment using default options.
func Build(htmlStr string) (*Box, error) {
	b, _, err := BuildWithOptions(htmlStr, Options{})
	return b, err
}

// BuildWithOptions constructs a box tree and returns warnings for recoverable resource failures.
func BuildWithOptions(htmlStr string, opts Options) (*Box, []string, error) {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return nil, nil, err
	}
	maxBytes := opts.MaxBytes
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBytes
	}
	ld := &loader{
		fetch:    opts.Fetch,
		maxBytes: maxBytes,
		warnings: []string{},
		images:   map[string]loadedImage{},
	}
	if ld.fetch == nil {
		ld.fetch = defaultFetch(maxBytes)
	}
	base := opts.Base
	if href := baseTagHref(doc); href != "" {
		base = resolveURL(base, href)
	}
	ld.base = base
	root := &Box{Tag: "root", Style: ComputedStyle{Display: DisplayBlock}, Fonts: map[string]*fonts.Face{}, FontVariants: map[string]fonts.Variants{}}
	body := findElement(doc, "body")
	if body == nil {
		body = doc
	}
	texts, err := ld.styleTexts(doc)
	if err != nil {
		return nil, ld.warnings, err
	}
	if err := root.loadFonts(texts, ld); err != nil {
		return nil, ld.warnings, err
	}
	root.loadPage(texts)
	root.Meta = readMetadata(doc)
	ss := newStyleSheet(texts)
	if hasDocumentRoot(htmlStr) {
		root.Style.BodyMargin = bodyMargin(ss, body)
	}
	bodyBox, err := buildBox(body, nil, ss, ld, textDecorations{})
	if err != nil {
		return nil, ld.warnings, err
	}
	root.Style.HasBg, root.Style.Bg = bodyBox.Style.HasBg, bodyBox.Style.Bg
	root.Children = append(root.Children, bodyBox.Children...)
	AssignSourcePaths(root)
	return root, ld.warnings, nil
}

// hasDocumentRoot distinguishes complete documents (including an implicit body)
// for HTML fragments. Fragments retain their existing origin without UA margins.
