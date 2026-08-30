package shape

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"path"
	"path/filepath"
	"sync"
)

// Index is the single owner of all shape information read from disk.
//
// Mental model, one-way data flow:
//
//	.go files --parse once--> Index --read only--> generators
//
// Every file is parsed at most once per Index lifetime, and every package
// directory is walked at most once. All lookups (by file, by package,
// by type reference) read from the same three maps below.
//
// The Index is safe for concurrent use. A single mutex guards the maps
// and counters. The mutex is never held while parsing a file or walking
// a directory, because parsing can call back into the Index
// (dot-import resolution); holding the lock across a parse would deadlock.
// The watch loop calls Reset() before each re-generation.
type Index struct {
	// mu guards every field below. Lock it only around map/counter
	// access, never across parseFile, filepath.WalkDir or
	// findPackagePathUncached calls (they can re-enter the Index).
	mu sync.Mutex

	// files: absolute file path -> parse result.
	// The (modTime, size) pair invalidates an entry when the file changes.
	files map[string]indexedFile

	// pkgShapes: package import name -> all shapes found in that package.
	// An entry exists (possibly nil) once the package was walked,
	// so a missing package is not re-walked on every lookup.
	pkgShapes map[string][]Shape

	// shapes: "pkgImportName.TypeName" -> shape, filled while walking packages.
	shapes map[string]Shape

	// pkgPaths: package import name -> directory on disk (memoized go.mod resolution).
	pkgPaths map[string]pkgPathResult

	// inferring: files currently being parsed, to break dot-import recursion.
	// A parse started while its own file is already being parsed runs with
	// dot-import resolution disabled and is NOT cached (it is incomplete).
	inferring map[string]bool

	// work counters, see Stats()
	fileParses    int64
	fileParseHits int64
	pkgWalks      int64
	pkgWalkHits   int64
}

type indexedFile struct {
	modTime int64
	size    int64
	info    *InferredInfo
}

type pkgPathResult struct {
	path string
	err  error
}

func NewIndex() *Index {
	return &Index{
		files:     map[string]indexedFile{},
		pkgShapes: map[string][]Shape{},
		shapes:    map[string]Shape{},
		pkgPaths:  map[string]pkgPathResult{},
		inferring: map[string]bool{},
	}
}

// DefaultIndex backs the package-level functions
// (InferFromFile, LookupShapeOnDisk, LookupPkgShapeOnDisk).
var DefaultIndex = NewIndex()

// ResetIndex drops all state of the DefaultIndex.
// Call it when files on disk changed (e.g. in watch mode) before re-generating.
func ResetIndex() {
	DefaultIndex.Reset()
}

// Reset drops all indexed state, so the next load re-reads from disk.
func (ix *Index) Reset() {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	ix.files = map[string]indexedFile{}
	ix.pkgShapes = map[string][]Shape{}
	ix.shapes = map[string]Shape{}
	ix.pkgPaths = map[string]pkgPathResult{}
	ix.inferring = map[string]bool{}
	ix.fileParses = 0
	ix.fileParseHits = 0
	ix.pkgWalks = 0
	ix.pkgWalkHits = 0
}

// Stats returns counters that describe how much work the index did.
func (ix *Index) Stats() map[string]int64 {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	return map[string]int64{
		"file_parses":     ix.fileParses,
		"file_parse_hits": ix.fileParseHits,
		"pkg_walks":       ix.pkgWalks,
		"pkg_walk_hits":   ix.pkgWalkHits,
	}
}

// LoadFile parses one Go file, at most once per file version.
func (ix *Index) LoadFile(filename string) (*InferredInfo, error) {
	if !path.IsAbs(filename) {
		cwd, _ := os.Getwd()
		filename = path.Join(cwd, filename)
	}

	var modTime, size int64
	cacheable := false
	if stat, err := os.Stat(filename); err == nil {
		modTime, size = stat.ModTime().UnixNano(), stat.Size()
		cacheable = true
	}

	ix.mu.Lock()
	if cacheable {
		if entry, ok := ix.files[filename]; ok && entry.modTime == modTime && entry.size == size {
			ix.fileParseHits++
			ix.mu.Unlock()
			return entry.info, nil
		}
	}

	// A nested parse of a file already being parsed runs with dot-import
	// resolution disabled (see InferredInfo.ResolveUnqualifiedType) and
	// produces an incomplete result, so it must not be cached.
	nested := ix.inferring[filename]
	if nested {
		cacheable = false
	}
	ix.inferring[filename] = true
	ix.fileParses++
	ix.mu.Unlock()

	// The lock is NOT held here: parseFile can call back into the Index.
	info, err := parseFile(filename)

	ix.mu.Lock()
	defer ix.mu.Unlock()
	if !nested {
		delete(ix.inferring, filename)
	}

	if err != nil {
		return nil, err
	}

	if cacheable {
		ix.files[filename] = indexedFile{modTime: modTime, size: size, info: info}
		ix.registerShapesLocked(info.RetrieveShapes())
	}

	return info, nil
}

// LoadPackage walks a package directory once and returns all shapes in it.
func (ix *Index) LoadPackage(pkgImportName string) []Shape {
	ix.mu.Lock()
	if shapes, ok := ix.pkgShapes[pkgImportName]; ok {
		ix.pkgWalkHits++
		ix.mu.Unlock()
		return shapes
	}
	ix.mu.Unlock()

	pkgPath, err := ix.packagePath(pkgImportName)
	if err != nil {
		log.Warnf("shape.Index.LoadPackage: could not find package path %s", err.Error())
		ix.mu.Lock()
		ix.pkgShapes[pkgImportName] = nil
		ix.mu.Unlock()
		return nil
	}

	ix.mu.Lock()
	ix.pkgWalks++
	ix.mu.Unlock()

	// The lock is NOT held during the walk: LoadFile locks on its own,
	// and parsing can call back into the Index.
	var result []Shape
	err = filepath.WalkDir(
		pkgPath,
		func(path string, d os.DirEntry, err error) error {
			if err != nil {
				// ignore errors
				return nil
			}

			if d.IsDir() {
				if path != pkgPath {
					return filepath.SkipDir
				}
				return nil
			}

			if filepath.Ext(path) != ".go" {
				return nil
			}

			inferred, err := ix.LoadFile(path)
			if err != nil {
				return fmt.Errorf("shape.Index.LoadPackage: error during infer %w", err)
			}

			result = append(result, inferred.RetrieveShapes()...)
			return nil
		})

	ix.mu.Lock()
	defer ix.mu.Unlock()

	if err != nil {
		log.Warnf("shape.Index.LoadPackage: error during walk %s", err.Error())
		ix.pkgShapes[pkgImportName] = nil
		return nil
	}

	ix.registerShapesLocked(result)
	ix.pkgShapes[pkgImportName] = result
	return result
}

// Lookup finds a shape by reference, loading its package on first miss.
func (ix *Index) Lookup(x *RefName) (Shape, bool) {
	key := shapeFullName(x)

	ix.mu.Lock()
	if v, ok := ix.shapes[key]; ok {
		ix.mu.Unlock()
		return v, true
	}
	ix.mu.Unlock()

	ix.LoadPackage(x.PkgImportName)

	ix.mu.Lock()
	defer ix.mu.Unlock()
	v, ok := ix.shapes[key]
	return v, ok
}

// isInferring reports if a file is currently being parsed by this index.
func (ix *Index) isInferring(filename string) bool {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	return ix.inferring[filename]
}

// packagePath resolves a package import name to a directory on disk, memoized.
func (ix *Index) packagePath(pkgImportName string) (string, error) {
	ix.mu.Lock()
	if entry, ok := ix.pkgPaths[pkgImportName]; ok {
		ix.mu.Unlock()
		return entry.path, entry.err
	}
	ix.mu.Unlock()

	p, err := findPackagePathUncached(pkgImportName)

	ix.mu.Lock()
	ix.pkgPaths[pkgImportName] = pkgPathResult{path: p, err: err}
	ix.mu.Unlock()
	return p, err
}

// registerShapesLocked stores shapes; the caller must hold ix.mu.
func (ix *Index) registerShapesLocked(shapes []Shape) {
	for _, y := range shapes {
		ix.shapes[shapeFullName(y)] = y
		if union, ok := y.(*UnionLike); ok {
			for _, v := range union.Variant {
				ix.shapes[shapeFullName(v)] = v
			}
		}
	}
}
