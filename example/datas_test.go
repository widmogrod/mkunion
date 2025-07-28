package example

import (
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/shared"
	"testing"

	"github.com/stretchr/testify/assert"
	. "github.com/widmogrod/mkunion/f"
)

func TestHandleFetch(t *testing.T) {
	tests := []struct {
		name     string
		input    FetchResult
		expected string
	}{
		{
			name:     "UserFound",
			input:    MkOk[APIError](MkSome(User{Name: "Alice"})),
			expected: "Found user: Alice",
		},
		{
			name:     "UserNotFound",
			input:    MkOk[APIError](MkNone[User]()),
			expected: "User not found",
		},
		{
			name:     "APIErrorOccurred",
			input:    MkErr[Option[User]](APIError{Code: 500, Message: "Internal Server Error"}),
			expected: "API error: {500 Internal Server Error}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handleFetch(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOptionUsage(t *testing.T) {
	// --8<-- [start:mk-option-use]

	// Usage examples
	userOpt := MkSome(User{Name: "Alice"}) // Some[User]
	// emptyOpt := MkNone[User]()             // None[User]

	// Safe access with pattern matching
	message := MatchOptionR1(userOpt,
		func(*None[User]) string {
			return "No user found"
		},
		func(some *Some[User]) string {
			return fmt.Sprintf("Hello, %s!", some.Value.Name)
		},
	)
	assert.Equal(t, "Hello, Alice!", message)
	// --8<-- [end:mk-option-use]
}

// Usage examples
func parseUser(data string) Result[User, string] {
	if data == "" {
		return MkErr[User]("empty input")
	}
	// Parse logic here
	return MkOk[string](User{Name: data})
}

func TestOptionComplexUse(t *testing.T) {
	result := parseUser("Alice")
	output := MatchResultR1(result,
		func(ok *Ok[User, string]) string {
			return fmt.Sprintf("Parsed user: %s", ok.Value.Name)
		},
		func(err *Err[User, string]) string {
			return fmt.Sprintf("Parse error: %s", err.Error)
		},
	)
	require.Equal(t, "Parsed user: Alice", output)
}

// --8<-- [start:creating-fetch]
func TestCreatingUnions(t *testing.T) {
	// Success with user
	fetchSuccess := MkOk[APIError](MkSome(User{Name: "Alice"}))

	// Success but user not found
	fetchNotFound := MkOk[APIError](MkNone[User]())

	// API error
	fetchError := MkErr[Option[User]](APIError{
		Code:    500,
		Message: "Internal Server Error",
	})

	assert.Equal(t, "Found user: Alice", handleFetch(fetchSuccess))
	assert.Equal(t, "User not found", handleFetch(fetchNotFound))
	assert.Equal(t, "API error: {500 Internal Server Error}", handleFetch(fetchError))
}

// --8<-- [end:creating-fetch]

func TestUnmarshaling(t *testing.T) {
	fetchSuccess := MkOk[APIError](MkSome(User{Name: "Alice"}))

	// First, let's test that marshaling works
	data, err := shared.JSONMarshal[Result[Option[User], APIError]](fetchSuccess)
	require.NoError(t, err)
	t.Log(string(data))

	// Then test unmarshalling
	result, err := shared.JSONUnmarshal[Result[Option[User], APIError]](data)
	require.NoError(t, err)
	t.Log(result)

	if diff := cmp.Diff(fetchSuccess, result); diff != "" {
		t.Fatalf("mismatch (-want +got):\n%s", diff)
	}
}
