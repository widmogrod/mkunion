package shape

import (
	"reflect"
	"strings"
)

func FromGo(x any) Shape {
	switch y := x.(type) {
	case string:
		return &PrimitiveLike{Kind: &StringLike{}}
	case bool:
		return &PrimitiveLike{Kind: &BooleanLike{}}
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float64, float32:
		return &PrimitiveLike{Kind: &NumberLike{}}
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
		return &PrimitiveLike{Kind: &StringLike{}}
	case reflect.Bool:
		return &PrimitiveLike{Kind: &BooleanLike{}}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float64, reflect.Float32:
		return &PrimitiveLike{Kind: &NumberLike{}}
	case reflect.Slice:
		return &ListLike{
			Element: FromGoReflect(x.Elem(), infiniteRecursionFix),
			//ElementIsPointer: x.Elem().Kind() == reflect.Ptr,
		}
	case reflect.Map:
		return &MapLike{
			Key: FromGoReflect(x.Key(), infiniteRecursionFix),
			//KeyIsPointer: x.Key().Kind() == reflect.Ptr,
			Val: FromGoReflect(x.Elem(), infiniteRecursionFix),
			//ValIsPointer: x.Elem().Kind() == reflect.Ptr,
		}

	case reflect.Ptr:
		return &PointerLike{
			Type: FromGoReflect(x.Elem(), infiniteRecursionFix),
		}

	case reflect.Interface:
		shape, found := LookupShape(MkRefNameFromReflect(x))

		union, isUnion := shape.(*UnionLike)
		if isUnion {
			if result, found := infiniteRecursionFix[x.String()]; found {
				result2 := result.(*UnionLike)
				return &RefName{
					Name:          result2.Name,
					PkgName:       result2.PkgName,
					PkgImportName: result2.PkgImportName,
				}
			}

			result := union

			infiniteRecursionFix[x.String()] = result
			return result
		}

		if found {
			return shape
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
			PkgName:       GuessPkgName(x),
			PkgImportName: x.PkgPath(),
		}

		infiniteRecursionFix[x.String()] = result

		fields := make([]*FieldLike, 0, x.NumField())
		for i := 0; i < x.NumField(); i++ {
			field := x.Field(i)

			var guard Guard
			if enum := field.Tag.Get("enum"); enum != "" {
				guard = ConcatGuard(guard, &Enum{
					Val: strings.Split(enum, ","),
				})
			}
			if required := field.Tag.Get("required"); required == "true" {
				guard = ConcatGuard(guard, &Required{})
			}

			tags := ExtractTags(string(field.Tag))
			desc := TagsToDesc(tags)
			guard = TagsToGuard(tags)

			fields = append(fields, &FieldLike{
				Name:  field.Name,
				Type:  FromGoReflect(field.Type, infiniteRecursionFix),
				Desc:  desc,
				Guard: guard,
				Tags:  tags,
			})
		}

		if len(fields) > 0 {
			result.Fields = fields
		}
		return result
	}

	return &Any{}
}

func GuessPkgName(x reflect.Type) string {
	return GuessPkgNameFromPkgImportName(x.PkgPath())
}

func GuessPkgNameFromPkgImportName(x string) string {
	parts := strings.Split(x, "/")
	return parts[len(parts)-1]
}
