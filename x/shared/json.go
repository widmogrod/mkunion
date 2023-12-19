package shared

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
)

var (
	registerJSONMarshaller = sync.Map{}
	registerType           = sync.Map{}
)

type serde[A any] struct {
	from func([]byte) (A, error)
	to   func(A) ([]byte, error)
}

func TypeRegistryLoad(typeFullName string) (any, bool) {
	return registerType.Load(typeFullName)
}

func TypeRegistryLoadFromReflect(x reflect.Type) (any, bool) {
	if x.Kind() == reflect.Ptr {
		x = x.Elem()
	}

	return TypeRegistryLoad(FullTypeName(x))
}

func FullTypeName(x reflect.Type) string {
	if x.Kind() == reflect.Ptr {
		x = x.Elem()
	}

	// native types
	if x.PkgPath() == "" {
		return x.Name()
	}

	return fmt.Sprintf("%s.%s", x.PkgPath(), x.Name())
}

func JSONMarshallerRegister[A any](
	fullName string,
	from func([]byte) (A, error),
	to func(A) ([]byte, error),
) {
	destinationTypePtr := new(A)
	registerType.Store(fullName, *destinationTypePtr)

	registerJSONMarshaller.Store(fullName, serde[any]{
		from: func(bytes []byte) (any, error) {
			return from(bytes)
		},
		to: func(a any) ([]byte, error) {
			if x, ok := a.(A); ok {
				return to(x)
			}

			return nil, fmt.Errorf("shared.JSONMarshallerRegister: expected %T, given %+#v", new(A), a)
		},
	})
}

func JSONUnmarshal[A any](data []byte) (A, error) {
	var destinationTypePtr *A = new(A)
	var destinationType A = *destinationTypePtr

	valuePtr, destinationPtrMarshaller := any(destinationTypePtr).(json.Unmarshaler)
	if destinationPtrMarshaller {
		// convert source to pointer, since this is only way to use native marshaller
		err := valuePtr.UnmarshalJSON(data)
		if err != nil {
			return destinationType, fmt.Errorf("shared.JSONUnmarshal: in shourt circut; destination ptr unmarshal; %w", err)
		}
		return *(any(valuePtr).(*A)), nil
	}

	if JSONIsNativePath(destinationType) {
		result := new(A)
		err := json.Unmarshal(data, result)
		if err != nil {
			return destinationType, fmt.Errorf("shared.JSONUnmarshal: use native; %w", err)
		}
		return *result, nil
	}

	key := FullTypeName(reflect.TypeOf(new(A)))
	fromTo, ok := registerJSONMarshaller.Load(key)
	if !ok {
		err := json.Unmarshal(data, &destinationType)
		if err != nil {
			return destinationType, fmt.Errorf("shared.JSONUnmarshal: use native fallback; %w", err)
		}
		return destinationType, nil
	}

	// no data, no need to unmarshall
	if data == nil || bytes.Equal(data, []byte("null")) {
		return destinationType, nil
	}

	result, err := fromTo.(serde[any]).from(data)
	if err != nil {
		return destinationType, fmt.Errorf("shared.JSONUnmarshal: serde err; %w", err)
	}

	if result == nil {
		return destinationType, nil
	}

	return result.(A), nil
}

func JSONMarshal[A any](x any) ([]byte, error) {
	if x == nil {
		return nil, nil
	}

	var destinationTypePtr *A = new(A)
	var destinationType A = *destinationTypePtr

	_, destinationMarshaller := any(destinationType).(json.Marshaler)
	_, destinationPtrMarshaller := any(destinationTypePtr).(json.Marshaler)
	value, valueMarshaller := x.(json.Marshaler)
	y, destinationAndSourceAreTheSame := x.(A)

	// union interfaces (visitor pattern), are not marshalable
	// but if destination type destinationAndSourceAreTheSame marshalable, we can use it
	if destinationAndSourceAreTheSame {
		// simple case when we can use native marshaller
		if destinationMarshaller && valueMarshaller {
			out, err := value.MarshalJSON()
			if err != nil {
				return out, fmt.Errorf("shared.JSONMarshal: in shourt circut; value marshaller; %w", err)
			}
			return out, nil
		} else if destinationPtrMarshaller {
			// convert source to pointer, since this is only way to use native marshaller
			if z, ok := any(&y).(json.Marshaler); ok {
				out, err := z.MarshalJSON()
				if err != nil {
					return out, fmt.Errorf("shared.JSONMarshal: in shourt circut; value marshaller; %w", err)
				}
				return out, nil
			}
		}
	}

	if JSONIsNativePath(destinationType) {
		out, err := json.Marshal(x)
		if err != nil {
			return out, fmt.Errorf("shared.JSONMarshal: in shourt circut; %w", err)
		}
		return out, nil
	}

	// choose the right marshaller
	// of field type, not the current value type
	key := FullTypeName(reflect.TypeOf(new(A)))
	fromTo, ok := registerJSONMarshaller.Load(key)
	if !ok {
		date, err := json.Marshal(x)
		if err != nil {
			return nil, fmt.Errorf("shared.JSONMarshal: in fallback; %w", err)
		}

		return date, nil
	}

	out, err := fromTo.(serde[any]).to(x)
	if err != nil {
		return nil, fmt.Errorf("shared.JSONMarshal: in serde; %w", err)
	}

	return out, nil
}

func JSONIsNativePath(x any) bool {
	switch x.(type) {
	case
		any,
		string,
		*string,
		int, int8, int16, int32, int64,
		*int, *int8, *int16, *int32, *int64,
		uint, uint8, uint16, uint32, uint64,
		*uint, *uint8, *uint16, *uint32, *uint64,
		float32, float64,
		*float32, *float64,
		complex64, complex128,
		*complex64, *complex128,
		bool,
		*bool,
		[]byte,
		*[]byte,
		//json.Marshaler,
		//json.Unmarshaler,
		json.RawMessage,
		[]json.RawMessage,
		*[]json.RawMessage,
		map[string]json.RawMessage,
		*map[string]json.RawMessage:
		return true
	}

	return false
}
