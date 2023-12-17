package shape

import (
	"fmt"
	"strings"
)

func ToGoPkgName(x Shape) string {
	return MustMatchShape(
		x,
		func(x *Any) string {
			return ""
		},
		func(x *RefName) string {
			return x.PkgName
		},
		func(x *AliasLike) string {
			return x.PkgName
		},
		func(x *BooleanLike) string {
			return ""
		},
		func(x *StringLike) string {
			return ""
		},
		func(x *NumberLike) string {
			return ""
		},
		func(x *ListLike) string {
			return ""
		},
		func(x *MapLike) string {
			return ""
		},
		func(x *StructLike) string {
			return x.PkgName
		},
		func(x *UnionLike) string {
			return x.PkgName
		},
	)
}

func ToGoPkgImportName(x Shape) string {
	return MustMatchShape(
		x,
		func(x *Any) string {
			return ""
		},
		func(x *RefName) string {
			return x.PkgImportName
		},
		func(x *AliasLike) string {
			return x.PkgImportName
		},
		func(x *BooleanLike) string {
			return ""
		},
		func(x *StringLike) string {
			return ""
		},
		func(x *NumberLike) string {
			return ""
		},
		func(x *ListLike) string {
			return ""
		},
		func(x *MapLike) string {
			return ""
		},
		func(x *StructLike) string {
			return x.PkgImportName
		},
		func(x *UnionLike) string {
			return x.PkgImportName
		},
	)
}

type ToGoTypeNameOption string

func WithInstantiation() ToGoTypeNameOption {
	return "instantiate"
}

const rootPackage = "root-package:"

func WithRootPackage(pkgName string) ToGoTypeNameOption {
	result := rootPackage + pkgName
	return ToGoTypeNameOption(result)
}

func packageWrap(options []ToGoTypeNameOption, pkgName string, value string) string {
	for _, option := range options {
		if !strings.HasPrefix(string(option), rootPackage) {
			continue
		}

		rootPkgName := string(option)[len(rootPackage):]
		if pkgName == rootPkgName {
			return value
		}
	}

	if pkgName == "" {
		return value
	}

	return fmt.Sprintf("%s.%s", pkgName, value)

}

func shouldInstantiate(options []ToGoTypeNameOption) bool {
	for _, x := range options {
		if x == "instantiate" {
			return true
		}
	}
	return false
}

func ToGoRemovePkgName(pkgName string, x string) string {
	return strings.ReplaceAll(x, fmt.Sprintf("%s.", pkgName), "")
}

func ToGoTypeName(x Shape, options ...ToGoTypeNameOption) string {
	return MustMatchShape(
		x,
		func(x *Any) string {
			return "any"
		},
		func(x *RefName) string {
			// this is when it's a parameter result or some other primitive type
			if x.PkgName == "" {
				return x.Name
			}

			result := packageWrap(options, x.PkgName, x.Name)
			if len(x.Indexed) > 0 {
				names := make([]string, 0)
				for _, name := range x.Indexed {
					names = append(names, ToGoTypeName(name, options...))
				}

				result = fmt.Sprintf("%s[%s]", result, strings.Join(names, ","))
			}

			return result
		},
		func(x *AliasLike) string {
			return packageWrap(options, x.PkgName, x.Name)
		},
		func(x *BooleanLike) string {
			return "bool"
		},
		func(x *StringLike) string {
			return "string"
		},
		func(x *NumberLike) string {
			return NumberKindToGoName(x.Kind)
		},
		func(x *ListLike) string {
			return fmt.Sprintf("[]%s",
				WrapPointerIf(ToGoTypeName(x.Element, options...), x.ElementIsPointer),
			)
		},
		func(x *MapLike) string {
			return fmt.Sprintf("map[%s]%s",
				WrapPointerIf(ToGoTypeName(x.Key, options...), x.KeyIsPointer),
				WrapPointerIf(ToGoTypeName(x.Val, options...), x.ValIsPointer),
			)
		},
		func(x *StructLike) string {
			if shouldInstantiate(options) {
				typeParams := ToGoTypeParamsTypes(x)
				if len(typeParams) > 0 {
					instantiatedTypeParams := make([]string, len(typeParams))
					for i, y := range typeParams {
						instantiatedTypeParams[i] = ToGoTypeName(y, options...)
					}

					result := fmt.Sprintf("%s[%s]",
						x.Name,
						strings.Join(instantiatedTypeParams, ","),
					)

					return packageWrap(options, x.PkgName, result)
				}
			} else {
				typeParams := ToGoTypeParamsNames(x)
				if len(typeParams) > 0 {
					result := fmt.Sprintf("%s[%s]",
						x.Name,
						strings.Join(typeParams, ","),
					)

					return packageWrap(options, x.PkgName, result)
				}
			}

			return packageWrap(options, x.PkgName, x.Name)
		},
		func(x *UnionLike) string {
			return packageWrap(options, x.PkgName, x.Name)
		},
	)
}

func ToGoTypeParamsNames(x Shape) []string {
	return MustMatchShape(
		x,
		func(x *Any) []string {
			return nil
		},
		func(x *RefName) []string {
			return nil
		},
		func(x *AliasLike) []string {
			return nil
		},
		func(x *BooleanLike) []string {
			return nil
		},
		func(x *StringLike) []string {
			return nil
		},
		func(x *NumberLike) []string {
			return nil
		},
		func(x *ListLike) []string {
			return nil
		},
		func(x *MapLike) []string {
			return nil
		},
		func(x *StructLike) []string {
			typeParamsNames := make([]string, len(x.TypeParams))
			for i, y := range x.TypeParams {
				typeParamsNames[i] = y.Name
			}
			return typeParamsNames
		},
		func(x *UnionLike) []string {
			return nil
		},
	)
}

func ToGoTypeParamsTypes(x Shape) []Shape {
	return MustMatchShape(
		x,
		func(x *Any) []Shape {
			return nil
		},
		func(x *RefName) []Shape {
			return nil
		},
		func(x *AliasLike) []Shape {
			return nil
		},
		func(x *BooleanLike) []Shape {
			return nil
		},
		func(x *StringLike) []Shape {
			return nil
		},
		func(x *NumberLike) []Shape {
			return nil
		},
		func(x *ListLike) []Shape {
			return nil
		},
		func(x *MapLike) []Shape {
			return nil
		},
		func(x *StructLike) []Shape {
			typeParamsNames := make([]Shape, len(x.TypeParams))
			for i, y := range x.TypeParams {
				typeParamsNames[i] = y.Type
			}
			return typeParamsNames
		},
		func(x *UnionLike) []Shape {
			return nil
		},
	)
}

func WrapPointerIf(name string, isPointer bool) string {
	if isPointer {
		return fmt.Sprintf("*%s", name)
	}
	return name
}

func IsPrimitive(x Shape) bool {
	return MustMatchShape(
		x,
		func(x *Any) bool {
			return true
		},
		func(x *RefName) bool {
			return false
		},
		func(x *AliasLike) bool {
			return false
		},
		func(x *BooleanLike) bool {
			return true
		},
		func(x *StringLike) bool {
			return true
		},
		func(x *NumberLike) bool {
			return true
		},
		func(x *ListLike) bool {
			return true
		},
		func(x *MapLike) bool {
			return true
		},
		func(x *StructLike) bool {
			return false
		},
		func(x *UnionLike) bool {
			return false
		},
	)
}
