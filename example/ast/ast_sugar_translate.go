package ast

var _ SyntaxSugarVisitor = (*TranslateSyntaxASTtoOperatorAST)(nil)

type TranslateSyntaxASTtoOperatorAST struct {
	currentField []string
}

func (a *TranslateSyntaxASTtoOperatorAST) VisitEqTo(v *EqTo) any {
	return &Eq{
		L: &Accessor{a.currentField},
		R: &Lit{Value: v.V},
	}
}

func (a *TranslateSyntaxASTtoOperatorAST) VisitGrThan(v *GrThan) any {
	return &Gt{
		L: &Accessor{a.currentField},
		R: &Lit{Value: v.V},
	}
}

func (a *TranslateSyntaxASTtoOperatorAST) VisitOrFields(v *OrFields) any {
	var result []Operator
	for field, value := range *v {
		a.currentField = append(a.currentField, field)
		result = append(result, value.AcceptSyntaxSugar(a).(Operator))
		a.currentField = a.currentField[:len(a.currentField)-1]
	}

	or := Or{result}
	return &or
}
