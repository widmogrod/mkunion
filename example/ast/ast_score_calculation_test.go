package ast

import (
	"github.com/stretchr/testify/assert"
	"sort"
	"testing"
)

func TestScoreCalculation_Calculate(t *testing.T) {
	ast := HumanFriendlyRules{
		AtLeastOneOf: []FiledBoostRule{
			{
				"question.thanks": BoostRuleOneOf{
					ConstBoost: &ConstBoost{
						Boost: 3.0,
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

	data := map[string]interface{}{
		"question": map[string]interface{}{
			"thanks":     22,
			"similarity": 0.99,
		},
	}

	calc := NewScoreCalculatorFromHumanFriendlyRules()
	res := calc.Calculate(ast, data)
	assert.Equal(t, 3.0, res)
}

func TestCalculationForListOfResults(t *testing.T) {
	ast := HumanFriendlyRules{
		AtLeastOneOf: []FiledBoostRule{
			{
				"question.thanks": BoostRuleOneOf{
					ConstBoost: &ConstBoost{
						Boost: 3.0,
						RuleOneOf: RuleOneOf{
							Gt: 10,
						},
					},
				},
				"question.verified": BoostRuleOneOf{
					ConstBoost: &ConstBoost{
						Boost: 100.0,
						RuleOneOf: RuleOneOf{
							Eq: true,
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

	data := []map[string]interface{}{
		{
			"question": map[string]interface{}{
				"thanks":     22,
				"similarity": 0.99,
			},
		},
		{
			"question": map[string]interface{}{
				"thanks":     2,
				"similarity": 0.99,
			},
		},
		{
			"question": map[string]interface{}{
				"thanks":     22,
				"similarity": 0.7,
			},
		},
		{
			"question": map[string]interface{}{
				"thanks":     2,
				"similarity": 0.99,
				"verified":   true,
			},
		},
	}

	calc := NewScoreCalculatorFromHumanFriendlyRules()
	for i, d := range data {
		score := calc.Calculate(ast, d)
		data[i]["score"] = score
	}

	assert.Equal(t, 3.0, data[0]["score"])
	assert.Equal(t, 0.0, data[1]["score"])
	assert.Equal(t, 0.0, data[2]["score"])
	assert.Equal(t, 100.0, data[3]["score"])

	// now sort by score
	sort.SliceStable(data, func(i, j int) bool {
		return data[i]["score"].(float64) > data[j]["score"].(float64)
	})

	// pick first result
	// notice that score 100 comes form last element, that is verified
	assert.Equal(t, 100.0, data[0]["score"])
}

func TestNewScoreCalculatorFromHDescriptionOfBestResult(t *testing.T) {
	ast := DescriptionOfBestResult{
		AtLeastOneOf: []BoostWhenFieldRuleOneOf{
			{
				Boost: PtrFloat(3.0),
				When: FieldRuleOneOf{
					Field: "question.verified", Eq: true,
				},
			},
			{
				MultiplyUsingFieldValue: PtrBool(true),
				When: FieldRuleOneOf{
					Field: "question.thanks", Gt: 10,
					And: &FieldRuleOneOf{
						Field: "question.avgScore",
						Gt:    3,
					},
				},
			},
		},
		MustMatchOneOf: []FieldRuleOneOf{
			{
				Field: "question.similarity",
				Gt:    0.98,
			},
		},
	}
	data := []map[string]interface{}{
		{
			"question": map[string]interface{}{
				"id":         1,
				"thanks":     22,
				"similarity": 0.99,
				"verified":   true,
				"avgScore":   3.0,
			},
		},
		{
			"question": map[string]interface{}{
				"id":         2,
				"thanks":     2,
				"similarity": 0.99,
				"verified":   false,
				"avgScore":   2.0,
			},
		},
		{
			"question": map[string]interface{}{
				"id":         3,
				"thanks":     22,
				"similarity": 0.7,
				"verified":   false,
				"avgScore":   4.0,
			},
		},
		{
			"question": map[string]interface{}{
				"id":         4,
				"thanks":     15,
				"similarity": 0.99,
				"verified":   false,
				"avgScore":   4.1,
			},
		},
	}

	calc := NewScoreCalculatorFromDescriptionOfBestResult()
	for i, d := range data {
		score := calc.Calculate(ast, d)
		data[i]["score"] = score
	}

	assert.Equal(t, 3.0, data[0]["score"])
	assert.Equal(t, 0.0, data[1]["score"])
	assert.Equal(t, 0.0, data[2]["score"])
	assert.Equal(t, 15.0, data[3]["score"])

	// now sort by score
	sort.SliceStable(data, func(i, j int) bool {
		return data[i]["score"].(float64) > data[j]["score"].(float64)
	})

	assert.Equal(t, 15.0, data[0]["score"])
	assert.Equal(t, 3.0, data[1]["score"])
	assert.Equal(t, 0.0, data[2]["score"])
	assert.Equal(t, 0.0, data[3]["score"])

	// pick first result

	// notice that score
}
