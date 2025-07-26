package shared

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
)

var (
	registerJSONMarshaller = sync.Map{}
	registerType           = sync.Map{}
	packageTags            = sync.Map{}
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

// PackageTagsStore stores package-level tags for runtime access with package namespace.
// This function is typically called from generated code during package initialization.
// Tags are stored with package-namespaced keys to prevent conflicts between packages.
func PackageTagsStore(pkgImportName string, tags map[string]interface{}) {
	for key, value := range tags {
		namespacedKey := pkgImportName + "." + key
		packageTags.Store(namespacedKey, value)
	}
}

// PackageTagsLoad retrieves package-level tags that were embedded at compile time.
// Returns a map of all stored package tags.
func PackageTagsLoad() map[string]interface{} {
	result := make(map[string]interface{})
	packageTags.Range(func(key, value interface{}) bool {
		if keyStr, ok := key.(string); ok {
			result[keyStr] = value
		}
		return true
	})
	return result
}

// PackageTagsLoadForPackage retrieves package-level tags for a specific package.
// Returns only the tags for the given package import name.
func PackageTagsLoadForPackage(pkgImportName string) map[string]interface{} {
	result := make(map[string]interface{})
	prefix := pkgImportName + "."
	packageTags.Range(func(key, value interface{}) bool {
		if keyStr, ok := key.(string); ok {
			if strings.HasPrefix(keyStr, prefix) {
				// Remove the package prefix to get the original tag name
				tagName := strings.TrimPrefix(keyStr, prefix)
				result[tagName] = value
			}
		}
		return true
	})
	return result
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
		json.RawMessage,
		[]json.RawMessage,
		*[]json.RawMessage,
		map[string]json.RawMessage,
		*map[string]json.RawMessage:
		return true
	}

	return false
}
