package ast

var _ SyntaxSugarVisitor = (*TranslateSyntaxASTtoOperatorAST)(nil)

type TranslateSyntaxASTtoOperatorAST struct {
	currentField []string
}

func (a *TranslateSyntaxASTtoOperatorAST) VisitEqTo(v *EqTo) any {
	// Create a copy of currentField to avoid sharing the underlying array
	pathCopy := make([]string, len(a.currentField))
	copy(pathCopy, a.currentField)
	return &Eq{
		L: &Accessor{pathCopy},
		R: &Lit{Value: v.V},
	}
}

func (a *TranslateSyntaxASTtoOperatorAST) VisitGrThan(v *GrThan) any {
	// Create a copy of currentField to avoid sharing the underlying array
	pathCopy := make([]string, len(a.currentField))
	copy(pathCopy, a.currentField)
	return &Gt{
		L: &Accessor{pathCopy},
		R: &Lit{Value: v.V},
	}
}

func (a *TranslateSyntaxASTtoOperatorAST) VisitOrFields(v *OrFields) any {
	var result []Operator
	for field, value := range v.M {
		a.currentField = append(a.currentField, field)
		result = append(result, value.AcceptSyntaxSugar(a).(Operator))
		a.currentField = a.currentField[:len(a.currentField)-1]
	}

	or := Or{result}
	return &or
}
