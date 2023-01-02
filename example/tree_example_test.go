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

	result = ReduceTreeBreadthFirst(red, tree, orderAgg{})
	assert.Equal(t, 10, result.Result)
	assert.Equal(t, []int{1, 4, 2, 3}, result.Order)

	result = ReduceTreeF(tree, func(x int, agg orderAgg) orderAgg {
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
	return MustMatchTree(
		x,
		func(x *Branch) B {
			return ReduceTreeF(x.L, f, ReduceTreeF(x.R, f, init))
		}, func(x *Leaf) B {
			return f(x.Value, init)
		},
	)
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

/*
Function reduction is faster than depth first, and depth first is faster than breadth first.
But function vs depth first is not that big difference.

BenchmarkReduceTreeBreadthFirst-8        1000000              1096 ns/op            1024 B/op         13 allocs/op
BenchmarkReduceTreeBreadthFirst-8        1000000              1090 ns/op            1024 B/op         13 allocs/op
BenchmarkReduceTreeBreadthFirst-8        1000000              1090 ns/op            1024 B/op         13 allocs/op
BenchmarkReduceTreeBreadthFirst-8        1000000              1086 ns/op            1024 B/op         13 allocs/op
BenchmarkReduceTreeBreadthFirst-8        1000000              1092 ns/op            1024 B/op         13 allocs/op
BenchmarkReduceTreeDepthFirst-8         20942499                57.10 ns/op           32 B/op          1 allocs/op
BenchmarkReduceTreeDepthFirst-8         20936881                57.75 ns/op           32 B/op          1 allocs/op
BenchmarkReduceTreeDepthFirst-8         20820109                57.58 ns/op           32 B/op          1 allocs/op
BenchmarkReduceTreeDepthFirst-8         20699052                57.60 ns/op           32 B/op          1 allocs/op
BenchmarkReduceTreeDepthFirst-8         20935268                57.53 ns/op           32 B/op          1 allocs/op
BenchmarkReduceTreeF-8                  26585260                44.96 ns/op            0 B/op          0 allocs/op
BenchmarkReduceTreeF-8                  26453421                45.33 ns/op            0 B/op          0 allocs/op
BenchmarkReduceTreeF-8                  26363992                45.13 ns/op            0 B/op          0 allocs/op
BenchmarkReduceTreeF-8                  26396396                48.38 ns/op            0 B/op          0 allocs/op
BenchmarkReduceTreeF-8                  26441884                45.18 ns/op            0 B/op          0 allocs/op
*/
var (
	benchTreeResult   int
	benchTreeExpected = 10
	benchTree         = &Branch{
		L: &Leaf{Value: 1},
		R: &Branch{
			L: &Branch{
				L: &Leaf{Value: 2},
				R: &Leaf{Value: 3},
			},
			R: &Leaf{Value: 4},
		},
	}

	benchTreeReducer TreeReducer[int] = &TreeDefaultReduction[int]{
		PanicOnFallback:      false,
		DefaultStopReduction: false,
		OnLeaf: func(x *Leaf, agg int) (int, bool) {
			return agg + x.Value, false
		},
	}
)

func BenchmarkReduceTreeBreadthFirst(b *testing.B) {
	var r int
	for i := 0; i < b.N; i++ {
		r = ReduceTreeBreadthFirst(benchTreeReducer, benchTree, 0)
		if r != benchTreeExpected {
			b.Fail()
		}
	}
	benchTreeResult = r
}

func BenchmarkReduceTreeDepthFirst(b *testing.B) {
	var r int
	for i := 0; i < b.N; i++ {
		r = ReduceTreeDepthFirst(benchTreeReducer, benchTree, 0)
		if r != benchTreeExpected {
			b.Fail()
		}
	}
	benchTreeResult = r
}

func BenchmarkReduceTreeF(b *testing.B) {
	var r int
	for i := 0; i < b.N; i++ {
		r = ReduceTreeF(benchTree, func(x int, agg int) int {
			return agg + x
		}, 0)
		if r != benchTreeExpected {
			b.Fail()
		}
	}
	benchTreeResult = r
}
