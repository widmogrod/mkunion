package schema

import (
	"reflect"
)

var none = &None{}

func MkNone() *None {
	return none
}

func MkBool(b bool) *Bool {
	return &Bool{B: b}
}

func MkInt(x int) *Number {
	return &Number{N: float64(x)}
}

func MkFloat(x float64) *Number {
	return &Number{N: x}
}

func MkBinary(b []byte) *Binary {
	return &Binary{B: b}
}

func MkString(s string) *String {
	return &String{S: s}
}

func MkList(items ...Schema) *List {
	return &List{
		Items: items,
	}
}
func MkMap(fields ...Field) *Map {
	return &Map{
		Field: fields,
	}
}

func MkField(name string, value Schema) Field {
	return Field{
		Name:  name,
		Value: value,
	}
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

	// mapBuilderCanProcessRawMapSchema returns marks special class of MapBuilder that they can work with raw Schema value,
	// and don't need go value that was decoded using default schemaToGo.
	// in technical terms, it disables recursive call to schemaToGo
	mapBuilderCanProcessRawMapSchema interface {
		BuildFromMapSchema(x *Map) (any, error)
	}
)

//go:generate go run ../../cmd/mkunion/main.go -name=Schema -skip-extension=schema,shape
type (
	None   struct{}
	Bool   struct{ B bool }
	Number struct{ N float64 }
	String struct{ S string }
	Binary struct{ B []byte }
	List   struct {
		Items []Schema
	}
	Map struct {
		Field []Field
	}
)

type (
	Marshaler interface {
		MarshalSchema() (*Map, error)
	}

	Unmarshaler interface {
		UnmarshalSchema(x *Map) error
	}
)

type Field struct {
	Name  string
	Value Schema
}

type UnionInformationRule interface {
	UnionType() reflect.Type
	VariantsTypes() []reflect.Type
	IsUnionOrUnionType(t reflect.Type) bool
}

func UseStruct(t any) TypeMapDefinition {
	// Optimisation: When struct has its own definition how to populate it from schema,
	// we can use it instead of costly StructDefinition, that is based on reflection.
	if from, ok := t.(Unmarshaler); ok {
		// here is assumption that t is pointer to struct
		tType := reflect.ValueOf(from).Type().Elem()
		return UseSelfUnmarshallingStruct(func() Unmarshaler {
			// that's why here we create new emtpy type using reflection
			return reflect.New(tType).Interface().(Unmarshaler)
		})
	}

	return UseReflectionUnmarshallingStruct(t)
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
	MapDefFor(x *Map, path []string, config *goConfig) (TypeMapDefinition, bool)
	SchemaToUnionType(x any, schema Schema, config *goConfig) (Schema, bool)
}
