package example

import (
	"fmt"
	"github.com/widmogrod/mkunion/f"
)

// --8<-- [start:example]

// Note: Option[T] and Result[T, E] types are imported from the f package via dot import

type User struct{ Name string }

type APIError struct {
	Code    int
	Message string
}

// FetchResult combine unions for rich error handling
type FetchResult = f.Result[f.Option[User], APIError]

// HandleFetch uses nested pattern matching to handle result (exported for testing)
func HandleFetch(result FetchResult) string {
	return f.MatchResultR1(result,
		func(ok *f.Ok[f.Option[User], APIError]) string {
			return f.MatchOptionR1(ok.Value,
				func(some *f.Some[User]) string {
					return fmt.Sprintf("Found user: %s", some.Value.Name)
				},
				func(*f.None[User]) string { return "User not found" },
			)
		},
		func(err *f.Err[f.Option[User], APIError]) string {
			return fmt.Sprintf("API error: %v", err.Error)
		},
	)
}

// --8<-- [end:example]
