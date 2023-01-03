// Code generated by mkunion. DO NOT EDIT.
package example

var _ WherePredicateVisitor = (*WherePredicateBreadthFirstVisitor[any])(nil)

type WherePredicateBreadthFirstVisitor[A any] struct {
	stop   bool
	result A
	reduce WherePredicateReducer[A]

	queue         []WherePredicate
	visited       map[WherePredicate]bool
	shouldExecute map[WherePredicate]bool
}

func (d *WherePredicateBreadthFirstVisitor[A]) VisitEq(v *Eq) any {
	d.queue = append(d.queue, v)

	if d.shouldExecute[v] {
		d.shouldExecute[v] = false
		d.result, d.stop = d.reduce.ReduceEq(v, d.result)
	} else {
		d.execute()
	}
	return nil
}

func (d *WherePredicateBreadthFirstVisitor[A]) VisitAnd(v *And) any {
	d.queue = append(d.queue, v)

	if d.shouldExecute[v] {
		d.shouldExecute[v] = false
		d.result, d.stop = d.reduce.ReduceAnd(v, d.result)
	} else {
		d.execute()
	}
	return nil
}

func (d *WherePredicateBreadthFirstVisitor[A]) VisitOr(v *Or) any {
	d.queue = append(d.queue, v)

	if d.shouldExecute[v] {
		d.shouldExecute[v] = false
		d.result, d.stop = d.reduce.ReduceOr(v, d.result)
	} else {
		d.execute()
	}
	return nil
}

func (d *WherePredicateBreadthFirstVisitor[A]) VisitPath(v *Path) any {
	d.queue = append(d.queue, v)
	d.queue = append(d.queue, v.Condition)
	for idx := range v.Then {
		d.queue = append(d.queue, v.Then[idx])
	}
	for idx, _ := range v.Y {
		d.queue = append(d.queue, v.Y[idx])
	}

	if d.shouldExecute[v] {
		d.shouldExecute[v] = false
		d.result, d.stop = d.reduce.ReducePath(v, d.result)
	} else {
		d.execute()
	}
	return nil
}

func (d *WherePredicateBreadthFirstVisitor[A]) execute() {
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
		i.Accept(d)
	}

	return
}

func (d *WherePredicateBreadthFirstVisitor[A]) pop() WherePredicate {
	i := d.queue[0]
	d.queue = d.queue[1:]
	return i
}

func ReduceWherePredicateBreadthFirst[A any](r WherePredicateReducer[A], v WherePredicate, init A) A {
	reducer := &WherePredicateBreadthFirstVisitor[A]{
		result:        init,
		reduce:        r,
		queue:         []WherePredicate{v},
		visited:       make(map[WherePredicate]bool),
		shouldExecute: make(map[WherePredicate]bool),
	}

	_ = v.Accept(reducer)

	return reducer.result
}