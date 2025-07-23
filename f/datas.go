package f

// --8<-- [start:either]

//go:tag mkunion:"Either,serde"
type (
	Left[A, B any]  struct{ Value A }
	Right[A, B any] struct{ Value B }
)

// --8<-- [end:either]
// --8<-- [start:option]

// Option type - represent nullable values explicitly
//
//go:tag mkunion:"Option"
type (
	Some[A any] struct{ Value A }
	None[A any] struct{}
)

// --8<-- [end:option]
// --8<-- [start:result]

// Result type - explicit error handling without exceptions
//
//go:tag mkunion:"Result"
type (
	Ok[A any, E any]  struct{ Value A }
	Err[A any, E any] struct{ Error E }
)

// --8<-- [end:result]
// --8<-- [start:map-either]

func MapEither[A, B, C any](x Either[A, B], f func(A) C) Either[C, B] {
	return MatchEitherR1(
		x,
		func(x *Left[A, B]) Either[C, B] {
			return &Left[C, B]{Value: f(x.Value)}
		},
		func(x *Right[A, B]) Either[C, B] {
			return &Right[C, B]{Value: x.Value}
		},
	)
}

// --8<-- [end:map-either]

func OrElseEither[A, B any](x Either[A, B], y Either[A, B]) Either[A, B] {
	return MatchEitherR1(
		x,
		func(x *Left[A, B]) Either[A, B] {
			return x
		},
		func(x *Right[A, B]) Either[A, B] {
			return y
		},
	)
}

// --8<-- [start:map-option]

func MapOption[A, B any](x Option[A], f func(A) B) Option[B] {
	return MatchOptionR1(
		x,
		func(x *Some[A]) Option[B] {
			return &Some[B]{Value: f(x.Value)}
		},
		func(x *None[A]) Option[B] {
			return &None[B]{}
		},
	)
}

// --8<-- [end:map-option]

func MapResult[A, E any, C any](x Result[A, E], f func(A) C) Result[C, E] {
	return MatchResultR1(
		x,
		func(x *Ok[A, E]) Result[C, E] {
			return &Ok[C, E]{Value: f(x.Value)}
		},
		func(x *Err[A, E]) Result[C, E] {
			return &Err[C, E]{Error: x.Error}
		},
	)
}

//
//func Map[
//	Z *Left[C, B] | *Right[C, B] | *Ok[C, E] | *Err[C, E] | *Some[C] | *None[C],
//	X *Left[A, B] | *Right[A, B] | *Ok[A, E] | *Err[A, E] | *Some[A] | *None[A],
//	A, B, C any, E any,
//](x X, f func(A) C) Z {
//	switch y := any(x).(type) {
//	case *Left[A, B]:
//		return MapEither[A, B, C](y, f).(Z)
//	case *Right[A, B]:
//		return MapEither[A, B, C](y, f).(Z)
//	case *Ok[A, E]:
//		return MapResult[A, E, C](y, f).(Z)
//	case *Err[A, E]:
//		return MapResult[A, E, C](y, f).(Z)
//	case *Some[A]:
//		return MapOption[A, C](y, f).(Z)
//	case *None[A]:
//		return MapOption[A, C](y, f).(Z)
//	}
//
//	panic(fmt.Errorf("f.Map: unexpected type %T", x))
//}

func MkOk[E, A any](x A) Result[A, E] {
	return &Ok[A, E]{Value: x}
}

func MkErr[A, E any](err E) Result[A, E] {
	return &Err[A, E]{Error: err}
}

func MkSome[A any](v A) Option[A] {
	return &Some[A]{Value: v}
}

func MkNone[A any]() Option[A] {
	return &None[A]{}
}
