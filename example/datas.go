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

type SimpleUser struct{ Name string }

type SimpleAPIError struct {
	Code    int
	Message string
}

// FetchResult combine unions for rich error handling
type SimpleFetchResult = Result[Option[SimpleUser], SimpleAPIError]

// handleFetch uses nested pattern matching to handle result
func handleFetch(result SimpleFetchResult) string {
	return MatchResultR1(result,
		func(ok *Ok[Option[SimpleUser], SimpleAPIError]) string {
			return MatchOptionR1(ok.Value,
				func(*None[SimpleUser]) string { return "User not found" },
				func(some *Some[SimpleUser]) string {
					return fmt.Sprintf("Found user: %s", some.Value.Name)
				},
			)
		},
		func(err *Err[Option[SimpleUser], SimpleAPIError]) string {
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
