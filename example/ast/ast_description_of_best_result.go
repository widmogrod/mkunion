package ast

type (
	DescriptionOfBestResult struct {
		// OneOf implies OR for list of rules - AtLeastOneOf
		// Lack  of it implies AND for list of rules - MustMatch
		AtLeastOneOf []BoostWhenFieldRuleOneOf `json:"atLeastOneOf"`
		MustMatch    *FieldRuleOneOf           `json:"mustMatch"`
	}
)

type (
	BoostWhenFieldRuleOneOf struct {
		Boost    *float64 `json:"boost,omitempty"`
		Multiply *float64 `json:"multiply,omitempty"`

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

func PtrFloat(f float64) *float64 {
	return &f
}
