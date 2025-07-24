package shape

import (
	"testing"
)

func TestExtractTags_MkMatchWithoutValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]Tag
	}{
		{
			name:  "mkmatch without value",
			input: "mkmatch",
			expected: map[string]Tag{
				"mkmatch": {Value: "", Options: nil},
			},
		},
		{
			name:  "mkmatch with value",
			input: `mkmatch:"CustomName"`,
			expected: map[string]Tag{
				"mkmatch": {Value: "CustomName", Options: nil},
			},
		},
		{
			name:  "mkmatch with dash",
			input: `mkmatch:"-"`,
			expected: map[string]Tag{
				"mkmatch": {Value: "-", Options: nil},
			},
		},
		{
			name:  "mkunion without value",
			input: "mkunion",
			expected: map[string]Tag{
				"mkunion": {Value: "", Options: nil},
			},
		},
		{
			name:  "multiple tags",
			input: `mkunion:"Example" json:"field,omitempty"`,
			expected: map[string]Tag{
				"mkunion": {Value: "Example", Options: nil},
				"json":    {Value: "field", Options: []string{"omitempty"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractTags(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("ExtractTags() returned %d tags, want %d", len(result), len(tt.expected))
			}

			for key, expectedTag := range tt.expected {
				if actualTag, ok := result[key]; !ok {
					t.Errorf("ExtractTags() missing key %q, got tags: %v", key, result)
				} else {
					if actualTag.Value != expectedTag.Value {
						t.Errorf("ExtractTags() tag %q has value %q, want %q", key, actualTag.Value, expectedTag.Value)
					}
					if len(actualTag.Options) != len(expectedTag.Options) {
						t.Errorf("ExtractTags() tag %q has %d options, want %d", key, len(actualTag.Options), len(expectedTag.Options))
					}
				}
			}
		})
	}
}
