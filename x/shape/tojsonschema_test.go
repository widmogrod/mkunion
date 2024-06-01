package shape

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type structB struct {
	Count int      `desc:"total number of words in sentence"`
	Words []string `desc:"list of words in sentence"`
	//EnumTest Temperature `desc:"temperature enum"`
}

func TestToJsonSchema(t *testing.T) {
	result := FromGo(structB{})
	schema := ToJsonSchema(result)

	t.Logf("schema: %s", schema)

	expected := `{
  "$ref": "#/$defs/structB",
  "$defs": {
    "structB": {
      "type": "object",
      "properties": {
        "Count": {
          "type": "number",
          "description": "total number of words in sentence"
        },
        "Words": {
          "type": "array",
          "items": {
            "type": "string"
          },
          "description": "list of words in sentence"
        }
      }
    }
  },
  "$schema": "https://json-schema.org/draft/2020-12/schema"
}`
	assert.JSONEq(t, expected, schema)
}
