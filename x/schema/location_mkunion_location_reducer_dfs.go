// Code generated by mkunion. DO NOT EDIT.
package schema

type (
	LocationReducer[A any] interface {
		ReduceLocationField(x *LocationField, agg A) (result A, stop bool)
		ReduceLocationIndex(x *LocationIndex, agg A) (result A, stop bool)
		ReduceLocationAnything(x *LocationAnything, agg A) (result A, stop bool)
	}
)

type LocationDepthFirstVisitor[A any] struct {
	stop   bool
	result A
	reduce LocationReducer[A]
}

var _ LocationVisitor = (*LocationDepthFirstVisitor[any])(nil)

func (d *LocationDepthFirstVisitor[A]) VisitLocationField(v *LocationField) any {
	d.result, d.stop = d.reduce.ReduceLocationField(v, d.result)
	if d.stop {
		return nil
	}

	return nil
}

func (d *LocationDepthFirstVisitor[A]) VisitLocationIndex(v *LocationIndex) any {
	d.result, d.stop = d.reduce.ReduceLocationIndex(v, d.result)
	if d.stop {
		return nil
	}

	return nil
}

func (d *LocationDepthFirstVisitor[A]) VisitLocationAnything(v *LocationAnything) any {
	d.result, d.stop = d.reduce.ReduceLocationAnything(v, d.result)
	if d.stop {
		return nil
	}

	return nil
}

func ReduceLocationDepthFirst[A any](r LocationReducer[A], v Location, init A) A {
	reducer := &LocationDepthFirstVisitor[A]{
		result: init,
		reduce: r,
	}

	_ = v.AcceptLocation(reducer)

	return reducer.result
}