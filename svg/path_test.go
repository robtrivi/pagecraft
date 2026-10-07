package svg

import (
	"reflect"
	"testing"
)

// Path parsing converts d-strings to operations (M/L/H/V/C/S/Q/T/t/Z), including
// absolute/relative, smooth, and close commands. Rendering is not covered here.
func TestSVG_ParsePath_Basic(t *testing.T) {
	ops, err := ParsePath("M10 10 L20 20 Z")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	want := []Op{
		{Kind: OpMove, Pts: []float64{10, 10}},
		{Kind: OpLine, Pts: []float64{20, 20}},
		{Kind: OpClose},
	}
	if !reflect.DeepEqual(ops, want) {
		t.Errorf("got %+v, want %+v", ops, want)
	}
}

func TestSVG_ParsePath_RelativeAndSmooth(t *testing.T) {
	ops, err := ParsePath("m0 0 h10 v10 c1 2 3 4 5 6 s7 8 9 10 q1 1 2 2 t3 3 a5 5 0 0 1 6 6")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(ops) != 8 {
		t.Fatalf("ops: got %d, want 8", len(ops))
	}
	if !reflect.DeepEqual(ops[0], Op{Kind: OpMove, Pts: []float64{0, 0}}) {
		t.Errorf("move: got %+v", ops[0])
	}
	if ops[1].Kind != OpLine || ops[1].Pts[0] != 10 || ops[1].Pts[1] != 0 {
		t.Errorf("h rel: got %+v", ops[1])
	}
	if ops[2].Kind != OpLine || ops[2].Pts[0] != 10 || ops[2].Pts[1] != 10 {
		t.Errorf("v rel: got %+v", ops[2])
	}
	// Relative c from (10,10): (11,12), (13,14), (15,16).
	if ops[3].Kind != OpCurve || ops[3].Pts[4] != 15 || ops[3].Pts[5] != 16 {
		t.Errorf("c rel: got %+v", ops[3])
	}
	// s reflects the control point from (15,16), using the previous point (13,14) and current point (17,18).
	if ops[4].Kind != OpCurve || ops[4].Pts[0] != 17 || ops[4].Pts[1] != 18 {
		t.Errorf("s smooth: got %+v", ops[4])
	}
	// t reflects from the current point; validate operation type and count.
	if ops[6].Kind != OpQuad {
		t.Errorf("t smooth: got %+v", ops[6])
	}
	if ops[7].Kind != OpArc {
		t.Errorf("arc: got %+v", ops[7])
	}
}

func TestSVG_ParsePath_Invalid(t *testing.T) {
	for _, d := range []string{"", "M10", "Q1 2", "Z5", "M 1 2 3"} {
		if _, err := ParsePath(d); err == nil {
			t.Errorf("expected error: %q", d)
		}
	}
}

func TestSVG_ParsePath_ImplicitLineto(t *testing.T) {
	// Additional coordinate pairs after M are implicit L commands (per spec).
	ops, err := ParsePath("M1 2 3 4")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	want := []Op{
		{Kind: OpMove, Pts: []float64{1, 2}},
		{Kind: OpLine, Pts: []float64{3, 4}},
	}
	if !reflect.DeepEqual(ops, want) {
		t.Errorf("got %+v, want %+v", ops, want)
	}
}
