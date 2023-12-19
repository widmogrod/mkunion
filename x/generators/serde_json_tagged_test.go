package generators

import (
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/shape"
	"testing"
)

func TestNewSerdeJSONTagged_Struct(t *testing.T) {
	//t.Skip("not implemented")
	inferred, err := shape.InferFromFile("testutils/tree.go")
	if err != nil {
		t.Fatal(err)
	}

	generator := NewSerdeJSONTagged(
		inferred.RetrieveShapeNamedAs("ListOf2"),
	)

	result, err := generator.Generate()
	assert.NoError(t, err)
	assert.Equal(t, `package testutils

import (
	"encoding/json"
	"fmt"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/shared"
	"time"
)

var (
	_ json.Unmarshaler = (*ListOf2[any,any])(nil)
	_ json.Marshaler   = (*ListOf2[any,any])(nil)
)

func (r *ListOf2[T1,T2]) MarshalJSON() ([]byte, error) {
	var err error
	result := make(map[string]json.RawMessage)

	fieldID, err := shared.JSONMarshal[string](r.ID)
	if err != nil {
		return nil, fmt.Errorf("testutils.ListOf2[T1,T2].MarshalJSON: field ID; %w", err)
	}
	result["ID"] = fieldID

	fieldData, err := shared.JSONMarshal[T1](r.Data)
	if err != nil {
		return nil, fmt.Errorf("testutils.ListOf2[T1,T2].MarshalJSON: field Data; %w", err)
	}
	result["Data"] = fieldData

	fieldList := make([]json.RawMessage, len(r.List))
	for i, v := range r.List {
		fieldList[i], err = shared.JSONMarshal[T2](v)
		if err != nil {
			return nil, fmt.Errorf("testutils.ListOf2[T1,T2].MarshalJSON: field List[%d]; %w", i, err)
		}
	}
	result["List"], err = json.Marshal(fieldList)
	if err != nil {
		return nil, fmt.Errorf("testutils.ListOf2[T1,T2].MarshalJSON: field List; %w", err)
	}

	fieldMap := make(map[string]json.RawMessage)
	for k, v := range r.Map {
		var key any
		key, ok := any(k).(string)
		if !ok {
			key, err = shared.JSONMarshal[T1](k)
			if err != nil {
				return nil, fmt.Errorf("testutils.ListOf2[T1,T2].MarshalJSON: field Map[%#v] key decoding; %w", key, err)
			}
			key = string(key.([]byte))
		}

		fieldMap[key.(string)], err = shared.JSONMarshal[T2](v)
		if err != nil {
			return nil, fmt.Errorf("testutils.ListOf2[T1,T2].MarshalJSON: field Map[%#v] value decoding %#v; %w", key, v, err)
		}
	}
	result["map_of_tree"], err = json.Marshal(fieldMap)
	if err != nil {
		return nil, fmt.Errorf("testutils.ListOf2[T1,T2].MarshalJSON: field Map; %w", err)
	}

	fieldListOf, err := shared.JSONMarshal[ListOf[T1]](r.ListOf)
	if err != nil {
		return nil, fmt.Errorf("testutils.ListOf2[T1,T2].MarshalJSON: field ListOf; %w", err)
	}
	result["ListOf"] = fieldListOf

	fieldListOfPtr, err := shared.JSONMarshal[*ListOf[T2]](r.ListOfPtr)
	if err != nil {
		return nil, fmt.Errorf("testutils.ListOf2[T1,T2].MarshalJSON: field ListOfPtr; %w", err)
	}
	result["ListOfPtr"] = fieldListOfPtr

	fieldTime, err := shared.JSONMarshal[time.Time](r.Time)
	if err != nil {
		return nil, fmt.Errorf("testutils.ListOf2[T1,T2].MarshalJSON: field Time; %w", err)
	}
	result["Time"] = fieldTime

	fieldValue, err := shared.JSONMarshal[schema.Schema](r.Value)
	if err != nil {
		return nil, fmt.Errorf("testutils.ListOf2[T1,T2].MarshalJSON: field Value; %w", err)
	}
	result["Value"] = fieldValue

	output, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("testutils.ListOf2[T1,T2].MarshalJSON: final step; %w", err)
	}

	return output, nil
}

func (r *ListOf2[T1,T2]) UnmarshalJSON(bytes []byte) error {
	return shared.JSONParseObject(bytes, func(key string, bytes []byte) error {
		switch key {
		case "ID":
			var err error
			r.ID, err = shared.JSONUnmarshal[string](bytes)
			if err != nil {
				return fmt.Errorf("testutils.ListOf2[T1,T2].UnmarshalJSON: field ID; %w", err)
			}
			return nil

		case "Data":
			var err error
			r.Data, err = shared.JSONUnmarshal[T1](bytes)
			if err != nil {
				return fmt.Errorf("testutils.ListOf2[T1,T2].UnmarshalJSON: field Data; %w", err)
			}
			return nil

		case "List":
			err := shared.JSONParseList(bytes, func(index int, bytes []byte) error {
				item, err := shared.JSONUnmarshal[T2](bytes)
				if err != nil {
					return fmt.Errorf("testutils.ListOf2[T1,T2].UnmarshalJSON: field List[%d]; %w", index, err)
				}
				r.List = append(r.List, item)
				return nil
			})
			if err != nil {
				return fmt.Errorf("testutils.ListOf2[T1,T2].UnmarshalJSON: field List; %w", err)
			}
			return nil

		case "map_of_tree":
			r.Map = make(map[T1]T2)
			err := shared.JSONParseObject(bytes, func(rawKey string, bytes []byte) error {
				item, err := shared.JSONUnmarshal[T2](bytes)
				if err != nil {
					return fmt.Errorf("key=%s to type=%T item error;  %w", bytes, item, err)
				}

				var key2 T1
				if _, ok := any(key2).(string); !ok {
					var err error
					key2, err = shared.JSONUnmarshal[T1]([]byte(rawKey))
					if err != nil {
						return fmt.Errorf("key=%s to type=%T key error; %w", rawKey, key2, err)
					}
				} else {
					key2 = any(rawKey).(T1)
				}

				r.Map[key2] = item
				return nil
			})
			if err != nil {
				return fmt.Errorf("testutils.ListOf2[T1,T2].UnmarshalJSON: field Map; %w", err)
			}
			return nil

		case "ListOf":
			var err error
			r.ListOf, err = shared.JSONUnmarshal[ListOf[T1]](bytes)
			if err != nil {
				return fmt.Errorf("testutils.ListOf2[T1,T2].UnmarshalJSON: field ListOf; %w", err)
			}
			return nil

		case "ListOfPtr":
			var err error
			r.ListOfPtr, err = shared.JSONUnmarshal[*ListOf[T2]](bytes)
			if err != nil {
				return fmt.Errorf("testutils.ListOf2[T1,T2].UnmarshalJSON: field ListOfPtr; %w", err)
			}
			return nil

		case "Time":
			var err error
			r.Time, err = shared.JSONUnmarshal[time.Time](bytes)
			if err != nil {
				return fmt.Errorf("testutils.ListOf2[T1,T2].UnmarshalJSON: field Time; %w", err)
			}
			return nil

		case "Value":
			var err error
			r.Value, err = shared.JSONUnmarshal[schema.Schema](bytes)
			if err != nil {
				return fmt.Errorf("testutils.ListOf2[T1,T2].UnmarshalJSON: field Value; %w", err)
			}
			return nil

		}

		return nil
	})
}

`, result)
}
func TestNewSerdeJSONTagged_Alias(t *testing.T) {
	inferred, err := shape.InferFromFile("testutils/tree.go")
	if err != nil {
		t.Fatal(err)
	}

	generator := NewSerdeJSONTagged(
		inferred.RetrieveShapeNamedAs("K"),
	)

	result, err := generator.Generate()
	assert.NoError(t, err)
	assert.Equal(t, `package testutils

import (
	"encoding/json"
	"fmt"
	"github.com/widmogrod/mkunion/x/shared"
)

var (
	_ json.Unmarshaler = (*K)(nil)
	_ json.Marshaler   = (*K)(nil)
)

func (r *K) MarshalJSON() ([]byte, error) {
	result, err := shared.JSONMarshal[string](string(*r))
	if err != nil {
		return nil, fmt.Errorf("testutils.K.MarshalJSON: %w", err)
	}
	return result, nil
}

func (r *K) UnmarshalJSON(bytes []byte) error {
	result, err := shared.JSONUnmarshal[string](bytes)
	if err != nil {
		return fmt.Errorf("testutils.K.UnmarshalJSON: %w", err)
	}
	*r = K(result)
	return nil
}

`, result)
}
