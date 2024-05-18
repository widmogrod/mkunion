package example

import (
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/widmogrod/mkunion/x/shared"
)

func ExampleTreeJson() {
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

	json, _ := shared.JSONMarshal[Tree[int]](tree)
	result, _ := shared.JSONUnmarshal[Tree[int]](json)

	fmt.Println(string(json))
	if diff := cmp.Diff(tree, result); diff != "" {
		fmt.Println("expected tree and result to be equal, but got diff:", diff)
	}
	//Output: {"$type":"example.Branch","example.Branch":{"L":{"$type":"example.Leaf","example.Leaf":{"Value":1}},"R":{"$type":"example.Branch","example.Branch":{"L":{"$type":"example.Branch","example.Branch":{"L":{"$type":"example.Leaf","example.Leaf":{"Value":2}},"R":{"$type":"example.Leaf","example.Leaf":{"Value":3}}}},"R":{"$type":"example.Leaf","example.Leaf":{"Value":4}}}}}}
}

//func TestMyTriesMatchR0(t *testing.T) {
//	MyTriesMatchR0(
//		&Leaf{Value: 1}, &Leaf{Value: 3},
//		func(x *Leaf, y *Leaf) {
//			assert.Equal(t, x.Value, 1)
//			assert.Equal(t, y.Value, 3)
//		},
//		func(x0 *Branch, x1 any) {
//			assert.Fail(t, "should not match")
//		},
//		func(x0 any, x1 any) {
//			assert.Fail(t, "should not match")
//		},
//	)
//}
