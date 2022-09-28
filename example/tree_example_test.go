package example

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

//go:generate go run ../cmd/mkunion/main.go golang -name=Tree -types=Branch,Leaf -output=tree_example_gen_test.go -package=example
type (
	Branch struct{ L, R Tree }
	Leaf   struct{ Value int }
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
