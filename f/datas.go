package f

//go:generate go run ../cmd/mkunion --type-registry

//go:tag mkunion:"Either,serde"
type (
	Left[A, B any]  struct{ Value A }
	Right[A, B any] struct{ Value B }
)

//go:tag mkunion:"Option"
type (
	Some[A any] struct{ Value A }
	None[A any] struct{}
)

//go:tag mkunion:"Result"
type (
	Ok[A any, E any]  struct{ Value A }
	Err[A any, E any] struct{ Value E }
)

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

func MapResult[A, E any, C any](x Result[A, E], f func(A) C) Result[C, E] {
	return MatchResultR1(
		x,
		func(x *Ok[A, E]) Result[C, E] {
			return &Ok[C, E]{Value: f(x.Value)}
		},
		func(x *Err[A, E]) Result[C, E] {
			return &Err[C, E]{Value: x.Value}
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
