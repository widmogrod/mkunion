package f

// --8<-- [start:either]

//go:tag mkunion:"Either[A, B],serde"
type (
	Left[A, B any]  struct{ Value A }
	Right[A, B any] struct{ Value B }
)

// --8<-- [end:either]
// --8<-- [start:option]

// Option type - represent nullable values explicitly
//
//go:tag mkunion:"Option[A]"
type (
	None[A any] struct{}
	Some[A any] struct{ Value A }
)

// --8<-- [end:option]
// --8<-- [start:result]

// Result type - explicit error handling without exceptions
//
//go:tag mkunion:"Result[A, E]"
type (
	Ok[A any, E any]  struct{ Value A }
	Err[A any, E any] struct{ Error E }
)

// --8<-- [end:result]

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

// --8<-- [start:mk-option]

// Helper functions for creating Options

func MkSome[A any](v A) Option[A] {
	return &Some[A]{Value: v}
}

func MkNone[A any]() Option[A] {
	return &None[A]{}
}

// --8<-- [end:mk-option]
// --8<-- [start:option-algebra]

// Providing default values
func GetOrDefault[T any](opt Option[T], defaultValue T) T {
	return MatchOptionR1(opt,
		func(*None[T]) T { return defaultValue },
		func(some *Some[T]) T { return some.Value },
	)
}

// --8<-- [start:map-option]

func MapOption[A, B any](x Option[A], f func(A) B) Option[B] {
	return MatchOptionR1(
		x,
		func(x *None[A]) Option[B] {
			return &None[B]{}
		},
		func(x *Some[A]) Option[B] {
			return &Some[B]{Value: f(x.Value)}
		},
	)
}

// --8<-- [end:map-option]

func FlatMapOption[A, B any](opt Option[A], f func(A) Option[B]) Option[B] {
	return MatchOptionR1(opt,
		func(*None[A]) Option[B] { return MkNone[B]() },
		func(some *Some[A]) Option[B] { return f(some.Value) },
	)
}

// --8<-- [end:option-algebra]

func ResultToOption[T, E any](r Result[T, E]) Option[T] {
	return MatchResultR1(r,
		func(ok *Ok[T, E]) Option[T] { return MkSome(ok.Value) },
		func(*Err[T, E]) Option[T] { return MkNone[T]() },
	)
}

func MapResult[A, B, E any](r Result[A, E], f func(A) B) Result[B, E] {
	return MatchResultR1(r,
		func(ok *Ok[A, E]) Result[B, E] {
			return MkOk[E](f(ok.Value))
		},
		func(err *Err[A, E]) Result[B, E] {
			return MkErr[B](err.Error)
		},
	)
}

func OrElse[T, E any](r Result[T, E], fallback func(E) Result[T, E]) Result[T, E] {
	return MatchResultR1(r,
		func(ok *Ok[T, E]) Result[T, E] { return ok },
		func(err *Err[T, E]) Result[T, E] { return fallback(err.Error) },
	)
}

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
