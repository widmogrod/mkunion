package example

import (
	"fmt"
	. "github.com/widmogrod/mkunion/f"
)

// --8<-- [start:example]

// Note: Option[T] and Result[T, E] types are imported from the f package via dot import

type User struct{ Name string }

type APIError struct {
	Code    int
	Message string
}

// FetchResult combine unions for rich error handling
type FetchResult = Result[Option[User], APIError]

// HandleFetch uses nested pattern matching to handle result (exported for testing)
func HandleFetch(result FetchResult) string {
	return MatchResultR1(result,
		func(ok *Ok[Option[User], APIError]) string {
			return MatchOptionR1(ok.Value,
				func(some *Some[User]) string {
					return fmt.Sprintf("Found user: %s", some.Value.Name)
				},
				func(*None[User]) string { return "User not found" },
			)
		},
		func(err *Err[Option[User], APIError]) string {
			return fmt.Sprintf("API error: %v", err.Error)
		},
	)
}

// --8<-- [end:example]
