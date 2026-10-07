package svg

import (
	"encoding/xml"
	"sort"
	"strconv"
	"strings"

	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"

	"github.com/robtrivi/pagecraft-go/css"
)

// Shape contains the shape's geometry and resolved paint attributes.
type Shape struct {
	Ops       []Op
	Text      string
	TextX     float64
	TextY     float64
	TextSize  float64
	HasFill   bool
	Fill      [4]float64
	HasStroke bool
	Stroke    [4]float64
	Width     float64 // stroke-width
	EvenOdd   bool    // fill-rule
}

// Document contains an SVG document's intrinsic size and shapes.
type Document struct {
	Width, Height       float64
	ViewBox             [4]float64
	PreserveAspectRatio string
	Shapes              []Shape
}

// Parse converts SVG into a document and ignores unknown elements.
func Parse(data []byte) (*Document, error) {
	var raw struct {
		XMLName xml.Name `xml:"svg"`
		Width   string   `xml:"width,attr"`
		Height  string   `xml:"height,attr"`
		ViewBox string   `xml:"viewBox,attr"`
		Aspect  string   `xml:"preserveAspectRatio,attr"`
	}
	// First pass: read the root element to determine the size.
	if err := xml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	doc := &Document{}
	doc.PreserveAspectRatio = raw.Aspect
	vb := parseViewBox(raw.ViewBox)
	doc.ViewBox = vb
	w, wOk := parseSize(raw.Width)
	h, hOk := parseSize(raw.Height)
	switch {
	case wOk && hOk:
		doc.Width, doc.Height = w, h
	case vb[2] > 0 && vb[3] > 0:
		doc.Width, doc.Height = vb[2], vb[3]
	default:
		doc.Width, doc.Height = 300, 150 // default replaced element (spec)
	}
	// Second pass: nodes.
	var root xmlNode
	if err := xml.Unmarshal(data, &root); err != nil {
		return nil, err
	}
	rules, err := svgStylesheets(root)
	if err != nil {
		return nil, err
	}
	shapes, err := collectShapes(root, toHTMLNode(root), inherited{}, IdentityTransform(), rules)
	if err != nil {
		return nil, err
	}
	doc.Shapes = shapes
	return doc, nil
}

// inherited carries paint properties from the ancestor.
type inherited struct {
	fill, stroke                       string
	fillSet, strokeSet                 bool
	width                              float64
	widthSet                           bool
	evenOdd                            bool
	opacity, fillOp, strokeOp          float64
	hasOpacity, hasFillOp, hasStrokeOp bool
}

type xmlNode struct {
	XMLName  xml.Name
	Attrs    []xml.Attr `xml:",any,attr"`
	Children []xmlNode  `xml:",any"`
	Text     string     `xml:",chardata"`
}

func attr(n xmlNode, key string) string {
	for _, a := range n.Attrs {
		if a.Name.Local == key {
			return a.Value
		}
	}
	return ""
}

type svgStyleRule struct {
	group cascadia.SelectorGroup
	decls css.Declarations
	order int
}

func svgStylesheets(root xmlNode) ([]svgStyleRule, error) {
	var rules []svgStyleRule
	order := 0
	var walk func(xmlNode) error
	walk = func(n xmlNode) error {
		if n.XMLName.Local == "style" {
			parsed, err := css.ParseStylesheet(n.Text)
			if err != nil {
				return err
			}
			for _, rule := range parsed {
				group, err := cascadia.ParseGroup(rule.Selector)
				if err != nil {
					continue
				}
				rules = append(rules, svgStyleRule{group: group, decls: rule.Decls, order: order})
				order++
			}
		}
		for _, child := range n.Children {
			if child.XMLName.Local != "" {
				if err := walk(child); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return rules, walk(root)
}

func toHTMLNode(n xmlNode) *html.Node {
	if n.XMLName.Local == "" {
		return nil
	}
	result := &html.Node{Type: html.ElementNode, Data: strings.ToLower(n.XMLName.Local)}
	for _, a := range n.Attrs {
		result.Attr = append(result.Attr, html.Attribute{Key: a.Name.Local, Val: a.Value})
	}
	for _, child := range n.Children {
		converted := toHTMLNode(child)
		if converted == nil {
			continue
		}
		converted.Parent = result
		if result.LastChild == nil {
			result.FirstChild = converted
		} else {
			result.LastChild.NextSibling = converted
			converted.PrevSibling = result.LastChild
		}
		result.LastChild = converted
	}
	return result
}

func resolvedSVGStyle(n xmlNode, node *html.Node, rules []svgStyleRule) css.Declarations {
	decls := css.Declarations{}
	for _, a := range n.Attrs {
		switch a.Name.Local {
		case "fill", "stroke", "stroke-width", "fill-rule", "opacity", "fill-opacity", "stroke-opacity", "transform", "font-size":
			decls[a.Name.Local] = a.Value
		}
	}
	type hit struct {
		spec  cascadia.Specificity
		order int
		decls css.Declarations
	}
	var hits []hit
	for _, rule := range rules {
		matched := false
		var best cascadia.Specificity
		for _, selector := range rule.group {
			if selector.Match(node) {
				spec := selector.Specificity()
				if !matched || best.Less(spec) {
					best = spec
				}
				matched = true
			}
		}
		if matched {
			hits = append(hits, hit{best, rule.order, rule.decls})
		}
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].spec != hits[j].spec {
			return hits[i].spec.Less(hits[j].spec)
		}
		return hits[i].order < hits[j].order
	})
	for _, h := range hits {
		for k, v := range h.decls {
			decls[k] = v
		}
	}
	inline, _ := css.ParseInlineStyle(attr(n, "style"))
	for k, v := range inline {
		decls[k] = v
	}
	return decls
}

// collectShapes walks the tree while inheriting paint properties.
func collectShapes(n xmlNode, htmlNode *html.Node, inh inherited, parent Matrix, rules []svgStyleRule) ([]Shape, error) {
	style := resolvedSVGStyle(n, htmlNode, rules)
	inh = inheritAttrs(inh, style)
	transform := parent
	if value := style["transform"]; value != "" {
		local, err := ParseTransform(value)
		if err != nil {
			return nil, err
		}
		transform = parent.Multiply(local)
	}
	var out []Shape
	if n.XMLName.Local == "text" {
		value := strings.TrimSpace(n.Text)
		if value != "" {
			x, _ := parseNumber(attr(n, "x"))
			y, _ := parseNumber(attr(n, "y"))
			x, y = transform.Apply(x, y)
			fontSize := 16.0
			if size, ok := parseNumber(style["font-size"]); ok && size > 0 {
				fontSize = size
			}
			shape := resolvePaint(style, inh, nil)
			shape.Text, shape.TextX, shape.TextY, shape.TextSize = value, x, y, fontSize
			out = append(out, shape)
		}
	} else if ops, ok := shapeOps(n); ok {
		ops = TransformOps(ops, transform)
		out = append(out, resolvePaint(style, inh, ops))
	}
	htmlChild := htmlNode.FirstChild
	for _, c := range n.Children {
		// Avoid processing text/character data twice; only tagged elements qualify.
		if c.XMLName.Local == "" {
			continue
		}
		children, err := collectShapes(c, htmlChild, inh, transform, rules)
		if err != nil {
			return nil, err
		}
		out = append(out, children...)
		htmlChild = htmlChild.NextSibling
	}
	return out, nil
}

// transformOps applies a matrix to every point; arcs are converted to cubic
// curves first so nonuniform scales and skews are supported.
// TransformOps returns a transformed copy of the operations.
func TransformOps(ops []Op, m Matrix) []Op {
	var expanded []Op
	var x, y float64
	for _, op := range ops {
		if op.Kind == OpArc && len(op.Pts) == 7 {
			p := op.Pts
			curves := ArcToBezier(x, y, p[0], p[1], p[2], p[3], p[4], p[5], p[6])
			expanded = append(expanded, curves...)
			x, y = p[5], p[6]
			continue
		}
		expanded = append(expanded, op)
		switch op.Kind {
		case OpMove, OpLine:
			if len(op.Pts) >= 2 {
				x, y = op.Pts[0], op.Pts[1]
			}
		case OpCurve:
			if len(op.Pts) >= 6 {
				x, y = op.Pts[4], op.Pts[5]
			}
		case OpQuad:
			if len(op.Pts) >= 4 {
				x, y = op.Pts[2], op.Pts[3]
			}
		}
	}
	for i := range expanded {
		for j := 0; j+1 < len(expanded[i].Pts); j += 2 {
			expanded[i].Pts[j], expanded[i].Pts[j+1] = m.Apply(expanded[i].Pts[j], expanded[i].Pts[j+1])
		}
	}
	return expanded
}

// inheritAttrs merges the node's paint attributes over inherited values.
func inheritAttrs(inh inherited, style css.Declarations) inherited {
	if v := style["fill"]; v != "" {
		inh.fill, inh.fillSet = v, true
	}
	if v := style["stroke"]; v != "" {
		inh.stroke, inh.strokeSet = v, true
	}
	if v := style["stroke-width"]; v != "" {
		if f, ok := parseNumber(v); ok {
			inh.width, inh.widthSet = f, true
		}
	}
	if v := style["fill-rule"]; v != "" {
		inh.evenOdd = v == "evenodd"
	}
	if v := style["opacity"]; v != "" {
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			inh.opacity, inh.hasOpacity = clamp01(f), true
		}
	}
	if v := style["fill-opacity"]; v != "" {
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			inh.fillOp, inh.hasFillOp = clamp01(f), true
		}
	}
	if v := style["stroke-opacity"]; v != "" {
		if f, err := strconv.ParseFloat(strings.TrimSpace(v), 64); err == nil {
			inh.strokeOp, inh.hasStrokeOp = clamp01(f), true
		}
	}
	return inh
}

func clamp01(f float64) float64 {
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}

// shapeOps converts a supported shape to operations; ok is false otherwise.
func shapeOps(n xmlNode) ([]Op, bool) {
	num := func(key string) float64 {
		f, _ := parseNumber(attr(n, key))
		return f
	}
	switch n.XMLName.Local {
	case "rect":
		x, y := num("x"), num("y")
		w, h := num("width"), num("height")
		if w <= 0 || h <= 0 {
			return nil, false
		}
		return RectOps(x, y, w, h, num("rx"), num("ry")), true
	case "circle":
		r := num("r")
		if r <= 0 {
			return nil, false
		}
		return CircleOps(num("cx"), num("cy"), r), true
	case "ellipse":
		rx, ry := num("rx"), num("ry")
		if rx <= 0 || ry <= 0 {
			return nil, false
		}
		return EllipseOps(num("cx"), num("cy"), rx, ry), true
	case "line":
		return LineOps(num("x1"), num("y1"), num("x2"), num("y2")), true
	case "polyline":
		ops, err := PolyOps(attr(n, "points"), false)
		if err != nil {
			return nil, false
		}
		return ops, true
	case "polygon":
		ops, err := PolyOps(attr(n, "points"), true)
		if err != nil {
			return nil, false
		}
		return ops, true
	case "path":
		ops, err := ParsePath(attr(n, "d"))
		if err != nil {
			return nil, false
		}
		return ops, true
	}
	return nil, false
}

// resolvePaint resolves inherited fill, stroke, widths, and opacity values.
func resolvePaint(style css.Declarations, inh inherited, ops []Op) Shape {
	s := Shape{Ops: ops}
	fill := inh.fill
	if v := style["fill"]; v != "" {
		fill = v
	}
	if fill != "" && fill != "none" {
		if c, ok := css.ParseColor(fill); ok {
			a := c[3]
			if inh.hasFillOp {
				a *= inh.fillOp
			}
			if inh.hasOpacity {
				a *= inh.opacity
			}
			s.HasFill = true
			s.Fill = [4]float64{c[0], c[1], c[2], a}
		}
	} else if fill == "" {
		// Default SVG: black fill.
		a := 1.0
		if inh.hasFillOp {
			a *= inh.fillOp
		}
		if inh.hasOpacity {
			a *= inh.opacity
		}
		s.HasFill = true
		s.Fill = [4]float64{0, 0, 0, a}
	}
	stroke := inh.stroke
	if v := style["stroke"]; v != "" {
		stroke = v
	}
	if stroke != "" && stroke != "none" {
		if c, ok := css.ParseColor(stroke); ok {
			a := c[3]
			if inh.hasStrokeOp {
				a *= inh.strokeOp
			}
			if inh.hasOpacity {
				a *= inh.opacity
			}
			s.HasStroke = true
			s.Stroke = [4]float64{c[0], c[1], c[2], a}
			s.Width = inh.width
		}
	}
	s.EvenOdd = inh.evenOdd
	return s
}

// parseNumber accepts a number with an optional px/pt unit.
func parseNumber(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(strings.TrimSuffix(s, "px"), "pt")
	if s == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

// parseViewBox parses "minx miny width height".
func parseViewBox(s string) [4]float64 {
	var vb [4]float64
	f := strings.Fields(strings.ReplaceAll(s, ",", " "))
	if len(f) != 4 {
		return vb
	}
	for i := range f {
		v, err := strconv.ParseFloat(f[i], 64)
		if err != nil {
			return [4]float64{}
		}
		vb[i] = v
	}
	return vb
}

// parseSize accepts a number or px value (percentages are resolved by layout).
func parseSize(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" || strings.HasSuffix(s, "%") {
		return 0, false
	}
	return parseNumber(s)
}
