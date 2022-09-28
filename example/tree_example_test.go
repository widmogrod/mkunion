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

func TestTree(t *testing.T) {
	tree := &Branch{
		L: &Leaf{Value: 1},
		R: &Branch{
			L: &Leaf{Value: 2},
			R: &Leaf{Value: 3},
		},
	}

	assert.Equal(t, 6, tree.Accept(&sumVisitor{}))
}

//func TestTreeReducer(t *testing.T) {
//	tree := &Branch{
//		L: &Leaf{Value: 1},
//		R: &Branch{
//			L: &Leaf{Value: 2},
//			R: &Leaf{Value: 3},
//		},
//	}
//
//	red := TreeReducer[int]{
//		Branch: func(x *Branch, agg int) (int, bool) {
//			return agg, false
//		},
//		Leaf: func(x *Leaf, agg int) (int, bool) {
//			return agg + 1, false
//		},
//	}
//
//	result := ReduceTree(red, tree, 0)
//	assert.Equal(t, 6, result)
//}

//func TestTreeNonExhaustive(t *testing.T) {
//	tree := &Branch{
//		L: &Leaf{Value: 1},
//		R: &Branch{
//			L: &Leaf{Value: 2},
//			R: &Leaf{Value: 3},
//		},
//	}
//
//	n := TreeNonExhaustiveG[int]{
//		Default: 10,
//		OnLeaf: func(x *Leaf) int {
//			return x.Value
//		},
//	}
//
//	assert.Equal(t, 11, tree.Accept(&n))
//}
