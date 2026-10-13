package boxes

// ComputedStyle stores the resolved CSS and user-agent values used for layout and painting.
// The Box tree stores the document's content, resources, and metadata.
type ComputedStyle struct {
	BodyMargin                               float64 // uniform UA margin for the body element
	Display                                  Display // supported display mode
	FlexWrap                                 FlexWrapMode
	FlexDirection                            FlexDirection
	JustifyContent                           JustifyContent
	Float                                    FloatSide
	Clear                                    ClearSide
	Position                                 Position
	Top, Left, Right, Bottom                 float64
	HasRight, HasBottom                      bool
	Sizing                                   SizeConstraints
	BoxSizing                                BoxSizing
	GridCols                                 int // columns from grid-template-columns
	GridTracks                               []GridTrack
	GridRowHeights                           []float64 // fixed or auto heights from grid-template-rows
	GridRowFixedZeroTracks                   []bool    // distinguishes 0pt from auto
	GridAutoRowHeights                       []float64 // Cyclic pattern of implicit row heights (0 = auto or 0%).
	GridAutoRowFixedZeroTracks               []bool    // Distinguishes fixed zero-size tracks from auto tracks.
	GridAutoColumnWidths                     []float64 // Cyclic pattern of implicit column widths.
	GridAutoColumnPercentages                []float64 // Cyclic percentages for implicit tracks.
	GridAutoColumnPercentageTracks           []bool    // distinguishes 0% from auto
	GridAutoColumnFixedZeroTracks            []bool    // distinguishes 0pt from auto
	GridAutoColumnFractions                  []float64 // Cyclic fr factors for implicit tracks.
	GridAutoColumnMinimums                   []float64 // Fixed minimums for implicit minmax(..., fr) tracks.
	GridAutoColumnMinimumPercentages         []float64 // Percentage minimums for implicit fr tracks.
	GridAutoColumnAutomaticMinimums          []bool    // Automatic minimums for implicit fr tracks.
	GridAreas                                [][]string
	GridAutoFlowDense                        bool
	GridAutoFlowColumn                       bool
	RowGap, ColumnGap                        float64 // CSS gap in points
	RowGapPercent, ColumnGapPercent          float64 // percentage component of the gap
	ColCount                                 int     // column-count (0 = no columns)
	ColumnWidth                              float64 // column-width in points
	HasColumnWidth                           bool
	ColumnSpanAll                            bool // column-span: all
	ColumnRuleWidth                          float64
	ColumnRuleColor                          [4]float64
	AlignItems                               FlexAlignment // flex-start | center | flex-end | baseline | stretch
	AlignSelf                                FlexAlignment // auto | stretch | flex-start | center | flex-end | baseline
	FlexGrow                                 float64
	FlexBasis                                FlexBasis
	FlexShrink                               float64
	FlexShrinkSet                            bool
	ColSpan                                  int // number of spanned cells (1 = normal)
	RowSpan                                  int
	GridColumnLine                           int    // 1-based start line (negative counts from the end); 0 = auto.
	GridColumnEndLine                        int    // Optional 1-based end line.
	GridRowSpan                              int    // grid rows spanned by grid-row: span N
	GridRowLine                              int    // 1-based grid-row start line; 0 = auto.
	GridRowEndLine                           int    // Optional 1-based end line.
	GridAreaName                             string // Area name specified by grid-area.
	MarginTop, MarginRight                   float64
	MarginBottom, MarginLeft                 float64
	MarginAutoTop, MarginAutoRight           bool
	MarginAutoBottom, MarginAutoLeft         bool
	PaddingTop, PaddingRight                 float64
	PaddingBottom, PaddingLeft               float64
	ImageWidth, ImageHeight                  float64 // absolute CSS dimensions in points
	ImageWidthPct, ImageHeightPct            float64 // percentage CSS dimensions
	SVGWidth, SVGHeight                      float64 // CSS viewport dimensions in pt; 0 = intrinsic.
	SVGWidthPct                              float64 // percentage of the viewport width
	SVGHeightPct                             float64 // percentage of the viewport height
	HasBg                                    bool
	Bg                                       [4]float64 // background-color RGBA 0-1
	BorderWidth                              float64    // px, uniform border
	BorderCollapse                           bool
	TableBorderSpacingX, TableBorderSpacingY float64
	TableLayoutFixed                         bool
	TableColumnSpan                          int
	BorderCornerRadii                        [4]CornerRadius
	Border                                   [4]float64 // border color
	BorderTop, BorderRight                   BorderSide
	BorderBottom, BorderLeft                 BorderSide
	FontSize                                 float64 // pt; 0 = unset (layout default)
	HasColor                                 bool
	Color                                    [4]float64 // RGBA text color, values from 0 to 1
	Align                                    string     // left | center | right | justify
	FontFamily                               string     // first font-family entry
	FontFamilies                             []string   // ordered CSS font-family fallback list
	Bold                                     bool
	Italic                                   bool
	Underline                                bool
	Strike                                   bool
	LineHeight                               float64 // factor (0 = default 1.2)
	LineHeightPt                             float64 // absolute value in points (0 = unset; overrides the factor)
	LetterSpacing                            float64 // extra pt per rune (0 = normal)
	BreakBefore                              BreakBeforeMode
	BreakAfter                               BreakAfterMode
	Overflow                                 OverflowMode
	OverflowX                                OverflowMode // horizontal axis; unset inherits the shorthand
	OverflowY                                OverflowMode // vertical axis; unset inherits the shorthand
	Widows, Orphans                          int          // Minimum lines per fragment (0 = initial value 2).
	BreakInside                              BreakInsideMode
	Opacity                                  float64 // 0-1 (used when HasOpacity is true)
	HasOpacity                               bool
	WhiteSpace                               string // pre | pre-wrap | pre-line | nowrap ("" = normal)
	ListInside                               bool   // list-style-position: inside (default outside)
	ListMarkerShape                          string // circle | square | ""
	VAlign                                   string // sub | super | top | middle | bottom
	Transform                                string // uppercase | lowercase | capitalize ("" = none)
}
