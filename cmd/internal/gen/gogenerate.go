package gen

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

// RunGoGenerate runs 'go generate ./...' in each directory that contains
// //go:generate directives. Failures are logged per directory; a single error
// is returned when any directory failed.
func RunGoGenerate(dirs []string) error {
	if len(dirs) == 0 {
		return nil
	}

	// Filter directories that actually contain Go files with generate directives
	dirsToProcess := []string{}
	for _, dir := range dedup(dirs) {
		if hasGoGenerateDirectives(dir) {
			dirsToProcess = append(dirsToProcess, dir)
		}
	}

	if len(dirsToProcess) == 0 {
		log.Debug("No directories with go:generate directives found, skipping go generate")
		return nil
	}

	hasErrors := false
	for _, dir := range dirsToProcess {
		log.Infof("Running 'go generate ./...' in %s", dir)

		cmd := exec.Command("go", "generate", "./...")
		cmd.Dir = dir

		output, err := cmd.CombinedOutput()
		if err != nil {
			// Check if it's a real error or just "no packages" warning
			outputStr := string(output)
			if strings.Contains(outputStr, "matched no packages") {
				log.Debugf("No packages to generate in %s", dir)
				continue
			}

			log.Warnf("'go generate ./...' failed in %s: %v\n%s", dir, err, outputStr)
			hasErrors = true
			continue
		}

		if len(output) > 0 {
			log.Debugf("go generate output for %s:\n%s", dir, string(output))
		}
	}

	if hasErrors {
		// Specific failures were already logged as warnings
		return fmt.Errorf("go generate failed in one or more directories")
	}

	log.Info("Finished running 'go generate ./...'")
	return nil
}

// hasGoGenerateDirectives checks if a directory (or any of its
// subdirectories) contains a Go file with a //go:generate directive.
func hasGoGenerateDirectives(dir string) bool {
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil || len(files) == 0 {
		return false
	}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		if bytes.Contains(content, []byte("//go:generate")) {
			return true
		}
	}

	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		if filepath.Ext(path) == ".go" {
			content, err := os.ReadFile(path)
			if err == nil && bytes.Contains(content, []byte("//go:generate")) {
				return fmt.Errorf("found") // Use error to stop walking
			}
		}
		return nil
	})

	return err != nil && err.Error() == "found"
}
