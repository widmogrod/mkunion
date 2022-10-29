package ast

func NewScoreCalculatorFromHumanFriendlyRules() *ScoreCalculationFromHumanFriendlyRules {
	return &ScoreCalculationFromHumanFriendlyRules{
		interpret: NewInterpreter(),
	}
}

type ScoreCalculationFromHumanFriendlyRules struct {
	interpret *IntrprateOperatorAST
}

// Calculate is based on rule that
// - must match - then allow should match calculation, otherwise score is 0
// - should - takes and adds score from all should rules
// - result is final score for given data record
func (s *ScoreCalculationFromHumanFriendlyRules) Calculate(ast HumanFriendlyRules, data MapAny) (score float64) {
	if !s.interpret.Eval(ast.MustMatchToOperation(), data) {
		return 0
	}

	score = 0

	for _, m := range ast.AtLeastOneOf {
		for field, rule := range m {
			if rule.ConstBoost != nil {
				op, err := ast.toOperator(rule.ConstBoost.RuleOneOf, field)
				if err != nil {
					// TODO in production code, we should log error
					continue
				}

				if s.interpret.Eval(op, data) {
					// TODO score function could be configurable
					score += rule.ConstBoost.Boost
				}
			} else if rule.ValueBoost != nil {
				// TODO implement score function
				// that will take value of attribute and use it as score indication
			}
		}
	}

	return score
}

func NewScoreCalculatorFromDescriptionOfBestResult() *ScoreCalculationFromDescriptionOfBestResult {
	return &ScoreCalculationFromDescriptionOfBestResult{
		interpret: NewInterpreter(),
	}
}

type ScoreCalculationFromDescriptionOfBestResult struct {
	interpret *IntrprateOperatorAST
}

// Calculate is based on rule that
// - must match - then allow should match calculation, otherwise score is 0
// - should - takes and adds score from all should rules
// - result is final score for given data record
func (s *ScoreCalculationFromDescriptionOfBestResult) Calculate(ast DescriptionOfBestResult, data MapAny) (score float64) {
	found := false
	for _, rule := range ast.MustMatchOneOf {
		op := rule.ToOperation()
		if op == nil {
			// TODO in production code, we should log warning
			continue
		}

		if s.interpret.Eval(op, data) {
			found = true
			// we found rule that classify must match
			break
		}
	}

	if !found {
		return 0
	}

	score = 0

	for _, boost := range ast.AtLeastOneOf {
		op := boost.When.ToOperation()
		if op == nil {
			// TODO in production code, we should log warning
			continue
		}

		if s.interpret.Eval(op, data) {
			if boost.Boost != nil {
				// TODO score function could be configurable
				score += *boost.Boost
			} else if boost.MultiplyUsingFieldValue != nil {
				val, ok := s.interpret.Value(&Accessor{Path: boost.When.Field.ToPath()}, data)
				if !ok {
					// TODO in production code, we should log warning
					continue
				}

				if score == 0 {
					score = 1
				}

				if v, ok := val.(float64); ok {
					score *= v
				} else if v, ok := val.(int); ok {
					score *= float64(v)
				}
			}
		}
	}

	return score
}
