package ast

func NewScoreCalculator() *ScoreCalculation {
	return &ScoreCalculation{
		interpret: NewInterpreter(),
	}
}

type ScoreCalculation struct {
	interpret *IntrprateOperatorAST
}

// Calculate is based on rule that
// - must match - then allow should match calculation, otherwise score is 0
// - should - takes and adds score from all should rules
// - result is final score for given data record
func (s *ScoreCalculation) Calculate(ast HumanFriendlyRules, data MapAny) (score float64) {
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
