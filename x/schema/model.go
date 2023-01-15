package schema

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
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

type (
	WhenField struct {
		path        []string
		setter      func() Setter
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

func UseStruct(t any) func() Setter {
	rt := reflect.TypeOf(t)

	isNotStruct := rt.Kind() != reflect.Struct
	isNotPointerToStruct :=
		rt.Kind() == reflect.Pointer &&
			rt.Elem().Kind() != reflect.Struct

	if isNotStruct && isNotPointerToStruct {
		panic(fmt.Sprintf("UseStruct: not a struct, but %T", t))
	}

	return func() Setter {
		return &StructSetter{
			r: reflect.New(rt),
		}
	}
}

func WhenPath(path []string, setter func() Setter) *WhenField {
	return &WhenField{
		path:        path,
		unwrapField: "",
		setter:      setter,
	}
}

type RuleMatcher interface {
	MatchPath(path []any, x Schema) (Setter, bool)
	UnwrapField(x *Map) (Schema, bool, string)
}

var (
	_ RuleMatcher = (*WhenField)(nil)
)

func (r *WhenField) MatchPath(path []any, x Schema) (Setter, bool) {
	if len(r.path) == 2 {
		if len(path) < 1 {
			return nil, false
		}

		isAnyPath := r.path[0] == "*"
		if isAnyPath {
			//parts := strings.Split(r.path[1], "?.")
			lastPathItem := path[len(path)-1]
			if r.path[1] == lastPathItem {
				return r.setter(), true
			}
			return nil, false
		}
	}

	if len(path) != len(r.path) {
		return nil, false
	}

	for i := range r.path {
		parts := strings.Split(r.path[i], "?.")
		if path[i] != parts[0] {
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

	return r.setter(), true
}

type (
	StructSetter struct {
		r     reflect.Value
		deref *reflect.Value
	}
	NativeMap struct {
		m map[string]any
	}
	NativeList struct {
		l []any
	}
)

type Setter interface {
	Set(k string, value any) error
	Get() any
}

var (
	_ Setter = (*StructSetter)(nil)
	_ Setter = (*NativeMap)(nil)
	_ Setter = (*NativeList)(nil)
)

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
		// Try to do graceful conversion of types
		// This is LOSS-FULL conversion for some types
		switch f.Type().Kind() {
		case reflect.Int,
			reflect.Int8,
			reflect.Int16,
			reflect.Int32,
			reflect.Int64:
			switch v := value.(type) {
			case float32:
				f.SetInt(int64(v))
			case float64:
				f.SetInt(int64(v))
			}
			return nil

		case reflect.Uint,
			reflect.Uint8,
			reflect.Uint16,
			reflect.Uint32,
			reflect.Uint64:
			switch v := value.(type) {
			case float32:
				f.SetUint(uint64(v))
			case float64:
				f.SetUint(uint64(v))
			}

		case reflect.Float32,
			reflect.Float64:
			switch v := value.(type) {
			case float32:
				f.SetFloat(float64(v))
			case float64:
				f.SetFloat(v)
			}

		case reflect.Slice:
			// when struct field has type like []string
			// and value that should be set is []interface{} but element inside is string
			// do conversion!
			v := reflect.ValueOf(value)
			if v.Kind() == reflect.Slice && v.Len() > 0 {
				destinationSliceType := f.Type().Elem().Kind()
				inputSliceType := v.Index(0).Elem().Kind()

				if destinationSliceType == inputSliceType {
					st := reflect.SliceOf(f.Type().Elem())
					ss := reflect.MakeSlice(st, v.Len(), v.Len())

					for i := 0; i < v.Len(); i++ {
						ss.Index(i).Set(v.Index(i).Elem())
					}

					f.Set(ss)
				}
			} else {
				f.Set(v)
			}

		default:
			f.Set(reflect.ValueOf(value))
		}

		return nil
	}

	return errors.New(fmt.Sprintf("StructSetter:Set can't set value of type %T for key %s", value, key))
}

func (s *StructSetter) Get() any {
	return s.r.Elem().Interface()
}

func (s *NativeMap) Set(k string, value any) error {
	s.m[k] = value
	return nil
}

func (s *NativeMap) Get() any {
	return s.m
}

func (s *NativeList) Set(k string, value any) error {
	s.l = append(s.l, value)
	return nil
}

func (s *NativeList) Get() any {
	return s.l
}

func MkInt(x int) *Number {
	v := Number(x)
	return &v
}

func MkString(s string) *String {
	return (*String)(&s)
}

type TransformFunc = func(x any, schema Schema) (Schema, bool)

func WrapStruct[A any](structt A, inField string) TransformFunc {
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

func FewTransformations(xs ...[]TransformFunc) []TransformFunc {
	var out []TransformFunc
	for _, x := range xs {
		out = append(out, x...)
	}
	return out
}

func FewRules(xs ...[]RuleMatcher) []RuleMatcher {
	var out []RuleMatcher
	for _, x := range xs {
		out = append(out, x...)
	}
	return out
}
