package example

//go:generate go run ../cmd/mkunion/main.go -name=Tree -types=Branch,Leaf
type (
	Branch struct{ L, R Tree }
	Leaf   struct{ Value int }
)

//var _ TreeVisitor = (*TreeBreatheFirstVisitor[any])(nil)
//
//type TreeBreatheFirstVisitor[A any] struct {
//	stop   bool
//	result A
//	reduce TreeReducer[A]
//
//	queue         []Tree
//	visited       map[Tree]bool
//	shouldExecute map[Tree]bool
//}
//
//func (d *TreeBreatheFirstVisitor[A]) VisitBranch(v *Branch) any {
//	d.queue = append(d.queue, v.L, v.R)
//	if d.shouldExecute[v] {
//		d.shouldExecute[v] = false
//		d.result, d.stop = d.reduce.ReduceBranch(v, d.result)
//	} else {
//		d.execute()
//	}
//	return nil
//}
//
//func (d *TreeBreatheFirstVisitor[A]) VisitLeaf(v *Leaf) any {
//	d.queue = append(d.queue, v)
//	if d.shouldExecute[v] {
//		d.shouldExecute[v] = false
//		d.result, d.stop = d.reduce.ReduceLeaf(v, d.result)
//	} else {
//		d.execute()
//	}
//	return nil
//}
//
//func (d *TreeBreatheFirstVisitor[A]) execute() {
//	for len(d.queue) > 0 {
//		if d.stop {
//			return
//		}
//
//		i := d.pop()
//		if d.visited[i] {
//			continue
//		}
//		d.visited[i] = true
//		d.shouldExecute[i] = true
//		i.Accept(d)
//	}
//
//	return
//}
//
//func (d *TreeBreatheFirstVisitor[A]) pop() Tree {
//	i := d.queue[0]
//	d.queue = d.queue[1:]
//	return i
//}
//
//func ReduceTreeBreatheFirst[A any](r TreeReducer[A], v Tree, init A) A {
//	reducer := &TreeBreatheFirstVisitor[A]{
//		result:        init,
//		reduce:        r,
//		queue:         []Tree{v},
//		visited:       make(map[Tree]bool),
//		shouldExecute: make(map[Tree]bool),
//	}
//
//	_ = v.Accept(reducer)
//
//	return reducer.result
//}
