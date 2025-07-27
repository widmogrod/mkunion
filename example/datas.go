package example

import (
	"fmt"
	. "github.com/widmogrod/mkunion/f"
)

// --8<-- [start:example]

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
