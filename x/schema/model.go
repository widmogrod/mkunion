package schema

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
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

func UnwrapStruct[A any](structt A, fromField string) *WhenField {
	return &WhenField{
		path:        []string{"*", fromField},
		unwrapField: fromField,
		setter:      UseStruct(structt),
	}
}

func UseStruct(t any) *StructDefinition {
	rt := reflect.TypeOf(t)
	isNotStruct := rt.Kind() != reflect.Struct
	isNotPointerToStruct :=
		rt.Kind() == reflect.Pointer &&
			rt.Elem().Kind() != reflect.Struct

	if isNotStruct && isNotPointerToStruct {
		panic(fmt.Sprintf("UseStruct: not a struct, but %T", t))
	}

	return &StructDefinition{
		t:  t,
		rt: rt,
	}
}

func UseTypeDef(definition TypeMapDefinition) TypeMapDefinition {
	return definition
}

func WhenPath(path []string, setter TypeMapDefinition) *WhenField {
	return &WhenField{
		path:        path,
		unwrapField: "",
		setter:      setter,
	}
}

type RuleMatcher interface {
	MatchPath(path []string, x Schema) (TypeMapDefinition, bool)
	UnwrapField(x *Map) (Schema, bool, string)
}

var (
	_ RuleMatcher = (*WhenField)(nil)
)

type (
	WhenField struct {
		path        []string
		setter      TypeMapDefinition
		unwrapField string
	}
)

func (r *WhenField) UnwrapField(x *Map) (Schema, bool, string) {
	if r.unwrapField == "" {
		return nil, false, ""
	}

	if len(x.Field) != 1 {
		return nil, false, ""
	}

	if x.Field[0].Name == r.unwrapField {
		return x.Field[0].Value, true, r.unwrapField
	}

	return nil, false, ""
}

func (r *WhenField) MatchPath(path []string, x Schema) (TypeMapDefinition, bool) {
	if len(r.path) == 1 && r.path[0] == "*" {
		return r.setter, true
	}

	if len(r.path) > 1 && r.path[0] == "*" {
		if len(path) < len(r.path)-1 {
			return nil, false
		}

		isAnyPath := r.path[0] == "*"
		if isAnyPath {
			pathLen := len(r.path)
			for i := 1; i < pathLen; i++ {
				//parts := strings.Split(r.path[1], "?.")
				// compare from the end
				if r.path[len(r.path)-i] != path[len(path)-i] && r.path[len(r.path)-i] != "*" {
					return nil, false
				}
			}
			return r.setter, true
		}
	}

	if len(path) != len(r.path) {
		return nil, false
	}

	for i := range r.path {
		parts := strings.Split(r.path[i], "?.")
		if path[i] != parts[0] && parts[0] != "*" {
			return nil, false
		}

		if len(parts) != 2 {
			continue
		}

		m, ok := x.(*Map)
		if !ok {
			return nil, false
		}

		found := false
		for _, f := range m.Field {
			if f.Name == parts[1] {
				found = true
				break
			}
		}

		if !found {
			return nil, false
		}
	}

	return r.setter, true
}

var _ ListBuilder = (*NativeList)(nil)

type NativeList struct {
	l []any
}

func (s *NativeList) NewListBuilder() ListBuilder {
	return &NativeList{
		l: nil,
	}
}

func (s *NativeList) Append(value any) error {
	s.l = append(s.l, value)
	return nil
}

func (s *NativeList) Build() any {
	return s.l
}

var _ MapBuilder = (*NativeMap)(nil)

type NativeMap struct {
	m map[string]any
}

func (s *NativeMap) NewMapBuilder() MapBuilder {
	return &NativeMap{
		m: make(map[string]any),
	}
}

func (s *NativeMap) Build() any {
	return s.m
}

func (s *NativeMap) Set(k string, value any) error {
	s.m[k] = value
	return nil
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

	return &StructSetter{
		orginal: s.t,
		r:       reflect.New(s.rt),
	}
}

type StructSetter struct {
	orginal any
	r       reflect.Value
	deref   *reflect.Value
}

func (s *StructSetter) Set(key string, value any) error {
	if value == nil {
		return nil
	}

	var f reflect.Value
	e := s.r.Elem()
	if e.Kind() == reflect.Ptr {
		if s.deref == nil {
			s.deref = &e
			s.deref.Set(reflect.New(e.Type().Elem()))
		}
		y := s.deref.Elem()
		f = y.FieldByName(key)
	} else if e.Kind() == reflect.Struct {
		f = e.FieldByName(key)
	}

	if f.IsValid() && f.CanSet() {
		// Try to do graceful conversion of reflections
		// This is LOSS-FULL conversion for some reflections
		return s.set(f, value)
	}

	return errors.New(fmt.Sprintf("schema.StructSetter.Set can't set value of type %T for key %s", value, key))
}

func (s *StructSetter) set(f reflect.Value, value any) error {
	switch f.Type().Kind() {
	case reflect.Pointer:
		v := reflect.ValueOf(value)
		if v.Type().AssignableTo(f.Type()) {
			f.Set(v)
			return nil
		} else if v.Type().ConvertibleTo(f.Type()) {
			f.Set(v.Convert(f.Type()))
			return nil
		}

		if f.IsNil() {
			vv := reflect.New(f.Type().Elem())
			err := s.set(vv, value)
			if err != nil {
				return err
			}
			f.Set(vv)

			return nil
		} else {
			return s.set(f.Elem(), value)
		}

	case reflect.Interface,
		reflect.Struct:
		v := reflect.ValueOf(value)
		if v.Type().AssignableTo(f.Type()) {
			f.Set(v)
			return nil
		} else if v.Type().ConvertibleTo(f.Type()) {
			f.Set(v.Convert(f.Type()))
			return nil
		}

	// Try to do graceful conversion of reflections
	// This is LOSS-FULL conversion for some reflections
	case reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64:
		switch v := value.(type) {
		case float32:
			f.SetInt(int64(v))
			return nil
		case float64:
			f.SetInt(int64(v))
			return nil
		}

	case reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64:
		switch v := value.(type) {
		case float32:
			f.SetUint(uint64(v))
			return nil
		case float64:
			f.SetUint(uint64(v))
			return nil
		}

	case reflect.Float32,
		reflect.Float64:
		switch v := value.(type) {
		case float32:
			f.SetFloat(float64(v))
			return nil
		case float64:
			f.SetFloat(v)
			return nil
		}

	case reflect.String:
		switch v := value.(type) {
		case string:
			f.SetString(v)
			return nil
		}

	case reflect.Bool:
		switch v := value.(type) {
		case bool:
			f.SetBool(v)
			return nil
		}

	case reflect.Slice:
		// when struct field has type like []string
		// and value that should be set is []interface{} but element inside is string
		// do conversion!
		v := reflect.ValueOf(value)
		if v.Len() == 0 {
			return nil
		}

		if v.Kind() == reflect.Slice {
			st := reflect.SliceOf(f.Type().Elem())
			ss := reflect.MakeSlice(st, v.Len(), v.Len())

			for i := 0; i < v.Len(); i++ {
				err := s.set(ss.Index(i), v.Index(i).Interface())
				if err != nil {
					return err
				}
			}

			f.Set(ss)
			return nil
		}

	case reflect.Map:
		// when struct field has type like map[string]string
		// and value that should be set is map[string]interface{} but element inside is string
		// do conversion!
		v := reflect.ValueOf(value)
		if v.Len() == 0 {
			return nil
		}

		if v.Kind() == reflect.Map {
			st := reflect.MapOf(f.Type().Key(), f.Type().Elem())
			ss := reflect.MakeMap(st)

			destinationType := f.Type().Elem()
			for _, key := range v.MapKeys() {
				newValue := reflect.New(destinationType).Elem()
				err := s.set(newValue, v.MapIndex(key).Interface())
				if err != nil {
					return err
				}
				ss.SetMapIndex(key, newValue)
			}

			f.Set(ss)
			return nil
		}
	}

	return errors.New(fmt.Sprintf("schema.StructSetter.set can't set value of type %T for key %s", value, f.String()))
}

func (s *StructSetter) Build() any {
	return s.r.Elem().Interface()
}

type TransformFunc = func(x any, schema Schema) (Schema, bool)

func WrapStruct[A any](_ A, inField string) TransformFunc {
	return func(x any, schema Schema) (Schema, bool) {
		_, ok := x.(A)
		if !ok {
			return nil, false
		}

		return &Map{
			Field: []Field{
				{
					Name:  inField,
					Value: schema,
				},
			},
		}, true
	}
}
