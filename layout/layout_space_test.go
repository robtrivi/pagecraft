package layout

import "testing"

func TestLayout_ChildSpaceDerivesDefiniteContainingBlockWithoutMutatingParent(t *testing.T) {
	parent := layoutSpace{
		containingSize: sizeContext{
			Inline: definiteSize(240),
			Block:  definiteSize(120),
		},
		marginLeft: 12,
	}
	child := parent.withContainingBlock(sizeContext{
		Inline: definiteSize(80),
		Block:  sizeBasis{},
	}).withMargins(4, 6, 0, 0)

	if got := child.containingSize.Inline; got != definiteSize(80) {
		t.Errorf("child inline basis=%+v, want definite 80", got)
	}
	if child.containingSize.Block.Definite {
		t.Errorf("child block basis=%+v, want indefinite", child.containingSize.Block)
	}
	if got := child.marginLeft; got != 4 {
		t.Errorf("child left margin=%g, want 4", got)
	}
	if got := parent.containingSize.Inline; got != definiteSize(240) {
		t.Errorf("parent inline basis changed to %+v", got)
	}
	if got := parent.containingSize.Block; got != definiteSize(120) {
		t.Errorf("parent block basis changed to %+v", got)
	}
	if got := parent.marginLeft; got != 12 {
		t.Errorf("parent left margin changed to %g", got)
	}
}
