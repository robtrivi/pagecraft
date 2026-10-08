package boxes

import (
	"strconv"
	"strings"

	"github.com/robtrivi/pagecraft-go/css"
)

func applyFlexStyle(b *Box, decls css.Declarations) {
	if d, ok := decls["display"]; ok {
		if display, valid := parseDisplay(d); valid {
			b.Style.Display = display
		}
	}
	if wrap, ok := decls["flex-wrap"]; ok {
		if mode, valid := parseFlexWrap(wrap); valid {
			b.Style.FlexWrap = mode
		}
	}
	if direction, ok := decls["flex-direction"]; ok {
		if mode, valid := parseFlexDirection(direction); valid {
			b.Style.FlexDirection = mode
		}
	}
	if justify, ok := decls["justify-content"]; ok {
		if mode, valid := parseJustifyContent(justify); valid {
			b.Style.JustifyContent = mode
		}
	}
	if ai, ok := decls["align-items"]; ok {
		if alignment, valid := parseFlexAlignment(ai, false, false); valid {
			b.Style.AlignItems = alignment
		}
	}
	if as, ok := decls["align-self"]; ok {
		if alignment, valid := parseFlexAlignment(as, true, true); valid {
			b.Style.AlignSelf = alignment
		}
	}
	if shorthand, ok := decls["flex"]; ok {
		if grow, shrink, basis, valid := parseFlexShorthand(shorthand); valid {
			b.Style.FlexGrow, b.Style.FlexBasis = grow, basis
			b.Style.FlexShrink, b.Style.FlexShrinkSet = shrink, true
		}
	}
	// Longhands are applied after the shorthand, matching their component-value
	// override behavior in the current declaration cascade.
	if grow, err := strconv.ParseFloat(strings.TrimSpace(decls["flex-grow"]), 64); err == nil && grow >= 0 {
		b.Style.FlexGrow = grow
	}
	if shrink, err := strconv.ParseFloat(strings.TrimSpace(decls["flex-shrink"]), 64); err == nil && shrink >= 0 {
		b.Style.FlexShrink, b.Style.FlexShrinkSet = shrink, true
	}
	if basis, ok := decls["flex-basis"]; ok {
		if parsed, valid := parseFlexBasis(basis); valid {
			b.Style.FlexBasis = parsed
		}
	}
}

func parseFlexBasis(value string) (FlexBasis, bool) {
	value = strings.TrimSpace(value)
	switch value {
	case "auto":
		return FlexBasis{Kind: FlexBasisAuto}, true
	case "content":
		return FlexBasis{Kind: FlexBasisContent}, true
	}
	dimension, valid := parseCSSDimension(value)
	if !valid {
		return FlexBasis{}, false
	}
	return FlexBasis{Kind: FlexBasisKind(dimension.Kind), Value: dimension.Value}, true
}

func parseFlexShorthand(value string) (float64, float64, FlexBasis, bool) {
	parts := strings.Fields(strings.TrimSpace(value))
	if len(parts) == 1 {
		switch parts[0] {
		case "none":
			return 0, 0, FlexBasis{Kind: FlexBasisAuto}, true
		case "auto":
			return 1, 1, FlexBasis{Kind: FlexBasisAuto}, true
		case "initial":
			return 0, 1, FlexBasis{Kind: FlexBasisAuto}, true
		}
	}
	if len(parts) == 0 || len(parts) > 3 {
		return 0, 0, FlexBasis{}, false
	}
	grow, shrink := 1.0, 1.0
	basis := FlexBasis{Kind: FlexBasisPercentage}
	factors, hasBasis := 0, false
	for _, part := range parts {
		if factor, err := strconv.ParseFloat(part, 64); err == nil {
			if factor < 0 || factors == 2 {
				return 0, 0, FlexBasis{}, false
			}
			if factors == 0 {
				grow = factor
			} else {
				shrink = factor
			}
			factors++
			continue
		}
		parsed, valid := parseFlexBasis(part)
		if !valid || hasBasis {
			return 0, 0, FlexBasis{}, false
		}
		basis, hasBasis = parsed, true
	}
	if len(parts) == 1 && factors == 1 {
		shrink = 1
	}
	return grow, shrink, basis, true
}

func applyFlowPositionStyle(b *Box, decls css.Declarations) {
	if fl, ok := decls["float"]; ok {
		if side, valid := parseFloatSide(fl); valid {
			b.Style.Float = side
		}
	}
	if clear, ok := decls["clear"]; ok {
		if side, valid := parseClearSide(clear); valid {
			b.Style.Clear = side
		}
	}
	if pos, ok := decls["position"]; ok {
		if position, valid := parsePosition(pos); valid {
			b.Style.Position = position
		}
	}
}
