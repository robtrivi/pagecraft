package css

import (
	"strconv"
	"strings"
)

// ParseColor converts a supported CSS color to RGBA components in the range [0,1].
// It returns false for invalid values and the transparent keyword. Colors without alpha use 1.
func ParseColor(s string) (c [4]float64, ok bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	switch {
	case strings.HasPrefix(s, "#"):
		return parseHex(s[1:])
	case strings.HasPrefix(s, "rgba(") && strings.HasSuffix(s, ")"):
		return parseRGBAFunc(s[len("rgba(") : len(s)-1])
	case strings.HasPrefix(s, "rgb(") && strings.HasSuffix(s, ")"):
		if r, g, b, ok := parseRGBFunc(s[len("rgb(") : len(s)-1]); ok {
			return [4]float64{r, g, b, 1}, true
		}
		return [4]float64{}, false
	default:
		if c, known := namedColors[s]; known {
			return [4]float64{c[0], c[1], c[2], 1}, true
		}
		return [4]float64{}, false
	}
}

func hexByte(h string) (float64, bool) {
	n, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return 0, false
	}
	return float64(n) / 255, true
}

func parseHex(h string) ([4]float64, bool) {
	// Expand to rrggbbaa.
	var full string
	switch len(h) {
	case 3:
		full = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]}) + "ff"
	case 4:
		full = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2], h[3], h[3]})
	case 6:
		full = h + "ff"
	case 8:
		full = h
	default:
		return [4]float64{}, false
	}
	var c [4]float64
	for i := 0; i < 4; i++ {
		v, ok := hexByte(full[2*i : 2*i+2])
		if !ok {
			return [4]float64{}, false
		}
		c[i] = v
	}
	return c, true
}

func parseChannel(p string) (float64, bool) {
	p = strings.TrimSpace(p)
	if strings.HasSuffix(p, "%") {
		f, err := strconv.ParseFloat(strings.TrimSuffix(p, "%"), 64)
		if err != nil || f < 0 || f > 100 {
			return 0, false
		}
		return f / 100, true
	}
	n, err := strconv.Atoi(p)
	if err != nil || n < 0 || n > 255 {
		return 0, false
	}
	return float64(n) / 255, true
}

func parseRGBFunc(inner string) (float64, float64, float64, bool) {
	parts := strings.Split(inner, ",")
	if len(parts) != 3 {
		return 0, 0, 0, false
	}
	var c [3]float64
	for i, p := range parts {
		v, ok := parseChannel(p)
		if !ok {
			return 0, 0, 0, false
		}
		c[i] = v
	}
	return c[0], c[1], c[2], true
}

func parseRGBAFunc(inner string) ([4]float64, bool) {
	parts := strings.Split(inner, ",")
	if len(parts) != 4 {
		return [4]float64{}, false
	}
	var c [4]float64
	for i := 0; i < 3; i++ {
		v, ok := parseChannel(parts[i])
		if !ok {
			return [4]float64{}, false
		}
		c[i] = v
	}
	a, err := strconv.ParseFloat(strings.TrimSpace(parts[3]), 64)
	if err != nil || a < 0 || a > 1 {
		return [4]float64{}, false
	}
	c[3] = a
	return c, true
}

var namedColors = map[string][3]float64{
	"black": {0, 0, 0}, "silver": {0.75, 0.75, 0.75}, "gray": {0.5, 0.5, 0.5},
	"white": {1, 1, 1}, "maroon": {0.5, 0, 0}, "red": {1, 0, 0},
	"purple": {0.5, 0, 0.5}, "fuchsia": {1, 0, 1}, "green": {0, 0.5, 0},
	"lime": {0, 1, 0}, "olive": {0.5, 0.5, 0}, "yellow": {1, 1, 0},
	"navy": {0, 0, 0.5}, "blue": {0, 0, 1}, "teal": {0, 0.5, 0.5},
	"aqua": {0, 1, 1}, "orange": {1, 0.65, 0},
}
