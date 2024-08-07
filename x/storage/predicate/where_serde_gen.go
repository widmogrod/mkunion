// Code generated by mkunion. DO NOT EDIT.
package predicate

import (
	"encoding/json"
	"fmt"
	"github.com/widmogrod/mkunion/x/shape"
	"github.com/widmogrod/mkunion/x/shared"
)

var (
	_ json.Unmarshaler = (*WherePredicates)(nil)
	_ json.Marshaler   = (*WherePredicates)(nil)
)

func (r *WherePredicates) MarshalJSON() ([]byte, error) {
	if r == nil {
		return nil, nil
	}
	return r._marshalJSONWherePredicates(*r)
}
func (r *WherePredicates) _marshalJSONWherePredicates(x WherePredicates) ([]byte, error) {
	partial := make(map[string]json.RawMessage)
	var err error
	var fieldPredicate []byte
	fieldPredicate, err = r._marshalJSONPredicate(x.Predicate)
	if err != nil {
		return nil, fmt.Errorf("predicate: WherePredicates._marshalJSONWherePredicates: field name Predicate; %w", err)
	}
	partial["Predicate"] = fieldPredicate
	var fieldParams []byte
	fieldParams, err = r._marshalJSONParamBinds(x.Params)
	if err != nil {
		return nil, fmt.Errorf("predicate: WherePredicates._marshalJSONWherePredicates: field name Params; %w", err)
	}
	partial["Params"] = fieldParams
	var fieldShape []byte
	fieldShape, err = r._marshalJSONshape_Shape(x.Shape)
	if err != nil {
		return nil, fmt.Errorf("predicate: WherePredicates._marshalJSONWherePredicates: field name Shape; %w", err)
	}
	partial["Shape"] = fieldShape
	result, err := json.Marshal(partial)
	if err != nil {
		return nil, fmt.Errorf("predicate: WherePredicates._marshalJSONWherePredicates: struct; %w", err)
	}
	return result, nil
}
func (r *WherePredicates) _marshalJSONPredicate(x Predicate) ([]byte, error) {
	result, err := shared.JSONMarshal[Predicate](x)
	if err != nil {
		return nil, fmt.Errorf("predicate: WherePredicates._marshalJSONPredicate:; %w", err)
	}
	return result, nil
}
func (r *WherePredicates) _marshalJSONParamBinds(x ParamBinds) ([]byte, error) {
	result, err := shared.JSONMarshal[ParamBinds](x)
	if err != nil {
		return nil, fmt.Errorf("predicate: WherePredicates._marshalJSONParamBinds:; %w", err)
	}
	return result, nil
}
func (r *WherePredicates) _marshalJSONshape_Shape(x shape.Shape) ([]byte, error) {
	result, err := shared.JSONMarshal[shape.Shape](x)
	if err != nil {
		return nil, fmt.Errorf("predicate: WherePredicates._marshalJSONshape_Shape:; %w", err)
	}
	return result, nil
}
func (r *WherePredicates) UnmarshalJSON(data []byte) error {
	result, err := r._unmarshalJSONWherePredicates(data)
	if err != nil {
		return fmt.Errorf("predicate: WherePredicates.UnmarshalJSON: %w", err)
	}
	*r = result
	return nil
}
func (r *WherePredicates) _unmarshalJSONWherePredicates(data []byte) (WherePredicates, error) {
	result := WherePredicates{}
	var partial map[string]json.RawMessage
	err := json.Unmarshal(data, &partial)
	if err != nil {
		return result, fmt.Errorf("predicate: WherePredicates._unmarshalJSONWherePredicates: native struct unwrap; %w", err)
	}
	if fieldPredicate, ok := partial["Predicate"]; ok {
		result.Predicate, err = r._unmarshalJSONPredicate(fieldPredicate)
		if err != nil {
			return result, fmt.Errorf("predicate: WherePredicates._unmarshalJSONWherePredicates: field Predicate; %w", err)
		}
	}
	if fieldParams, ok := partial["Params"]; ok {
		result.Params, err = r._unmarshalJSONParamBinds(fieldParams)
		if err != nil {
			return result, fmt.Errorf("predicate: WherePredicates._unmarshalJSONWherePredicates: field Params; %w", err)
		}
	}
	if fieldShape, ok := partial["Shape"]; ok {
		result.Shape, err = r._unmarshalJSONshape_Shape(fieldShape)
		if err != nil {
			return result, fmt.Errorf("predicate: WherePredicates._unmarshalJSONWherePredicates: field Shape; %w", err)
		}
	}
	return result, nil
}
func (r *WherePredicates) _unmarshalJSONPredicate(data []byte) (Predicate, error) {
	result, err := shared.JSONUnmarshal[Predicate](data)
	if err != nil {
		return result, fmt.Errorf("predicate: WherePredicates._unmarshalJSONPredicate: native ref unwrap; %w", err)
	}
	return result, nil
}
func (r *WherePredicates) _unmarshalJSONParamBinds(data []byte) (ParamBinds, error) {
	result, err := shared.JSONUnmarshal[ParamBinds](data)
	if err != nil {
		return result, fmt.Errorf("predicate: WherePredicates._unmarshalJSONParamBinds: native ref unwrap; %w", err)
	}
	return result, nil
}
func (r *WherePredicates) _unmarshalJSONshape_Shape(data []byte) (shape.Shape, error) {
	result, err := shared.JSONUnmarshal[shape.Shape](data)
	if err != nil {
		return result, fmt.Errorf("predicate: WherePredicates._unmarshalJSONshape_Shape: native ref unwrap; %w", err)
	}
	return result, nil
}
