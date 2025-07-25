package shape

import (
	"go/parser"
	"go/token"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackageLevelTagExtraction(t *testing.T) {
	tests := []struct {
		name           string
		packageContent string
		expectedTags   map[string]Tag
	}{
		{
			name: "single package-level mkunion tag",
			packageContent: `//go:tag mkunion:",no-type-registry"
package example

type Example struct{}`,
			expectedTags: map[string]Tag{
				"mkunion": {Value: "", Options: []string{"no-type-registry"}},
			},
		},
		{
			name: "multiple package-level tags",
			packageContent: `//go:tag mkunion:",no-type-registry"
//go:tag custom:"value,option1,option2"
package example

type Example struct{}`,
			expectedTags: map[string]Tag{
				"mkunion": {Value: "", Options: []string{"no-type-registry"}},
				"custom":  {Value: "value", Options: []string{"option1", "option2"}},
			},
		},
		{
			name: "package-level tag with generic type syntax",
			packageContent: `//go:tag container:"Container[T]"
//go:tag config:"Config[A, B],immutable"
package example

type Example struct{}`,
			expectedTags: map[string]Tag{
				"container": {Value: "Container[T]", Options: nil},
				"config":    {Value: "Config[A, B]", Options: []string{"immutable"}},
			},
		},
		{
			name: "no package-level tags",
			packageContent: `package example

type Example struct{}`,
			expectedTags: nil,
		},
		{
			name: "package-level tag in separate comments",
			packageContent: `//go:tag module:"example"
//go:tag version:"1.0.0,stable"
package example

type Example struct{}`,
			expectedTags: map[string]Tag{
				"module":  {Value: "example", Options: nil},
				"version": {Value: "1.0.0", Options: []string{"stable"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "test.go", tt.packageContent, parser.ParseComments)
			require.NoError(t, err)

			// Extract package-level tags
			var packageTags map[string]Tag
			if f.Doc != nil {
				packageTags = ExtractDocumentTags(f.Doc)
			}

			if tt.expectedTags == nil {
				assert.Nil(t, packageTags)
				return
			}

			require.NotNil(t, packageTags)
			assert.Equal(t, len(tt.expectedTags), len(packageTags))

			for key, expectedTag := range tt.expectedTags {
				require.Contains(t, packageTags, key)
				actualTag := packageTags[key]
				assert.Equal(t, expectedTag.Value, actualTag.Value,
					"Tag %s value mismatch", key)
				assert.Equal(t, expectedTag.Options, actualTag.Options,
					"Tag %s options mismatch", key)
			}
		})
	}
}

func TestIndexedTypeWalkerPackageTags(t *testing.T) {
	content := `//go:tag mkunion:",no-type-registry"
//go:tag module:"testmodule"
//go:tag version:"1.0.0,stable,experimental"
package testpkg

type TestStruct struct {
	Field string
}

//go:tag mkunion:"TestUnion"
type (
	VariantA struct{ A int }
	VariantB struct{ B string }
)`

	walker := newIndexedTypeWalkerWithContentBody(content)

	packageTags := walker.PackageTags()
	require.NotNil(t, packageTags)

	expected := map[string]Tag{
		"mkunion": {Value: "", Options: []string{"no-type-registry"}},
		"module":  {Value: "testmodule", Options: nil},
		"version": {Value: "1.0.0", Options: []string{"stable", "experimental"}},
	}

	assert.Equal(t, len(expected), len(packageTags))

	for key, expectedTag := range expected {
		require.Contains(t, packageTags, key)
		actualTag := packageTags[key]
		assert.Equal(t, expectedTag.Value, actualTag.Value,
			"Tag %s value mismatch", key)
		assert.Equal(t, expectedTag.Options, actualTag.Options,
			"Tag %s options mismatch", key)
	}
}

func TestPackageTagsInDirectoryWalk(t *testing.T) {
	// This tests that package tags work with the actual directory walking functionality
	// Create a temporary directory structure for testing
	tempDir := t.TempDir()

	testFile := `//go:tag mkunion:",no-type-registry"
//go:tag version:"1.0.0"
package testpkg

type TestStruct struct {
	Field string
}`

	err := writeTestFile(tempDir, "test.go", testFile)
	require.NoError(t, err)

	walker, err := NewIndexTypeInDir(tempDir)
	require.NoError(t, err)

	packageTags := walker.PackageTags()
	require.NotNil(t, packageTags)

	expectedTags := map[string]Tag{
		"mkunion": {Value: "", Options: []string{"no-type-registry"}},
		"version": {Value: "1.0.0", Options: nil},
	}

	assert.Equal(t, len(expectedTags), len(packageTags))

	for key, expectedTag := range expectedTags {
		require.Contains(t, packageTags, key)
		actualTag := packageTags[key]
		assert.Equal(t, expectedTag.Value, actualTag.Value,
			"Tag %s value mismatch", key)
		assert.Equal(t, expectedTag.Options, actualTag.Options,
			"Tag %s options mismatch", key)
	}
}

// Helper function to write test files
func writeTestFile(dir, filename, content string) error {
	return writeFile(dir+"/"+filename, content)
}

func writeFile(path, content string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}

func TestExtractPackageTagsFromFile(t *testing.T) {
	tempDir := t.TempDir()
	
	testContent := `//go:tag mkunion:",no-type-registry"
//go:tag version:"1.2.3,stable"
//go:tag module:"testmodule"
package testpkg

type TestStruct struct {
	Field string
}`

	testFile := tempDir + "/test.go"
	err := writeFile(testFile, testContent)
	require.NoError(t, err)

	tags, err := ExtractPackageTagsFromFile(testFile)
	require.NoError(t, err)
	require.NotNil(t, tags)

	expected := map[string]Tag{
		"mkunion": {Value: "", Options: []string{"no-type-registry"}},
		"version": {Value: "1.2.3", Options: []string{"stable"}},
		"module":  {Value: "testmodule", Options: nil},
	}

	assert.Equal(t, len(expected), len(tags))

	for key, expectedTag := range expected {
		require.Contains(t, tags, key)
		actualTag := tags[key]
		assert.Equal(t, expectedTag.Value, actualTag.Value)
		assert.Equal(t, expectedTag.Options, actualTag.Options)
	}
}

func TestExtractPackageTagsFromDir(t *testing.T) {
	tempDir := t.TempDir()
	
	testContent := `//go:tag mkunion:",no-type-registry"
//go:tag version:"1.2.3,stable"
package testpkg

type TestStruct struct {
	Field string
}`

	err := writeTestFile(tempDir, "test.go", testContent)
	require.NoError(t, err)

	tags, err := ExtractPackageTagsFromDir(tempDir)
	require.NoError(t, err)
	require.NotNil(t, tags)

	expected := map[string]Tag{
		"mkunion": {Value: "", Options: []string{"no-type-registry"}},
		"version": {Value: "1.2.3", Options: []string{"stable"}},
	}

	assert.Equal(t, len(expected), len(tags))

	for key, expectedTag := range expected {
		require.Contains(t, tags, key)
		actualTag := tags[key]
		assert.Equal(t, expectedTag.Value, actualTag.Value)
		assert.Equal(t, expectedTag.Options, actualTag.Options)
	}
}

func TestGetPackageTagValue(t *testing.T) {
	tags := map[string]Tag{
		"version": {Value: "1.0.0", Options: []string{"stable"}},
		"module":  {Value: "example", Options: nil},
		"empty":   {Value: "", Options: []string{"option1"}},
	}

	tests := []struct {
		name         string
		tagName      string
		defaultValue string
		expected     string
	}{
		{
			name:         "existing tag with value",
			tagName:      "version",
			defaultValue: "unknown",
			expected:     "1.0.0",
		},
		{
			name:         "existing tag with empty value",
			tagName:      "empty",
			defaultValue: "default",
			expected:     "default",
		},
		{
			name:         "non-existing tag",
			tagName:      "nonexistent",
			defaultValue: "fallback",
			expected:     "fallback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPackageTagValue(tags, tt.tagName, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}

	// Test with nil tags
	result := GetPackageTagValue(nil, "version", "default")
	assert.Equal(t, "default", result)
}

func TestHasPackageTagOption(t *testing.T) {
	tags := map[string]Tag{
		"mkunion": {Value: "", Options: []string{"no-type-registry", "serde"}},
		"build":   {Value: "debug", Options: []string{"verbose", "warnings"}},
		"empty":   {Value: "test", Options: nil},
	}

	tests := []struct {
		name     string
		tagName  string
		option   string
		expected bool
	}{
		{
			name:     "existing tag with matching option",
			tagName:  "mkunion",
			option:   "no-type-registry",
			expected: true,
		},
		{
			name:     "existing tag with non-matching option",
			tagName:  "mkunion",
			option:   "unknown",
			expected: false,
		},
		{
			name:     "existing tag with no options",
			tagName:  "empty",
			option:   "any",
			expected: false,
		},
		{
			name:     "non-existing tag",
			tagName:  "nonexistent",
			option:   "any",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasPackageTagOption(tags, tt.tagName, tt.option)
			assert.Equal(t, tt.expected, result)
		})
	}

	// Test with nil tags
	result := HasPackageTagOption(nil, "mkunion", "no-type-registry")
	assert.False(t, result)
}