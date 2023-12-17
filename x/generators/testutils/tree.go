package testutils

import (
	"encoding/json"
	"time"
)

//go:generate go run ../../../cmd/mkunion/main.go serde

//go:generate go run ../../../cmd/mkunion/main.go -name=Tree
type (
	Branch struct {
		Lit  Tree
		List []Tree
		Map  map[string]Tree
	}
	Leaf struct{ Value int64 }
	K    string
	P    ListOf2[ListOf[any], *ListOf2[int64, *time.Duration]]
)

//go:tag serde:"json"
type ListOf[T any] struct {
	Data T
}

//go:tag serde:"json"
type ListOf2[T1 comparable, T2 any] struct {
	ID        string
	Data      T1
	List      []T2
	Map       map[T1]T2
	ListOf    ListOf[T1]
	ListOfPtr *ListOf[T2]
}

var (
	_ json.Unmarshaler = (*ListOf2[any, any])(nil)
	_ json.Marshaler   = (*ListOf2[any, any])(nil)
)

//func (r *ListOf2[T1, T2]) MarshalJSON() ([]byte, error) {
//	result := make(map[string]json.RawMessage)
//
//	field_ID, err := shared.JSONMarshal[string](r.ID)
//	if err != nil {
//		return nil, fmt.Errorf("testutils.ListOf2[T1,T2].MarshalJSON: field ID; %w", err)
//	}
//	result["ID"] = field_ID
//
//	field_Data, err := shared.JSONMarshal[T1](r.Data)
//	if err != nil {
//		return nil, fmt.Errorf("testutils.ListOf2[T1,T2].MarshalJSON: field Data; %w", err)
//	}
//	result["Data"] = field_Data
//
//	field_List := make([]json.RawMessage, len(r.List))
//	for i, v := range r.List {
//		field_List[i], err = shared.JSONMarshal[T2](v)
//		if err != nil {
//			return nil, fmt.Errorf("testutils.ListOf2[T1,T2].MarshalJSON: field List[%d]; %w", i, err)
//		}
//	}
//	result["List"], err = json.Marshal(field_List)
//	if err != nil {
//		return nil, fmt.Errorf("testutils.ListOf2[T1,T2].MarshalJSON: field List; %w", err)
//	}
//
//	field_Map := make(map[string]json.RawMessage)
//	for k, v := range r.Map {
//		var key any
//		key, ok := any(k).(string)
//		if !ok {
//			key, err = shared.JSONMarshal[T1](k)
//			if err != nil {
//				return nil, fmt.Errorf("testutils.ListOf2[T1,T2].MarshalJSON: field Map[%#v] key decoding; %w", key, err)
//			}
//			key = string(key.([]byte))
//		}
//
//		field_Map[key.(string)], err = shared.JSONMarshal[T2](v)
//		if err != nil {
//			return nil, fmt.Errorf("testutils.ListOf2[T1,T2].MarshalJSON: field Map[%#v] value decoding %#v; %w", key, v, err)
//		}
//	}
//	result["Map"], err = json.Marshal(field_Map)
//	if err != nil {
//		return nil, fmt.Errorf("testutils.ListOf2[T1,T2].MarshalJSON: field Map; %w", err)
//	}
//
//	output, err := json.Marshal(result)
//	if err != nil {
//		return nil, fmt.Errorf("testutils.ListOf2[T1,T2].MarshalJSON: final step; %w", err)
//	}
//
//	return output, nil
//}
//
//func (r *ListOf2[T1, T2]) UnmarshalJSON(bytes []byte) error {
//	return shared.JSONParseObject(bytes, func(key string, bytes []byte) error {
//		switch key {
//		case "ID":
//			var err error
//			r.ID, err = shared.JSONUnmarshal[string](bytes)
//			if err != nil {
//				return fmt.Errorf("testutils.ListOf2[T1,T2].UnmarshalJSON: field ID; %w", err)
//			}
//			return nil
//		case "Data":
//			var err error
//			r.Data, err = shared.JSONUnmarshal[T1](bytes)
//			if err != nil {
//				return fmt.Errorf("testutils.ListOf2[T1,T2].UnmarshalJSON: field Data; %w", err)
//			}
//			return nil
//
//		case "List":
//			err := shared.JSONParseList(bytes, func(index int, bytes []byte) error {
//				item, err := shared.JSONUnmarshal[T2](bytes)
//				if err != nil {
//					return err
//				}
//				r.List = append(r.List, item)
//				return nil
//			})
//			if err != nil {
//				return fmt.Errorf("testutils.ListOf2[T1,T2].UnmarshalJSON: field List; %w", err)
//			}
//			return nil
//
//		case "Map":
//			r.Map = make(map[T1]T2)
//			err := shared.JSONParseObject(bytes, func(rawKey string, bytes []byte) error {
//				item, err := shared.JSONUnmarshal[T2](bytes)
//				if err != nil {
//					return fmt.Errorf("key=%s to type=%T item error;  %w", bytes, item, err)
//				}
//
//				var key2 T1
//				if _, ok := any(key2).(string); !ok {
//					var err error
//					key2, err = shared.JSONUnmarshal[T1]([]byte(rawKey))
//					if err != nil {
//						return fmt.Errorf("key=%s to type=%T key error; %w", rawKey, key2, err)
//					}
//				} else {
//					key2 = any(rawKey).(T1)
//				}
//
//				r.Map[key2] = item
//				return nil
//			})
//			if err != nil {
//				return fmt.Errorf("testutils.ListOf2[T1,T2].UnmarshalJSON: field Map; %w", err)
//			}
//			return nil
//		}
//
//		return fmt.Errorf("testutils.ListOf2[T1,T2].UnmarshalJSON: unknown key: %s", key)
//	})
//}
