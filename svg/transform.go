package svg

import (
	"errors"
	"math"
	"strconv"
	"strings"
)

// Matrix is a 2D affine matrix using the SVG convention:
// x'=A*x+C*y+E, y'=B*x+D*y+F.
type Matrix struct {
	A, B, C, D, E, F float64
}

// ViewBoxTransform maps viewBox coordinates to the viewport, centering the
// drawing and preserving its aspect ratio like preserveAspectRatio="xMidYMid meet".
func (d *Document) ViewBoxTransform(width, height float64) Matrix {
	if width <= 0 || height <= 0 {
		return IdentityTransform()
	}
	vb := d.ViewBox
	if vb[2] <= 0 || vb[3] <= 0 {
		vb = [4]float64{0, 0, d.Width, d.Height}
	}
	if vb[2] <= 0 || vb[3] <= 0 {
		return IdentityTransform()
	}
	sx, sy := width/vb[2], height/vb[3]
	parts := strings.Fields(strings.TrimSpace(strings.TrimPrefix(d.PreserveAspectRatio, "defer ")))
	align, mode := "xMidYMid", "meet"
	if len(parts) > 0 {
		align = parts[0]
	}
	if len(parts) > 1 && parts[1] == "slice" {
		mode = "slice"
	}
	if align == "none" {
		return Matrix{A: sx, D: sy, E: -vb[0] * sx, F: -vb[1] * sy}
	}
	scale := math.Min(sx, sy)
	if mode == "slice" {
		scale = math.Max(sx, sy)
	}
	extraX, extraY := width-vb[2]*scale, height-vb[3]*scale
	left, top := extraX/2, extraY/2
	if strings.HasPrefix(align, "xMin") {
		left = 0
	} else if strings.HasPrefix(align, "xMax") {
		left = extraX
	}
	if strings.HasSuffix(align, "YMin") {
		top = 0
	} else if strings.HasSuffix(align, "YMax") {
		top = extraY
	}
	return Matrix{A: scale, D: scale, E: left - vb[0]*scale, F: top - vb[1]*scale}
}

// IdentityTransform returns the identity matrix.
func IdentityTransform() Matrix { return Matrix{A: 1, D: 1} }

// Apply transforms a point.
func (m Matrix) Apply(x, y float64) (float64, float64) {
	return m.A*x + m.C*y + m.E, m.B*x + m.D*y + m.F
}

// ParseTransform parses an SVG list of translate, scale, rotate, skew, and matrix operations.
// Transforms are composed in written order, multiplying each new matrix on the right.
func ParseTransform(s string) (Matrix, error) {
	result := IdentityTransform()
	s = strings.TrimSpace(s)
	if s == "" {
		return Matrix{}, errors.New("svg: empty transform")
	}
	for len(s) > 0 {
		s = strings.TrimLeft(s, " \t\r\n,")
		if s == "" {
			break
		}
		i := 0
		for i < len(s) && ((s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z')) {
			i++
		}
		if i == 0 {
			return Matrix{}, errors.New("svg: invalid transform")
		}
		name := s[:i]
		rest := strings.TrimLeft(s[i:], " \t\r\n")
		if len(rest) == 0 || rest[0] != '(' {
			return Matrix{}, errors.New("svg: missing parentheses in transform")
		}
		end := strings.IndexByte(rest, ')')
		if end < 0 {
			return Matrix{}, errors.New("svg: transform without cerrar")
		}
		args, err := transformNumbers(rest[1:end])
		if err != nil {
			return Matrix{}, err
		}
		next, err := transformMatrix(name, args)
		if err != nil {
			return Matrix{}, err
		}
		result = result.Multiply(next)
		s = rest[end+1:]
	}
	return result, nil
}

// Multiply returns m*n; n is applied to the point first.
func (m Matrix) Multiply(n Matrix) Matrix {
	return Matrix{
		A: m.A*n.A + m.C*n.B,
		B: m.B*n.A + m.D*n.B,
		C: m.A*n.C + m.C*n.D,
		D: m.B*n.C + m.D*n.D,
		E: m.A*n.E + m.C*n.F + m.E,
		F: m.B*n.E + m.D*n.F + m.F,
	}
}

func transformNumbers(s string) ([]float64, error) {
	s = strings.ReplaceAll(s, ",", " ")
	fields := strings.Fields(s)
	values := make([]float64, len(fields))
	for i, field := range fields {
		v, err := strconv.ParseFloat(field, 64)
		if err != nil {
			return nil, errors.New("svg: invalid transform argument")
		}
		values[i] = v
	}
	return values, nil
}

func transformMatrix(name string, p []float64) (Matrix, error) {
	bad := func() (Matrix, error) { return Matrix{}, errors.New("svg: invalid arguments for " + name) }
	switch name {
	case "matrix":
		if len(p) != 6 {
			return bad()
		}
		return Matrix{A: p[0], B: p[1], C: p[2], D: p[3], E: p[4], F: p[5]}, nil
	case "translate":
		if len(p) < 1 || len(p) > 2 {
			return bad()
		}
		m := IdentityTransform()
		m.E = p[0]
		if len(p) == 2 {
			m.F = p[1]
		}
		return m, nil
	case "scale":
		if len(p) < 1 || len(p) > 2 {
			return bad()
		}
		y := p[0]
		if len(p) == 2 {
			y = p[1]
		}
		return Matrix{A: p[0], D: y}, nil
	case "rotate":
		if len(p) != 1 && len(p) != 3 {
			return bad()
		}
		a := p[0] * math.Pi / 180
		c, s := math.Cos(a), math.Sin(a)
		rot := Matrix{A: c, B: s, C: -s, D: c}
		if len(p) == 3 {
			toCenter := IdentityTransform()
			toCenter.E, toCenter.F = p[1], p[2]
			fromCenter := IdentityTransform()
			fromCenter.E, fromCenter.F = -p[1], -p[2]
			rot = toCenter.Multiply(rot).Multiply(fromCenter)
		}
		return rot, nil
	case "skewX", "skewY":
		if len(p) != 1 {
			return bad()
		}
		t := math.Tan(p[0] * math.Pi / 180)
		if name == "skewX" {
			return Matrix{A: 1, C: t, D: 1}, nil
		}
		return Matrix{A: 1, B: t, D: 1}, nil
	default:
		return bad()
	}
}
