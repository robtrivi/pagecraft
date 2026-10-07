package svg

import (
	"testing"
)

// Arc conversion produces one or more Bezier curves from endpoints (spec F.6.5).
func TestSVG_ArcToBezier_Quarter(t *testing.T) {
	// Quarter circle with r=10 from (10,0) to (0,10).
	segs := ArcToBezier(10, 0, 10, 10, 0, 0, 1, 0, 10)
	if len(segs) != 1 {
		t.Fatalf("segments: got %d, want 1", len(segs))
	}
	end := segs[0].Pts[4:]
	if diff := end[0] - 0; diff < -1e-9 || diff > 1e-9 {
		t.Errorf("end x: got %v, want 0", end[0])
	}
	if diff := end[1] - 10; diff < -1e-9 || diff > 1e-9 {
		t.Errorf("end y: got %v, want 10", end[1])
	}
}

func TestSVG_ArcToBezier_Half(t *testing.T) {
	segs := ArcToBezier(0, 0, 10, 10, 0, 1, 1, 20, 0)
	if len(segs) != 2 {
		t.Fatalf("half arc: got %d segments, want 2", len(segs))
	}
}

func TestSVG_ArcToBezier_Degenerate(t *testing.T) {
	// Same point or zero radius becomes a line to the endpoint.
	segs := ArcToBezier(5, 5, 0, 0, 0, 0, 0, 5, 5)
	if len(segs) != 1 || segs[0].Kind != OpLine {
		t.Errorf("degenerado: got %+v", segs)
	}
	segs = ArcToBezier(1, 1, 10, 10, 0, 0, 0, 1, 1)
	if len(segs) != 1 || segs[0].Kind != OpLine {
		t.Errorf("same point: got %+v", segs)
	}
}
