// Package gen contains the code-generation pipeline behind the mkunion CLI:
// expanding path arguments, generating union/shape/serde/match files, the type
// registry, cleaning generated files, running go generate, and watching for
// changes. The cmd/mkunion binary is a thin CLI layer over this package.
package gen

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"
)

var (
	ErrUseWorkingDirectory = fmt.Errorf("cannot retrive path of working directory")
	ErrFindingRecursiveDir = fmt.Errorf("failed to find recursive directories")
)

// ExpandPathArgs turns CLI path arguments (".", "./...", explicit paths) into a
// deduplicated list of directories. An empty argument list means the current
// directory.
func ExpandPathArgs(args []string) ([]string, error) {
	if len(args) == 0 {
		args = []string{"."}
	}

	var result []string
	for _, arg := range args {
		expanded, err := argToPaths(arg)
		if err != nil {
			return nil, err
		}
		result = append(result, expanded...)
	}

	return dedup(result), nil
}

// SourcePathsOrGOFILE returns the given paths, or, when empty, the file named
// by the GOFILE environment variable resolved against the current working
// directory (the go:generate convention). Returns nil when neither is set.
func SourcePathsOrGOFILE(paths []string) []string {
	if len(paths) > 0 {
		return paths
	}

	if goFile := os.Getenv("GOFILE"); goFile != "" {
		cwd, _ := syscall.Getwd()
		return []string{path.Join(cwd, path.Base(goFile))}
	}

	return nil
}

func dedup(in []string) []string {
	m := make(map[string]bool)
	for _, x := range in {
		m[x] = true
	}

	out := make([]string, 0, len(m))
	for x := range m {
		out = append(out, x)
	}

	return out
}

func cwdOrPath(x string) (string, error) {
	if x == "." {
		cwd, err := syscall.Getwd()
		if err != nil {
			return ".", fmt.Errorf("cwdOrPath: %s; %w", x, ErrUseWorkingDirectory)
		}

		return cwd, nil
	}

	return x, nil
}

func argToPaths(x string) ([]string, error) {
	switch x {
	case ".":
		cwd, err := cwdOrPath(x)
		if err != nil {
			return nil, fmt.Errorf("argToPaths: %s; %w", x, err)
		}

		return []string{cwd}, nil

	default:
		if path.Base(x) == "..." {
			var result []string
			// recursively walk through all directories starting from the directory
			dir, err := cwdOrPath(path.Dir(x))
			if err != nil {
				return nil, fmt.Errorf("argToPaths: recursive %s; %w", x, err)
			}

			err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}

				if !d.IsDir() {
					return nil
				}

				// if is hidden directory, skip
				if strings.HasPrefix(d.Name(), ".") {
					return filepath.SkipDir
				}

				result = append(result, path)
				return nil
			})

			if err != nil {
				return nil, fmt.Errorf("argToPaths: %s; %w; %w", x, err, ErrFindingRecursiveDir)
			}

			return result, nil
		}

		return []string{x}, nil
	}
}

// GoFilesFromDirs lists non-generated .go files directly inside each directory
// (no recursion into subdirectories).
func GoFilesFromDirs(dirs []string) ([]string, error) {
	var result []string
	for _, dir := range dirs {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if dir != path && d.IsDir() {
				return filepath.SkipDir
			}

			// if is hidden file, skip
			if strings.HasPrefix(d.Name(), ".") {
				return nil
			}

			// if is not go file, skip
			if filepath.Ext(path) != ".go" {
				return nil
			}

			// if is generated file, skip
			if strings.HasSuffix(path, "_gen.go") {
				return nil
			}

			result = append(result, path)
			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("GoFilesFromDirs: %w", err)
		}
	}

	return result, nil
}
