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
  "$type": "github.com/widmogrod/mkunion/example/ast.Lit",
  "github.com/widmogrod/mkunion/example/ast.Lit": {
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
  "$type": "github.com/widmogrod/mkunion/example/ast.Eq",
  "github.com/widmogrod/mkunion/example/ast.Eq": {
    "L": {
      "$type": "github.com/widmogrod/mkunion/example/ast.Accessor",
      "github.com/widmogrod/mkunion/example/ast.Accessor": {
        "Path": [
          "foo"
        ]
      }
    },
    "R": {
      "$type": "github.com/widmogrod/mkunion/example/ast.Lit",
      "github.com/widmogrod/mkunion/example/ast.Lit": {
        "Value": "baz"
      }
    }
  }
}
`, string(jsonData))
}
