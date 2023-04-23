package schema

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
)

func MustDefineUnion[A any](xs ...A) *UnionVariants[A] {
	result := UnionVariants[A]{
		unique:      make(map[string]struct{}),
		pathToUnion: make(map[string]TypeMapDefinition),
	}

	for _, x := range xs {
		t := reflect.TypeOf(x)
		if _, ok := result.unique[t.String()]; ok {
			panic(fmt.Errorf("schema.MustDefineUnion: union variant %s already defined %T", t.String(), x))
		}
		result.variants = append(result.variants, x)
		result.reflections = append(result.reflections, t)
		result.unique[t.String()] = struct{}{}
	}

	return &result
}

var _ RuleMatcher = (*UnionVariants[any])(nil)

type UnionVariants[A any] struct {
	variants    []A
	reflections []reflect.Type
	unique      map[string]struct{}
	pathToUnion map[string]TypeMapDefinition

	lock sync.RWMutex
}

func (u *UnionVariants[A]) SchemaToUnionType(x any, schema Schema, config *goConfig) (Schema, bool) {
	_, ok := x.(A)
	if !ok {
		return nil, false
	}

	for i := range u.variants {
		// TODO: fix reflection!
		if u.reflections[i] == reflect.TypeOf(x) {
			return &Map{
				Field: []Field{
					{
						Name:  config.variantName(u.reflections[i]),
						Value: schema,
					},
				},
			}, true
		}
	}

	panic("schema.UnionVariants.SchemaToUnionType: unreachable")
}

func (u *UnionVariants[A]) MapDefFor(x *Map, path []string, config *goConfig) (TypeMapDefinition, bool) {
	// by using the path as a key, that's why this is first operation
	// to build correct type, it needs to be cached, and this is done
	// needs to unwrap it and then build the union type
	// this functions when it detects a map with only one field
	// Since union type is a map with only one field
	k := strings.Join(path, ".")
	u.lock.RLock()
	if mapDef, ok := u.pathToUnion[k]; ok {
		u.lock.RUnlock()
		return mapDef, true
	}
	u.lock.RUnlock()

	if len(x.Field) != 1 {
		return nil, false
	}

	for i := range u.variants {
		if x.Field[0].Name == config.variantName(u.reflections[i]) {
			ss := make([]string, len(path)+1)
			copy(ss, path)
			ss[len(path)] = config.variantName(u.reflections[i])

			u.lock.Lock()
			u.pathToUnion[strings.Join(ss, ".")] = UseStruct(u.variants[i])
			u.lock.Unlock()
			return unionMap, true
		}
	}

	return nil, false
}
