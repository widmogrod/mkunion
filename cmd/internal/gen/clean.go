package gen

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/widmogrod/mkunion/x/generators"
)

// CleanGeneratedFiles removes all mkunion-generated files from the specified
// directories (recursively). A file is removed only when its name matches a
// generated pattern AND it starts with the mkunion generated-code header.
// With dryRun, it only reports what would be removed.
func CleanGeneratedFiles(dirs []string, dryRun bool) ([]string, error) {
	var allFiles []string

	for _, dir := range dirs {
		files, err := findGeneratedFiles(dir)
		if err != nil {
			return nil, fmt.Errorf("failed to find generated files in %s: %w", dir, err)
		}
		allFiles = append(allFiles, files...)
	}

	uniqueFiles := dedup(allFiles)
	var removedFiles []string

	for _, file := range uniqueFiles {
		if !dryRun {
			err := os.Remove(file)
			if err != nil {
				log.Warnf("failed to remove %s: %v", file, err)
				continue
			}
		}
		removedFiles = append(removedFiles, file)
	}

	return removedFiles, nil
}

// findGeneratedFiles finds all generated files in a directory
func findGeneratedFiles(dir string) ([]string, error) {
	var generatedFiles []string

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}

		if !d.IsDir() && filepath.Ext(path) == ".go" {
			if isGeneratedFile(path) && hasGeneratedHeader(path) {
				generatedFiles = append(generatedFiles, path)
			}
		}

		return nil
	})

	return generatedFiles, err
}

// isGeneratedFile checks if a file is a generated file based on naming patterns
func isGeneratedFile(path string) bool {
	fileName := filepath.Base(path)

	generatedSuffixes := []string{
		"_union_gen.go",
		"_shape_gen.go",
		"_serde_gen.go",
		"_match_gen.go",
	}

	for _, suffix := range generatedSuffixes {
		if strings.HasSuffix(fileName, suffix) {
			return true
		}
	}

	return fileName == "types_reg_gen.go"
}

// hasGeneratedHeader confirms the file starts with the mkunion generated-code
// header, so clean never removes a hand-written file that happens to match a
// generated file name.
func hasGeneratedHeader(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, len(generators.Header))
	n, err := io.ReadFull(f, buf)
	if err != nil {
		return false
	}

	return string(buf[:n]) == generators.Header
}
