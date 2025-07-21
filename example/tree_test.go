package example

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/schema"
	"testing"
)

// --8<-- [start:example-sum-values]

func Example_treeSumValues() {
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

// --8<-- [end:example-sum-values]

// --8<-- [start:example-custom-agg]

type orderAgg struct {
	Order  []int
	Result int
}

func Example_treeCustomReduction() {
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

// --8<-- [end:example-custom-agg]

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

func TestMyNameMatch(t *testing.T) {
	leaf1 := &Leaf[any]{Value: 1}
	leaf2 := &Leaf[any]{Value: 2}

	result := treeDoNumbers(leaf1, leaf2)
	assert.Equal(t, 3, result)

	branch1 := &Branch[any]{L: leaf1, R: leaf2}

	result = treeDoNumbers(branch1, leaf1)
	assert.Equal(t, -1, result)
}
