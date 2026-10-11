package layout

import (
	"math"
	"testing"

	"github.com/robtrivi/pagecraft/boxes"
)

func TestLayout_BlockAutoMarginsResolveAgainstContainingWidth(t *testing.T) {
	tests := []struct {
		name string
		css  string
		want float64
	}{
		{name: "both sides center a narrow block", css: "width:120pt;margin-left:auto;margin-right:auto", want: 237.5},
		{name: "shorthand centers with asymmetric padding", css: "width:180pt;margin:0 auto;padding-left:12pt;padding-right:8pt", want: 197.5},
		{name: "left auto consumes remaining space", css: "width:120pt;margin-left:auto;margin-right:10pt", want: 393},
		{name: "right auto stays at content origin", css: "width:120pt;margin-left:10pt;margin-right:auto", want: 82},
		{name: "oversized block does not gain negative margins", css: "width:500pt;margin:0 auto", want: 72},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, err := boxes.Build(`<div style="` + tt.css + `">content with variable-length text</div>`)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			blocks := Flow(root, testOptions())[0].Blocks
			if len(blocks) != 1 {
				t.Fatalf("blocks: got %d, want 1", len(blocks))
			}
			if math.Abs(blocks[0].X-tt.want) > 0.01 {
				t.Errorf("x: got %.2f, want %.2f", blocks[0].X, tt.want)
			}
		})
	}
}
