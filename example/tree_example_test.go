package example

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type sumVisitor struct{}

func (s sumVisitor) VisitBranch(v *Branch) any {
	return v.L.Accept(s).(int) + v.R.Accept(s).(int)
}

func (s sumVisitor) VisitLeaf(v *Leaf) any {
	return v.Value
}

var _ TreeVisitor = (*sumVisitor)(nil)

func TestTreeSumValues(t *testing.T) {
	tree := &Branch{
		L: &Leaf{Value: 1},
		R: &Branch{
			L: &Leaf{Value: 2},
			R: &Leaf{Value: 3},
		},
	}

	assert.Equal(t, 6, tree.Accept(&sumVisitor{}))
}

func TestTreeSumUsingReducer(t *testing.T) {
	tree := &Branch{
		L: &Leaf{Value: 1},
		R: &Branch{
			L: &Leaf{Value: 2},
			R: &Leaf{Value: 3},
		},
	}

	var red TreeReducer[int] = &TreeDefaultReduction[int]{
		PanicOnFallback:      false,
		DefaultStopReduction: false,
		//OnBranch: func(x *Branch, agg int) (result int, stop bool) {
		//	return agg, false
		//},
		OnLeaf: func(x *Leaf, agg int) (int, bool) {
			return agg + x.Value, false
		},
	}

	result := ReduceTree(red, tree, 0)
	assert.Equal(t, 6, result)
}

func TestTreeNonExhaustive(t *testing.T) {
	tree := &Branch{
		L: &Leaf{Value: 1},
		R: &Branch{
			L: &Leaf{Value: 2},
			R: &Leaf{Value: 3},
		},
	}

	n := TreeDefaultVisitor[int]{
		Default: 10,
		OnLeaf: func(x *Leaf) int {
			return x.Value
		},
	}

	assert.Equal(t, 10, tree.Accept(&n))
}
