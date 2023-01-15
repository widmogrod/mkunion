package schema

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestJsonSchema(t *testing.T) {
	useCases := map[string]struct {
		in []byte
	}{
		"simple": {
			in: []byte(`{"Foo": 1, "Bar": 2}`),
		},
		"nested": {
			in: []byte(`{"Foo": 1, "Bar": {"Foo": 1, "Bar": 2}}`),
		},
		"array": {
			in: []byte(`{"Foo": 1, "Bar": [{"Foo": 1, "Bar": 2}]}`),
		},
	}
	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			schema, err := JsonToSchema(uc.in)
			assert.NoError(t, err)
			data, err := SchemaToJson(schema)
			assert.NoError(t, err)
			assert.JSONEq(t, string(uc.in), string(data))
		})
	}
}
