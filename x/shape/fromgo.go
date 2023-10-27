package shape

import (
	"github.com/widmogrod/mkunion/x/schema"
	"reflect"
	"strings"
)

func FromGo(x any) Shape {
	switch y := x.(type) {
	case string:
		return &StringLike{}
	case bool:
		return &BooleanLike{}
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float64, float32:
		return &NumberLike{}
	case []any:
		return &ListLike{
			Element: FromGo(y[0]),
		}
	case map[any]any:
		return &MapLike{
			Key: FromGo(y[0]),
			Val: FromGo(y[1]),
		}
	}

	return FromGoReflect(reflect.TypeOf(x), make(map[string]Shape))
}

func FromGoReflect(x reflect.Type, infiniteRecursionFix map[string]Shape) Shape {
	switch x.Kind() {
	case reflect.String:
		return &StringLike{}
	case reflect.Bool:
		return &BooleanLike{}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float64, reflect.Float32:
		return &NumberLike{}
	case reflect.Slice:
		return &ListLike{
			Element: FromGoReflect(x.Elem(), infiniteRecursionFix),
		}
	case reflect.Map:
		return &MapLike{
			Key: FromGoReflect(x.Key(), infiniteRecursionFix),
			Val: FromGoReflect(x.Elem(), infiniteRecursionFix),
		}

	case reflect.Ptr:
		return FromGoReflect(x.Elem(), infiniteRecursionFix)

	case reflect.Interface:
		union, variantTypes, found := schema.UnionOf(x)
		if found {
			if result, found := infiniteRecursionFix[x.String()]; found {
				result2 := result.(*UnionLike)
				return &RefName{
					Name:          result2.Name,
					PkgName:       result2.PkgName,
					PkgImportName: result2.PkgImportName,
				}
			}

			result := &UnionLike{
				Name:          union.Name(),
				PkgName:       guessPkgName(union),
				PkgImportName: union.PkgPath(),
			}

			infiniteRecursionFix[x.String()] = result

			variants := make([]Shape, 0, len(variantTypes))
			for _, variant := range variantTypes {
				variants = append(variants, FromGoReflect(variant.Elem(), infiniteRecursionFix))
			}

			result.Variant = variants
			return result
		}

		return &Any{}

	case reflect.Struct:
		if result, found := infiniteRecursionFix[x.String()]; found {
			result2 := result.(*StructLike)
			return &RefName{
				Name:          result2.Name,
				PkgName:       result2.PkgName,
				PkgImportName: result2.PkgImportName,
			}
		}

		result := &StructLike{
			Name:          x.Name(),
			PkgName:       guessPkgName(x),
			PkgImportName: x.PkgPath(),
		}

		infiniteRecursionFix[x.String()] = result

		fields := make([]*FieldLike, 0, x.NumField())
		for i := 0; i < x.NumField(); i++ {
			field := x.Field(i)
			fields = append(fields, &FieldLike{
				Name: field.Name,
				Type: FromGoReflect(field.Type, infiniteRecursionFix),
			})
		}

		result.Fields = fields
		return result
	}

	return &Any{}
}

func guessPkgName(x reflect.Type) string {
	parts := strings.Split(x.PkgPath(), "/")
	return parts[len(parts)-1]
}
