package schema

import (
	"fmt"
	"reflect"
)

func NewStructDef[A any]() *StructDefinition {
	var r A

	return &StructDefinition{
		t: r,
	}
}

var _ TypeMapDefinition = &StructDefinition{}

type StructDefinition struct {
	t any
}

func (s *StructDefinition) NewMapBuilder() MapBuilder {
	if builder, ok := s.t.(MapBuilder); ok {
		return builder
	}

	// TODO: fix this, reflection is done on every call
	return NewStructBuilder(s.t)
}

var _ MapBuilder = &StructSetter{}

func NewStructBuilder(t any) *StructSetter {
	rt := reflect.TypeOf(t)
	isNotStruct := rt.Kind() != reflect.Struct
	isNotPointerToStruct :=
		rt.Kind() == reflect.Pointer &&
			rt.Elem().Kind() != reflect.Struct

	if isNotStruct && isNotPointerToStruct {
		panic(fmt.Sprintf("UseStruct: not a struct, but %T", t))
	}

	return &StructSetter{
		orginal: t,
		r:       reflect.New(rt),
	}
}

type UnionMap struct {
	last any
}

var (
	_ TypeMapDefinition = (*UnionMap)(nil)
	_ MapBuilder        = (*UnionMap)(nil)
)

func (u *UnionMap) NewMapBuilder() MapBuilder {
	return &UnionMap{}
}

func (u *UnionMap) Set(key string, value any) error {
	u.last = value
	return nil
}

func (u *UnionMap) Build() any {
	return u.last
}
