package shape

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"go/ast"
)

type FromASTOption func(x Shape)

func FromAST(x any, fx ...FromASTOption) Shape {
	switch y := x.(type) {
	case *ast.Ident:
		switch y.Name {
		case "any":
			return &Any{}
		case "string":
			return &PrimitiveLike{Kind: &StringLike{}}

		case "bool":
			return &PrimitiveLike{Kind: &BooleanLike{}}
		case "int", "int8", "int16", "int32", "int64",
			"uint", "uint8", "uint16", "uint32", "uint64",
			"float64", "float32", "byte", "rune":
			return &PrimitiveLike{Kind: &NumberLike{
				Kind: TypeStringToNumberKindMap[y.Name],
			}}
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
		result := FromAST(y.X, fx...)
		switch z := result.(type) {
		case *RefName:
			z.Indexed = append(z.Indexed, FromAST(y.Index, fx...))
			return z
		}

		panic(fmt.Errorf("shape.FromAST: unsupported IndexExpr: %#v", y))

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
			Element:          FromAST(y.Elt, fx...),
			ElementIsPointer: IsStarExpr(y.Elt),
			ArrayLen:         tryGetArrayLen(y.Len),
		}

	case *ast.MapType:
		return &MapLike{
			Key:          FromAST(y.Key, fx...),
			KeyIsPointer: IsStarExpr(y.Key),
			Val:          FromAST(y.Value, fx...),
			ValIsPointer: IsStarExpr(y.Value),
		}

	case *ast.SelectorExpr:
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
		}

		panic(fmt.Errorf("shape.FromAST: unsupported SelectorExpr: %#v", y))

	case *ast.StarExpr:
		result := FromAST(y.X, fx...)
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
