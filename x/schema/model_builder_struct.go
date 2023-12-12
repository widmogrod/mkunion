package schema

import (
	"errors"
	"fmt"
	"reflect"
)

func UseReflectionUnmarshallingStruct(t any) TypeMapDefinition {
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

var _ MapBuilder = (*StructBuilder)(nil)

type StructBuilder struct {
	original any
	r        reflect.Value
	deref    *reflect.Value

	wellDefinedFromConversion func(x Schema, r reflect.Type) any
}

type wellDefinedSupported interface {
	WithWellDefinedTypesConversion(from func(x Schema, r reflect.Type) any)
}

func (s *StructBuilder) WithWellDefinedTypesConversion(from func(x Schema, r reflect.Type) any) {
	s.wellDefinedFromConversion = from
}

func (s *StructBuilder) Set(key string, value any) error {
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

	return errors.New(fmt.Sprintf("schema.StructBuilder.Set can't set value of type %T for key %s", value, key))
}

func (s *StructBuilder) set(f reflect.Value, value any) error {
	if s.wellDefinedFromConversion != nil {
		// TODO this is not optimal, because we are doing conversion twice
		v := FromGo(value)
		if result := s.wellDefinedFromConversion(v, f.Type()); result != nil {
			val := reflect.ValueOf(result)
			if f.Type().Kind() == reflect.Ptr {
				val = reflect.New(f.Type().Elem())
				val.Elem().Set(reflect.ValueOf(result))
			}

			f.Set(val.Convert(f.Type()))
			return nil
		}
	}

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
		switch v := value.(type) {
		case []byte:
			f.SetBytes(v)
			return nil
		}

		// when struct field has type like []string
		// and value that should be set is []interface{} but element inside is string
		// do conversion!
		v := reflect.ValueOf(value)
		if v.Len() == 0 {
			return nil
		}

		if f.Type().Elem().Kind() == reflect.Uint8 { // Check if the field is a byte slice
			switch vv := v.Interface().(type) {
			case []byte:
				f.SetBytes(vv)
				return nil
			case Binary:
				f.SetBytes(vv)
				return nil
			}
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

	return errors.New(fmt.Sprintf("schema.StructBuilder.set can't set value of type %T for key that expects type %s", value, f.String()))
}

func (s *StructBuilder) Build() any {
	return s.r.Elem().Interface()
}
