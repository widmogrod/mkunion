package ast

func PtrFloat(f float64) *float64 {
	return &f
}

func PtrBool(b bool) *bool {
	return &b
}

type (
	DescriptionOfBestResult struct {
		AtLeastOneOf   []BoostWhenFieldRuleOneOf `json:"atLeastOneOf"`
		MustMatchOneOf []FieldRuleOneOf          `json:"mustMatchOneOf"`
	}
)

type (
	BoostWhenFieldRuleOneOf struct {
		Boost *float64 `json:"boost,omitempty"`
		// multiply score by value in field
		MultiplyUsingFieldValue *bool `json:"multiply,omitempty"`

		// There can be more operations supported
		When FieldRuleOneOf `json:"when"`
	}

	FieldRuleOneOf struct {
		Field Field           `json:"field,omitempty"`
		Eq    interface{}     `json:"eq,omitempty"`
		Gt    interface{}     `json:"gt,omitempty"`
		Or    *FieldRuleOneOf `json:"or,omitempty"`
		And   *FieldRuleOneOf `json:"and,omitempty"`
		Not   *FieldRuleOneOf `json:"not,omitempty"`
	}
)

func (a FieldRuleOneOf) ToOperation() Operator {
	var res Operator
	if a.Eq != nil {
		res = &Eq{
			L: &Accessor{Path: a.Field.ToPath()},
			R: &Lit{Value: a.Eq},
		}
	} else if a.Gt != nil {
		res = &Gt{
			L: &Accessor{Path: a.Field.ToPath()},
			R: &Lit{Value: a.Gt},
		}
	}

	if a.Or != nil {
		if res == nil {
			res = a.Or.ToOperation()
		} else {
			res = &Or{List: []Operator{res, a.Or.ToOperation()}}
		}
	} else if a.And != nil {
		if res == nil {
			res = a.And.ToOperation()
		} else {
			res = &And{List: []Operator{res, a.And.ToOperation()}}
		}
	} else if a.Not != nil {
		return &Not{Operator: a.Not.ToOperation()}
	}

	return res
}
