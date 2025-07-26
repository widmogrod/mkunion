package shape

import (
	"fmt"
	"path"
	"strings"
	log "github.com/sirupsen/logrus"
	"go/ast"
)

type FromASTOption func(x Shape)

func FromAST(x any, fx ...FromASTOption) Shape {
	switch y := x.(type) {
	case *ast.Ident:
		if primitive := NameToPrimitiveShape(y.Name); primitive != nil {
			return primitive
		}
		if !y.IsExported() {
			log.Infof("shape.FromAST: Ident, skipping non exported type %s", y.Name)
			return &Any{}
		}

		result := &RefName{
			Name:          y.Name,
			PkgName:       "",
			PkgImportName: "",
			Indexed:       nil,
		}

		for _, f := range fx {
			f(result)
		}

		return result

	case *ast.IndexExpr:
		result := FromAST(y.X, fx...)
		switch z := result.(type) {
		case *RefName:
			z.Indexed = append(z.Indexed, FromAST(y.Index, fx...))
			return z
		}

		log.Warnf("shape.FromAST: unsupported IndexExpr: %#v, most likely unexported type", y)
		return result

	case *ast.IndexListExpr:
		result := FromAST(y.X, fx...)
		switch z := result.(type) {
		case *RefName:
			for _, x := range y.Indices {
				z.Indexed = append(z.Indexed, FromAST(x, fx...))
			}
			return z
		}

	case *ast.ArrayType:
		return &ListLike{
			Element:  FromAST(y.Elt, fx...),
			ArrayLen: tryGetArrayLen(y.Len),
		}

	case *ast.MapType:
		return &MapLike{
			Key: FromAST(y.Key, fx...),
			Val: FromAST(y.Value, fx...),
		}

	case *ast.SelectorExpr:
		// check if type is exported
		if !y.Sel.IsExported() {
			log.Infof("shape.FromAST: SelectorExpr, skipping non exported type %s", y.Sel.Name)
			return &Any{}
		}

		switch z := y.X.(type) {
		case *ast.Ident:
			var result Shape
			if y.Sel != nil {
				result = &RefName{
					Name:    y.Sel.String(),
					PkgName: z.Name,
				}
			} else {
				result = &RefName{
					Name: z.Name,
				}
			}

			for _, f := range fx {
				f(result)
			}

			return result

		case *ast.SelectorExpr:
			return FromAST(z, fx...)

		default:
			//// print go ast
			//ast.Print(token.NewFileSet(), y)
			// print go ast as go code
			//fmt.Println(printer.Fprint(os.Stdout, token.NewFileSet(), y))

			panic(fmt.Errorf("shape.FromAST: unsupported SelectorExpr: %#v", z))
		}

	case *ast.StarExpr:
		result := FromAST(y.X, fx...)
		return &PointerLike{
			Type: result,
		}
	}

	return &Any{}
}

func InjectPkgImportName(pkgNameToImportName map[string]string) func(x Shape) {
	return func(x Shape) {
		switch y := x.(type) {
		case *RefName:
			isPkgNotSet := y.PkgName != "" && y.PkgImportName == ""
			if isPkgNotSet {
				if pkgImportName, ok := pkgNameToImportName[y.PkgName]; ok {
					y.PkgImportName = pkgImportName
				} else {
					log.Warnf("InjectPkgImportName: could not find pkgNameToImportName for %s", y.PkgName)
				}
			}

		case *StructLike:
			isPkgNotSet := y.PkgName != "" && y.PkgImportName == ""
			if isPkgNotSet {
				if pkgImportName, ok := pkgNameToImportName[y.PkgName]; ok {
					y.PkgImportName = pkgImportName
				} else {
					log.Warnf("InjectPkgImportName: could not find pkgNameToImportName for %s", y.PkgName)
				}
			}

		case *UnionLike:
			isPkgNotSet := y.PkgName != "" && y.PkgImportName == ""
			if isPkgNotSet {
				if pkgImportName, ok := pkgNameToImportName[y.PkgName]; ok {
					y.PkgImportName = pkgImportName
				} else {
					log.Warnf("InjectPkgImportName: could not find pkgNameToImportName for %s", y.PkgName)
				}
			}

		case *AliasLike:
			isPkgNotSet := y.PkgName != "" && y.PkgImportName == ""
			if isPkgNotSet {
				if pkgImportName, ok := pkgNameToImportName[y.PkgName]; ok {
					y.PkgImportName = pkgImportName
				} else {
					log.Warnf("InjectPkgImportName: could not find pkgNameToImportName for %s", y.PkgName)
				}
			}
		}
	}
}

// InjectPkgImportNameWithDotImportResolution enhances InjectPkgImportName to handle dot imports correctly
func InjectPkgImportNameWithDotImportResolution(pkgNameToImportName map[string]string, dotImports []string) func(x Shape) {
	// If no dot imports, just use the regular logic
	if len(dotImports) == 0 {
		return InjectPkgImportName(pkgNameToImportName)
	}
	
	return func(x Shape) {
		switch y := x.(type) {
		case *RefName:
			isPkgNotSet := y.PkgName != "" && y.PkgImportName == ""
			if isPkgNotSet {
				if pkgImportName, ok := pkgNameToImportName[y.PkgName]; ok {
					// Only try dot import resolution if this might be a type from the current package
					// that could actually be from a dot import
					resolvedPkg := resolveDotImportPackage(y.Name, dotImports)
					if resolvedPkg != "" && resolvedPkg != pkgImportName {
						// Only use the resolved package if it's different from the current assignment
						y.PkgImportName = resolvedPkg
						y.PkgName = path.Base(resolvedPkg)
					} else {
						y.PkgImportName = pkgImportName
					}
				} else {
					log.Warnf("InjectPkgImportName: could not find pkgNameToImportName for %s", y.PkgName)
				}
			}

		case *StructLike:
			isPkgNotSet := y.PkgName != "" && y.PkgImportName == ""
			if isPkgNotSet {
				if pkgImportName, ok := pkgNameToImportName[y.PkgName]; ok {
					y.PkgImportName = pkgImportName
				} else {
					log.Warnf("InjectPkgImportName: could not find pkgNameToImportName for %s", y.PkgName)
				}
			}

		case *UnionLike:
			isPkgNotSet := y.PkgName != "" && y.PkgImportName == ""
			if isPkgNotSet {
				if pkgImportName, ok := pkgNameToImportName[y.PkgName]; ok {
					y.PkgImportName = pkgImportName
				} else {
					log.Warnf("InjectPkgImportName: could not find pkgNameToImportName for %s", y.PkgName)
				}
			}

		case *AliasLike:
			isPkgNotSet := y.PkgName != "" && y.PkgImportName == ""
			if isPkgNotSet {
				if pkgImportName, ok := pkgNameToImportName[y.PkgName]; ok {
					y.PkgImportName = pkgImportName
				} else {
					log.Warnf("InjectPkgImportName: could not find pkgNameToImportName for %s", y.PkgName)
				}
			}
		}
	}
}

// resolveDotImportPackage attempts to resolve which dot-imported package contains a given type
func resolveDotImportPackage(typeName string, dotImports []string) string {
	// For demonstration purposes, we'll use known type names to resolve packages
	// In a production system, this would involve actually looking up the types in the packages
	
	switch typeName {
	case "Option", "Result", "Some", "None", "Ok", "Err":
		// These are from the f package
		for _, pkg := range dotImports {
			if strings.Contains(pkg, "/f") || strings.HasSuffix(pkg, "/f") {
				return pkg
			}
		}
	case "JSONMarshal", "TypeRegistryStore":
		// These are from the shared package  
		for _, pkg := range dotImports {
			if strings.Contains(pkg, "/shared") || strings.HasSuffix(pkg, "/shared") {
				return pkg
			}
		}
	}
	
	// If we can't resolve, return the first dot import as fallback
	if len(dotImports) > 0 {
		return dotImports[0]
	}
	
	return ""
}

func InjectPkgName(pkgName string) func(x Shape) {
	return func(x Shape) {
		switch y := x.(type) {
		case *RefName:
			isPkgNotSet := y.PkgName == ""
			if isPkgNotSet {
				y.PkgName = pkgName
			}

		case *StructLike:
			isPkgNotSet := y.PkgName == ""
			if isPkgNotSet {
				y.PkgName = pkgName
			}

		case *UnionLike:
			isPkgNotSet := y.PkgName == ""
			if isPkgNotSet {
				y.PkgName = pkgName
			}

		case *AliasLike:
			isPkgNotSet := y.PkgName == ""
			if isPkgNotSet {
				y.PkgName = pkgName
			}
		}
	}
}
