package schema

import (
	"encoding/json"
	"fmt"
	"github.com/widmogrod/mkunion/x/shared"
)

//go:generate go run ../../cmd/mkunion/main.go serde

//go:generate go run ../../cmd/mkunion/main.go -name=Schema -skip-extension=schema
type (
	None   struct{}
	Bool   bool
	Number float64
	String string
	Binary []byte
	List   []Schema
	Map    map[string]Schema
)

var (
	_ json.Unmarshaler = (*Map)(nil)
	_ json.Marshaler   = (*Map)(nil)
)

func (x *Map) UnmarshalJSON(bytes []byte) error {
	*x = make(Map)
	return shared.JSONParseObject(bytes, func(key string, value []byte) error {
		val, err := SchemaFromJSON(value)
		if err != nil {
			return fmt.Errorf("schema.Map.UnmarshalJSON: %w", err)
		}

		(*x)[key] = val
		return nil
	})
}

func (x *Map) MarshalJSON() ([]byte, error) {
	result := make(map[string]json.RawMessage)
	for key, value := range *x {
		bytes, err := SchemaToJSON(value)
		if err != nil {
			return nil, fmt.Errorf("schema.Map.MarshalJSON: %w", err)
		}
		result[key] = bytes
	}
	return json.Marshal(result)
}

//go:tag serde:"json"
type Field struct {
	Name  string
	Value Schema
}

var none = &None{}

func MkNone() *None {
	return none
}

func IsNone(x Schema) bool {
	_, ok := x.(*None)
	return ok
}

func MkBool(b bool) *Bool {
	return (*Bool)(&b)
}

func MkInt(x int) *Number {
	v := float64(x)
	return (*Number)(&v)
}

func MkFloat(x float64) *Number {
	return (*Number)(&x)
}

func MkBinary(b []byte) *Binary {
	v := Binary(b)
	return &v
}

func MkString(s string) *String {
	return (*String)(&s)
}

func MkList(items ...Schema) *List {
	result := make(List, len(items))
	copy(result, items)
	return &result
}
func MkMap(fields ...Field) *Map {
	var result = make(Map)
	for _, field := range fields {
		result[field.Name] = field.Value
	}
	return &result
}

func MkField(name string, value Schema) Field {
	return Field{
		Name:  name,
		Value: value,
	}
}
