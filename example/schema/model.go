package schema

import (
	"errors"
	"reflect"
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
	//TopLevel struct {
	//	rt reflect.Type
	//}
	WhenField struct {
		path   []string
		setter func() Setter
	}
)

//func MustTopLevel(t any) *TopLevel {
//	rt := reflect.TypeOf(t)
//	if rt.Kind() == reflect.Ptr {
//		rt = rt.Elem()
//	}
//
//	if rt.Kind() != reflect.Struct {
//		panic("MustTopLevel: not a struct")
//	}
//
//	return &TopLevel{rt: rt}
//}

func UseStruct(t any) func() Setter {
	rt := reflect.TypeOf(t)
	//if rt.Kind() == reflect.Ptr {
	//	rt = rt.Elem()
	//}
	//
	//if rt.Kind() != reflect.Struct {
	//	panic("MustTopLevel: not a struct")
	//}

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
	Match(path []any) (Setter, bool)
}

var (
	_ RuleMatcher = (*WhenField)(nil)
)

func (r *WhenField) Match(path []any) (Setter, bool) {
	if len(path) != len(r.path) {
		return nil, false
	}

	for i := range r.path {
		if path[i] != r.path[i] {
			return nil, false
		}
	}

	return r.setter(), true
}

type (
	StructSetter struct {
		r reflect.Value
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
		e.Set(reflect.New(e.Type().Elem()))
		y := e.Elem()
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

	panic(errors.New("StructSetter:Set can't set"))
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
