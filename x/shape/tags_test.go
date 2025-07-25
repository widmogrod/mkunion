package shape

import (
	"go/ast"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractTags(t *testing.T) {
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
		{
			name:  "simple generic",
			input: `mkunion:"Option[T]"`,
			expected: map[string]Tag{
				"mkunion": {Value: "Option[T]", Options: nil},
			},
		},
		{
			name:  "generic with multiple params",
			input: `mkunion:"Either[A, B]"`,
			expected: map[string]Tag{
				"mkunion": {Value: "Either[A, B]", Options: nil},
			},
		},
		{
			name:  "generic with options",
			input: `mkunion:"Either[A, B],serde,no-type-registry"`,
			expected: map[string]Tag{
				"mkunion": {Value: "Either[A, B]", Options: []string{"serde", "no-type-registry"}},
			},
		},
		{
			name:  "nested generics",
			input: `mkunion:"Tree[Node[K, V]]"`,
			expected: map[string]Tag{
				"mkunion": {Value: "Tree[Node[K, V]]", Options: nil},
			},
		},
		{
			name:  "complex nested generics",
			input: `sometag:"Map[String, List[Option[T]]]"`,
			expected: map[string]Tag{
				"sometag": {Value: "Map[String, List[Option[T]]]", Options: nil},
			},
		},
		// Non-mkunion tags with generics
		{
			name:  "shape tag with generics",
			input: `shape:"Container[T]"`,
			expected: map[string]Tag{
				"shape": {Value: "Container[T]", Options: nil},
			},
		},
		{
			name:  "custom tag with generics and options",
			input: `mytag:"Handler[Request, Response],async,retry"`,
			expected: map[string]Tag{
				"mytag": {Value: "Handler[Request, Response]", Options: []string{"async", "retry"}},
			},
		},
		{
			name:  "tags with spaces",
			input: `desc:"The city and state e.g. San Francisco, CA" name:"location"`,
			expected: map[string]Tag{
				"desc": {Value: "The city and state e.g. San Francisco", Options: []string{"CA"}},
				"name": {Value: "location", Options: nil},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractTags(tt.input)

			require.Equal(t, len(tt.expected), len(result),
				"Expected %d tags, got %d. Result: %+v", len(tt.expected), len(result), result)

			for key, expectedTag := range tt.expected {
				require.Containsf(t, result, key, "Missing expected tag: %s", key)
				actualTag := result[key]
				assert.Equal(t, expectedTag.Value, actualTag.Value,
					"Tag %s value mismatch", key)
				assert.Equal(t, expectedTag.Options, actualTag.Options,
					"Tag %s options mismatch", key)
			}
		})
	}
}

func TestExtractDocumentTags(t *testing.T) {
	tests := []struct {
		name     string
		comments []string
		expected map[string]Tag
	}{
		{
			name:     "generic mkunion tag",
			comments: []string{"//go:tag mkunion:\"Option[T]\""},
			expected: map[string]Tag{
				"mkunion": {Value: "Option[T]", Options: nil},
			},
		},
		{
			name:     "multiple generic tags",
			comments: []string{"//go:tag type1:\"Map[K, V]\" type2:\"List[T]\""},
			expected: map[string]Tag{
				"type1": {Value: "Map[K, V]", Options: nil},
				"type2": {Value: "List[T]", Options: nil},
			},
		},
		{
			name: "multiline tags",
			comments: []string{
				"//go:tag mkunion:\"Either[A, B],serde\"",
				"//go:tag json:\",omitempty\"",
			},
			expected: map[string]Tag{
				"mkunion": {Value: "Either[A, B]", Options: []string{"serde"}},
				"json":    {Value: "", Options: []string{"omitempty"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commentList := make([]*ast.Comment, len(tt.comments))
			for i, c := range tt.comments {
				commentList[i] = &ast.Comment{Text: c}
			}

			commentGroup := &ast.CommentGroup{List: commentList}
			result := ExtractDocumentTags(commentGroup)

			assert.Equal(t, len(tt.expected), len(result),
				"Expected %d tags, got %d", len(tt.expected), len(result))

			for key, expectedTag := range tt.expected {
				require.Containsf(t, result, key, "Missing expected tag: %s", key)
				actualTag := result[key]
				assert.Equal(t, expectedTag.Value, actualTag.Value,
					"Tag %s value mismatch", key)
				assert.Equal(t, expectedTag.Options, actualTag.Options,
					"Tag %s options mismatch", key)
			}
		})
	}
}

func TestFindTagValueEndUnified(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		startIdx int
		expected int
	}{
		{
			name:     "no brackets, comma",
			input:    "value,option",
			startIdx: 0,
			expected: 5,
		},
		{
			name:     "no brackets, quote",
			input:    `value"`,
			startIdx: 0,
			expected: 5,
		},
		{
			name:     "brackets with comma inside",
			input:    "Type[A, B],option",
			startIdx: 0,
			expected: 10,
		},
		{
			name:     "nested brackets",
			input:    "Outer[Inner[A, B], C],opt",
			startIdx: 0,
			expected: 21,
		},
		{
			name:     "unbalanced brackets",
			input:    "Bad[A, B",
			startIdx: 0,
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findTagValueEndUnified(tt.input, tt.startIdx)
			assert.Equal(t, tt.expected, result,
				"findTagValueEndUnified(%q, %d) = %d, want %d",
				tt.input, tt.startIdx, result, tt.expected)
		})
	}
}
