package ast

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSimpleAST(t *testing.T) {
	ast := HumanFriendlyRules{
		AtLeastOneOf: []FiledBoostRule{
			{
				"question.thanks": BoostRuleOneOf{
					ConstBoost: &ConstBoost{
						Boost: 1.0,
						RuleOneOf: RuleOneOf{
							Gt: 10,
						},
					},
				},
			},
		},
		MustMatch: []FiledRule{
			{
				"question.similarity": RuleOneOf{Gt: 0.98},
			},
		},
	}

	// json serialise ast_score_calculation.go, and print it
	res, err := json.Marshal(ast)
	assert.NoError(t, err)
	t.Log(string(res))

	expected := `{
  "atLeastOneOf": [
    {
      "question.thanks": {
        "boost": {
          "boost": 1,
          "gt": 10
        }
      }
    }
  ],
  "mustMatch": [
    {
      "question.similarity": {
        "gt": 0.98
      }
    }
  ]
}`
	assert.JSONEq(t, expected, string(res))

	unm := HumanFriendlyRules{}
	err = json.Unmarshal([]byte(expected), &unm)
	assert.NoError(t, err)

	res2, err := json.Marshal(unm)
	assert.NoError(t, err)
	assert.JSONEq(t, expected, string(res2))
}
