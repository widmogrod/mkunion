package shape

import (
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
				Name: y.String(),
			}

			for _, f := range fx {
				f(result)
			}

			return result
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
		return FromAst(y.X, fx...)

	case *ast.StarExpr:
		return FromAst(y.X, fx...)
	}

	return &Any{}
}

func IsStarExpr(x ast.Expr) bool {
	_, ok := x.(*ast.StarExpr)
	return ok
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

		case *StringLike:
			isPkgNotSet := y.Named.PkgName == ""
			if isPkgNotSet {
				y.Named.PkgName = pkgName
				y.Named.PkgImportName = pkgImportName
			}

		case *NumberLike:
			isPkgNotSet := y.Named.PkgName == ""
			if isPkgNotSet {
				y.Named.PkgName = pkgName
				y.Named.PkgImportName = pkgImportName
			}

		case *BooleanLike:
			isPkgNotSet := y.Named.PkgName == ""
			if isPkgNotSet {
				y.Named.PkgName = pkgName
				y.Named.PkgImportName = pkgImportName
			}

		case *ListLike:
			isPkgNotSet := y.Named.PkgName == ""
			if isPkgNotSet {
				y.Named.PkgName = pkgName
				y.Named.PkgImportName = pkgImportName
			}

		case *MapLike:
			isPkgNotSet := y.Named.PkgName == ""
			if isPkgNotSet {
				y.Named.PkgName = pkgName
				y.Named.PkgImportName = pkgImportName
			}
		}
	}
}

func IsNamed(x Shape) bool {
	switch y := x.(type) {
	case *RefName, *StructLike, *UnionLike:
		return true

	case *StringLike:
		if y.Named == nil {
			return false
		}

		return y.Named.Name != ""

	case *NumberLike:
		if y.Named == nil {
			return false
		}

		return y.Named.Name != ""

	case *BooleanLike:
		if y.Named == nil {
			return false
		}

		return y.Named.Name != ""

	case *ListLike:
		if y.Named == nil {
			return false
		}

		return y.Named.Name != ""

	case *MapLike:
		if y.Named == nil {
			return false
		}

		return y.Named.Name != ""

	case *Any:
		return false
	}

	return false
}
