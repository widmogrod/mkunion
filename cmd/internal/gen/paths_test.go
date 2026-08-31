package gen

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandPathArgs(t *testing.T) {
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "sub")
	hiddenDir := filepath.Join(tempDir, ".hidden")
	require.NoError(t, os.MkdirAll(subDir, 0755))
	require.NoError(t, os.MkdirAll(hiddenDir, 0755))

	t.Run("explicit_path_is_kept_as_is", func(t *testing.T) {
		paths, err := ExpandPathArgs([]string{tempDir})
		require.NoError(t, err)
		assert.Equal(t, []string{tempDir}, paths)
	})

	t.Run("recursive_pattern_walks_subdirectories_and_skips_hidden", func(t *testing.T) {
		paths, err := ExpandPathArgs([]string{filepath.Join(tempDir, "...")})
		require.NoError(t, err)
		sort.Strings(paths)
		assert.Equal(t, []string{tempDir, subDir}, paths)
	})

	t.Run("duplicate_arguments_are_deduplicated", func(t *testing.T) {
		paths, err := ExpandPathArgs([]string{tempDir, tempDir})
		require.NoError(t, err)
		assert.Equal(t, []string{tempDir}, paths)
	})

	t.Run("empty_arguments_default_to_working_directory", func(t *testing.T) {
		cwd, err := os.Getwd()
		require.NoError(t, err)

		paths, err := ExpandPathArgs(nil)
		require.NoError(t, err)
		assert.Equal(t, []string{cwd}, paths)
	})
}

func TestGoFilesFromDirs(t *testing.T) {
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "sub")
	require.NoError(t, os.MkdirAll(subDir, 0755))

	files := map[string]bool{
		"source.go":           true,  // plain go file, included
		"source_union_gen.go": false, // generated, skipped
		".hidden.go":          false, // hidden, skipped
		"readme.md":           false, // not a go file, skipped
	}
	for name := range files {
		require.NoError(t, os.WriteFile(filepath.Join(tempDir, name), []byte("package x"), 0644))
	}
	// files in subdirectories are not included (no recursion)
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "nested.go"), []byte("package sub"), 0644))

	result, err := GoFilesFromDirs([]string{tempDir})
	require.NoError(t, err)

	assert.Equal(t, []string{filepath.Join(tempDir, "source.go")}, result)
}

func TestSourcePathsOrGOFILE(t *testing.T) {
	t.Run("explicit_paths_win", func(t *testing.T) {
		t.Setenv("GOFILE", "ignored.go")
		assert.Equal(t, []string{"a.go"}, SourcePathsOrGOFILE([]string{"a.go"}))
	})

	t.Run("falls_back_to_GOFILE_in_working_directory", func(t *testing.T) {
		t.Setenv("GOFILE", "from_env.go")
		cwd, err := os.Getwd()
		require.NoError(t, err)

		paths := SourcePathsOrGOFILE(nil)
		assert.Equal(t, []string{filepath.Join(cwd, "from_env.go")}, paths)
	})

	t.Run("nil_when_nothing_is_set", func(t *testing.T) {
		t.Setenv("GOFILE", "")
		assert.Nil(t, SourcePathsOrGOFILE(nil))
	})
}
