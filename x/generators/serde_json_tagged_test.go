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
	if r == nil {
		return nil, nil
	}
	return r._marshalJSONListOf2Lb_T1CommaT2_bL(*r)
}
func (r *ListOf2[T1,T2]) _marshalJSONListOf2Lb_T1CommaT2_bL(x ListOf2[T1,T2]) ([]byte, error) {
	partial := make(map[string]json.RawMessage)
	var err error
	var fieldID []byte
	fieldID, err = r._marshalJSONstring(x.ID)
	if err != nil {
		return nil, fmt.Errorf("testutils: ListOf2[T1,T2]._marshalJSONListOf2Lb_T1CommaT2_bL: field name ID; %w", err)
	}
	partial["ID"] = fieldID
	var fieldData []byte
	fieldData, err = r._marshalJSONT1(x.Data)
	if err != nil {
		return nil, fmt.Errorf("testutils: ListOf2[T1,T2]._marshalJSONListOf2Lb_T1CommaT2_bL: field name Data; %w", err)
	}
	partial["Data"] = fieldData
	var fieldList []byte
	fieldList, err = r._marshalJSONSliceT2(x.List)
	if err != nil {
		return nil, fmt.Errorf("testutils: ListOf2[T1,T2]._marshalJSONListOf2Lb_T1CommaT2_bL: field name List; %w", err)
	}
	partial["List"] = fieldList
	var fieldMap []byte
	fieldMap, err = r._marshalJSONmapLb_T1_bLT2(x.Map)
	if err != nil {
		return nil, fmt.Errorf("testutils: ListOf2[T1,T2]._marshalJSONListOf2Lb_T1CommaT2_bL: field name Map; %w", err)
	}
	partial["map_of_tree"] = fieldMap
	var fieldListOf []byte
	fieldListOf, err = r._marshalJSONListOfLb_T1_bL(x.ListOf)
	if err != nil {
		return nil, fmt.Errorf("testutils: ListOf2[T1,T2]._marshalJSONListOf2Lb_T1CommaT2_bL: field name ListOf; %w", err)
	}
	partial["ListOf"] = fieldListOf
	if x.ListOfPtr != nil {
		var fieldListOfPtr []byte
		fieldListOfPtr, err = r._marshalJSONPtrListOfLb_T2_bL(x.ListOfPtr)
		if err != nil {
			return nil, fmt.Errorf("testutils: ListOf2[T1,T2]._marshalJSONListOf2Lb_T1CommaT2_bL: field name ListOfPtr; %w", err)
		}
		partial["ListOfPtr"] = fieldListOfPtr
	}
	var fieldTime []byte
	fieldTime, err = r._marshalJSONtime_Time(x.Time)
	if err != nil {
		return nil, fmt.Errorf("testutils: ListOf2[T1,T2]._marshalJSONListOf2Lb_T1CommaT2_bL: field name Time; %w", err)
	}
	partial["Time"] = fieldTime
	var fieldValue []byte
	fieldValue, err = r._marshalJSONschema_Schema(x.Value)
	if err != nil {
		return nil, fmt.Errorf("testutils: ListOf2[T1,T2]._marshalJSONListOf2Lb_T1CommaT2_bL: field name Value; %w", err)
	}
	partial["Value"] = fieldValue
	result, err := json.Marshal(partial)
	if err != nil {
		return nil, fmt.Errorf("testutils: ListOf2[T1,T2]._marshalJSONListOf2Lb_T1CommaT2_bL: struct; %w", err)
	}
	return result, nil
}
func (r *ListOf2[T1,T2]) _marshalJSONstring(x string) ([]byte, error) {
	result, err := json.Marshal(x)
	if err != nil {
		return nil, fmt.Errorf("testutils: ListOf2[T1,T2]._marshalJSONstring:; %w", err)
	}
	return result, nil
}
func (r *ListOf2[T1,T2]) _marshalJSONT1(x T1) ([]byte, error) {
	result, err := shared.JSONMarshal[T1](x)
	if err != nil {
		return nil, fmt.Errorf("testutils: ListOf2[T1,T2]._marshalJSONT1:; %w", err)
	}
	return result, nil
}
func (r *ListOf2[T1,T2]) _marshalJSONSliceT2(x []T2) ([]byte, error) {
	partial := make([]json.RawMessage, len(x))
	for i, v := range x {
		item, err := r._marshalJSONT2(v)
		if err != nil {
			return nil, fmt.Errorf("testutils: ListOf2[T1,T2]._marshalJSONSliceT2: at index %d; %w", i, err)
		}
		partial[i] = item
	}
	result, err := json.Marshal(partial)
	if err != nil {
		return nil, fmt.Errorf("testutils: ListOf2[T1,T2]._marshalJSONSliceT2:; %w", err)
	}
	return result, nil
}
func (r *ListOf2[T1,T2]) _marshalJSONT2(x T2) ([]byte, error) {
	result, err := shared.JSONMarshal[T2](x)
	if err != nil {
		return nil, fmt.Errorf("testutils: ListOf2[T1,T2]._marshalJSONT2:; %w", err)
	}
	return result, nil
}
func (r *ListOf2[T1,T2]) _marshalJSONmapLb_T1_bLT2(x map[T1]T2) ([]byte, error) {
	partial := make(map[string]json.RawMessage)
	var err error
	var keyType T1
	_, isString := any(keyType).(string)
	for k, v := range x {
		var key []byte
		if isString {
			key = []byte(any(k).(string))
		} else {
			key, err = r._marshalJSONT1(k)
			if err != nil {
				return nil, fmt.Errorf("testutils: ListOf2[T1,T2]._marshalJSONmapLb_T1_bLT2: key; %w", err)
			}
		}
		value, err := r._marshalJSONT2(v)
		if err != nil {
			return nil, fmt.Errorf("testutils: ListOf2[T1,T2]._marshalJSONmapLb_T1_bLT2: value; %w", err)
		}
		partial[string(key)] = value
	}
	result, err := json.Marshal(partial)
	if err != nil {
		return nil, fmt.Errorf("testutils: ListOf2[T1,T2]._marshalJSONmapLb_T1_bLT2:; %w", err)
	}
	return result, nil
}
func (r *ListOf2[T1,T2]) _marshalJSONListOfLb_T1_bL(x ListOf[T1]) ([]byte, error) {
	result, err := shared.JSONMarshal[ListOf[T1]](x)
	if err != nil {
		return nil, fmt.Errorf("testutils: ListOf2[T1,T2]._marshalJSONListOfLb_T1_bL:; %w", err)
	}
	return result, nil
}
func (r *ListOf2[T1,T2]) _marshalJSONPtrListOfLb_T2_bL(x *ListOf[T2]) ([]byte, error) {
	result, err := shared.JSONMarshal[*ListOf[T2]](x)
	if err != nil {
		return nil, fmt.Errorf("testutils: ListOf2[T1,T2]._marshalJSONPtrListOfLb_T2_bL:; %w", err)
	}
	return result, nil
}
func (r *ListOf2[T1,T2]) _marshalJSONtime_Time(x time.Time) ([]byte, error) {
	result, err := shared.JSONMarshal[time.Time](x)
	if err != nil {
		return nil, fmt.Errorf("testutils: ListOf2[T1,T2]._marshalJSONtime_Time:; %w", err)
	}
	return result, nil
}
func (r *ListOf2[T1,T2]) _marshalJSONschema_Schema(x schema.Schema) ([]byte, error) {
	result, err := shared.JSONMarshal[schema.Schema](x)
	if err != nil {
		return nil, fmt.Errorf("testutils: ListOf2[T1,T2]._marshalJSONschema_Schema:; %w", err)
	}
	return result, nil
}
func (r *ListOf2[T1,T2]) UnmarshalJSON(data []byte) error {
	result, err := r._unmarshalJSONListOf2Lb_T1CommaT2_bL(data)
	if err != nil {
		return fmt.Errorf("testutils: ListOf2[T1,T2].UnmarshalJSON: %w", err)
	}
	*r = result
	return nil
}
func (r *ListOf2[T1,T2]) _unmarshalJSONListOf2Lb_T1CommaT2_bL(data []byte) (ListOf2[T1,T2], error) {
	result := ListOf2[T1,T2]{}
	var partial map[string]json.RawMessage
	err := json.Unmarshal(data, &partial)
	if err != nil {
		return result, fmt.Errorf("testutils: ListOf2[T1,T2]._unmarshalJSONListOf2Lb_T1CommaT2_bL: native struct unwrap; %w", err)
	}
	if fieldID, ok := partial["ID"]; ok {
		result.ID, err = r._unmarshalJSONstring(fieldID)
		if err != nil {
			return result, fmt.Errorf("testutils: ListOf2[T1,T2]._unmarshalJSONListOf2Lb_T1CommaT2_bL: field ID; %w", err)
		}
	}
	if fieldData, ok := partial["Data"]; ok {
		result.Data, err = r._unmarshalJSONT1(fieldData)
		if err != nil {
			return result, fmt.Errorf("testutils: ListOf2[T1,T2]._unmarshalJSONListOf2Lb_T1CommaT2_bL: field Data; %w", err)
		}
	}
	if fieldList, ok := partial["List"]; ok {
		result.List, err = r._unmarshalJSONSliceT2(fieldList)
		if err != nil {
			return result, fmt.Errorf("testutils: ListOf2[T1,T2]._unmarshalJSONListOf2Lb_T1CommaT2_bL: field List; %w", err)
		}
	}
	if fieldMap, ok := partial["map_of_tree"]; ok {
		result.Map, err = r._unmarshalJSONmapLb_T1_bLT2(fieldMap)
		if err != nil {
			return result, fmt.Errorf("testutils: ListOf2[T1,T2]._unmarshalJSONListOf2Lb_T1CommaT2_bL: field Map; %w", err)
		}
	}
	if fieldListOf, ok := partial["ListOf"]; ok {
		result.ListOf, err = r._unmarshalJSONListOfLb_T1_bL(fieldListOf)
		if err != nil {
			return result, fmt.Errorf("testutils: ListOf2[T1,T2]._unmarshalJSONListOf2Lb_T1CommaT2_bL: field ListOf; %w", err)
		}
	}
	if fieldListOfPtr, ok := partial["ListOfPtr"]; ok {
		result.ListOfPtr, err = r._unmarshalJSONPtrListOfLb_T2_bL(fieldListOfPtr)
		if err != nil {
			return result, fmt.Errorf("testutils: ListOf2[T1,T2]._unmarshalJSONListOf2Lb_T1CommaT2_bL: field ListOfPtr; %w", err)
		}
	}
	if fieldTime, ok := partial["Time"]; ok {
		result.Time, err = r._unmarshalJSONtime_Time(fieldTime)
		if err != nil {
			return result, fmt.Errorf("testutils: ListOf2[T1,T2]._unmarshalJSONListOf2Lb_T1CommaT2_bL: field Time; %w", err)
		}
	}
	if fieldValue, ok := partial["Value"]; ok {
		result.Value, err = r._unmarshalJSONschema_Schema(fieldValue)
		if err != nil {
			return result, fmt.Errorf("testutils: ListOf2[T1,T2]._unmarshalJSONListOf2Lb_T1CommaT2_bL: field Value; %w", err)
		}
	}
	return result, nil
}
func (r *ListOf2[T1,T2]) _unmarshalJSONstring(data []byte) (string, error) {
	var result string
	err := json.Unmarshal(data, &result)
	if err != nil {
		return result, fmt.Errorf("testutils: ListOf2[T1,T2]._unmarshalJSONstring: native primitive unwrap; %w", err)
	}
	return result, nil
}
func (r *ListOf2[T1,T2]) _unmarshalJSONT1(data []byte) (T1, error) {
	result, err := shared.JSONUnmarshal[T1](data)
	if err != nil {
		return result, fmt.Errorf("testutils: ListOf2[T1,T2]._unmarshalJSONT1: native ref unwrap; %w", err)
	}
	return result, nil
}
func (r *ListOf2[T1,T2]) _unmarshalJSONSliceT2(data []byte) ([]T2, error) {
	result := make([]T2, 0)
	var partial []json.RawMessage
	err := json.Unmarshal(data, &partial)
	if err != nil {
		return result, fmt.Errorf("testutils: ListOf2[T1,T2]._unmarshalJSONSliceT2: native list unwrap; %w", err)
	}
	for i, v := range partial {
		item, err := r._unmarshalJSONT2(v)
		if err != nil {
			return result, fmt.Errorf("testutils: ListOf2[T1,T2]._unmarshalJSONSliceT2: at index %d; %w", i, err)
		}
		result = append(result, item)
	}
	return result, nil
}
func (r *ListOf2[T1,T2]) _unmarshalJSONT2(data []byte) (T2, error) {
	result, err := shared.JSONUnmarshal[T2](data)
	if err != nil {
		return result, fmt.Errorf("testutils: ListOf2[T1,T2]._unmarshalJSONT2: native ref unwrap; %w", err)
	}
	return result, nil
}
func (r *ListOf2[T1,T2]) _unmarshalJSONmapLb_T1_bLT2(data []byte) (map[T1]T2, error) {
	var partial map[string]json.RawMessage
	err := json.Unmarshal(data, &partial)
	if err != nil {
		return nil, fmt.Errorf("testutils: ListOf2[T1,T2]._unmarshalJSONmapLb_T1_bLT2: native map unwrap; %w", err)
	}
	result := make(map[T1]T2)
	var keyType T1
	_, isString := any(keyType).(string)
	for k, v := range partial {
		var key T1
		if isString {
			key = any(k).(T1)
		} else {
			key, err = r._unmarshalJSONT1([]byte(k))
			if err != nil {
				return nil, fmt.Errorf("testutils: ListOf2[T1,T2]._unmarshalJSONmapLb_T1_bLT2: key; %w", err)
			}
		}
		value, err := r._unmarshalJSONT2(v)
		if err != nil {
			return nil, fmt.Errorf("testutils: ListOf2[T1,T2]._unmarshalJSONmapLb_T1_bLT2: value; %w", err)
		}
		result[key] = value
	}
	return result, nil
}
func (r *ListOf2[T1,T2]) _unmarshalJSONListOfLb_T1_bL(data []byte) (ListOf[T1], error) {
	result, err := shared.JSONUnmarshal[ListOf[T1]](data)
	if err != nil {
		return result, fmt.Errorf("testutils: ListOf2[T1,T2]._unmarshalJSONListOfLb_T1_bL: native ref unwrap; %w", err)
	}
	return result, nil
}
func (r *ListOf2[T1,T2]) _unmarshalJSONPtrListOfLb_T2_bL(data []byte) (*ListOf[T2], error) {
	result, err := shared.JSONUnmarshal[*ListOf[T2]](data)
	if err != nil {
		return result, fmt.Errorf("testutils: ListOf2[T1,T2]._unmarshalJSONPtrListOfLb_T2_bL: native ref unwrap; %w", err)
	}
	return result, nil
}
func (r *ListOf2[T1,T2]) _unmarshalJSONtime_Time(data []byte) (time.Time, error) {
	result, err := shared.JSONUnmarshal[time.Time](data)
	if err != nil {
		return result, fmt.Errorf("testutils: ListOf2[T1,T2]._unmarshalJSONtime_Time: native ref unwrap; %w", err)
	}
	return result, nil
}
func (r *ListOf2[T1,T2]) _unmarshalJSONschema_Schema(data []byte) (schema.Schema, error) {
	result, err := shared.JSONUnmarshal[schema.Schema](data)
	if err != nil {
		return result, fmt.Errorf("testutils: ListOf2[T1,T2]._unmarshalJSONschema_Schema: native ref unwrap; %w", err)
	}
	return result, nil
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
)

var (
	_ json.Unmarshaler = (*K)(nil)
	_ json.Marshaler   = (*K)(nil)
)

func (r *K) MarshalJSON() ([]byte, error) {
	if r == nil {
		return nil, nil
	}
	return r._marshalJSONK(*r)
}
func (r *K) _marshalJSONK(x K) ([]byte, error) {
	return r._marshalJSONstring(string(x))
}
func (r *K) _marshalJSONstring(x string) ([]byte, error) {
	result, err := json.Marshal(x)
	if err != nil {
		return nil, fmt.Errorf("testutils: K._marshalJSONstring:; %w", err)
	}
	return result, nil
}
func (r *K) UnmarshalJSON(data []byte) error {
	result, err := r._unmarshalJSONK(data)
	if err != nil {
		return fmt.Errorf("testutils: K.UnmarshalJSON: %w", err)
	}
	*r = result
	return nil
}
func (r *K) _unmarshalJSONK(data []byte) (K, error) {
	var result K
	intermidiary, err := r._unmarshalJSONstring(data)
	if err != nil {
		return result, fmt.Errorf("testutils: K._unmarshalJSONK: alias; %w", err)
	}
	result = K(intermidiary)
	return result, nil
}
func (r *K) _unmarshalJSONstring(data []byte) (string, error) {
	var result string
	err := json.Unmarshal(data, &result)
	if err != nil {
		return result, fmt.Errorf("testutils: K._unmarshalJSONstring: native primitive unwrap; %w", err)
	}
	return result, nil
}
`, result)
}

func TestNewSerdeJSONTagged_Ka_Serde(t *testing.T) {
	inferred, err := shape.InferFromFile("testutils/tree.go")
	if err != nil {
		t.Fatal(err)
	}

	generator := NewSerdeJSONTagged(
		inferred.RetrieveShapeNamedAs("Ka"),
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
	_ json.Unmarshaler = (*Ka)(nil)
	_ json.Marshaler   = (*Ka)(nil)
)

func (r *Ka) MarshalJSON() ([]byte, error) {
	if r == nil {
		return nil, nil
	}
	return r._marshalJSONKa(*r)
}
func (r *Ka) _marshalJSONKa(x Ka) ([]byte, error) {
	return r._marshalJSONSlicemapLb_string_bLTree([]map[string]Tree(x))
}
func (r *Ka) _marshalJSONSlicemapLb_string_bLTree(x []map[string]Tree) ([]byte, error) {
	partial := make([]json.RawMessage, len(x))
	for i, v := range x {
		item, err := r._marshalJSONmapLb_string_bLTree(v)
		if err != nil {
			return nil, fmt.Errorf("testutils: Ka._marshalJSONSlicemapLb_string_bLTree: at index %d; %w", i, err)
		}
		partial[i] = item
	}
	result, err := json.Marshal(partial)
	if err != nil {
		return nil, fmt.Errorf("testutils: Ka._marshalJSONSlicemapLb_string_bLTree:; %w", err)
	}
	return result, nil
}
func (r *Ka) _marshalJSONmapLb_string_bLTree(x map[string]Tree) ([]byte, error) {
	partial := make(map[string]json.RawMessage)
	for k, v := range x {
		key := string(k)
		value, err := r._marshalJSONTree(v)
		if err != nil {
			return nil, fmt.Errorf("testutils: Ka._marshalJSONmapLb_string_bLTree: value; %w", err)
		}
		partial[string(key)] = value
	}
	result, err := json.Marshal(partial)
	if err != nil {
		return nil, fmt.Errorf("testutils: Ka._marshalJSONmapLb_string_bLTree:; %w", err)
	}
	return result, nil
}
func (r *Ka) _marshalJSONstring(x string) ([]byte, error) {
	result, err := json.Marshal(x)
	if err != nil {
		return nil, fmt.Errorf("testutils: Ka._marshalJSONstring:; %w", err)
	}
	return result, nil
}
func (r *Ka) _marshalJSONTree(x Tree) ([]byte, error) {
	result, err := shared.JSONMarshal[Tree](x)
	if err != nil {
		return nil, fmt.Errorf("testutils: Ka._marshalJSONTree:; %w", err)
	}
	return result, nil
}
func (r *Ka) UnmarshalJSON(data []byte) error {
	result, err := r._unmarshalJSONKa(data)
	if err != nil {
		return fmt.Errorf("testutils: Ka.UnmarshalJSON: %w", err)
	}
	*r = result
	return nil
}
func (r *Ka) _unmarshalJSONKa(data []byte) (Ka, error) {
	var result Ka
	intermidiary, err := r._unmarshalJSONSlicemapLb_string_bLTree(data)
	if err != nil {
		return result, fmt.Errorf("testutils: Ka._unmarshalJSONKa: alias; %w", err)
	}
	result = Ka(intermidiary)
	return result, nil
}
func (r *Ka) _unmarshalJSONSlicemapLb_string_bLTree(data []byte) ([]map[string]Tree, error) {
	result := make([]map[string]Tree, 0)
	var partial []json.RawMessage
	err := json.Unmarshal(data, &partial)
	if err != nil {
		return result, fmt.Errorf("testutils: Ka._unmarshalJSONSlicemapLb_string_bLTree: native list unwrap; %w", err)
	}
	for i, v := range partial {
		item, err := r._unmarshalJSONmapLb_string_bLTree(v)
		if err != nil {
			return result, fmt.Errorf("testutils: Ka._unmarshalJSONSlicemapLb_string_bLTree: at index %d; %w", i, err)
		}
		result = append(result, item)
	}
	return result, nil
}
func (r *Ka) _unmarshalJSONmapLb_string_bLTree(data []byte) (map[string]Tree, error) {
	var partial map[string]json.RawMessage
	err := json.Unmarshal(data, &partial)
	if err != nil {
		return nil, fmt.Errorf("testutils: Ka._unmarshalJSONmapLb_string_bLTree: native map unwrap; %w", err)
	}
	result := make(map[string]Tree)
	for k, v := range partial {
		key := string(k)
		value, err := r._unmarshalJSONTree(v)
		if err != nil {
			return nil, fmt.Errorf("testutils: Ka._unmarshalJSONmapLb_string_bLTree: value; %w", err)
		}
		result[key] = value
	}
	return result, nil
}
func (r *Ka) _unmarshalJSONstring(data []byte) (string, error) {
	var result string
	err := json.Unmarshal(data, &result)
	if err != nil {
		return result, fmt.Errorf("testutils: Ka._unmarshalJSONstring: native primitive unwrap; %w", err)
	}
	return result, nil
}
func (r *Ka) _unmarshalJSONTree(data []byte) (Tree, error) {
	result, err := shared.JSONUnmarshal[Tree](data)
	if err != nil {
		return result, fmt.Errorf("testutils: Ka._unmarshalJSONTree: native ref unwrap; %w", err)
	}
	return result, nil
}
`, result)
}

func TestNewSerdeJSONTagged_P_Serde(t *testing.T) {
	inferred, err := shape.InferFromFile("testutils/tree.go")
	if err != nil {
		t.Fatal(err)
	}

	generator := NewSerdeJSONTagged(
		inferred.RetrieveShapeNamedAs("P"),
	)

	result, err := generator.Generate()
	assert.NoError(t, err)
	assert.Equal(t, `package testutils

import (
	"encoding/json"
	"fmt"
	"github.com/widmogrod/mkunion/x/shared"
	"time"
)

var (
	_ json.Unmarshaler = (*P)(nil)
	_ json.Marshaler   = (*P)(nil)
)

func (r *P) MarshalJSON() ([]byte, error) {
	if r == nil {
		return nil, nil
	}
	return r._marshalJSONP(*r)
}
func (r *P) _marshalJSONP(x P) ([]byte, error) {
	return r._marshalJSONListOf2Lb_ListOfLb_any_bLCommaPtrListOf2Lb_int64CommaPtrtime_Duration_bL_bL(ListOf2[ListOf[any],*ListOf2[int64,*time.Duration]](x))
}
func (r *P) _marshalJSONListOf2Lb_ListOfLb_any_bLCommaPtrListOf2Lb_int64CommaPtrtime_Duration_bL_bL(x ListOf2[ListOf[any],*ListOf2[int64,*time.Duration]]) ([]byte, error) {
	result, err := shared.JSONMarshal[ListOf2[ListOf[any],*ListOf2[int64,*time.Duration]]](x)
	if err != nil {
		return nil, fmt.Errorf("testutils: P._marshalJSONListOf2Lb_ListOfLb_any_bLCommaPtrListOf2Lb_int64CommaPtrtime_Duration_bL_bL:; %w", err)
	}
	return result, nil
}
func (r *P) UnmarshalJSON(data []byte) error {
	result, err := r._unmarshalJSONP(data)
	if err != nil {
		return fmt.Errorf("testutils: P.UnmarshalJSON: %w", err)
	}
	*r = result
	return nil
}
func (r *P) _unmarshalJSONP(data []byte) (P, error) {
	var result P
	intermidiary, err := r._unmarshalJSONListOf2Lb_ListOfLb_any_bLCommaPtrListOf2Lb_int64CommaPtrtime_Duration_bL_bL(data)
	if err != nil {
		return result, fmt.Errorf("testutils: P._unmarshalJSONP: alias; %w", err)
	}
	result = P(intermidiary)
	return result, nil
}
func (r *P) _unmarshalJSONListOf2Lb_ListOfLb_any_bLCommaPtrListOf2Lb_int64CommaPtrtime_Duration_bL_bL(data []byte) (ListOf2[ListOf[any],*ListOf2[int64,*time.Duration]], error) {
	result, err := shared.JSONUnmarshal[ListOf2[ListOf[any],*ListOf2[int64,*time.Duration]]](data)
	if err != nil {
		return result, fmt.Errorf("testutils: P._unmarshalJSONListOf2Lb_ListOfLb_any_bLCommaPtrListOf2Lb_int64CommaPtrtime_Duration_bL_bL: native ref unwrap; %w", err)
	}
	return result, nil
}
`, result)
}
