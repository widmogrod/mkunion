package ast

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAstJsonConversionOnSimpleType(t *testing.T) {
	in := &Lit{Value: float64(12)}
	jsonData, err := ValueToJSON(in)
	assert.NoError(t, err)
	assert.JSONEq(t, `{
  "$type": "ast.Lit",
  "ast.Lit": {
    "Value": 12
  }
}`, string(jsonData))
}
func TestAstToJSONOnSumTypes(t *testing.T) {
	in := &Eq{
		L: &Accessor{[]string{"foo"}},
		R: &Lit{"baz"},
	}

	jsonData, err := OperatorToJSON(in)
	assert.NoError(t, err)
	t.Log(string(jsonData))
	assert.JSONEq(t, `{
  "$type": "ast.Eq",
  "ast.Eq": {
    "L": {
      "$type": "ast.Accessor",
      "ast.Accessor": {
        "Path": [
          "foo"
        ]
      }
    },
    "R": {
      "$type": "ast.Lit",
      "ast.Lit": {
        "Value": "baz"
      }
    }
  }
}
`, string(jsonData))
}
