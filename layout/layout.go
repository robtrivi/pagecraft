// Package layout resolves the supported visual formatting rules and paginates
// content while preserving computed styles until each box is placed.
package layout

import (
	"math"

	"github.com/robtrivi/pagecraft/boxes"
	"github.com/robtrivi/pagecraft/css"
	"github.com/robtrivi/pagecraft/fonts"
	"github.com/robtrivi/pagecraft/images"
	"github.com/robtrivi/pagecraft/svg"
)

// Options configures the page and base typography.
type Options struct {
	PageWidth, PageHeight float64
	PageBackground        [4]float64
	HasPageBackground     bool
	PageRotation          int
	PageBleed             float64
	PageMarks             css.PageMarks
	PageRules             []css.PageRule
	pageNumber            int
	pageStyle             css.PageStyle
	pageBodyMargin        float64
	pageDefaults          *pageGeometryDefaults
	Margin                float64 // equal on all four sides (v0.3)
	PageMargins           css.PageMargins
	space                 layoutSpace
	FontSize              float64 // pt
	LineHeight            float64 // Global override; 0 uses the font's normal value.
	Measure               func(text string, fontSize float64) float64
	MeasureBold           func(text string, fontSize float64) float64 // nil = Measure
	Faces                 map[string]*fonts.Face                      // @font-face registry (nil = Helvetica only)
	FontVariants          map[string]fonts.Variants
	groupSequence         *int
	groupStack            []OpacityGroup
	tableOpacityCache     map[*boxes.Box]OpacityGroup
}

type pageGeometryDefaults struct {
	width, height, margin float64
	margins               css.PageMargins
	background            [4]float64
	hasBackground         bool
	rotation              int
	bleed                 float64
	marks                 css.PageMarks
}

func (or Options) left() float64  { return or.pageLeftMargin() + or.space.marginLeft }
func (or Options) right() float64 { return or.pageRightMargin() + or.space.marginRight }
func (or Options) top() float64   { return or.pageTopMargin() + or.space.marginTop }

// withinPageHorizontalRange sets a child coordinate range relative to the
// resolved page margins while retaining the page-box coordinate system.
func (or Options) withinPageHorizontalRange(x, width float64) Options {
	or.space.marginLeft = x - or.pageLeftMargin()
	or.space.marginRight = or.PageWidth - or.pageRightMargin() - x - width
	return or
}

func (or Options) pageBottom() float64 {
	return or.PageHeight - or.pageBottomMargin() - or.space.marginBottom
}

func (or Options) forPage(pageNumber int) Options {
	if pageNumber < 1 {
		pageNumber = 1
	}
	style := css.ResolvePageStyle(or.PageRules, "", pageNumber)
	if or.pageDefaults != nil && or.pageNumber == pageNumber {
		return or
	}
	if or.pageNumber != 0 && or.pageNumber != pageNumber && or.pageStyle == style {
		or.pageNumber = pageNumber
		return or
	}
	if or.pageDefaults == nil {
		or.pageDefaults = &pageGeometryDefaults{or.PageWidth, or.PageHeight, or.Margin, or.PageMargins, or.PageBackground, or.HasPageBackground, or.PageRotation, or.PageBleed, or.PageMarks}
	}
	defaults := or.pageDefaults
	or.PageWidth, or.PageHeight, or.Margin, or.PageMargins = defaults.width, defaults.height, defaults.margin, defaults.margins
	or.PageBackground, or.HasPageBackground = defaults.background, defaults.hasBackground
	or.PageRotation, or.PageBleed, or.PageMarks = defaults.rotation, defaults.bleed, defaults.marks
	or.pageNumber = pageNumber
	if style.HasSize {
		or.PageWidth, or.PageHeight = style.Width, style.Height
	}
	margins := style.Margins.Resolve(or.PageWidth, or.PageHeight)
	if margins.HasTop {
		or.PageMargins.Top, or.PageMargins.HasTop = margins.Top, true
	}
	if margins.HasRight {
		or.PageMargins.Right, or.PageMargins.HasRight = margins.Right, true
	}
	if margins.HasBottom {
		or.PageMargins.Bottom, or.PageMargins.HasBottom = margins.Bottom, true
	}
	if margins.HasLeft {
		or.PageMargins.Left, or.PageMargins.HasLeft = margins.Left, true
	}
	if style.HasBackground {
		or.PageBackground, or.HasPageBackground = style.Background, true
	}
	if style.HasRotation {
		or.PageRotation = style.Rotation
	}
	if style.HasBleed {
		or.PageBleed = style.Bleed
	}
	if style.HasMarks {
		or.PageMarks = style.Marks
	}
	or.pageStyle = style
	left, right := or.Margin, or.Margin
	top, bottom := or.Margin, or.Margin
	if or.PageMargins.HasLeft {
		left = or.PageMargins.Left
	}
	if or.PageMargins.HasRight {
		right = or.PageMargins.Right
	}
	if or.PageMargins.HasTop {
		top = or.PageMargins.Top
	}
	if or.PageMargins.HasBottom {
		bottom = or.PageMargins.Bottom
	}
	or.space = or.space.withPageMargins(left, right, top, bottom).withBodyMargin(or.pageBodyMargin)
	or.Margin += or.pageBodyMargin
	return or
}

func (or Options) pageLeftMargin() float64 {
	if or.space.pageMarginSet {
		return or.space.pageMarginLeft + or.space.bodyMargin
	}
	if or.PageMargins.HasLeft {
		return or.PageMargins.Left
	}
	return or.Margin
}

// pageAreaLeftMargin returns the @page margin without the body's margin.
// Fixed boxes are positioned in the page area, independently of body flow.
func (or Options) pageAreaLeftMargin() float64 {
	if or.space.pageMarginSet {
		return or.space.pageMarginLeft
	}
	if or.PageMargins.HasLeft {
		return or.PageMargins.Left
	}
	return or.Margin
}

func (or Options) pageAreaRightMargin() float64 {
	if or.space.pageMarginSet {
		return or.space.pageMarginRight
	}
	if or.PageMargins.HasRight {
		return or.PageMargins.Right
	}
	return or.Margin
}

func (or Options) pageAreaTopMargin() float64 {
	if or.space.pageMarginSet {
		return or.space.pageMarginTop
	}
	if or.PageMargins.HasTop {
		return or.PageMargins.Top
	}
	return or.Margin
}

func (or Options) pageAreaBottomMargin() float64 {
	if or.space.pageMarginSet {
		return or.space.pageMarginBottom
	}
	if or.PageMargins.HasBottom {
		return or.PageMargins.Bottom
	}
	return or.Margin
}

func (or Options) pageRightMargin() float64 {
	if or.space.pageMarginSet {
		return or.space.pageMarginRight + or.space.bodyMargin
	}
	if or.PageMargins.HasRight {
		return or.PageMargins.Right
	}
	return or.Margin
}
func (or Options) pageTopMargin() float64 {
	if or.space.pageMarginSet {
		return or.space.pageMarginTop + or.space.bodyMargin
	}
	if or.PageMargins.HasTop {
		return or.PageMargins.Top
	}
	return or.Margin
}
func (or Options) pageBottomMargin() float64 {
	if or.space.pageMarginSet {
		return or.space.pageMarginBottom
	}
	if or.PageMargins.HasBottom {
		return or.PageMargins.Bottom
	}
	return or.Margin
}

// DefaultOptions uses A4, a 1-inch margin, and a 12 pt base font size.
func DefaultOptions() Options {
	return Options{
		PageWidth: 595, PageHeight: 842,
		Margin: 72, FontSize: 12,
	}
}

// PlacedBox is a positioned block. X and and are its top-left coordinates in points.
type PlacedBox struct {
	fragmentNode             *FragmentNode
	source                   *boxes.Box
	sourceParent             *boxes.Box
	fragmentRef              *fragmentOccurrence
	parentFragmentRef        *fragmentOccurrence
	repeated                 bool
	FragmentainerIndex       int // 0 is page flow; columns start at 1.
	Text                     string
	Image                    *images.Raster // resolved <img> content
	SVG                      *svg.Document  // SVG inline
	HasBg                    bool
	Bg                       [4]float64
	BorderWidth              float64
	BorderCornerRadii        [4]boxes.CornerRadius
	Border                   [4]float64
	BorderTop, BorderRight   boxes.BorderSide
	BorderBottom, BorderLeft boxes.BorderSide
	FontSize                 float64 // resolved (Options default when the box does not set one)
	HasColor                 bool
	Color                    [4]float64
	Link                     string
	Align                    string // left | center | right | justify
	PadLeft                  float64
	PadRight                 float64
	Font                     *fonts.Face // nil = Helvetica base-14
	Bold                     bool
	Italic                   bool
	Underline                bool
	Strike                   bool
	Opacity                  float64
	HasOpacity               bool
	OpacityGroups            []OpacityGroup
	TablePaintUnderlays      []TablePaintLayer
	TablePaintOverlays       []TablePaintLayer
	ClipRects                []ClipRect
	LetterSpacing            float64
	Runs                     [][]TextRun // Inline runs per line; nil outside normal flow.
	X, Y                     float64
	Width                    float64
	Height                   float64
}

// TablePaintLayer retains the source box and fragment geometry for table,
// row-group, and row paint layers that the cell grid does not place directly.
type TablePaintLayer struct {
	Source *boxes.Box
	Bounds FragmentBounds
	Edges  TablePaintEdges
}

// TablePaintEdges selects the border edges visible on a structural fragment.
type TablePaintEdges uint8

const (
	TablePaintTop TablePaintEdges = 1 << iota
	TablePaintRight
	TablePaintBottom
	TablePaintLeft
	TablePaintAllEdges = TablePaintTop | TablePaintRight | TablePaintBottom | TablePaintLeft
)

// fragmentOccurrence is the identity shared by one container fragment and
// direct child placements emitted inside that fragment.
type fragmentOccurrence struct {
	source  *boxes.Box
	node    *FragmentNode
	pending []*FragmentNode
}

func newFragmentOccurrence(source *boxes.Box) *fragmentOccurrence {
	return &fragmentOccurrence{source: source}
}

func bindDirectFragmentChild(parent *fragmentOccurrence, child *PlacedBox) {
	if parent == nil || child == nil || child.parentFragmentRef != nil || !sameSourceBox(child.sourceParent, parent.source) {
		return
	}
	bindFragmentChild(parent, child)
}

func bindFragmentChild(parent *fragmentOccurrence, child *PlacedBox) {
	if parent == nil || child == nil || child.parentFragmentRef != nil {
		return
	}
	child.parentFragmentRef = parent
	if child.fragmentNode != nil {
		parent.attach(child.fragmentNode)
	}
}

func (parent *fragmentOccurrence) attach(child *FragmentNode) {
	if parent.node == nil {
		for _, pending := range parent.pending {
			if pending == child {
				return
			}
		}
		parent.pending = append(parent.pending, child)
		return
	}
	if child.Parent == parent.node {
		return
	}
	if child.state != nil {
		child.state.removeRoot(child)
	}
	child.Parent = parent.node
	parent.node.Children = append(parent.node.Children, child)
}

func sameSourceBox(left, right *boxes.Box) bool {
	if left == nil || right == nil {
		return false
	}
	if left == right {
		return true
	}
	leftPath, rightPath := left.SourcePath(), right.SourcePath()
	if len(leftPath) == 0 || len(leftPath) != len(rightPath) {
		return false
	}
	for index := range leftPath {
		if leftPath[index] != rightPath[index] {
			return false
		}
	}
	return true
}

// ClipRect defines a clipping rectangle in page coordinates (top-left origin).
type ClipRect struct {
	X, Y, Width, Height float64
}

// OpacityGroup identifies a compositing group for a box and its descendants.
type OpacityGroup struct {
	ID      int
	Opacity float64
}

// placed builds a PlacedBox and carries the box's background/border/color style.
func placed(src *boxes.Box, text string, img *images.Raster, x, y, w, h, fs float64, font *fonts.Face) PlacedBox {
	return PlacedBox{
		source: src, sourceParent: src.Parent,
		Text: text, Image: img, SVG: src.SVG,
		HasBg: src.Style.HasBg, Bg: src.Style.Bg, BorderWidth: src.Style.BorderWidth, BorderCornerRadii: src.Style.BorderCornerRadii, Border: src.Style.Border,
		BorderTop: src.Style.BorderTop, BorderRight: src.Style.BorderRight,
		BorderBottom: src.Style.BorderBottom, BorderLeft: src.Style.BorderLeft,
		FontSize: fs, HasColor: src.Style.HasColor, Color: src.Style.Color, Link: src.Link, Align: src.Style.Align,
		PadLeft: src.Style.PaddingLeft, PadRight: src.Style.PaddingRight, Font: font,
		Bold: src.Style.Bold, Italic: src.Style.Italic, Underline: src.Style.Underline, Strike: src.Style.Strike,
		Opacity: src.Style.Opacity, HasOpacity: src.Style.HasOpacity,
		LetterSpacing: src.Style.LetterSpacing,
		X:             x, Y: y, Width: w, Height: h,
	}
}

// fontFor resolves the box's font family (nil means Helvetica).
func fontFor(opts Options, b *boxes.Box) *fonts.Face {
	families := b.Style.FontFamilies
	if len(families) == 0 && b.Style.FontFamily != "" {
		families = []string{b.Style.FontFamily}
	}
	for _, family := range families {
		variants := opts.FontVariants[family]
		var face *fonts.Face
		switch {
		case b.Style.Bold && b.Style.Italic:
			face = variants.BoldItalic
			if face == nil {
				face = variants.Bold
			}
			if face == nil {
				face = variants.Italic
			}
		case b.Style.Bold:
			face = variants.Bold
		case b.Style.Italic:
			face = variants.Italic
		}
		if face == nil {
			face = variants.Regular
		}
		if face == nil {
			face = opts.Faces[family]
		}
		if face != nil {
			return face
		}
	}
	return nil
}

// measureFor selects the measurer and font: the box font, bold font, or global font.
// letter-spacing adds ls per rune, including spaces.
func measureFor(opts Options, b *boxes.Box) (func(string, float64) float64, *fonts.Face) {
	font := fontFor(opts, b)
	var base func(string, float64) float64
	if font != nil {
		base = func(s string, size float64) float64 { return font.Shape(s, size).Width }
	} else if b.Style.Bold && opts.MeasureBold != nil {
		base = opts.MeasureBold
	} else {
		base = opts.Measure
	}
	if ls := b.Style.LetterSpacing; ls != 0 {
		return func(s string, size float64) float64 {
			return base(s, size) + ls*float64(len([]rune(s)))
		}, font
	}
	return base, font
}

// resolveSize returns the box font size or the default.
func resolveSize(b *boxes.Box, opts Options) float64 {
	if b.Style.FontSize > 0 {
		return b.Style.FontSize
	}
	return opts.FontSize
}

// Page is a page numbered from 1. FragmentTree is the structural output;
// Blocks and Fragments are flat views published when Flow completes.
type Page struct {
	Number        int
	Geometry      PageGeometry
	Blocks        []PlacedBox
	Fragments     []BoxFragment
	FragmentTree  []*FragmentNode
	fragmentState *pageFragmentState
}

// PageGeometry is the resolved physical and content geometry for one output page.
type PageGeometry struct {
	Width, Height float64
	Margins       css.PageMargins
	Background    [4]float64
	HasBackground bool
	Rotation      int
	Bleed         float64
	Marks         css.PageMarks
}

func pageGeometryFromOptions(opts Options) PageGeometry {
	return PageGeometry{Width: opts.PageWidth, Height: opts.PageHeight, Margins: opts.PageMargins,
		Background: opts.PageBackground, HasBackground: opts.HasPageBackground, Rotation: opts.PageRotation,
		Bleed: opts.PageBleed, Marks: opts.PageMarks}
}

// BoxFragment describes one placed piece of a source box. PaintIndex addresses
// Page.Blocks; Path locates the source under Flow's root. Page.FragmentTree is
// the sole representation of page-local hierarchy.
type BoxFragment struct {
	Source             *boxes.Box
	Path               []int
	PageNumber         int
	FragmentainerIndex int
	FragmentIndex      int
	FragmentCount      int
	IsFirst            bool
	IsLast             bool
	Repeated           bool
	PaintIndex         int
	Bounds             FragmentBounds
}

type FragmentNodeKind string

const (
	FragmentPageBlock  FragmentNodeKind = "page-block"
	FragmentDecoration FragmentNodeKind = "decoration"
	FragmentTextRun    FragmentNodeKind = "text-run"
)

// FragmentNode is an emitted occurrence. Fragment and the flat views receive
// their final paint indices and chain metadata when Flow closes the page.
type FragmentNode struct {
	Fragment        BoxFragment
	Placed          PlacedBox
	Kind            FragmentNodeKind
	Parent          *FragmentNode // concrete instance of the parent fragment on this page
	OwnerPaintIndex int           // page block whose paint operation emits this decoration
	DecorationIndex int           // order inside the owning page block's decoration sequence
	Children        []*FragmentNode
	owner           *FragmentNode
	state           *pageFragmentState
}

// FragmentBounds is a border-box rectangle in page points.
type FragmentBounds struct {
	X, Y, Width, Height float64
}

// Flow lays out the direct children of root and opens a new page when the
// block does not fit in the remaining content area.
func Flow(root *boxes.Box, opts Options) []Page {
	boxes.AssignSourcePaths(root)
	opts.pageBodyMargin = root.Style.BodyMargin
	groupSequence := 0
	opts.groupSequence = &groupSequence
	pages, _ := flowLayout(root, opts)
	for i := range pages {
		pageOpts := opts
		if len(opts.PageRules) != 0 {
			pageOpts = opts.forPage(i + 1)
		}
		pages[i].Geometry = pageGeometryFromOptions(pageOpts)
	}
	finalizePageFragments(root, pages)
	return pages
}

func flowLayout(root *boxes.Box, opts Options) ([]Page, float64) {
	if len(opts.PageRules) != 0 {
		opts.pageBodyMargin = root.Style.BodyMargin
		opts = opts.forPage(1)
	} else {
		if !opts.space.pageMarginSet {
			left, right, top, bottom := opts.Margin, opts.Margin, opts.Margin, opts.Margin
			if opts.PageMargins.HasLeft {
				left = opts.PageMargins.Left
			}
			if opts.PageMargins.HasRight {
				right = opts.PageMargins.Right
			}
			if opts.PageMargins.HasTop {
				top = opts.PageMargins.Top
			}
			if opts.PageMargins.HasBottom {
				bottom = opts.PageMargins.Bottom
			}
			opts.space = opts.space.withPageMargins(left, right, top, bottom).withBodyMargin(root.Style.BodyMargin)
		}
		opts.Margin += root.Style.BodyMargin
	}
	usable := opts.PageWidth - opts.left() - opts.right()
	bottom := opts.pageBottom()
	if opts.Faces == nil {
		opts.Faces = root.Fonts
	}

	pages := []Page{{Number: 1}}
	y := opts.top()
	if opts.space.startYSet {
		y = opts.space.startY
	}
	var floats []activeFloat
	var fixed []*boxes.Box
	var previousBlockMarginBottom float64
	hasPreviousBlock := false
	for bi, b := range root.Children {
		if len(opts.PageRules) != 0 {
			opts = opts.forPage(len(pages))
		}
		usable = opts.PageWidth - opts.left() - opts.right()
		bottom = opts.pageBottom()
		if pages[len(pages)-1].blockCount() == 0 && len(pages) > 1 {
			y = opts.top()
		}
		if b.Style.Position == boxes.PositionFixed {
			fixed = append(fixed, b)
			continue
		}
		if b.Style.Position == boxes.PositionAbsolute {
			y = flowAbsolute(b, y, &pages, opts, usable)
			continue
		}
		if hasForcedPageBreakBefore(root.Children, bi) {
			y = forcePageBreak(y, opts, bottom, &pages)
			if len(opts.PageRules) != 0 {
				opts = opts.forPage(len(pages))
			}
			usable = opts.PageWidth - opts.left() - opts.right()
			bottom = opts.pageBottom()
			previousBlockMarginBottom, hasPreviousBlock = 0, false
		}
		if b.Style.Float == boxes.FloatNone && b.Style.Clear != boxes.ClearNone {
			clearedY := clearanceY(floats, len(pages)-1, b.Style.Clear, y)
			if clearedY > y {
				y = clearedY
				previousBlockMarginBottom, hasPreviousBlock = 0, false
			}
		}
		if b.Style.Display == boxes.DisplayFlex {
			gap := blockTopMarginGap(previousBlockMarginBottom, hasPreviousBlock, root.Style.BodyMargin, b.Style.MarginTop)
			contextY, contextOpts, contextWidth := placeFormattingContextAroundFloats(b, y+gap-b.Style.MarginTop, floats, len(pages)-1, opts, usable)
			y = flowFlex(b, contextY, &pages, contextOpts, contextWidth, bottom)
			previousBlockMarginBottom, hasPreviousBlock = b.Style.MarginBottom, true
			continue
		}
		if b.Style.Display == boxes.DisplayGrid {
			gap := blockTopMarginGap(previousBlockMarginBottom, hasPreviousBlock, root.Style.BodyMargin, b.Style.MarginTop)
			contextY, contextOpts, contextWidth := placeFormattingContextAroundFloats(b, y+gap-b.Style.MarginTop, floats, len(pages)-1, opts, usable)
			y = flowGrid(b, contextY, &pages, contextOpts, contextWidth, bottom)
			previousBlockMarginBottom, hasPreviousBlock = b.Style.MarginBottom, true
			continue
		}
		if b.Style.ColCount > 1 {
			gap := blockTopMarginGap(previousBlockMarginBottom, hasPreviousBlock, root.Style.BodyMargin, b.Style.MarginTop)
			contextY, contextOpts, contextWidth := placeFormattingContextAroundFloats(b, y+gap-b.Style.MarginTop, floats, len(pages)-1, opts, usable)
			y = flowColumns(b, contextY, &pages, contextOpts, contextWidth, bottom)
			previousBlockMarginBottom, hasPreviousBlock = b.Style.MarginBottom, true
			continue
		}
		if b.Style.Float == boxes.FloatLeft || b.Style.Float == boxes.FloatRight {
			y = dropBelowFloats(floats, len(pages)-1, y)
			floats = append(floats, flowFloat(b, y, &pages, opts, usable, bottom))
			continue
		}
		if b.Tag == "table" {
			gap := blockTopMarginGap(previousBlockMarginBottom, hasPreviousBlock, root.Style.BodyMargin, b.Style.MarginTop)
			y += gap
			tableY, tableOpts, tableUsable := placeFormattingContextAroundFloats(b, y, floats, len(pages)-1, opts, usable)
			y = flowTable(b, tableY, &pages, tableOpts, tableUsable, bottom)
			y += b.Style.MarginBottom
			previousBlockMarginBottom, hasPreviousBlock = b.Style.MarginBottom, true
			continue
		}
		b = resolveHorizontalAutoMargins(b, usable)
		inset, nested := nestedContainerInsets(b)
		if nested {
			items, topMargin, bottomMargin, collapseBottom := nestedContainerFlowMargins(b, inset)
			topGap := topMargin
			if hasPreviousBlock {
				topGap = collapseMargins(previousBlockMarginBottom, topMargin) - previousBlockMarginBottom
			} else if root.Style.BodyMargin != 0 {
				topGap = collapseMargins(root.Style.BodyMargin, topMargin) - root.Style.BodyMargin
			}
			if bi+1 < len(root.Children) && shouldMoveDecoratedBlockWithFollowingLines(b, root.Children[bi+1], y+topGap, usable, bottom, opts) {
				pageHasContent := pages[len(pages)-1].blockCount() > 0
				fragment := placeFragmentUnitForPage(fragmentUnit{breakBefore: fragmentBreakPage}, y+topGap, opts.top(), bottom, pageHasContent, &pages, opts)
				if fragment.pageOffset > 0 {
					y = fragment.y
					previousBlockMarginBottom, hasPreviousBlock = 0, false
					topGap = topMargin
					if root.Style.BodyMargin != 0 {
						topGap = collapseMargins(root.Style.BodyMargin, topMargin) - root.Style.BodyMargin
					}
				}
			}
			y += topGap
			y = flowNestedContainer(b, y, inset, items, collapseBottom, &pages, opts, usable)
			previousBlockMarginBottom = bottomMargin
			hasPreviousBlock = true
			continue
		}
		items := normalFlowItems(b)
		if b.Tag == "ul" || b.Tag == "ol" {
			if lis := liChildren(b); len(lis) > 0 {
				items = lis
			}
		}
		for idx, it := range items {
			if hasForcedPageBreakBefore(items, idx) {
				y = forcePageBreak(y, opts, bottom, &pages)
				if len(opts.PageRules) != 0 {
					opts = opts.forPage(len(pages))
				}
				usable = opts.PageWidth - opts.left() - opts.right()
				bottom = opts.pageBottom()
				previousBlockMarginBottom, hasPreviousBlock = 0, false
			}
			if it.Style.Float == boxes.FloatNone && it.Style.Clear != boxes.ClearNone {
				clearedY := clearanceY(floats, len(pages)-1, it.Style.Clear, y)
				if clearedY > y {
					y = clearedY
					previousBlockMarginBottom, hasPreviousBlock = 0, false
				}
			}
			if it.Style.Display == boxes.DisplayFlex {
				gap := blockTopMarginGap(previousBlockMarginBottom, hasPreviousBlock, root.Style.BodyMargin, it.Style.MarginTop)
				contextY, contextOpts, contextWidth := placeFormattingContextAroundFloats(it, y+gap-it.Style.MarginTop, floats, len(pages)-1, opts, usable)
				y = flowFlex(it, contextY, &pages, contextOpts, contextWidth, bottom)
				previousBlockMarginBottom, hasPreviousBlock = it.Style.MarginBottom, true
				continue
			}
			if it.Style.Display == boxes.DisplayGrid {
				gap := blockTopMarginGap(previousBlockMarginBottom, hasPreviousBlock, root.Style.BodyMargin, it.Style.MarginTop)
				contextY, contextOpts, contextWidth := placeFormattingContextAroundFloats(it, y+gap-it.Style.MarginTop, floats, len(pages)-1, opts, usable)
				y = flowGrid(it, contextY, &pages, contextOpts, contextWidth, bottom)
				previousBlockMarginBottom, hasPreviousBlock = it.Style.MarginBottom, true
				continue
			}
			if it.Style.ColCount > 1 {
				gap := blockTopMarginGap(previousBlockMarginBottom, hasPreviousBlock, root.Style.BodyMargin, it.Style.MarginTop)
				contextY, contextOpts, contextWidth := placeFormattingContextAroundFloats(it, y+gap-it.Style.MarginTop, floats, len(pages)-1, opts, usable)
				y = flowColumns(it, contextY, &pages, contextOpts, contextWidth, bottom)
				previousBlockMarginBottom, hasPreviousBlock = it.Style.MarginBottom, true
				continue
			}
			if it.Tag == "table" {
				gap := blockTopMarginGap(previousBlockMarginBottom, hasPreviousBlock, root.Style.BodyMargin, it.Style.MarginTop)
				y += gap
				tableY, tableOpts, tableUsable := placeFormattingContextAroundFloats(it, y, floats, len(pages)-1, opts, usable)
				y = flowTable(it, tableY, &pages, tableOpts, tableUsable, bottom)
				y += it.Style.MarginBottom
				previousBlockMarginBottom, hasPreviousBlock = it.Style.MarginBottom, true
				continue
			}
			it = resolveHorizontalAutoMargins(it, usable)
			fs := resolveSize(it, opts)
			_, font := measureFor(opts, it)
			insets := boxInsets(it)
			insetWidth := insets.left + insets.right
			width := usedWidth(it, math.Max(0, usable-it.Style.MarginLeft-it.Style.MarginRight))
			contentW := math.Max(0, width-insetWidth)
			words := buildInlineStream(it, opts)
			linesW := breakInlineStream(words, contentW, wraps(it))
			height := usedHeight(it, inlineHeight(linesW, it, opts), opts)
			var img *images.Raster
			var vector *svg.Document
			if it.Image != nil {
				img = it.Image
				width, height = rasterDimensions(it, usable, 0)
			} else if it.SVG != nil {
				vector = it.SVG
				width = vector.Width * css.PxToPt
				height = vector.Height * css.PxToPt
				hasWidth, hasHeight := false, false
				if it.Style.SVGWidth > 0 {
					width, hasWidth = it.Style.SVGWidth, true
				} else if it.Style.SVGWidthPct > 0 {
					width, hasWidth = usable*it.Style.SVGWidthPct/100, true
				}
				if it.Style.SVGHeight > 0 {
					height, hasHeight = it.Style.SVGHeight, true
				} else if it.Style.SVGHeightPct > 0 && opts.space.containingSize.Block.Definite {
					height, hasHeight = opts.space.containingSize.Block.Value*it.Style.SVGHeightPct/100, true
				}
				if hasWidth && !hasHeight {
					height = width * vector.Height / vector.Width
				} else if hasHeight && !hasWidth {
					width = height * vector.Width / vector.Height
				}
				if width > usable {
					scale := usable / width
					width, height = usable, height*scale
				}
			}
			topGap := blockTopMarginGap(previousBlockMarginBottom, hasPreviousBlock, root.Style.BodyMargin, it.Style.MarginTop)
			y += topGap
			// break-after:avoid protects the boundary between boxes. It only requires
			// Preserve the first fragmentable portion of the next block.
			var nxt *boxes.Box
			if idx+1 < len(items) {
				nxt = items[idx+1]
			} else if bi+1 < len(root.Children) {
				nxt = root.Children[bi+1]
			}
			if it.Style.BreakAfter == boxes.BreakAfterAvoid && img == nil && nxt != nil && isKeepable(nxt) {
				words2 := buildInlineStream(nxt, opts)
				cw2 := math.Max(0, usedWidth(nxt, usable-nxt.Style.MarginLeft-nxt.Style.MarginRight)-horizontalInsets(nxt))
				nextLines := breakInlineStream(words2, cw2, wraps(nxt))
				orphans := nxt.Style.Orphans
				if orphans <= 0 {
					orphans = 2
				}
				keepLines := min(len(nextLines), orphans)
				if keepLines == 0 {
					keepLines = 1
				}
				firstFragmentHeight := inlineHeight(nextLines[:keepLines], nxt, opts)
				if y+height <= bottom && y+height+nxt.Style.MarginTop+firstFragmentHeight > bottom {
					pages = append(pages, Page{Number: len(pages) + 1})
					y = opts.top()
				}
			}
			pageIdx := len(pages) - 1
			floats = floatsOnPage(floats, pageIdx)
			bx := opts.left() + it.Style.MarginLeft
			bw := width
			var positionedLines []inlineLine
			if img == nil && vector == nil {
				if len(floats) > 0 {
					contentX := bx + it.Style.BorderLeft.Width + it.Style.PaddingLeft
					contentWidth := math.Max(0, bw-horizontalInsets(it))
					contentY := y + it.Style.BorderTop.Width + it.Style.PaddingTop
					positionedLines = layoutInlineLines(it, words, contentX, contentWidth, contentY, floats, pageIdx, opts)
					linesW = inlineLineWords(positionedLines)
					if len(positionedLines) > 0 {
						contentHeight := positionedLines[len(positionedLines)-1].y + positionedLines[len(positionedLines)-1].height - contentY
						height = usedHeight(it, contentHeight, opts)
					}
				}
			}
			if img == nil && vector == nil && len(floatsOnPage(floats, pageIdx)) == 0 &&
				canFragmentTextBlock(it) && y+height > bottom &&
				!breakInsideAvoidFits(it.Style.BreakInside, height, bottom-opts.top()) {
				if len(opts.PageRules) != 0 {
					y = flowResponsiveInlineFragments(it, words, y, &pages, opts)
				} else {
					y = flowInlineBlockFragments(it, linesW, bx, bw, y, bottom, &pages, opts)
				}
				y += it.Style.MarginBottom
				previousBlockMarginBottom = it.Style.MarginBottom
				hasPreviousBlock = true
				continue
			}
			if y+height > bottom {
				pageHasContent := pages[len(pages)-1].blockCount() > 0
				fragment := placeFragmentUnitForPage(fragmentUnit{height: height}, y, opts.top(), bottom, pageHasContent, &pages, opts)
				y = fragment.y
			}
			var runs [][]TextRun
			if vector == nil {
				if positionedLines != nil {
					runs = inlineLineRuns(it, positionedLines, opts)
				} else {
					contentX := bx + it.Style.BorderLeft.Width + it.Style.PaddingLeft
					contentY := y + it.Style.BorderTop.Width + it.Style.PaddingTop
					contentWidth := bw - horizontalInsets(it)
					runs = assignRuns(it, linesW, contentX, contentWidth, contentY, opts)
				}
			}
			// Hanging: with outside, the marker sits outside the block and the
			// content returns to its place.
			if it.Tag == "li" && !it.Style.ListInside && len(runs) > 0 && len(runs[0]) > 1 {
				m := &runs[0][0]
				shift := m.Width + (runs[0][1].X - (m.X + m.Width))
				m.X -= shift
				for i := 1; i < len(runs[0]); i++ {
					runs[0][i].X -= shift
				}
			}
			p := &pages[len(pages)-1]
			pb := placed(it, it.Text, img, bx, y, bw, height, fs, font)
			pb.Runs = runs
			if it.Style.Position == boxes.PositionRelative {
				shiftPlacedBox(&pb, it.Style.Left, it.Style.Top)
			}
			p.emitBlock(pb)
			y += height + it.Style.MarginBottom
			previousBlockMarginBottom = it.Style.MarginBottom
			hasPreviousBlock = true
		}
	}
	// Fixed boxes: one copy per page at the same position (watermark).
	for pi := range pages {
		for _, fb := range fixed {
			pages[pi].emitBlock(placeFixed(fb, opts, usable))
		}
	}
	return pages, y
}

func isInFlowBox(b *boxes.Box) bool {
	return b != nil && b.Style.Position != boxes.PositionAbsolute && b.Style.Position != boxes.PositionFixed && b.Style.Float == boxes.FloatNone
}

func hasForcedPageBreakBefore(siblings []*boxes.Box, index int) bool {
	if index < 0 || index >= len(siblings) {
		return false
	}
	next := siblings[index]
	var previous *boxes.Box
	if isInFlowBox(next) {
		for i := index - 1; i >= 0; i-- {
			if isInFlowBox(siblings[i]) {
				previous = siblings[i]
				break
			}
		}
	}
	return forcedPageBreakBetween(previous, next)
}

func forcePageBreak(y float64, opts Options, bottom float64, pages *[]Page) float64 {
	if len(opts.PageRules) == 0 {
		pageHasContent := (*pages)[len(*pages)-1].blockCount() > 0
		fragment := placeFragmentUnitForPage(fragmentUnit{breakBefore: fragmentBreakPage}, y, opts.top(), bottom, pageHasContent, pages, opts)
		return fragment.y
	}
	current := len(*pages) - 1
	if (*pages)[current].blockCount() == 0 && y <= opts.top()+fragmentEpsilon {
		return y
	}
	ensurePageIndex(pages, current+1)
	return opts.forPage(current + 2).top()
}
