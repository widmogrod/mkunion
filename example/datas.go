package example

import (
	"fmt"
)

// --8<-- [start:example]

//go:tag mkunion:"Option[T]"
type (
	None[T any] struct{}
	Some[T any] struct{ Value T }
)

//go:tag mkunion:"Result[T, E]"
type (
	Ok[T, E any]  struct{ Value T }
	Err[T, E any] struct{ Error E }
)

type User struct{ Name string }

type APIError struct {
	Code    int
	Message string
}

// --8<-- [start:fetch-type]

// FetchResult combine unions for rich error handling
type FetchResult = Result[Option[User], APIError]

// --8<-- [end:fetch-type]
// --8<-- [start:fetch-handling]

// handleFetch uses nested pattern matching to handle result
func handleFetch(result FetchResult) string {
	return MatchResultR1(result,
		func(ok *Ok[Option[User], APIError]) string {
			return MatchOptionR1(ok.Value,
				func(*None[User]) string { return "User not found" },
				func(some *Some[User]) string {
					return fmt.Sprintf("Found user: %s", some.Value.Name)
				},
			)
		},
		func(err *Err[Option[User], APIError]) string {
			return fmt.Sprintf("API error: %v", err.Error)
		},
	)
}

// --8<-- [end:fetch-handling]
// --8<-- [end:example]

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

// Transforming Option values
func MapOption[A, B any](opt Option[A], f func(A) B) Option[B] {
	return MatchOptionR1(opt,
		func(*None[A]) Option[B] { return MkNone[B]() },
		func(some *Some[A]) Option[B] {
			return MkSome(f(some.Value))
		},
	)
}

// Chaining operations
func FlatMapOption[A, B any](opt Option[A], f func(A) Option[B]) Option[B] {
	return MatchOptionR1(opt,
		func(*None[A]) Option[B] { return MkNone[B]() },
		func(some *Some[A]) Option[B] { return f(some.Value) },
	)
}

// --8<-- [end:option-algebra]

// Converting Result to Option
func ResultToOption[T, E any](r Result[T, E]) Option[T] {
	return MatchResultR1(r,
		func(ok *Ok[T, E]) Option[T] { return MkSome(ok.Value) },
		func(*Err[T, E]) Option[T] { return MkNone[T]() },
	)
}

// Mapping over success values
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

// Error recovery
func OrElse[T, E any](r Result[T, E], fallback func(E) Result[T, E]) Result[T, E] {
	return MatchResultR1(r,
		func(ok *Ok[T, E]) Result[T, E] { return ok },
		func(err *Err[T, E]) Result[T, E] { return fallback(err.Error) },
	)
}
