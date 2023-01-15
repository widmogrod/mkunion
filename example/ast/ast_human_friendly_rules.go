package ast

import (
	"errors"
	"strings"
)

type (
	HumanFriendlyRules struct {
		// OneOf implies OR for list of rules - AtLeastOneOf
		// Lack  of it implies AND for list of rules - MustMatch
		AtLeastOneOf []FiledBoostRule `json:"atLeastOneOf"`
		MustMatch    []FiledRule      `json:"mustMatch"`
	}
)

type (
	Field          string
	FiledRule      map[Field]RuleOneOf
	FiledBoostRule map[Field]BoostRuleOneOf
)

type (
	RuleOneOf struct {
		Eq interface{} `json:"eq,omitempty"`
		Gt interface{} `json:"gt,omitempty"`
		// There can be more operations supported
	}
	BoostRuleOneOf struct {
		ConstBoost *ConstBoost `json:"boost,omitempty"`
		ValueBoost *ValueBoost `json:"valueBoost,omitempty"`
	}
	ConstBoost struct {
		Boost float64 `json:"boost"`
		RuleOneOf
	}
	ValueBoost struct {
		RuleOneOf
		// There could be fields like
		// - multiply: 1.2
		// - exponential
		// - or logarithmic
		// That take value of attribute, and apply specific function
	}
)

func (ast HumanFriendlyRules) MustMatchToOperation() Operator {
	var res []Operator
	for _, m := range ast.MustMatch {
		for field, rule := range m {
			op, err := ast.toOperator(rule, field)
			if err != nil {
				// TODO in production code, we should log error
			}
			res = append(res, op)
		}
	}
	// TODO should be AND, but operator is not implemented yet
	and := Or{res}
	return &and
}

func (ast HumanFriendlyRules) toOperator(rule RuleOneOf, field Field) (Operator, error) {
	if rule.Eq != nil {
		return &Eq{
			L: &Accessor{Path: field.ToPath()},
			R: &Lit{Value: rule.Eq},
		}, nil
	} else if rule.Gt != nil {
		return &Gt{
			L: &Accessor{Path: field.ToPath()},
			R: &Lit{Value: rule.Gt},
		}, nil
	}
	return nil, errors.New("unknown rule")
}

func (f Field) ToPath() []string {
	return strings.Split(string(f), ".")
}
