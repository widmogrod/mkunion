// Code generated by mkunion. DO NOT EDIT.
package schema

var _ LocationReducer[any] = (*LocationDefaultReduction[any])(nil)

type (
	LocationDefaultReduction[A any] struct {
		PanicOnFallback      bool
		DefaultStopReduction bool
		OnLocationField      func(x *LocationField, agg A) (result A, stop bool)
		OnLocationIndex      func(x *LocationIndex, agg A) (result A, stop bool)
		OnLocationAnything   func(x *LocationAnything, agg A) (result A, stop bool)
	}
)

func (t *LocationDefaultReduction[A]) ReduceLocationField(x *LocationField, agg A) (result A, stop bool) {
	if t.OnLocationField != nil {
		return t.OnLocationField(x, agg)
	}
	if t.PanicOnFallback {
		panic("no fallback allowed on undefined ReduceBranch")
	}
	return agg, t.DefaultStopReduction
}

func (t *LocationDefaultReduction[A]) ReduceLocationIndex(x *LocationIndex, agg A) (result A, stop bool) {
	if t.OnLocationIndex != nil {
		return t.OnLocationIndex(x, agg)
	}
	if t.PanicOnFallback {
		panic("no fallback allowed on undefined ReduceBranch")
	}
	return agg, t.DefaultStopReduction
}

func (t *LocationDefaultReduction[A]) ReduceLocationAnything(x *LocationAnything, agg A) (result A, stop bool) {
	if t.OnLocationAnything != nil {
		return t.OnLocationAnything(x, agg)
	}
	if t.PanicOnFallback {
		panic("no fallback allowed on undefined ReduceBranch")
	}
	return agg, t.DefaultStopReduction
}