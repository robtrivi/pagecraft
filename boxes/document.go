package boxes

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/andybalholm/cascadia"
	"github.com/robtrivi/pagecraft/css"
	"github.com/robtrivi/pagecraft/images"
	"github.com/robtrivi/pagecraft/svg"
	"golang.org/x/net/html"
)

func hasDocumentRoot(src string) bool {
	z := html.NewTokenizer(strings.NewReader(src))
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			return false
		}
		if tt == html.DoctypeToken {
			return true
		}
		if tt == html.StartTagToken {
			name, _ := z.TagName()
			if strings.EqualFold(string(name), "html") || strings.EqualFold(string(name), "body") {
				return true
			}
		}
	}
}

// bodyMargin resolves uniform UA/author margins; current layout uses
// a single margin value on all four sides.
func bodyMargin(ss *styleSheet, body *html.Node) float64 {
	decls := ss.cascaded(body)
	inline, _ := css.ParseInlineStyle(getAttr(body, "style"))
	for k, v := range inline {
		decls[k] = v
	}
	top, right, bottom, left := 0.0, 0.0, 0.0, 0.0
	if m, ok := decls["margin"]; ok {
		top, right, bottom, left = expandSides(m)
	}
	for _, side := range []struct {
		name string
		dst  *float64
	}{
		{"margin-top", &top}, {"margin-right", &right},
		{"margin-bottom", &bottom}, {"margin-left", &left},
	} {
		if v, ok := decls[side.name]; ok {
			if n, valid := tryLengthPx(v); valid {
				*side.dst = n
			}
		}
	}
	if top == right && top == bottom && top == left {
		return top
	}
	return 0
}

// loader resolves and downloads external resources.
type loader struct {
	base     string
	fetch    func(url string) ([]byte, error)
	maxBytes int64
	warnings []string
	images   map[string]loadedImage
}

type loadedImage struct {
	raster *images.Raster
	vector *svg.Document
}

func (ld *loader) warn(format string, args ...any) {
	ld.warnings = append(ld.warnings, fmt.Sprintf(format, args...))
}

// defaultFetch downloads HTTP(S) resources with a timeout and size limit; local files are read directly.
func defaultFetch(maxBytes int64) func(string) ([]byte, error) {
	client := &http.Client{Timeout: defaultHTTPTimeout}
	return func(rawurl string) ([]byte, error) {
		if !strings.HasPrefix(rawurl, "http://") && !strings.HasPrefix(rawurl, "https://") {
			data, err := os.ReadFile(rawurl)
			if err != nil {
				return nil, err
			}
			if int64(len(data)) > maxBytes {
				return nil, fmt.Errorf("boxes: exceeds size limit %d: %s", maxBytes, rawurl)
			}
			return data, nil
		}
		resp, err := client.Get(rawurl)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("boxes: HTTP %d: %s", resp.StatusCode, rawurl)
		}
		data, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
		if err != nil {
			return nil, err
		}
		if int64(len(data)) > maxBytes {
			return nil, fmt.Errorf("boxes: exceeds size limit %d: %s", maxBytes, rawurl)
		}
		return data, nil
	}
}

// baseTagHref reads the document's base URL element.
func baseTagHref(doc *html.Node) string {
	if base := findElement(doc, "base"); base != nil {
		return getAttr(base, "href")
	}
	return ""
}

// resolveURL joins a base and reference (URLs or local paths).
func resolveURL(base, ref string) string {
	if ref == "" {
		return ""
	}
	if strings.Contains(ref, "://") || strings.HasPrefix(ref, "data:") || strings.HasPrefix(ref, "file:") {
		return ref
	}
	if base == "" {
		return ref
	}
	if strings.HasPrefix(base, "http://") || strings.HasPrefix(base, "https://") {
		bu, err := url.Parse(base)
		if err != nil {
			return ref
		}
		ru, err := url.Parse(ref)
		if err != nil {
			return ref
		}
		return bu.ResolveReference(ru).String()
	}
	base = strings.TrimPrefix(base, "file://")
	if st, err := os.Stat(base); err == nil && st.IsDir() {
		return filepath.Join(base, ref)
	}
	return filepath.Join(filepath.Dir(base), ref)
}

// styleRule is a compiled rule with its global source order.
type styleRule struct {
	decls css.Declarations
	group cascadia.SelectorGroup
	order int
}

// styleSheet collects <style> rules in document order (v1: not @media,
// user-agent stylesheet, or !important support).
type styleSheet struct {
	rules []styleRule
}

func newStyleSheet(texts []string) *styleSheet {
	ss := &styleSheet{}
	order := 0
	for _, t := range append([]string{css.UAStylesheet}, texts...) {
		rules, _ := css.ParseStylesheet(t)
		for _, r := range rules {
			decls := css.Declarations{}
			for property, value := range r.Decls {
				decls[property] = value
			}
			expandFontShorthand(decls)
			g, err := cascadia.ParseGroup(r.Selector)
			if err != nil {
				continue
			}
			ss.rules = append(ss.rules, styleRule{decls: decls, group: g, order: order})
			order++
		}
	}
	return ss
}

// cascaded returns declarations from rules matching the element, in
// application order (specificity, then source order).
func (ss *styleSheet) cascaded(the *html.Node) css.Declarations {
	type hit struct {
		spec  cascadia.Specificity
		order int
		decls css.Declarations
	}
	var hits []hit
	for _, r := range ss.rules {
		matched := false
		var best cascadia.Specificity
		for _, sel := range r.group {
			if sel.Match(the) {
				if !matched || best.Less(sel.Specificity()) {
					best = sel.Specificity()
				}
				matched = true
			}
		}
		if matched {
			hits = append(hits, hit{best, r.order, r.decls})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].spec != hits[j].spec {
			return hits[i].spec.Less(hits[j].spec)
		}
		return hits[i].order < hits[j].order
	})
	out := css.Declarations{}
	for _, h := range hits {
		for k, v := range expandCascadeShorthands(h.decls) {
			out[k] = v
		}
	}
	return out
}

// expandCascadeShorthands applies box-model shorthands to their longhands
// within each cascade step. This lets a more specific shorthand replace a
// less specific longhand while still allowing a more specific longhand to win.
func expandCascadeShorthands(decls css.Declarations) css.Declarations {
	expanded := make(css.Declarations, len(decls)+8)
	for property, value := range decls {
		expanded[property] = value
	}
	for _, shorthand := range []struct {
		name string
	}{
		{name: "margin"},
		{name: "padding"},
	} {
		value, ok := decls[shorthand.name]
		if !ok {
			continue
		}
		sides, valid := expandMarginTokens(value)
		if !valid {
			continue
		}
		prefix := shorthand.name + "-"
		for index, side := range [...]string{"top", "right", "bottom", "left"} {
			expanded[prefix+side] = sides[index]
		}
	}
	// Keep explicit longhands in the same declaration block above the
	// shorthand expansion, consistent with the existing package contract.
	for property, value := range decls {
		if strings.HasPrefix(property, "margin-") || strings.HasPrefix(property, "padding-") {
			expanded[property] = value
		}
	}
	return expanded
}

// styleTexts collects <style> and <link rel=stylesheet> rules in document order.
// A missing local file returns an error; a failed remote request adds a warning and continues.
func (ld *loader) styleTexts(doc *html.Node) ([]string, error) {
	var texts []string
	var walk func(*html.Node) error
	walk = func(n *html.Node) error {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "style":
				texts = append(texts, rawText(n))
			case "link":
				if rel := strings.ToLower(getAttr(n, "rel")); strings.Contains(rel, "stylesheet") {
					href := getAttr(n, "href")
					data, err := ld.loadCSS(href)
					if err != nil {
						if href != "" && isRemote(href) {
							ld.warn("stylesheet %s: %v (skipped)", href, err)
							break
						}
						return fmt.Errorf("boxes: stylesheet: %w", err)
					}
					texts = append(texts, string(data))
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if err := walk(c); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(doc); err != nil {
		return nil, err
	}
	return texts, nil
}

// loadCSS loads CSS from a local/relative file or a remote resource.
func (ld *loader) loadCSS(href string) ([]byte, error) {
	if href == "" {
		return nil, errors.New("without href")
	}
	if isRemote(href) {
		return ld.fetch(href)
	}
	return os.ReadFile(resolveURL(ld.base, href))
}

func rawText(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return sb.String()
}

// inheritable is the set of properties inherited in v1.
