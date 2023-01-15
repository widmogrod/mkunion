package schema

import (
	"encoding/json"
)

func JsonToSchema(data []byte) (Schema, error) {
	var x any
	if err := json.Unmarshal(data, &x); err != nil {
		return nil, err
	}

	return GoToSchema(x), nil
}

func SchemaToJson(schema Schema, rules ...RuleMatcher) ([]byte, error) {
	return json.Marshal(SchemaToGo(schema, rules...))
}
