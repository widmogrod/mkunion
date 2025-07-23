package example

import (
	"fmt"
)

// --8<-- [start:example]

//go:tag mkunion:"Option"
type (
	None[T any] struct{}
	Some[T any] struct{ Value T }
)

//go:tag mkunion:"Result"
type (
	Ok[T, E any]  struct{ Value T }
	Err[T, E any] struct{ Error E }
)

type User struct{ Name string }

type APIError struct {
	Code    int
	Message string
}

// FetchResult combine unions for rich error handling
type FetchResult = Result[Option[User], APIError]

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

// --8<-- [end:example]

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
