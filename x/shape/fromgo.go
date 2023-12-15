package shape

import (
	"github.com/fatih/structtag"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/shared"
	"go/ast"
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
			Element:          FromGoReflect(x.Elem(), infiniteRecursionFix),
			ElementIsPointer: x.Elem().Kind() == reflect.Ptr,
		}
	case reflect.Map:
		return &MapLike{
			Key:          FromGoReflect(x.Key(), infiniteRecursionFix),
			KeyIsPointer: x.Key().Kind() == reflect.Ptr,
			Val:          FromGoReflect(x.Elem(), infiniteRecursionFix),
			ValIsPointer: x.Elem().Kind() == reflect.Ptr,
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
				Name:      field.Name,
				Type:      FromGoReflect(field.Type, infiniteRecursionFix),
				Desc:      desc,
				Guard:     guard,
				Tags:      tags,
				IsPointer: field.Type.Kind() == reflect.Ptr,
			})
		}

		result.Fields = fields
		return result
	}

	return &Any{}
}

func TagsToGuard(tags map[string]Tag) Guard {
	var result Guard
	if enum, ok := tags["enum"]; ok {
		result = ConcatGuard(result, &Enum{
			Val: append(strings.Split(enum.Value, ","), enum.Options...),
		})
	}
	if required, ok := tags["required"]; ok && required.Value == "true" {
		result = ConcatGuard(result, &Required{})
	}

	return result
}

func TagsToDesc(tags map[string]Tag) *string {
	if desc, ok := tags["desc"]; ok {
		// because tags are parsed according to the spec, we need to normalize options
		// since description field does not support options
		value := strings.Join(append([]string{desc.Value}, desc.Options...), ",")
		descStr := strings.Trim(value, `"`)
		if descStr != "" {
			return &descStr
		}
	}

	return nil
}

func ExtractDocumentTags(doc *ast.CommentGroup) map[string]Tag {
	result := make(map[string]Tag)

	comments := strings.Split(shared.Comment(doc), "\n")
	for _, comment := range comments {
		if strings.HasPrefix(comment, "go:tag") {
			tagString := strings.TrimPrefix(comment, "go:tag")
			tags := ExtractTags(tagString)
			for k, v := range tags {
				result[k] = v
			}
		}
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

func ExtractTags(tag string) map[string]Tag {
	tag = strings.Trim(tag, "`")
	tags, err := structtag.Parse(tag)
	if err != nil {
		return nil
	}

	if len(tags.Tags()) == 0 {
		return nil
	}

	result := make(map[string]Tag)
	for _, t := range tags.Tags() {
		result[t.Key] = Tag{
			Value:   t.Name,
			Options: t.Options,
		}
	}

	return result
}

func guessPkgName(x reflect.Type) string {
	parts := strings.Split(x.PkgPath(), "/")
	return parts[len(parts)-1]
}
