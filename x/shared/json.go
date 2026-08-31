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

//go:tag shape:"-"
type serde[A any] struct {
	from func([]byte) (A, error)
	to   func(A) ([]byte, error)
}

func TypeRegistryLoad(typeFullName string) (any, bool) {
	return registerType.Load(typeFullName)
}

func TypeRegistryStore[A any](typeFullName string) {
	destinationTypePtr := new(A)
	registerType.Store(typeFullName, *destinationTypePtr)
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
	TypeRegistryStore[A](fullName)

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

// JSONUnmarshal is a generic function to unmarshal json data into destination type
// that supports union types and fallback to native json.Unmarshal when available.
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
		// null is always a valid zero value, even for unregistered interfaces
		if data == nil || bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
			return destinationType, nil
		}

		if isUnregisterableInterface[A]() {
			return destinationType, fmt.Errorf("shared.JSONUnmarshal: type %q is an interface with no registered marshaller; import the package that registers it (generated init in *_union_gen.go) or check //go:tag mkunion:\",no-type-registry\"", key)
		}

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

// JSONMarshal is a generic function to marshal destination type into json data
// that supports union types and fallback to native json.Marshal when available
func JSONMarshal[A any](in A) ([]byte, error) {
	x := any(in)
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
		// Marshalling an interface value without a registered marshaller
		// would silently produce JSON without the $type discriminator,
		// and such data can never be unmarshalled back into the interface.
		// Fail loudly instead of corrupting data at rest.
		if isUnregisterableInterface[A]() {
			return nil, fmt.Errorf("shared.JSONMarshal: type %q is an interface with no registered marshaller; marshalling would drop the $type discriminator and the data could not be read back; import the package that registers it (generated init in *_union_gen.go) or check //go:tag mkunion:\",no-type-registry\"", key)
		}

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

// isUnregisterableInterface reports whether A is a non-empty interface type
// (like a union visitor interface). Such types cannot round-trip through
// plain encoding/json - they need a registered marshaller that writes the
// $type discriminator.
func isUnregisterableInterface[A any]() bool {
	t := reflect.TypeOf(new(A)).Elem()
	return t.Kind() == reflect.Interface && t.NumMethod() > 0
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
		json.RawMessage,
		[]json.RawMessage,
		*[]json.RawMessage,
		map[string]json.RawMessage,
		*map[string]json.RawMessage:
		return true
	}

	return false
}
