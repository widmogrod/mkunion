package shape

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

type weatherInput struct {
	Location string `desc:"The city and state e.g. San Francisco, CA" name:"location"`
	Unit     string `desc:"c or f" name:"unit" enum:"c,f"`
}

func TestToOpenAIFunctionDefinition(t *testing.T) {
	in := FromGo(weatherInput{})
	def := ToOpenAIFunctionDefinition("get_weather", "Determine weather in my location", in)
	defJSON, err := json.Marshal(def)
	assert.NoError(t, err)

	t.Logf("defJSON: %s", defJSON)

	expected := `{
  "name": "get_weather",
  "description": "Determine weather in my location",
  "parameters": {
    "type": "object",
    "properties": {
      "location": {
        "type": "string",
        "description": "The city and state e.g. San Francisco, CA",
        "properties": {}
      },
      "unit": {
        "type": "string",
        "description": "c or f",
        "enum": [
          "c",
          "f"
        ],
        "properties": {}
      }
    }
  }
}`
	assert.JSONEq(t, expected, string(defJSON))
}
