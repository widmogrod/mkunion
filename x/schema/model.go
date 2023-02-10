package schema

import (
	"fmt"
	"reflect"
)

func MkInt(x int) *Number {
	v := Number(x)
	return &v
}

func MkString(s string) *String {
	return (*String)(&s)
}

type (
	TypeListDefinition interface {
		NewListBuilder() ListBuilder
	}
	TypeMapDefinition interface {
		NewMapBuilder() MapBuilder
	}
)

type (
	ListBuilder interface {
		Append(value any) error
		Build() any
	}

	MapBuilder interface {
		Set(key string, value any) error
		Build() any
	}
)

//go:generate go run ../../cmd/mkunion/main.go -name=Schema -skip-extension=schema
type (
	None   struct{}
	Bool   bool
	Number float64
	String string
	List   struct {
		Items []Schema
	}
	Map struct {
		Field []Field
	}
)

type Field struct {
	Name  string
	Value Schema
}

func UseStruct(t any) *StructDefinition {
	rt := reflect.TypeOf(t)
	isNotStruct := rt.Kind() != reflect.Struct
	isNotPointerToStruct :=
		rt.Kind() == reflect.Pointer &&
			rt.Elem().Kind() != reflect.Struct

	if isNotStruct && isNotPointerToStruct {
		panic(fmt.Sprintf("schema.UseStruct: not a struct, but %T", t))
	}

	return &StructDefinition{
		t:  t,
		rt: rt,
	}
}

var _ TypeMapDefinition = &StructDefinition{}

type StructDefinition struct {
	t any

	rt reflect.Type
}

func (s *StructDefinition) NewMapBuilder() MapBuilder {
	if builder, ok := s.t.(MapBuilder); ok {
		return builder
	}

	return &StructBuilder{
		original: s.t,
		r:        reflect.New(s.rt),
	}
}

func UseTypeDef(definition TypeMapDefinition) TypeMapDefinition {
	return definition
}

func WhenPath(path []string, setter TypeMapDefinition) *WhenField[struct{}] {
	return &WhenField[struct{}]{
		path:       path,
		typeMapDef: setter,
	}
}

type RuleMatcher interface {
	MapDefFor(x *Map, path []string) (TypeMapDefinition, bool)
	SchemaToUnionType(x any, schema Schema) (Schema, bool)
}

var _ RuleMatcher = (*WrapInMap[any])(nil)

type WrapInMap[A any] struct {
	ForType A
	InField string
}

func (w *WrapInMap[A]) MapDefFor(x *Map, path []string) (TypeMapDefinition, bool) {
	return nil, false
}

func (w *WrapInMap[A]) SchemaToUnionType(x any, schema Schema) (Schema, bool) {
	if _, ok := x.(A); !ok {
		return nil, false
	}

	return &Map{
		Field: []Field{
			{
				Name:  w.InField,
				Value: schema,
			},
		},
	}, true
}
