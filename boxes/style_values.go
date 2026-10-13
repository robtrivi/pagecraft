package boxes

import (
	"strconv"
	"strings"
)

// Display is the supported computed display mode.
type Display string

const (
	DisplayNone   Display = "none"
	DisplayBlock  Display = "block"
	DisplayInline Display = "inline"
	DisplayFlex   Display = "flex"
	DisplayGrid   Display = "grid"
)

// Position is the computed CSS positioning mode. Its zero value represents static flow.
type Position string

const (
	PositionStatic   Position = ""
	PositionRelative Position = "relative"
	PositionAbsolute Position = "absolute"
	PositionFixed    Position = "fixed"
)

// FloatSide identifies which side receives a floated box.
type FloatSide string

const (
	FloatNone  FloatSide = ""
	FloatLeft  FloatSide = "left"
	FloatRight FloatSide = "right"
)

// ClearSide controls which float exclusions an in-flow box clears.
type ClearSide string

const (
	ClearNone  ClearSide = ""
	ClearLeft  ClearSide = "left"
	ClearRight ClearSide = "right"
	ClearBoth  ClearSide = "both"
)

// OverflowMode preserves whether overflow is inherited or explicitly set.
type OverflowMode string

const (
	OverflowUnset   OverflowMode = ""
	OverflowVisible OverflowMode = "visible"
	OverflowHidden  OverflowMode = "hidden"
)

// BoxSizing selects whether declared dimensions include border and padding.
type BoxSizing string

// FlexWrapMode controls whether flex items may form additional lines.
type FlexWrapMode string

// FlexDirection is the main axis of a Flex container.
type FlexDirection string

// JustifyContent distributes flex items along the main axis.
type JustifyContent string

// FlexAlignment is shared by align-items and align-self.
type FlexAlignment string

// FlexBasisKind describes how a Flex item's main-size basis is determined.
type FlexBasisKind uint8

const (
	FlexBasisAuto FlexBasisKind = iota
	FlexBasisLength
	FlexBasisPercentage
	FlexBasisContent
)

// FlexBasis preserves the authored flex-basis until the container size is known.
type FlexBasis struct {
	Kind  FlexBasisKind
	Value float64
}

// BreakBeforeMode selects the supported forced page break before a box.
type BreakBeforeMode string

// BreakAfterMode selects the supported forced page break after a box.
type BreakAfterMode string

// BreakInsideMode controls whether a box should stay intact when it fits.
type BreakInsideMode string

// DimensionKind identifies whether the declared CSS size is auto, absolute or relative.
type DimensionKind uint8

// CSSDimension stores an authored dimension without resolving its containing block.
type CSSDimension struct {
	Kind  DimensionKind
	Value float64
}

// SizeConstraints keeps authored dimensions and axis limits together until layout resolves them.
type SizeConstraints struct {
	Width, Height        CSSDimension
	MinWidth, MaxWidth   CSSDimension
	MinHeight, MaxHeight CSSDimension
}

func (d CSSDimension) IsAuto() bool       { return d.Kind == DimensionAuto }
func (d CSSDimension) IsLength() bool     { return d.Kind == DimensionLength }
func (d CSSDimension) IsPercentage() bool { return d.Kind == DimensionPercentage }

const (
	BoxSizingContentBox BoxSizing = ""
	BoxSizingBorderBox  BoxSizing = "border-box"
)

const (
	FlexNoWrap FlexWrapMode = ""
	FlexWrap   FlexWrapMode = "wrap"
)

const (
	FlexDirectionRow    FlexDirection = ""
	FlexDirectionColumn FlexDirection = "column"
)

const (
	JustifyFlexStart    JustifyContent = ""
	JustifyCenter       JustifyContent = "center"
	JustifyFlexEnd      JustifyContent = "flex-end"
	JustifySpaceBetween JustifyContent = "space-between"
)

const (
	AlignUnset     FlexAlignment = ""
	AlignStretch   FlexAlignment = "stretch"
	AlignAuto      FlexAlignment = "auto"
	AlignCenter    FlexAlignment = "center"
	AlignFlexStart FlexAlignment = "flex-start"
	AlignFlexEnd   FlexAlignment = "flex-end"
	AlignBaseline  FlexAlignment = "baseline"
)

const (
	BreakBeforeAuto BreakBeforeMode = ""
	BreakBeforePage BreakBeforeMode = "page"
)

const (
	BreakAfterAuto  BreakAfterMode = ""
	BreakAfterAvoid BreakAfterMode = "avoid"
	BreakAfterPage  BreakAfterMode = "page"
)

const (
	BreakInsideAuto  BreakInsideMode = ""
	BreakInsideAvoid BreakInsideMode = "avoid"
)

const (
	DimensionAuto DimensionKind = iota
	DimensionLength
	DimensionPercentage
)

func parseDisplay(value string) (Display, bool) {
	switch value {
	case "none":
		return DisplayNone, true
	case "block":
		return DisplayBlock, true
	case "inline":
		return DisplayInline, true
	case "flex":
		return DisplayFlex, true
	case "grid":
		return DisplayGrid, true
	default:
		return "", false
	}
}

func parsePosition(value string) (Position, bool) {
	switch value {
	case "static":
		return PositionStatic, true
	case "relative":
		return PositionRelative, true
	case "absolute":
		return PositionAbsolute, true
	case "fixed":
		return PositionFixed, true
	default:
		return PositionStatic, false
	}
}

func parseFloatSide(value string) (FloatSide, bool) {
	switch value {
	case "none":
		return FloatNone, true
	case "left":
		return FloatLeft, true
	case "right":
		return FloatRight, true
	default:
		return FloatNone, false
	}
}

func parseClearSide(value string) (ClearSide, bool) {
	switch value {
	case "none":
		return ClearNone, true
	case "left":
		return ClearLeft, true
	case "right":
		return ClearRight, true
	case "both":
		return ClearBoth, true
	default:
		return ClearNone, false
	}
}

func parseOverflowMode(value string) (OverflowMode, bool) {
	switch value {
	case "visible":
		return OverflowVisible, true
	case "hidden":
		return OverflowHidden, true
	default:
		return OverflowUnset, false
	}
}

func parseBoxSizing(value string) (BoxSizing, bool) {
	switch value {
	case "content-box":
		return BoxSizingContentBox, true
	case "border-box":
		return BoxSizingBorderBox, true
	default:
		return BoxSizingContentBox, false
	}
}

func parseFlexWrap(value string) (FlexWrapMode, bool) {
	switch value {
	case "nowrap":
		return FlexNoWrap, true
	case "wrap":
		return FlexWrap, true
	default:
		return FlexNoWrap, false
	}
}

func parseFlexDirection(value string) (FlexDirection, bool) {
	switch value {
	case "row":
		return FlexDirectionRow, true
	case "column":
		return FlexDirectionColumn, true
	default:
		return FlexDirectionRow, false
	}
}

func parseJustifyContent(value string) (JustifyContent, bool) {
	switch value {
	case "flex-start":
		return JustifyFlexStart, true
	case "center":
		return JustifyCenter, true
	case "flex-end":
		return JustifyFlexEnd, true
	case "space-between":
		return JustifySpaceBetween, true
	default:
		return JustifyFlexStart, false
	}
}

func parseFlexAlignment(value string, allowAuto, allowStretch bool) (FlexAlignment, bool) {
	switch value {
	case "stretch":
		return AlignStretch, allowStretch
	case "auto":
		return AlignAuto, allowAuto
	case "center":
		return AlignCenter, true
	case "flex-start":
		return AlignFlexStart, true
	case "flex-end":
		return AlignFlexEnd, true
	case "baseline":
		return AlignBaseline, true
	default:
		return AlignUnset, false
	}
}

func parseBreakBefore(value string) (BreakBeforeMode, bool) {
	switch value {
	case "page", "always":
		return BreakBeforePage, true
	default:
		return BreakBeforeAuto, false
	}
}

func parseBreakAfter(value string) (BreakAfterMode, bool) {
	switch value {
	case "auto":
		return BreakAfterAuto, true
	case "page", "always":
		return BreakAfterPage, true
	case "avoid", "avoid-page":
		return BreakAfterAvoid, true
	default:
		return BreakAfterAuto, false
	}
}

func parseBreakInside(value string) (BreakInsideMode, bool) {
	switch value {
	case "auto":
		return BreakInsideAuto, true
	case "avoid":
		return BreakInsideAvoid, true
	default:
		return BreakInsideAuto, false
	}
}

func parseCSSDimension(value string) (CSSDimension, bool) {
	value = strings.TrimSpace(value)
	if strings.HasSuffix(value, "%") {
		percent, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(value, "%")), 64)
		if err == nil && percent >= 0 {
			return CSSDimension{Kind: DimensionPercentage, Value: percent}, true
		}
	}
	if length, valid := tryLengthPx(value); valid {
		return CSSDimension{Kind: DimensionLength, Value: length}, true
	}
	return CSSDimension{}, false
}
