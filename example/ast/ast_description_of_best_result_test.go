package ast

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestHumanFriendlyTwo(t *testing.T) {
	ast := DescriptionOfBestResult{
		AtLeastOneOf: []BoostWhenFieldRuleOneOf{
			{
				Boost: PtrFloat(3.0),
				When: FieldRuleOneOf{
					Field: "question.thanks", Gt: 10,
					And: &FieldRuleOneOf{
						Field: "question.avgRating",
						Gt:    3,
					},
				},
			},
		},
		MustMatch: &FieldRuleOneOf{
			Field: "question.similarity",
			Gt:    0.98,
		},
	}

	// json serialise ast_score_calculation.go, and print it
	res, err := json.Marshal(ast)
	assert.NoError(t, err)
	t.Log(string(res))

	expected := `{
  "atLeastOneOf": [
    {
      "boost": 3,
      "when": {
        "field": "question.thanks",
        "gt": 10,
        "and": {
          "field": "question.avgRating",
          "gt": 3
        }
      }
    }
  ],
  "mustMatch": {
    "field": "question.similarity",
    "gt": 0.98
  }
}`

	assert.JSONEq(t, expected, string(res))

	unm := DescriptionOfBestResult{}
	err = json.Unmarshal([]byte(expected), &unm)
	assert.NoError(t, err)

	res2, err := json.Marshal(unm)
	assert.NoError(t, err)
	assert.JSONEq(t, expected, string(res2))
}
