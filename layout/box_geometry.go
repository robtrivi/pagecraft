package layout

import "github.com/robtrivi/pagecraft/boxes"

func horizontalInsets(b *boxes.Box) float64 {
	insets := boxInsets(b)
	return insets.left + insets.right
}

func verticalInsets(b *boxes.Box) float64 {
	insets := boxInsets(b)
	return insets.top + insets.bottom
}

type containerInsets struct{ top, right, bottom, left float64 }

func boxInsets(b *boxes.Box) containerInsets {
	return containerInsets{
		top:    b.Style.PaddingTop + b.Style.BorderTop.Width,
		right:  b.Style.PaddingRight + b.Style.BorderRight.Width,
		bottom: b.Style.PaddingBottom + b.Style.BorderBottom.Width,
		left:   b.Style.PaddingLeft + b.Style.BorderLeft.Width,
	}
}

// fragmentBox returns a shallow copy with slice decorations removed at cut edges.
func fragmentBox(source *boxes.Box, firstFragment, lastFragment bool) *boxes.Box {
	fragment := *source
	if !firstFragment {
		fragment.Style.PaddingTop = 0
		fragment.Style.BorderTop = boxes.BorderSide{}
	}
	if !lastFragment {
		fragment.Style.PaddingBottom = 0
		fragment.Style.BorderBottom = boxes.BorderSide{}
	}
	return &fragment
}

// fragmentPlacedBox keeps paintable borders only at the outer edges of fragments.
func fragmentPlacedBox(source PlacedBox, firstFragment, lastFragment bool) PlacedBox {
	fragment := source
	fragment.BorderWidth = 0
	if !firstFragment {
		fragment.BorderTop = boxes.BorderSide{}
	}
	if !lastFragment {
		fragment.BorderBottom = boxes.BorderSide{}
	}
	return fragment
}
