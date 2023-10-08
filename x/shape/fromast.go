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
			return &StringLike{}
		case "bool":
			return &BooleanLike{}
		case "int", "int8", "int16", "int32", "int64",
			"uint", "uint8", "uint16", "uint32", "uint64",
			"float64", "float32":
			return &NumberLike{}
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
			Element: FromAst(y.Elt, fx...),
		}
	case *ast.MapType:
		return &MapLike{
			Key: FromAst(y.Key, fx...),
			Val: FromAst(y.Value, fx...),
		}

	case *ast.SelectorExpr:
		return FromAst(y.X, fx...)

	case *ast.StarExpr:
		return FromAst(y.X, fx...)
	}

	return &Any{}
}

func InjectPkgName(pkgImportName, pkgName string) func(x Shape) {
	return func(x Shape) {
		y, ok := x.(*RefName)
		if ok && y.PkgName == "" {
			y.PkgName = pkgName
			y.PkgImportName = pkgImportName
		}
	}
}
