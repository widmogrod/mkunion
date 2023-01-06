package schema

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

//go:generate go run ../../cmd/mkunion/main.go -name=Schema
type (
	Value struct {
		V any
	}

	List struct {
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

type (
	WhenField struct {
		path   []string
		setter func() Setter
	}
)

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
		path:   path,
		setter: setter,
	}
}

type RuleMatcher interface {
	Match(path []any, x Schema) (Setter, bool)
}

var (
	_ RuleMatcher = (*WhenField)(nil)
)

func (r *WhenField) Match(path []any, x Schema) (Setter, bool) {
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
	e := s.r.Elem()
	if e.Kind() == reflect.Ptr {
		if s.deref == nil {
			s.deref = &e
			s.deref.Set(reflect.New(e.Type().Elem()))
		}
		y := s.deref.Elem()
		f := y.FieldByName(key)
		if f.IsValid() && f.CanSet() {
			f.Set(reflect.ValueOf(value))
			return nil
		}
	} else if e.Kind() == reflect.Struct {
		f := e.FieldByName(key)
		if f.IsValid() && f.CanSet() {
			f.Set(reflect.ValueOf(value))
			return nil
		}
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
