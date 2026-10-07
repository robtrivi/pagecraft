package svg

import (
	"math"
)

// ArcToBezier converts an SVG endpoint arc into one or more Bezier curves.
// The large-arc and sweep flags are true when nonzero.
func ArcToBezier(x1, y1, rx, ry, rot, large, sweep, x2, y2 float64) []Op {
	line := []Op{{Kind: OpLine, Pts: []float64{x2, y2}}}
	if rx == 0 || ry == 0 || (x1 == x2 && y1 == y2) {
		return line
	}
	rx, ry = math.Abs(rx), math.Abs(ry)
	phi := rot * math.Pi / 180
	cosPhi, sinPhi := math.Cos(phi), math.Sin(phi)
	dx, dy := (x1-x2)/2, (y1-y2)/2
	x1p := cosPhi*dx + sinPhi*dy
	y1p := -sinPhi*dx + cosPhi*dy
	lam := x1p*x1p/(rx*rx) + y1p*y1p/(ry*ry)
	if lam > 1 {
		s := math.Sqrt(lam)
		rx, ry = rx*s, ry*s
	}
	sign := 1.0
	if (large != 0) == (sweep != 0) {
		sign = -1
	}
	num := rx*rx*ry*ry - rx*rx*y1p*y1p - ry*ry*x1p*x1p
	den := rx*rx*y1p*y1p + ry*ry*x1p*x1p
	factor := 0.0
	if den > 0 {
		factor = sign * math.Sqrt(math.Max(0, num/den))
	}
	cxp := factor * rx * y1p / ry
	cyp := -factor * ry * x1p / rx
	cx := cosPhi*cxp - sinPhi*cyp + (x1+x2)/2
	cy := sinPhi*cxp + cosPhi*cyp + (y1+y2)/2
	theta1 := vecAngle(1, 0, (x1p-cxp)/rx, (y1p-cyp)/ry)
	dtheta := vecAngle((x1p-cxp)/rx, (y1p-cyp)/ry, (-x1p-cxp)/rx, (-y1p-cyp)/ry)
	if sweep == 0 && dtheta > 0 {
		dtheta -= 2 * math.Pi
	} else if sweep != 0 && dtheta < 0 {
		dtheta += 2 * math.Pi
	}
	n := int(math.Ceil(math.Abs(dtheta) / (math.Pi / 2)))
	if n < 1 {
		n = 1
	}
	segs := make([]Op, 0, n)
	for i := 0; i < n; i++ {
		t1 := theta1 + dtheta*float64(i)/float64(n)
		t2 := theta1 + dtheta*float64(i+1)/float64(n)
		segs = append(segs, arcSegment(cx, cy, rx, ry, cosPhi, sinPhi, t1, t2))
	}
	return segs
}

// vecAngle returns the oriented angle from u to v.
func vecAngle(ux, uy, vx, vy float64) float64 {
	dot := ux*vx + uy*vy
	lu := math.Hypot(ux, uy)
	lv := math.Hypot(vx, vy)
	if lu == 0 || lv == 0 {
		return 0
	}
	c := dot / (lu * lv)
	c = math.Max(-1, math.Min(1, c))
	a := math.Acos(c)
	if ux*vy-uy*vx < 0 {
		a = -a
	}
	return a
}

// arcSegment approximates the [t1, t2] segment with a cubic curve.
func arcSegment(cx, cy, rx, ry, cosPhi, sinPhi, t1, t2 float64) Op {
	pt := func(t float64) (float64, float64) {
		x := rx * math.Cos(t)
		y := ry * math.Sin(t)
		return cosPhi*x - sinPhi*y + cx, sinPhi*x + cosPhi*y + cy
	}
	dt := t2 - t1
	k := 4.0 / 3.0 * math.Tan(dt/4)
	x1, y1 := pt(t1)
	x2, y2 := pt(t2)
	dx1 := -rx * math.Sin(t1)
	dy1 := ry * math.Cos(t1)
	dx2 := -rx * math.Sin(t2)
	dy2 := ry * math.Cos(t2)
	return Op{Kind: OpCurve, Pts: []float64{
		x1 + k*(cosPhi*dx1-sinPhi*dy1), y1 + k*(sinPhi*dx1+cosPhi*dy1),
		x2 - k*(cosPhi*dx2-sinPhi*dy2), y2 - k*(sinPhi*dx2+cosPhi*dy2),
		x2, y2,
	}}
}
