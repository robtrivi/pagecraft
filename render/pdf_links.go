package render

import (
	"fmt"
	"strings"

	"github.com/robtrivi/pagecraft-go/layout"
)

type linkAnnot struct {
	rect [4]float64
	uri  string
}

// collectLinks gathers blocks and runs with href values into per-page annotations.
func collectLinks(pages []layout.Page, pageH float64) ([]linkAnnot, map[int][]int) {
	var annots []linkAnnot
	byPage := map[int][]int{}
	add := func(pi int, x, topY, w, h float64, uri string) {
		byPage[pi] = append(byPage[pi], len(annots))
		annots = append(annots, linkAnnot{
			rect: [4]float64{x, pageH - topY - h, x + w, pageH - topY},
			uri:  uri,
		})
	}
	for pi, pg := range pages {
		for _, b := range pg.Blocks {
			if len(b.Runs) > 0 {
				for _, ln := range b.Runs {
					for _, r := range ln {
						if r.Link == "" {
							continue
						}
						add(pi, r.X, r.Y, r.Width, r.LineH, r.Link)
					}
				}
				continue
			}
			if b.Link == "" {
				continue
			}
			add(pi, b.X, b.Y, b.Width, b.Height, b.Link)
		}
	}
	return annots, byPage
}

// annotsRefs builds " /Annots [12 0 R ...]" or an empty string when there are none.
func annotsRefs(idxs []int, base int) string {
	if len(idxs) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(" /Annots [")
	for _, i := range idxs {
		fmt.Fprintf(&sb, " %d 0 R", base+i)
	}
	sb.WriteString(" ]")
	return sb.String()
}

// base14Refs builds " /F1 3 0 R /F2 4 0 R ..." for the base font variants.
