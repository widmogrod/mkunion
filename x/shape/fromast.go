package shape

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"go/ast"
)

func FromAst(x any, fx ...func(x Shape)) Shape {
	switch y := x.(type) {
	case *ast.Ident:
		switch y.Name {
		case "any":
			return &Any{}
		case "string":
			result := &StringLike{}
			for _, f := range fx {
				f(result)
			}
			return result

		case "bool":
			return &BooleanLike{}
		case "int", "int8", "int16", "int32", "int64",
			"uint", "uint8", "uint16", "uint32", "uint64",
			"float64", "float32", "byte", "rune":
			return &NumberLike{
				Kind: TypeStringToNumberKindMap[y.Name],
			}
		default:
			if !y.IsExported() {
				log.Infof("formast: skipping non exported type %s", y.Name)
				return &Any{}
			}

			result := &RefName{
				Name:          y.String(),
				PkgName:       "",
				PkgImportName: "",
				Indexed:       nil,
			}

			for _, f := range fx {
				f(result)
			}

			return result
		}

	case *ast.IndexExpr:
		result := FromAst(y.X, fx...)
		switch z := result.(type) {
		case *RefName:
			z.Indexed = append(z.Indexed, FromAst(y.Index, fx...))
			return z
		}

		panic(fmt.Errorf("shape.FromAst: unsupported IndexExpr: %#v", y))

	case *ast.IndexListExpr:
		result := FromAst(y.X, fx...)
		switch z := result.(type) {
		case *RefName:
			for _, x := range y.Indices {
				z.Indexed = append(z.Indexed, FromAst(x, fx...))
			}
			return z
		}

	case *ast.ArrayType:
		return &ListLike{
			Element:          FromAst(y.Elt, fx...),
			ElementIsPointer: IsStarExpr(y.Elt),
		}

	case *ast.MapType:
		return &MapLike{
			Key:          FromAst(y.Key, fx...),
			KeyIsPointer: IsStarExpr(y.Key),
			Val:          FromAst(y.Value, fx...),
			ValIsPointer: IsStarExpr(y.Value),
		}

	case *ast.SelectorExpr:
		switch z := y.X.(type) {
		case *ast.Ident:
			var result Shape
			if y.Sel != nil {
				result = &RefName{
					Name:          y.Sel.String(),
					PkgName:       z.Name,
					PkgImportName: "",
					//IsPointer: IsStarExpr(y),
					Indexed: nil,
				}
			} else {
				result = &RefName{
					Name:          z.Name,
					PkgName:       "",
					PkgImportName: "",
					Indexed:       nil,
				}
			}
			for _, f := range fx {
				f(result)
			}

			return result
		}

		panic(fmt.Errorf("shape.FromAst: unsupported SelectorExpr: %#v", y))

	case *ast.StarExpr:
		result := FromAst(y.X, fx...)
		switch z := result.(type) {
		case *RefName:
			z.IsPointer = true
		}

		return result
	}

	return &Any{}
}

func IsStarExpr(x ast.Expr) bool {
	_, ok := x.(*ast.StarExpr)
	return ok
}

func InjectPkgImportName(pkgNameToImportName map[string]string) func(x Shape) {
	return func(x Shape) {
		switch y := x.(type) {
		case *RefName:
			isPkgNotSet := y.PkgName != "" && y.PkgImportName == ""
			if isPkgNotSet {
				y.PkgImportName = pkgNameToImportName[y.PkgName]
			}

		case *StructLike:
			isPkgNotSet := y.PkgName != "" && y.PkgImportName == ""
			if isPkgNotSet {
				y.PkgImportName = pkgNameToImportName[y.PkgName]
			}

		case *UnionLike:
			isPkgNotSet := y.PkgName != "" && y.PkgImportName == ""
			if isPkgNotSet {
				y.PkgImportName = pkgNameToImportName[y.PkgName]
			}

		case *AliasLike:
			isPkgNotSet := y.PkgName != "" && y.PkgImportName == ""
			if isPkgNotSet {
				y.PkgImportName = pkgNameToImportName[y.PkgName]
			}
		}
	}
}
func InjectPkgName(pkgImportName, pkgName string) func(x Shape) {
	return func(x Shape) {
		if !IsNamed(x) {
			return
		}

		switch y := x.(type) {
		case *RefName:
			isPkgNotSet := y.PkgName == ""
			if isPkgNotSet {
				y.PkgName = pkgName
				y.PkgImportName = pkgImportName
			}

		case *StructLike:
			isPkgNotSet := y.PkgName == ""
			if isPkgNotSet {
				y.PkgName = pkgName
				y.PkgImportName = pkgImportName
			}

		case *UnionLike:
			isPkgNotSet := y.PkgName == ""
			if isPkgNotSet {
				y.PkgName = pkgName
				y.PkgImportName = pkgImportName
			}

		case *AliasLike:
			isPkgNotSet := y.PkgName == ""
			if isPkgNotSet {
				y.PkgName = pkgName
				y.PkgImportName = pkgImportName
			}
		}
	}
}

func IsNamed(x Shape) bool {
	switch x.(type) {
	case *RefName, *StructLike, *UnionLike, *AliasLike:
		return true
	}

	return false
}
