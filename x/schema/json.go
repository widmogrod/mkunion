package schema

import (
	"encoding/json"
)

func FromJSON(data []byte) (Schema, error) {
	var x any
	if err := json.Unmarshal(data, &x); err != nil {
		return nil, err
	}

	return FromGo(x), nil
}

func ToJSON(schema Schema, rules ...RuleMatcher) ([]byte, error) {
	return json.Marshal(ToGo(schema, rules...))
}
