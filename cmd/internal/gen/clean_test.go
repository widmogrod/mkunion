package gen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/generators"
)

func TestCleanGeneratedFiles(t *testing.T) {
	t.Run("removes_generated_files_with_correct_patterns", func(t *testing.T) {
		// Create temporary directory
		tempDir := t.TempDir()

		// Create various generated files
		generatedFiles := []string{
			"test_union_gen.go",
			"test_shape_gen.go",
			"test_serde_gen.go",
			"test_match_gen.go",
			"types_reg_gen.go",
		}

		// Create non-generated files that should not be removed
		nonGeneratedFiles := []string{
			"test.go",
			"test_test.go",
			"helper.go",
			"types.go",
		}

		// Write all files; only generated files carry the mkunion header
		for _, fileName := range generatedFiles {
			filePath := filepath.Join(tempDir, fileName)
			err := os.WriteFile(filePath, []byte(generators.Header+"\npackage test"), 0644)
			require.NoError(t, err)
		}
		for _, fileName := range nonGeneratedFiles {
			filePath := filepath.Join(tempDir, fileName)
			err := os.WriteFile(filePath, []byte("package test"), 0644)
			require.NoError(t, err)
		}

		// Test clean without dry-run
		removedFiles, err := CleanGeneratedFiles([]string{tempDir}, false)
		require.NoError(t, err)

		// Verify correct files were removed
		assert.Len(t, removedFiles, len(generatedFiles))
		for _, expectedFile := range generatedFiles {
			found := false
			for _, removedFile := range removedFiles {
				if filepath.Base(removedFile) == expectedFile {
					found = true
					break
				}
			}
			assert.True(t, found, "Expected %s to be removed", expectedFile)
		}

		// Verify generated files no longer exist
		for _, fileName := range generatedFiles {
			filePath := filepath.Join(tempDir, fileName)
			_, err := os.Stat(filePath)
			assert.True(t, os.IsNotExist(err), "Generated file %s should be removed", fileName)
		}

		// Verify non-generated files still exist
		for _, fileName := range nonGeneratedFiles {
			filePath := filepath.Join(tempDir, fileName)
			_, err := os.Stat(filePath)
			assert.NoError(t, err, "Non-generated file %s should not be removed", fileName)
		}
	})

	t.Run("dry_run_does_not_remove_files", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create generated files
		generatedFiles := []string{
			"test_union_gen.go",
			"types_reg_gen.go",
		}

		for _, fileName := range generatedFiles {
			filePath := filepath.Join(tempDir, fileName)
			err := os.WriteFile(filePath, []byte(generators.Header+"\npackage test"), 0644)
			require.NoError(t, err)
		}

		// Test dry-run
		removedFiles, err := CleanGeneratedFiles([]string{tempDir}, true)
		require.NoError(t, err)

		// Verify files would be removed but still exist
		assert.Len(t, removedFiles, len(generatedFiles))
		for _, fileName := range generatedFiles {
			filePath := filepath.Join(tempDir, fileName)
			_, err := os.Stat(filePath)
			assert.NoError(t, err, "File %s should still exist in dry-run", fileName)
		}
	})

	t.Run("handles_recursive_directories", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create subdirectories
		subDir1 := filepath.Join(tempDir, "sub1")
		subDir2 := filepath.Join(tempDir, "sub2")
		err := os.MkdirAll(subDir1, 0755)
		require.NoError(t, err)
		err = os.MkdirAll(subDir2, 0755)
		require.NoError(t, err)

		// Create generated files in different directories
		files := map[string]string{
			filepath.Join(tempDir, "root_union_gen.go"): generators.Header + "\npackage test",
			filepath.Join(subDir1, "sub1_shape_gen.go"): generators.Header + "\npackage sub1",
			filepath.Join(subDir2, "sub2_serde_gen.go"): generators.Header + "\npackage sub2",
			filepath.Join(subDir2, "types_reg_gen.go"):  generators.Header + "\npackage sub2",
		}

		for filePath, content := range files {
			err := os.WriteFile(filePath, []byte(content), 0644)
			require.NoError(t, err)
		}

		// Test clean
		removedFiles, err := CleanGeneratedFiles([]string{tempDir}, false)
		require.NoError(t, err)

		// Should find all generated files recursively
		assert.Len(t, removedFiles, len(files))

		// Verify all files are removed
		for filePath := range files {
			_, err := os.Stat(filePath)
			assert.True(t, os.IsNotExist(err), "File %s should be removed", filePath)
		}
	})

	t.Run("handles_multiple_directories", func(t *testing.T) {
		tempDir1 := t.TempDir()
		tempDir2 := t.TempDir()

		// Create generated files in both directories
		files1 := []string{"test1_union_gen.go", "types_reg_gen.go"}
		files2 := []string{"test2_shape_gen.go"}

		for _, fileName := range files1 {
			filePath := filepath.Join(tempDir1, fileName)
			err := os.WriteFile(filePath, []byte(generators.Header+"\npackage test1"), 0644)
			require.NoError(t, err)
		}

		for _, fileName := range files2 {
			filePath := filepath.Join(tempDir2, fileName)
			err := os.WriteFile(filePath, []byte(generators.Header+"\npackage test2"), 0644)
			require.NoError(t, err)
		}

		// Test clean with multiple directories
		removedFiles, err := CleanGeneratedFiles([]string{tempDir1, tempDir2}, false)
		require.NoError(t, err)

		// Should remove files from both directories
		totalExpected := len(files1) + len(files2)
		assert.Len(t, removedFiles, totalExpected)
	})

	t.Run("handles_empty_directory", func(t *testing.T) {
		tempDir := t.TempDir()

		// Test clean on empty directory
		removedFiles, err := CleanGeneratedFiles([]string{tempDir}, false)
		require.NoError(t, err)
		assert.Len(t, removedFiles, 0)
	})

	t.Run("handles_nonexistent_directory", func(t *testing.T) {
		nonExistentDir := "/path/that/does/not/exist"

		// Should return error for non-existent directory
		_, err := CleanGeneratedFiles([]string{nonExistentDir}, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find generated files")
	})

	t.Run("keeps_files_without_generated_header_even_if_name_matches", func(t *testing.T) {
		tempDir := t.TempDir()

		// Hand-written file whose name matches a generated pattern
		handWritten := filepath.Join(tempDir, "custom_union_gen.go")
		err := os.WriteFile(handWritten, []byte("package test\n\n// hand written"), 0644)
		require.NoError(t, err)

		// Real generated file with the mkunion header
		generated := filepath.Join(tempDir, "real_union_gen.go")
		err = os.WriteFile(generated, []byte(generators.Header+"\npackage test"), 0644)
		require.NoError(t, err)

		removedFiles, err := CleanGeneratedFiles([]string{tempDir}, false)
		require.NoError(t, err)

		assert.Len(t, removedFiles, 1)
		assert.Equal(t, generated, removedFiles[0])

		_, err = os.Stat(handWritten)
		assert.NoError(t, err, "hand-written file should not be removed")
	})

	t.Run("deduplicates_files_from_overlapping_directories", func(t *testing.T) {
		tempDir := t.TempDir()
		subDir := filepath.Join(tempDir, "sub")
		err := os.MkdirAll(subDir, 0755)
		require.NoError(t, err)

		// Create a generated file in subdirectory
		filePath := filepath.Join(subDir, "test_union_gen.go")
		err = os.WriteFile(filePath, []byte(generators.Header+"\npackage test"), 0644)
		require.NoError(t, err)

		// Test with both parent and child directory
		removedFiles, err := CleanGeneratedFiles([]string{tempDir, subDir}, false)
		require.NoError(t, err)

		// Should only remove the file once (deduplicated)
		assert.Len(t, removedFiles, 1)
		assert.Equal(t, filePath, removedFiles[0])
	})
}

func TestIsGeneratedFile(t *testing.T) {
	tests := []struct {
		fileName string
		expected bool
	}{
		// Generated files
		{"test_union_gen.go", true},
		{"test_shape_gen.go", true},
		{"test_serde_gen.go", true},
		{"test_match_gen.go", true},
		{"types_reg_gen.go", true},

		// Non-generated files
		{"test.go", false},
		{"test_test.go", false},
		{"helper.go", false},
		{"types.go", false},
		{"main.go", false},
		{"gen.go", false},
		{"_gen.go", false},
		{"test_gen.go", false},   // Missing specific suffix
		{"test_union.go", false}, // Missing _gen suffix

		// Edge cases
		{"", false},
		{"a_union_gen.go", true},
		{"very_long_filename_union_gen.go", true},
	}

	for _, tt := range tests {
		t.Run(tt.fileName, func(t *testing.T) {
			// Create a temporary file path for testing
			tempDir := t.TempDir()
			fullPath := filepath.Join(tempDir, tt.fileName)

			result := isGeneratedFile(fullPath)
			assert.Equal(t, tt.expected, result, "isGeneratedFile(%q) = %v, want %v", tt.fileName, result, tt.expected)
		})
	}
}

func TestFindGeneratedFiles(t *testing.T) {
	t.Run("finds_all_generated_files", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create mix of generated and non-generated files
		files := map[string]bool{
			"test_union_gen.go": true,  // generated
			"test_shape_gen.go": true,  // generated
			"types_reg_gen.go":  true,  // generated
			"test.go":           false, // not generated
			"helper.go":         false, // not generated
			"README.md":         false, // not a go file
		}

		for fileName, isGenerated := range files {
			filePath := filepath.Join(tempDir, fileName)
			content := "content"
			if isGenerated {
				content = generators.Header + "\npackage test"
			}
			err := os.WriteFile(filePath, []byte(content), 0644)
			require.NoError(t, err)
		}

		generatedFiles, err := findGeneratedFiles(tempDir)
		require.NoError(t, err)

		// Count expected generated files
		expectedCount := 0
		for _, isGenerated := range files {
			if isGenerated {
				expectedCount++
			}
		}

		assert.Len(t, generatedFiles, expectedCount)

		// Verify all found files are actually generated
		for _, filePath := range generatedFiles {
			fileName := filepath.Base(filePath)
			assert.True(t, files[fileName], "File %s should be marked as generated", fileName)
		}
	})

	t.Run("handles_subdirectories", func(t *testing.T) {
		tempDir := t.TempDir()
		subDir := filepath.Join(tempDir, "subdir")
		err := os.MkdirAll(subDir, 0755)
		require.NoError(t, err)

		// Create generated files in both root and subdirectory
		rootFile := filepath.Join(tempDir, "root_union_gen.go")
		subFile := filepath.Join(subDir, "sub_shape_gen.go")

		err = os.WriteFile(rootFile, []byte(generators.Header+"\npackage root"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(subFile, []byte(generators.Header+"\npackage sub"), 0644)
		require.NoError(t, err)

		generatedFiles, err := findGeneratedFiles(tempDir)
		require.NoError(t, err)

		assert.Len(t, generatedFiles, 2)
		assert.Contains(t, generatedFiles, rootFile)
		assert.Contains(t, generatedFiles, subFile)
	})

	t.Run("skips_hidden_directories", func(t *testing.T) {
		tempDir := t.TempDir()
		hiddenDir := filepath.Join(tempDir, ".hidden")
		err := os.MkdirAll(hiddenDir, 0755)
		require.NoError(t, err)

		// Create generated file in hidden directory
		hiddenFile := filepath.Join(hiddenDir, "hidden_union_gen.go")
		err = os.WriteFile(hiddenFile, []byte(generators.Header+"\npackage hidden"), 0644)
		require.NoError(t, err)

		// Create generated file in visible directory
		visibleFile := filepath.Join(tempDir, "visible_union_gen.go")
		err = os.WriteFile(visibleFile, []byte(generators.Header+"\npackage visible"), 0644)
		require.NoError(t, err)

		generatedFiles, err := findGeneratedFiles(tempDir)
		require.NoError(t, err)

		// Should only find the visible file
		assert.Len(t, generatedFiles, 1)
		assert.Contains(t, generatedFiles, visibleFile)
		assert.NotContains(t, generatedFiles, hiddenFile)
	})
}
