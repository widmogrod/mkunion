package example

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var _ TreeVisitor = (*sumVisitor)(nil)

type sumVisitor struct{}

func (s sumVisitor) VisitBranch(v *Branch) any {
	return v.L.Accept(s).(int) + v.R.Accept(s).(int)
}

func (s sumVisitor) VisitLeaf(v *Leaf) any {
	return v.Value
}

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

type orderAgg struct {
	Order  []int
	Result int
}

func TestTreeSumUsingReducer(t *testing.T) {
	tree := &Branch{
		L: &Leaf{Value: 1},
		R: &Branch{
			L: &Branch{
				L: &Leaf{Value: 2},
				R: &Leaf{Value: 3},
			},
			R: &Leaf{Value: 4},
		},
	}

	var red TreeReducer[orderAgg] = &TreeDefaultReduction[orderAgg]{
		PanicOnFallback:      false,
		DefaultStopReduction: false,
		OnLeaf: func(x *Leaf, agg orderAgg) (orderAgg, bool) {
			return orderAgg{
				Order:  append(agg.Order, x.Value),
				Result: agg.Result + x.Value,
			}, false
		},
	}

	result := ReduceTreeDepthFirst(red, tree, orderAgg{})
	assert.Equal(t, 10, result.Result)
	assert.Equal(t, []int{1, 2, 3, 4}, result.Order)

	result = ReduceTreeBreatheFirst(red, tree, orderAgg{})
	assert.Equal(t, 10, result.Result)
	assert.Equal(t, []int{1, 4, 2, 3}, result.Order)
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
