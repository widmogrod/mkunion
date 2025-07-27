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
	return fmt.Sprintf("%s.%s", ToGoPkgImportName(x), Name(x))
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
	if x.PkgName == "" {
		if primitive := NameToPrimitiveShape(x.Name); primitive != nil {
			return primitive, true
		}
	}

	key := shapeFullName(x)
	if v, ok := shapeRegistry.Load(key); ok {
		return v.(Shape), true
	}

	return nil, false
}

var onDiskCache = sync.Map{}

// LookupShapeOnDisk scans filesystem for shapes.
// it's suited for generators, that parse AST
func LookupShapeOnDisk(x *RefName) (Shape, bool) {
	key := shapeFullName(x)
	log.Debugf("shape.LookupShapeOnDisk: %s", key)
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
				if path != pkgPath {
					return filepath.SkipDir
				}
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
	log.Debugf("LookupPkgShapeOnDisk: looking for shapes in %s", pkgImportName)
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
				if path != pkgPath {
					return filepath.SkipDir
				}

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

	log.Debugf("LookupPkgShapeOnDisk: found %d shapes in %s", len(result), pkgImportName)
	for _, shape := range result {
		log.Debugf("LookupPkgShapeOnDisk:   - %s", Name(shape))
	}
	return result
}

func findPackagePath(pkgImportName string) (string, error) {
	if strings.Trim(pkgImportName, " ") == "" {
		return "", fmt.Errorf("shape.findPackagePath: empty package name")
	}

	// optimisation: assumption that packages like fmt, context, etc
	// are standard library packages
	if len(strings.Split(pkgImportName, "/")) == 1 {
		return checkPkgExistsInPaths(pkgImportName)
	}

	cwd := os.Getenv("PWD")
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	if cwd != "" {
		// hack: to make sure code is simple, we start with the current directory
		// add append nonsense to the path,
		// because it will be stripped by path.Dir
		cwd = path.Join(cwd, "nonsense")
	}

	// if path has "go.mod" and package name is the same as x.PkgName
	// then we can assume that it's root of the package

	for {
		cwd = path.Dir(cwd)
		if cwd == "." || cwd == "/" || cwd == "" {
			log.Debugf("shape.findPackagePath: %s could not find go.mod file in CWD or parent directories %s, continue with other paths", pkgImportName, cwd)
			break
		}

		modpath := path.Join(cwd, "go.mod")
		_, err := os.Stat(modpath)
		//log.Debugf("shape.findPackagePath: %s checking modpath %s; err=%s", pkgImportName, modpath, err)
		if err == nil {
			f, err := os.Open(modpath)
			if err != nil {
				//log.Debugf("shape.findPackagePath: %s could not open %s", pkgImportName, cwd)
				continue
			}
			defer f.Close()

			data, err := io.ReadAll(f)
			if err != nil {
				log.Errorf("shape.findPackagePath: %s could not read go.mod in %s", pkgImportName, cwd)
				continue
			} else {
				parsed, err := modfile.Parse(modpath, data, nil)
				if err != nil {
					log.Debugf("shape.findPackagePath: %s could not parse go.mod %s", pkgImportName, cwd)
					break
				} else {
					if strings.Contains(pkgImportName, parsed.Module.Mod.Path) {
						log.Infof("shape.findPackagePath: mod name Contains(%s. %s)", pkgImportName, parsed.Module.Mod.Path)
						return filepath.Join(cwd, strings.TrimPrefix(pkgImportName, parsed.Module.Mod.Path)), nil
					}

					// check if package is in replace
					// and if found, return path to replace
					//for _, r := range parsed.Replace {
					//	if strings.Contains(pkgImportName, r.Old.Path) {
					//		log.Infof("shape.findPackagePath: %s found package in go.mod replace %s", pkgImportName, filepath.Join(cwd, r.New.Path))
					//		return filepath.Join(cwd, r.New.Path), nil
					//	}
					//}

					// check if package is in require
					// it means that it's a dependency
					// and we should look for it in vendor
					for _, r := range parsed.Require {
						if !strings.Contains(pkgImportName, r.Mod.Path) {
							continue
						}

						log.Debugf("shape.findPackagePath: require Contains(%s, %s) ", pkgImportName, r.Mod.Path)
						rest := strings.TrimPrefix(pkgImportName, r.Mod.Path)

						// check if package is in replace
						// and if found, return path to replace
						for _, re := range parsed.Replace {
							if strings.Contains(r.Mod.Path, re.Old.Path) {
								p := filepath.Join(cwd, re.New.Path, rest)
								log.Infof("shape.findPackagePath: %s found in replace %s", pkgImportName, p)
								return p, nil
							}
						}

						found, err := checkPkgExistsInPaths(r.Mod.String())
						if err != nil {
							return "", fmt.Errorf("shape.findPackagePath: %s could not find package in vendor %s", pkgImportName, err.Error())
						}

						log.Infof("shape.findPackagePath: %s found package in go.mod require %s", pkgImportName, filepath.Join(cwd, strings.TrimPrefix(pkgImportName, r.Mod.Path)))
						return filepath.Join(found, rest), nil
					}
				}
			}
		}
	}

	log.Debugf("shape.findPackagePath: %s continue with other paths", pkgImportName)
	return checkPkgExistsInPaths(pkgImportName)
}

func checkPkgExistsInPaths(pkgImportName string) (string, error) {
	gocache := os.Getenv("GOMODCACHE")
	if gocache == "" {
		gocache = os.Getenv("GOPATH")
		if gocache == "" {
			gocache = os.Getenv("HOME")
			if gocache != "" {
				gocache = filepath.Join(gocache, "go")
			}
		}

		if gocache != "" {
			gocache = filepath.Join(gocache, "pkg/mod")
		}
	}

	paths := []string{}

	if gocache != "" {
		paths = append(paths, gocache)
	}

	paths = append(paths, filepath.Join(os.Getenv("GOROOT"), "src"))

	for _, p := range paths {
		packPath := filepath.Join(p, pkgImportName)
		if _, err := os.Stat(packPath); err == nil {
			log.Infof("shape.checkPkgExistsInPaths: '%s' found package in fallback %s", pkgImportName, packPath)
			return packPath, nil
		} else {
			log.Debugf("shape.checkPkgExistsInPaths: '%s' could not find package in fallback path %s", pkgImportName, packPath)
		}
	}

	return "", fmt.Errorf("shape.checkPkgExistsInPaths: could not find package %s", pkgImportName)
}
