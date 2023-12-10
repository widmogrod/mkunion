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

func RefFullName(x *RefName) string {
	return fmt.Sprintf("%s:%s.%s", x.PkgImportName, x.PkgName, x.Name)
}

func StructFullName(x *StructLike) string {
	return fmt.Sprintf("%s:%s.%s", x.PkgImportName, x.PkgName, x.Name)
}

func UnionFullName(x *UnionLike) string {
	return fmt.Sprintf("%s:%s.%s", x.PkgImportName, x.PkgName, x.Name)
}

func LookupShape(x *RefName) (Shape, bool) {
	key := RefFullName(x)
	if v, ok := cache.Load(key); ok {
		return v.(Shape), true
	}

	pkgPath, err := findPackagePath(x.PkgImportName)
	if err != nil {
		log.Errorf("shape.LookupShape: could not find package path %s", err.Error())
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

			for _, y := range inferred.RetrieveStruct() {
				cache.Store(StructFullName(y), y)
			}
			for _, y := range inferred.RetrieveUnions() {
				cache.Store(UnionFullName(y), y)
			}

			// continue scanning
			return nil
		})

	if err != nil {
		log.Errorf("shape.LookupShape: error during walk %s", err.Error())
		return nil, false
	}

	// if not found, return nil, false
	if v, ok := cache.Load(key); ok {
		return v.(Shape), true
	}

	return nil, false
}

func findPackagePath(pkgImportName string) (string, error) {
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
			log.Infof("shape.findPackagePath: could not find go.mod file in CWD or parent directories %s, continue with other paths", cwd)
			break
		}

		modpath := path.Join(cwd, "go.mod")
		_, err := os.Stat(modpath)
		log.Infof("shape.findPackagePath: checking %s; %s", modpath, err)
		if err == nil {
			f, err := os.Open(modpath)
			if err != nil {
				log.Infof("shape.findPackagePath: could not open %s", cwd)
				continue
			}
			defer f.Close()

			data, err := io.ReadAll(f)
			if err != nil {
				log.Infof("shape.findPackagePath: could not read %s", cwd)
				continue
				//return "", fmt.Errorf("shape.findPackagePath: could not read %s; %w", cwd, err)
			} else {
				parsed, err := modfile.Parse(modpath, data, nil)
				if err != nil {
					log.Infof("shape.findPackagePath: could not parse go.mod %s", cwd)
					break
				} else {
					log.Infof("shape.findPackagePath: parsed go.mod %s", parsed.Module.Mod.Path)
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
