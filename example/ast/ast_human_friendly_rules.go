package ast

import (
	"errors"
	"strings"
)

type (
	HumanFriendlyRules struct {
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
	}
	BoostRuleOneOf struct {
		ConstBoost *ConstBoost `json:"boost,omitempty"`
		ValueBoost *ValueBoost `json:"boostValue,omitempty"`
	}
	ConstBoost struct {
		Boost float64 `json:"boost"`
		RuleOneOf
	}
	ValueBoost struct {
		RuleOneOf
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
	and := Or(res)
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
