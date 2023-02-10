package schema

import (
	"reflect"
	"strings"
)

var _ GoRuleMatcher = (*UnionVariants[any])(nil)

type UnionVariants[A any] struct {
	variants    []A
	reflections []reflect.Type
	unique      map[string]struct{}
	pathToUnion map[string]*StructDefinition
}

func (u *UnionVariants[A]) SchemaToUnionType(x any, schema Schema) (Schema, bool) {
	_, ok := x.(A)
	if !ok {
		return nil, false
	}

	for i, variant := range u.variants {
		if reflect.TypeOf(variant) == reflect.TypeOf(x) {
			return &Map{
				Field: []Field{
					{
						Name:  u.reflections[i].Elem().Name(),
						Value: schema,
					},
				},
			}, true
		}
	}

	panic("schema.UnionVariants.SchemaToUnionType: unreachable")
}

func (u *UnionVariants[A]) MapDefFor(x *Map, path []string) (TypeMapDefinition, bool) {
	k := strings.Join(path, ".")
	if mapDef, ok := u.pathToUnion[k]; ok {
		return mapDef, true
	}

	if len(x.Field) != 1 {
		return nil, false
	}

	for i, _ := range u.variants {
		if x.Field[0].Name == u.reflections[i].Elem().Name() {
			ss := make([]string, len(path)+1)
			copy(ss, path)
			ss[len(path)] = u.reflections[i].Elem().Name()

			u.pathToUnion[strings.Join(ss, ".")] = UseStruct(u.variants[i])
			return unionMap, true
		}
	}

	return nil, false
}
