package predicate

import (
	"encoding/json"
	"github.com/widmogrod/mkunion/x/schema"
)

//go:generate go run ../../../cmd/mkunion/main.go -name=Predicate
type (
	And struct {
		L []Predicate
	}
	Or struct {
		L []Predicate
	}
	Not struct {
		P Predicate
	}
	Compare struct {
		Location  string
		Operation string
		BindValue Bindable
	}
)

//go:generate go run ../../../cmd/mkunion/main.go -name=Bindable
type (
	BindValue struct{ BindName BindName }
	Literal   struct{ Value schema.Schema }
	Locatable struct{ Location string }
)

type BindName = string
type ParamBinds map[BindName]schema.Schema

var (
	_ json.Unmarshaler = (*ParamBinds)(nil)
	_ json.Marshaler   = (*ParamBinds)(nil)
)

func (p *ParamBinds) UnmarshalJSON(bytes []byte) error {
	var data map[string]json.RawMessage
	if err := json.Unmarshal(bytes, &data); err != nil {
		return err
	}

	result := ParamBinds{}
	for k, v := range data {
		value, err := schema.SchemaFromJSON(v)
		if err != nil {
			return err
		}

		result[k] = value
	}

	*p = result
	return nil
}

func (p *ParamBinds) MarshalJSON() ([]byte, error) {
	result := map[string]json.RawMessage{}
	for k, v := range *p {
		data, err := schema.SchemaToJSON(v)
		if err != nil {
			return nil, err
		}
		result[k] = data
	}

	return json.Marshal(result)
}
