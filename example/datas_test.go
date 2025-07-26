package example

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandleFetch(t *testing.T) {
	tests := []struct {
		name     string
		input    SimpleFetchResult
		expected string
	}{
		{
			name:     "UserFound",
			input:    MkOk[SimpleAPIError](MkSome(SimpleUser{Name: "Alice"})),
			expected: "Found user: Alice",
		},
		{
			name:     "UserNotFound",
			input:    MkOk[SimpleAPIError](MkNone[SimpleUser]()),
			expected: "User not found",
		},
		{
			name:     "APIErrorOccurred",
			input:    MkErr[Option[SimpleUser]](SimpleAPIError{Code: 500, Message: "Internal Server Error"}),
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
