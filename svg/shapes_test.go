package svg

import (
	"reflect"
	"testing"
)

// Shape parsing converts rect/circle/ellipse/line/poly elements to operations;
// the document inherits fill and computes intrinsic size.
func TestSVG_Shapes_ToOps(t *testing.T) {
	if ops := RectOps(0, 0, 10, 5, 0, 0); len(ops) != 5 {
		t.Errorf("rect: got %d ops, want 5 (M+3L+Z)", len(ops))
	}
	if ops := CircleOps(0, 0, 10); len(ops) != 6 {
		t.Errorf("circle: got %d ops, want 6 (M+4C+Z)", len(ops))
	}
	if ops := LineOps(1, 2, 3, 4); len(ops) != 2 {
		t.Errorf("line: got %d ops, want 2", len(ops))
	}
	ops, err := PolyOps("0,0 10,0 10,10", true)
	if err != nil || len(ops) != 4 {
		t.Errorf("polygon: got %v %v", ops, err)
	}
}

func TestSVG_Parse_Document(t *testing.T) {
	doc, err := Parse([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="40" height="20"><g fill="red"><rect width="40" height="20"/></g><circle cx="5" cy="5" r="2" fill="none" stroke="blue"/></svg>`))
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if doc.Width != 40 || doc.Height != 20 {
		t.Errorf("size: got %vx%v", doc.Width, doc.Height)
	}
	if len(doc.Shapes) != 2 {
		t.Fatalf("shapes: got %d, want 2", len(doc.Shapes))
	}
	if !doc.Shapes[0].HasFill || doc.Shapes[0].Fill[0] != 1 {
		t.Errorf("inherited fill: %+v", doc.Shapes[0])
	}
	if doc.Shapes[1].HasFill || doc.Shapes[1].Stroke != ([4]float64{0, 0, 1, 1}) {
		t.Errorf("stroke: %+v", doc.Shapes[1])
	}
}

func TestSVG_ParseTextNode(t *testing.T) {
	doc, err := Parse([]byte(`<svg width="300" height="170"><text x="40" y="155" font-size="12" fill="#334155">JUL</text></svg>`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(doc.Shapes) != 1 {
		t.Fatalf("shapes: got %d, want one text node", len(doc.Shapes))
	}
	shape := doc.Shapes[0]
	if shape.Text != "JUL" || shape.TextX != 40 || shape.TextY != 155 || shape.TextSize != 12 {
		t.Errorf("text geometry: %+v", shape)
	}
	if !shape.HasFill || shape.Fill != [4]float64{0x33 / 255.0, 0x41 / 255.0, 0x55 / 255.0, 1} {
		t.Errorf("text fill: %+v", shape)
	}
}

func TestSVG_Parse_AppliesInheritedTransforms(t *testing.T) {
	doc, err := Parse([]byte(`<svg><g transform="translate(10 20)"><rect x="1" y="2" width="3" height="4" transform="scale(2)"/></g></svg>`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(doc.Shapes) != 1 || len(doc.Shapes[0].Ops) < 2 {
		t.Fatalf("got %d shapes", len(doc.Shapes))
	}
	// List order composes as translate * scale, so (1,2) maps to (12,24).
	got := doc.Shapes[0].Ops[0].Pts
	if got[0] != 12 || got[1] != 24 {
		t.Errorf("first point = (%v,%v), want (12,24)", got[0], got[1])
	}
}

func TestSVG_Parse_InlineStylesInheritAndOverride(t *testing.T) {
	doc, err := Parse([]byte(`<svg><g style="fill: blue; stroke: red; stroke-width: 2"><rect width="5" height="4" style="fill: #00ff00"/></g></svg>`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(doc.Shapes) != 1 {
		t.Fatalf("got %d shapes, want 1", len(doc.Shapes))
	}
	s := doc.Shapes[0]
	if !s.HasFill || s.Fill[0] != 0 || s.Fill[1] != 1 || s.Fill[2] != 0 {
		t.Errorf("child inline style should override inherited fill: %+v", s)
	}
	if !s.HasStroke || s.Stroke[0] != 1 || s.Stroke[1] != 0 || s.Width != 2 {
		t.Errorf("stroke style should inherit from group: %+v", s)
	}
}

func TestSVG_Parse_StylesheetSelectorsApply(t *testing.T) {
	doc, err := Parse([]byte(`<svg><style>.hot { fill: #123456; stroke: red; stroke-width: 3 }</style><rect class="hot" width="8" height="6"/></svg>`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(doc.Shapes) != 1 {
		t.Fatalf("got %d shapes, want 1", len(doc.Shapes))
	}
	s := doc.Shapes[0]
	if !s.HasFill || s.Fill[0] != 18.0/255 || s.Fill[1] != 52.0/255 || s.Fill[2] != 86.0/255 {
		t.Errorf("class rule fill not applied: %+v", s)
	}
	if !s.HasStroke || s.Stroke[0] != 1 || s.Width != 3 {
		t.Errorf("class rule stroke not applied: %+v", s)
	}
}

func TestSVG_Rect_RxWithoutRyUsesSameRadius(t *testing.T) {
	got := RectOps(0, 0, 20, 10, 4, 0)
	want := RectOps(0, 0, 20, 10, 4, 4)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("rx without ry must round both axes: got %#v, want %#v", got, want)
	}
}
