package shape

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"golang.org/x/mod/modfile"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

var cache = sync.Map{}

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

func LookupShape(x *RefName) (Shape, bool) {
	key := shapeFullName(x)
	if v, ok := cache.Load(key); ok {
		return v.(Shape), true
	}

	pkgPath, err := findPackagePath(x.PkgImportName)
	if err != nil {
		log.Warnf("shape.LookupShape: could not find package path %s", err.Error())
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
				return fmt.Errorf("shape.LookupShape: error during infer %w", err)
			}

			for _, y := range inferred.RetrieveShapes() {
				cache.Store(shapeFullName(y), y)
				switch z := y.(type) {
				case *UnionLike:
					for _, v := range z.Variant {
						cache.Store(shapeFullName(v), v)
					}
				}
			}

			// continue scanning
			return nil
		})

	if err != nil {
		log.Warnf("shape.LookupShape: error during walk %s", err.Error())
		return nil, false
	}

	c := cache
	_ = c

	// if not found, return nil, false
	if v, ok := cache.Load(key); ok {
		return v.(Shape), true
	}

	return nil, false
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
