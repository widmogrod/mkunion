package example

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/schema"
	"testing"
)

func ExampleTreeSumValues() {
	tree := &Branch[int]{
		L: &Leaf[int]{Value: 1},
		R: &Branch[int]{
			L: &Branch[int]{
				L: &Leaf[int]{Value: 2},
				R: &Leaf[int]{Value: 3},
			},
			R: &Leaf[int]{Value: 4},
		},
	}

	result := ReduceTree(tree, func(x int, agg int) int {
		return agg + x
	}, 0)

	fmt.Println(result)
	// Output: 10
}

type orderAgg struct {
	Order  []int
	Result int
}

func ExampleTreeCustomReduction() {
	tree := &Branch[int]{
		L: &Leaf[int]{Value: 1},
		R: &Branch[int]{
			L: &Branch[int]{
				L: &Leaf[int]{Value: 2},
				R: &Leaf[int]{Value: 3},
			},
			R: &Leaf[int]{Value: 4},
		},
	}

	result := ReduceTree(tree, func(x int, agg orderAgg) orderAgg {
		return orderAgg{
			Order:  append(agg.Order, x),
			Result: agg.Result + x,
		}
	}, orderAgg{
		Order:  []int{},
		Result: 0,
	})
	fmt.Println(result.Order)
	fmt.Println(result.Result)
	// Output: [1 2 3 4]
	// 10
}

func TestTreeSchema(t *testing.T) {
	tree := &Branch[int]{
		L: &Leaf[int]{Value: 1},
		R: &Branch[int]{
			L: &Branch[int]{
				L: &Leaf[int]{Value: 2},
				R: &Leaf[int]{Value: 3},
			},
			R: &Leaf[int]{Value: 4},
		},
	}

	sch := schema.FromGo[Tree[int]](tree)
	result := schema.ToGo[Tree[int]](sch)
	assert.Equal(t, tree, result)
}
