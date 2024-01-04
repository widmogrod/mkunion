package shape

import (
	"fmt"
	"strings"
)

func ToStr(x Shape) string {
	return MatchShapeR1(
		x,
		func(x *Any) string {
			return "any"
		},
		func(x *RefName) string {
			return fmt.Sprintf("%s:%s.%s", x.PkgImportName, x.PkgName, x.Name)
		},
		func(x *AliasLike) string {
			panic("not implemented")
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
			return fmt.Sprintf("%s[]", ToStr(x.Element))
		},
		func(x *MapLike) string {
			return fmt.Sprintf("{[%s]: %s}", ToStr(x.Key), ToStr(x.Val))
		},
		func(x *StructLike) string {
			result := &strings.Builder{}
			_, _ = fmt.Fprintf(result, "%s:%s.%s={\n", x.PkgImportName, x.PkgName, x.Name)
			for _, field := range x.Fields {
				result.WriteString(fmt.Sprintf("\t%s %s\n", field.Name, ToStr(field.Type)))
			}
			result.WriteString("}")

			return result.String()
		},
		func(x *UnionLike) string {
			result := &strings.Builder{}
			_, _ = fmt.Fprintf(result, "union %s= ", x.Name)
			for idx, variant := range x.Variant {
				if idx > 0 {
					result.WriteString(" | ")
				}

				_, _ = fmt.Fprintf(result, "%s", ToStr(variant))
			}
			result.WriteString("}")

			return result.String()
		},
	)
}
