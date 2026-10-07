package svg

import (
	"errors"
	"strconv"
)

// OpKind identifies an SVG path operation.
type OpKind int

const (
	OpMove OpKind = iota
	OpLine
	OpCurve // Cubic curve: c1, c2, end (6 values).
	OpQuad  // Quadratic curve: end point (4 values).
	OpArc   // rx ry rotation large-arc sweep x and (7 raw values).
	OpClose
)

// Op represents a path operation and its points in absolute coordinates.
type Op struct {
	Kind OpKind
	Pts  []float64
}

// ParsePath converts an SVG d attribute into operations with absolute coordinates.
func ParsePath(d string) ([]Op, error) {
	toks, err := tokenizePath(d)
	if err != nil {
		return nil, err
	}
	var ops []Op
	var cx, cy float64         // current point
	var sx, sy float64         // initial subpath (for Z)
	var lastCx, lastCy float64 // Previous control point, for smooth curves.
	var lastQ bool             // Whether the previous command was q or t.
	var lastC bool             // Whether the previous command was c or s.
	cmd := byte(0)
	i := 0
	next := func(n int) ([]float64, error) {
		if i+n > len(toks) {
			return nil, errors.New("svg: missing parameters")
		}
		vals := make([]float64, n)
		for k := 0; k < n; k++ {
			v, err := strconv.ParseFloat(toks[i+k], 64)
			if err != nil {
				return nil, errors.New("svg: invalid number")
			}
			vals[k] = v
		}
		i += n
		return vals, nil
	}
	flushMove := func() {
		sx, sy = cx, cy
	}
	for i < len(toks) {
		t := toks[i]
		if len(t) == 1 && isPathCmd(t[0]) {
			cmd = t[0]
			i++
			if cmd == 'Z' || cmd == 'z' {
				ops = append(ops, Op{Kind: OpClose})
				cx, cy = sx, sy
				lastC, lastQ = false, false
				continue
			}
			if cmd == 'M' || cmd == 'm' {
				p, err := next(2)
				if err != nil {
					return nil, err
				}
				if cmd == 'm' {
					p[0] += cx
					p[1] += cy
				}
				cx, cy = p[0], p[1]
				flushMove()
				ops = append(ops, Op{Kind: OpMove, Pts: p})
				lastC, lastQ = false, false
				cmd = map[bool]byte{true: 'l', false: 'L'}[cmd == 'm']
				continue
			}
		}
		if cmd == 0 {
			return nil, errors.New("svg: path without comando")
		}
		switch cmd {
		case 'L', 'l':
			p, err := next(2)
			if err != nil {
				return nil, err
			}
			if cmd == 'l' {
				p[0] += cx
				p[1] += cy
			}
			cx, cy = p[0], p[1]
			ops = append(ops, Op{Kind: OpLine, Pts: p})
			lastC, lastQ = false, false
		case 'H', 'h':
			p, err := next(1)
			if err != nil {
				return nil, err
			}
			x := p[0]
			if cmd == 'h' {
				x += cx
			}
			cx = x
			ops = append(ops, Op{Kind: OpLine, Pts: []float64{cx, cy}})
			lastC, lastQ = false, false
		case 'V', 'v':
			p, err := next(1)
			if err != nil {
				return nil, err
			}
			y := p[0]
			if cmd == 'v' {
				y += cy
			}
			cy = y
			ops = append(ops, Op{Kind: OpLine, Pts: []float64{cx, cy}})
			lastC, lastQ = false, false
		case 'C', 'c':
			p, err := next(6)
			if err != nil {
				return nil, err
			}
			if cmd == 'c' {
				for k := 0; k < 6; k += 2 {
					p[k] += cx
					p[k+1] += cy
				}
			}
			lastCx, lastCy = p[2], p[3]
			lastC, lastQ = true, false
			cx, cy = p[4], p[5]
			ops = append(ops, Op{Kind: OpCurve, Pts: p})
		case 'S', 's':
			p, err := next(4)
			if err != nil {
				return nil, err
			}
			var c1x, c1y float64
			if lastC {
				c1x, c1y = 2*cx-lastCx, 2*cy-lastCy
			} else {
				c1x, c1y = cx, cy
			}
			pts := []float64{c1x, c1y, p[0], p[1], p[2], p[3]}
			if cmd == 's' {
				for k := 2; k < 6; k += 2 {
					pts[k] += cx
					pts[k+1] += cy
				}
			}
			lastCx, lastCy = pts[2], pts[3]
			lastC, lastQ = true, false
			cx, cy = pts[4], pts[5]
			ops = append(ops, Op{Kind: OpCurve, Pts: pts})
		case 'Q', 'q':
			p, err := next(4)
			if err != nil {
				return nil, err
			}
			if cmd == 'q' {
				for k := 0; k < 4; k += 2 {
					p[k] += cx
					p[k+1] += cy
				}
			}
			lastCx, lastCy = p[0], p[1]
			lastC, lastQ = false, true
			cx, cy = p[2], p[3]
			ops = append(ops, Op{Kind: OpQuad, Pts: p})
		case 'T', 't':
			p, err := next(2)
			if err != nil {
				return nil, err
			}
			var c1x, c1y float64
			if lastQ {
				c1x, c1y = 2*cx-lastCx, 2*cy-lastCy
			} else {
				c1x, c1y = cx, cy
			}
			pts := []float64{c1x, c1y, p[0], p[1]}
			if cmd == 't' {
				pts[2] += cx
				pts[3] += cy
			}
			lastCx, lastCy = pts[0], pts[1]
			lastC, lastQ = false, true
			cx, cy = pts[2], pts[3]
			ops = append(ops, Op{Kind: OpQuad, Pts: pts})
		case 'A', 'a':
			p, err := next(7)
			if err != nil {
				return nil, err
			}
			if cmd == 'a' {
				p[5] += cx
				p[6] += cy
			}
			lastC, lastQ = false, false
			cx, cy = p[5], p[6]
			ops = append(ops, Op{Kind: OpArc, Pts: p})
		default:
			return nil, errors.New("svg: invalid command")
		}
	}
	if len(ops) == 0 {
		return nil, errors.New("svg: empty path")
	}
	return ops, nil
}

func isPathCmd(c byte) bool {
	switch c {
	case 'M', 'm', 'L', 'l', 'H', 'h', 'V', 'v', 'C', 'c',
		'S', 's', 'Q', 'q', 'T', 't', 'A', 'a', 'Z', 'z':
		return true
	}
	return false
}

// tokenizePath splits a path into commands and numbers (spec: signs, decimal points,
// exponents; commas and whitespace separate tokens).
func tokenizePath(d string) ([]string, error) {
	var toks []string
	i := 0
	for i < len(d) {
		c := d[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == ',' {
			i++
			continue
		}
		if isPathCmd(c) && !(c == 'e' || c == 'E') {
			toks = append(toks, string(c))
			i++
			continue
		}
		j := i
		if d[j] == '+' || d[j] == '-' {
			j++
		}
		dot := false
		dig := false
		for j < len(d) {
			ch := d[j]
			if ch >= '0' && ch <= '9' {
				dig = true
				j++
			} else if ch == '.' && !dot {
				dot = true
				j++
			} else {
				break
			}
		}
		if j < len(d) && (d[j] == 'e' || d[j] == 'E') {
			k := j + 1
			if k < len(d) && (d[k] == '+' || d[k] == '-') {
				k++
			}
			m := k
			for m < len(d) && d[m] >= '0' && d[m] <= '9' {
				m++
			}
			if m > k {
				j = m
			}
		}
		if !dig && !dot {
			return nil, errors.New("svg: invalid token")
		}
		toks = append(toks, d[i:j])
		i = j
	}
	return toks, nil
}
