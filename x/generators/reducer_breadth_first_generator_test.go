package generators

import (
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/shape"
	"testing"
)

func TestReducerBreadthFirstGenerator(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	inferred, err := shape.InferFromFile("testutils/tree_example_lit.go")
	assert.NoError(t, err)

	g := NewReducerBreadthFirstGenerator(
		inferred.RetrieveUnion("Tree2"),
		NewHelper(WithPackageName("testutils")),
	)

	result, err := g.Generate()
	assert.NoError(t, err)
	assert.Equal(t, `// Code generated by mkunion. DO NOT EDIT.
package testutils

var _ Tree2Visitor = (*Tree2BreadthFirstVisitor[any])(nil)

type Tree2BreadthFirstVisitor[A any] struct {
	stop   bool
	result A
	reduce Tree2Reducer[A]

	queue         []Tree2
	visited       map[Tree2]bool
	shouldExecute map[Tree2]bool
}

func (d *Tree2BreadthFirstVisitor[A]) VisitBranch2(v *Branch2) any {
	d.queue = append(d.queue, v)
	d.queue = append(d.queue, v.Lit)
	for idx := range v.List {
		d.queue = append(d.queue, v.List[idx])
	}
	for idx, _ := range v.Map {
		d.queue = append(d.queue, v.Map[idx])
	}

	if d.shouldExecute[v] {
		d.shouldExecute[v] = false
		d.result, d.stop = d.reduce.ReduceBranch2(v, d.result)
	} else {
		d.execute()
	}
	return nil
}

func (d *Tree2BreadthFirstVisitor[A]) VisitLeaf2(v *Leaf2) any {
	d.queue = append(d.queue, v)

	if d.shouldExecute[v] {
		d.shouldExecute[v] = false
		d.result, d.stop = d.reduce.ReduceLeaf2(v, d.result)
	} else {
		d.execute()
	}
	return nil
}

func (d *Tree2BreadthFirstVisitor[A]) execute() {
	for len(d.queue) > 0 {
		if d.stop {
			return
		}

		i := d.pop()
		if d.visited[i] {
			continue
		}
		d.visited[i] = true
		d.shouldExecute[i] = true
		i.AcceptTree2(d)
	}

	return
}

func (d *Tree2BreadthFirstVisitor[A]) pop() Tree2 {
	i := d.queue[0]
	d.queue = d.queue[1:]
	return i
}

func ReduceTree2BreadthFirst[A any](r Tree2Reducer[A], v Tree2, init A) A {
	reducer := &Tree2BreadthFirstVisitor[A]{
		result:        init,
		reduce:        r,
		queue:         []Tree2{v},
		visited:       make(map[Tree2]bool),
		shouldExecute: make(map[Tree2]bool),
	}

	_ = v.AcceptTree2(reducer)

	return reducer.result
}
`, string(result))
}