package ast

var _ SyntaxSugarVisitor = (*TranslateSyntaxASTtoOperatorAST)(nil)

type TranslateSyntaxASTtoOperatorAST struct {
	currentField []string
}

func (a *TranslateSyntaxASTtoOperatorAST) VisitEqTo(v *EqTo) any {
	return &AEq{
		L: &AAccessor{a.currentField},
		R: &ALit{Value: v.V},
	}
}

func (a *TranslateSyntaxASTtoOperatorAST) VisitGrThan(v *GrThan) any {
	return &AGt{
		L: &AAccessor{a.currentField},
		R: &ALit{Value: v.V},
	}
}

func (a *TranslateSyntaxASTtoOperatorAST) VisitOrFields(v *OrFields) any {
	var result []Operator
	for field, value := range *v {
		a.currentField = append(a.currentField, field)
		result = append(result, value.Accept(a).(Operator))
		a.currentField = a.currentField[:len(a.currentField)-1]
	}

	or := AOr(result)
	return &or
}
