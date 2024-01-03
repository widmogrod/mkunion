package shape

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"golang.org/x/mod/modfile"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
)

var shapeRegistry = sync.Map{}

func Register(x Shape) {
	shapeRegistry.Store(shapeFullName(x), x)
}

func shapeFullName(x Shape) string {
	return MustMatchShape(
		x,
		func(x *Any) string {
			return "any"
		},
		func(x *RefName) string {
			if x.PkgName == "" {
				return x.Name
			}

			// intentionaly skip indexing

			return fmt.Sprintf("%s.%s", x.PkgName, x.Name)
		},
		func(x *AliasLike) string {
			if x.PkgName == "" {
				return x.Name
			}

			return fmt.Sprintf("%s.%s", x.PkgName, x.Name)
		},
		func(x *BooleanLike) string {
			return "bool"
		},
		func(x *StringLike) string {
			return "string"
		},
		func(x *NumberLike) string {
			return "number"
		},
		func(x *ListLike) string {
			return fmt.Sprintf("[]%s", shapeFullName(x.Element))
		},
		func(x *MapLike) string {
			return fmt.Sprintf("map[%s]%s", shapeFullName(x.Key), shapeFullName(x.Val))
		},
		func(x *StructLike) string {
			if x.PkgName == "" {
				return x.Name
			}

			return fmt.Sprintf("%s.%s", x.PkgName, x.Name)
		},
		func(x *UnionLike) string {
			if x.PkgName == "" {
				return x.Name
			}

			return fmt.Sprintf("%s.%s", x.PkgName, x.Name)
		},
	)
}

func LookupShapeReflectAndIndex[A any]() (Shape, bool) {
	v := reflect.TypeOf(new(A)).Elem()
	original := MkRefNameFromReflect(v)

	s, found := LookupShape(original)
	if !found {
		return nil, false
	}

	s = IndexWith(s, original)
	return s, true
}

var (
	ErrShapeNotFound = fmt.Errorf(`To register shape manually, use shape.Register(myShape) in your package init() function or, use shape.LookupShapeOnDisk(x) to scan your filesystem for shapes.`)
)

// LookupShape scans registry for shapes.
// it's suited for runtime and compiled code
func LookupShape(x *RefName) (Shape, bool) {
	key := shapeFullName(x)
	if v, ok := shapeRegistry.Load(key); ok {
		return v.(Shape), true
	}

	//log.Warnf("shape.LookupShape: %s not found in shapeRegistry, trying to lookup on disk", key)
	//return LookupShapeOnDisk(x)

	return nil, false
}

var onDiskCache = sync.Map{}

// LookupShapeOnDisk scans filesystem for shapes.
// it's suited for generators, that parse AST
func LookupShapeOnDisk(x *RefName) (Shape, bool) {
	key := shapeFullName(x)
	if v, ok := onDiskCache.Load(key); ok {
		return v.(Shape), true
	}

	pkgPath, err := findPackagePath(x.PkgImportName)
	if err != nil {
		log.Warnf("shape.LookupShapeOnDisk: could not find package path %s", err.Error())
		return nil, false
	}

	err = filepath.WalkDir(
		pkgPath,
		func(path string, d os.DirEntry, err error) error {
			if err != nil {
				// ignore errors
				return nil
			}

			if d.IsDir() {
				return nil
			}

			if filepath.Ext(path) != ".go" {
				return nil
			}

			inferred, err := InferFromFile(path)
			if err != nil {
				return fmt.Errorf("shape.LookupShapeOnDisk: error during infer %w", err)
			}

			for _, y := range inferred.RetrieveShapes() {
				onDiskCache.Store(shapeFullName(y), y)
				switch z := y.(type) {
				case *UnionLike:
					for _, v := range z.Variant {
						onDiskCache.Store(shapeFullName(v), v)
					}
				}
			}

			// continue scanning
			return nil
		})

	if err != nil {
		log.Warnf("shape.LookupShapeOnDisk: error during walk %s", err.Error())
		return nil, false
	}

	// if not found, return nil, false
	if v, ok := onDiskCache.Load(key); ok {
		return v.(Shape), true
	}

	return nil, false
}

// LookupPkgShapeOnDisk scans filesystem for all shapes in pkgImportName.
// it's suited for generators, that parse AST
func LookupPkgShapeOnDisk(pkgImportName string) []Shape {
	pkgPath, err := findPackagePath(pkgImportName)
	if err != nil {
		log.Warnf("shape.LookupPkgShapeOnDisk: could not find package path %s", err.Error())
		return nil
	}

	var result []Shape
	err = filepath.WalkDir(
		pkgPath,
		func(path string, d os.DirEntry, err error) error {
			if err != nil {
				// ignore errors
				return nil
			}

			if d.IsDir() {
				return nil
			}

			if filepath.Ext(path) != ".go" {
				return nil
			}

			inferred, err := InferFromFile(path)
			if err != nil {
				return fmt.Errorf("shape.LookupPkgShapeOnDisk: error during infer %w", err)
			}

			for _, y := range inferred.RetrieveShapes() {
				result = append(result, y)
				onDiskCache.Store(shapeFullName(y), y)
				switch z := y.(type) {
				case *UnionLike:
					for _, v := range z.Variant {
						onDiskCache.Store(shapeFullName(v), v)
					}
				}
			}

			// continue scanning
			return nil
		})

	if err != nil {
		log.Warnf("shape.LookupPkgShapeOnDisk: error during walk %s", err.Error())
		return nil
	}

	return result
}

func findPackagePath(pkgImportName string) (string, error) {
	if strings.Trim(pkgImportName, " ") == "" {
		return "", fmt.Errorf("shape.findPackagePath: empty package name")
	}

	cwd := os.Getenv("PWD")
	if cwd == "" {
		cwd, _ = os.Getwd()
	}

	// if path has "go.mod" and package name is the same as x.PkgName
	// then we can assume that it's root of the package

	// hack: to make sure code is simple, we start with the current directory
	// add append nonsense to the path,
	// because it will be stripped by path.Dir
	cwd = path.Join(cwd, "nonsense")
	for {
		cwd = path.Dir(cwd)
		if cwd == "." || cwd == "/" {
			log.Infof("shape.findPackagePath: %s could not find go.mod file in CWD or parent directories %s, continue with other paths", pkgImportName, cwd)
			break
		}

		modpath := path.Join(cwd, "go.mod")
		_, err := os.Stat(modpath)
		log.Infof("shape.findPackagePath: %s checking modpath %s; %s", pkgImportName, modpath, err)
		if err == nil {
			f, err := os.Open(modpath)
			if err != nil {
				log.Infof("shape.findPackagePath: %s could not open %s", pkgImportName, cwd)
				continue
			}
			defer f.Close()

			data, err := io.ReadAll(f)
			if err != nil {
				log.Infof("shape.findPackagePath: %s could not read %s", pkgImportName, cwd)
				continue
				//return "", fmt.Errorf("shape.findPackagePath: could not read %s; %w", cwd, err)
			} else {
				parsed, err := modfile.Parse(modpath, data, nil)
				if err != nil {
					log.Infof("shape.findPackagePath: %s could not parse go.mod %s", pkgImportName, cwd)
					break
				} else {
					log.Infof("shape.findPackagePath: %s parsed go.mod %s", pkgImportName, parsed.Module.Mod.Path)
					if strings.Contains(pkgImportName, parsed.Module.Mod.Path) {
						return filepath.Join(cwd, strings.TrimPrefix(pkgImportName, parsed.Module.Mod.Path)), nil
					}
				}
			}
		}
	}

	//otherwise fallback to GOPATH/pkg/mod
	gopath := os.Getenv("GOPATH")
	paths := []string{
		filepath.Join(cwd),
		filepath.Join(cwd, "vendor"),
		filepath.Join(gopath, "pkg/mod"),
		filepath.Join(gopath, "src"),
	}

	for _, p := range paths {
		packPath := filepath.Join(p, pkgImportName)
		if _, err := os.Stat(packPath); err == nil {
			return packPath, nil
		}
	}

	return "", fmt.Errorf("shape.findPackagePath: could not find package %s", pkgImportName)
}
