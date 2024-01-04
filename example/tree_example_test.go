package example

import (
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/schema"
	"testing"
)

var _ TreeVisitor = (*sumVisitor)(nil)

type sumVisitor struct{}

func (s sumVisitor) VisitBranch(v *Branch) any {
	return v.L.AcceptTree(s).(int) + v.R.AcceptTree(s).(int)
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

	assert.Equal(t, 6, tree.AcceptTree(&sumVisitor{}))
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

	result := ReduceTreeF(tree, func(x int, agg orderAgg) orderAgg {
		return orderAgg{
			Order:  append(agg.Order, x),
			Result: agg.Result + x,
		}
	}, orderAgg{
		Order:  []int{},
		Result: 0,
	})
	assert.Equal(t, 10, result.Result)
}

func ReduceTreeF[B any](x Tree, f func(int, B) B, init B) B {
	return MatchTreeR1(
		x,
		func(x *Branch) B {
			return ReduceTreeF(x.L, f, ReduceTreeF(x.R, f, init))
		}, func(x *Leaf) B {
			return f(x.Value, init)
		},
	)
}

func TestTreeSchema(t *testing.T) {
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

	sch := schema.FromGo[Tree](tree)
	result := schema.ToGo[Tree](sch)
	assert.Equal(t, tree, result)
}

func TestMyTriesMatchR0(t *testing.T) {
	MyTriesMatchR0(
		&Leaf{Value: 1}, &Leaf{Value: 3},
		func(x *Leaf, y *Leaf) {
			assert.Equal(t, x.Value, 1)
			assert.Equal(t, y.Value, 3)
		},
		func(x0 *Branch, x1 any) {
			assert.Fail(t, "should not match")
		},
		func(x0 any, x1 any) {
			assert.Fail(t, "should not match")
		},
	)
}
