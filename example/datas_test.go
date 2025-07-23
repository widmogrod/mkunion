package example

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
