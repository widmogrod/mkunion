package gen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunGoGenerate(t *testing.T) {
	t.Run("executes_go_generate_successfully", func(t *testing.T) {
		// Create temporary directory
		tempDir := t.TempDir()

		// Create go.mod file
		goMod := filepath.Join(tempDir, "go.mod")
		modContent := `module test

go 1.21
`
		err := os.WriteFile(goMod, []byte(modContent), 0644)
		require.NoError(t, err)

		// Create a simple Go file with generate directive
		goFile := filepath.Join(tempDir, "test.go")
		// Split the directive so 'go generate' over this repo does not execute it
		content := "package test\n\n" +
			"//go" + ":generate touch generated.txt\n"
		err = os.WriteFile(goFile, []byte(content), 0644)
		require.NoError(t, err)

		// Test the function
		err = RunGoGenerate([]string{tempDir})
		require.NoError(t, err)

		// Verify that go generate was executed
		generatedFile := filepath.Join(tempDir, "generated.txt")
		_, err = os.Stat(generatedFile)
		assert.NoError(t, err, "generated file should exist")
	})

	t.Run("handles_multiple_directories", func(t *testing.T) {
		// Create multiple temporary directories
		tempDir1 := t.TempDir()
		tempDir2 := t.TempDir()

		// Create Go files in both directories
		for i, dir := range []string{tempDir1, tempDir2} {
			// Create go.mod file
			goMod := filepath.Join(dir, "go.mod")
			modContent := `module test` + string(rune('1'+i)) + `

go 1.21
`
			err := os.WriteFile(goMod, []byte(modContent), 0644)
			require.NoError(t, err)

			goFile := filepath.Join(dir, "test.go")
			// Split the directive so 'go generate' over this repo does not execute it
			content := "package test\n\n" +
				"//go" + ":generate touch generated.txt\n"
			err = os.WriteFile(goFile, []byte(content), 0644)
			require.NoError(t, err)
		}

		// Test the function with multiple directories
		err := RunGoGenerate([]string{tempDir1, tempDir2})
		require.NoError(t, err)

		// Verify that go generate was executed in both directories
		for _, dir := range []string{tempDir1, tempDir2} {
			generatedFile := filepath.Join(dir, "generated.txt")
			_, err = os.Stat(generatedFile)
			assert.NoError(t, err, "generated file should exist in %s", dir)
		}
	})

	t.Run("continues_on_failure_in_one_directory", func(t *testing.T) {
		// Create multiple temporary directories
		tempDir1 := t.TempDir()
		tempDir2 := t.TempDir()

		// First directory with valid command
		goMod1 := filepath.Join(tempDir1, "go.mod")
		modContent1 := `module test1

go 1.21
`
		err := os.WriteFile(goMod1, []byte(modContent1), 0644)
		require.NoError(t, err)

		goFile1 := filepath.Join(tempDir1, "test.go")
		// Split the directive so 'go generate' over this repo does not execute it
		content1 := "package test\n\n" +
			"//go" + ":generate touch generated.txt\n"
		err = os.WriteFile(goFile1, []byte(content1), 0644)
		require.NoError(t, err)

		// Second directory with invalid command
		goMod2 := filepath.Join(tempDir2, "go.mod")
		modContent2 := `module test2

go 1.21
`
		err = os.WriteFile(goMod2, []byte(modContent2), 0644)
		require.NoError(t, err)

		goFile2 := filepath.Join(tempDir2, "test.go")
		// Using string concatenation to prevent go generate from picking this up
		content2 := "package test\n\n" +
			"//go" + ":generate invalid-command-that-does-not-exist\n"
		err = os.WriteFile(goFile2, []byte(content2), 0644)
		require.NoError(t, err)

		// Test the function with multiple directories - should return error but continue processing
		err = RunGoGenerate([]string{tempDir1, tempDir2})
		assert.Error(t, err, "should return error when one directory fails")

		// Verify that go generate was still executed in the first directory
		generatedFile := filepath.Join(tempDir1, "generated.txt")
		_, err = os.Stat(generatedFile)
		assert.NoError(t, err, "generated file should exist in first directory despite second directory failure")
	})

	t.Run("returns_error_on_go_generate_failure", func(t *testing.T) {
		// Create temporary directory
		tempDir := t.TempDir()

		// Create go.mod file
		goMod := filepath.Join(tempDir, "go.mod")
		modContent := `module test

go 1.21
`
		err := os.WriteFile(goMod, []byte(modContent), 0644)
		require.NoError(t, err)

		// Create a Go file with invalid generate directive
		goFile := filepath.Join(tempDir, "test.go")
		// Note: Using a comment prefix to prevent go generate from picking this up in the test file itself
		content := "package test\n\n" +
			"//go" + ":generate invalid-command-that-does-not-exist\n"
		err = os.WriteFile(goFile, []byte(content), 0644)
		require.NoError(t, err)

		// Test the function - should return error
		err = RunGoGenerate([]string{tempDir})
		assert.Error(t, err, "should return error for invalid command")
	})

	t.Run("handles_empty_directory_list", func(t *testing.T) {
		// Test with empty directory list
		err := RunGoGenerate([]string{})
		assert.NoError(t, err, "should handle empty directory list gracefully")
	})

	t.Run("handles_nil_directory_list", func(t *testing.T) {
		// Test with nil directory list
		err := RunGoGenerate(nil)
		assert.NoError(t, err, "should handle nil directory list gracefully")
	})

	t.Run("skips_directories_without_generate_directives", func(t *testing.T) {
		// Create temporary directory without go:generate
		tempDir := t.TempDir()

		// Create go.mod file
		goMod := filepath.Join(tempDir, "go.mod")
		modContent := `module test

go 1.21
`
		err := os.WriteFile(goMod, []byte(modContent), 0644)
		require.NoError(t, err)

		// Create a Go file WITHOUT generate directive
		goFile := filepath.Join(tempDir, "test.go")
		content := `package test

func main() {
	// No generate directive here
}
`
		err = os.WriteFile(goFile, []byte(content), 0644)
		require.NoError(t, err)

		// Test the function - should not run go generate
		err = RunGoGenerate([]string{tempDir})
		assert.NoError(t, err, "should skip directories without generate directives")
	})
}

func TestWatchCommandWithGoGenerate(t *testing.T) {
	// This test would require a more complex setup to test the integration
	// of the --dont-run-go-generate flag with the watch command.
	// For now, we'll focus on unit testing the runGoGenerate function.
	t.Skip("Integration test for watch command with go generate")
}
