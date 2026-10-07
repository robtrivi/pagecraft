package svg

import (
	"errors"
	"strconv"
	"strings"
)

// kappa approximates circles with Bezier curves.
const kappa = 0.5522847498

// RectOps returns the path operations for a rectangle.
func RectOps(x, y, w, h, rx, ry float64) []Op {
	if w <= 0 || h <= 0 {
		return []Op{{Kind: OpMove, Pts: []float64{x, y}}, {Kind: OpClose}}
	}
	if rx == 0 && ry > 0 {
		rx = ry
	} else if ry == 0 && rx > 0 {
		ry = rx
	}
	if rx <= 0 || ry <= 0 {
		return []Op{
			{Kind: OpMove, Pts: []float64{x, y}},
			{Kind: OpLine, Pts: []float64{x + w, y}},
			{Kind: OpLine, Pts: []float64{x + w, y + h}},
			{Kind: OpLine, Pts: []float64{x, y + h}},
			{Kind: OpClose},
		}
	}
	if rx > w/2 {
		rx = w / 2
	}
	if ry > h/2 {
		ry = h / 2
	}
	kx, ky := kappa*rx, kappa*ry
	return []Op{
		{Kind: OpMove, Pts: []float64{x + rx, y}},
		{Kind: OpLine, Pts: []float64{x + w - rx, y}},
		{Kind: OpCurve, Pts: []float64{x + w - rx + kx, y, x + w, y + ry - ky, x + w, y + ry}},
		{Kind: OpLine, Pts: []float64{x + w, y + h - ry}},
		{Kind: OpCurve, Pts: []float64{x + w, y + h - ry + ky, x + w - rx + kx, y + h, x + w - rx, y + h}},
		{Kind: OpLine, Pts: []float64{x + rx, y + h}},
		{Kind: OpCurve, Pts: []float64{x + rx - kx, y + h, x, y + h - ry + ky, x, y + h - ry}},
		{Kind: OpLine, Pts: []float64{x, y + ry}},
		{Kind: OpCurve, Pts: []float64{x, y + ry - ky, x + rx - kx, y, x + rx, y}},
		{Kind: OpClose},
	}
}

// EllipseOps returns the path operations for an ellipse.
func EllipseOps(cx, cy, rx, ry float64) []Op {
	if rx <= 0 || ry <= 0 {
		return []Op{{Kind: OpMove, Pts: []float64{cx, cy}}, {Kind: OpClose}}
	}
	kx, ky := kappa*rx, kappa*ry
	return []Op{
		{Kind: OpMove, Pts: []float64{cx + rx, cy}},
		{Kind: OpCurve, Pts: []float64{cx + rx, cy + ky, cx + kx, cy + ry, cx, cy + ry}},
		{Kind: OpCurve, Pts: []float64{cx - kx, cy + ry, cx - rx, cy + ky, cx - rx, cy}},
		{Kind: OpCurve, Pts: []float64{cx - rx, cy - ky, cx - kx, cy - ry, cx, cy - ry}},
		{Kind: OpCurve, Pts: []float64{cx + kx, cy - ry, cx + rx, cy - ky, cx + rx, cy}},
		{Kind: OpClose},
	}
}

// CircleOps returns the path operations for a circle.
func CircleOps(cx, cy, r float64) []Op {
	return EllipseOps(cx, cy, r, r)
}

// LineOps returns the path operations for a line.
func LineOps(x1, y1, x2, y2 float64) []Op {
	return []Op{
		{Kind: OpMove, Pts: []float64{x1, y1}},
		{Kind: OpLine, Pts: []float64{x2, y2}},
	}
}

// PolyOps converts a list of points into a polyline or polygon path.
func PolyOps(points string, close bool) ([]Op, error) {
	nums, err := splitPoints(points)
	if err != nil {
		return nil, err
	}
	if len(nums) < 4 || len(nums)%2 != 0 {
		return nil, errors.New("svg: invalid points")
	}
	ops := []Op{{Kind: OpMove, Pts: []float64{nums[0], nums[1]}}}
	for i := 2; i < len(nums); i += 2 {
		ops = append(ops, Op{Kind: OpLine, Pts: []float64{nums[i], nums[i+1]}})
	}
	if close {
		ops = append(ops, Op{Kind: OpClose})
	}
	return ops, nil
}

// splitPoints splits numbers separated by spaces or commas.
func splitPoints(s string) ([]float64, error) {
	s = strings.ReplaceAll(s, ",", " ")
	var out []float64
	for _, f := range strings.Fields(s) {
		v, err := strconv.ParseFloat(f, 64)
		if err != nil {
			return nil, errors.New("svg: invalid points")
		}
		out = append(out, v)
	}
	return out, nil
}
