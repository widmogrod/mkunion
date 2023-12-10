package ast

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAstSyntaxSugar(t *testing.T) {
	data := MapAny{
		"foo": "bar",
		"question": MapAny{
			"thanks": 22,
		},
	}

	sugarAST := OrFields{
		M: map[string]SyntaxSugar{
			"foo": &EqTo{"baz"},
			"question": &OrFields{
				M: map[string]SyntaxSugar{
					"thanks": &GrThan{10},
				},
			},
		},
	}

	operatorAST := &Or{
		[]Operator{
			&Eq{
				L: &Accessor{[]string{"foo"}},
				R: &Lit{"baz"},
			},
			&Gt{
				L: &Accessor{[]string{"question", "thanks"}},
				R: &Lit{10},
			},
		},
	}

	translatedAST := sugarAST.AcceptSyntaxSugar(&TranslateSyntaxASTtoOperatorAST{}).(Operator)

	interpreter := NewInterpreter()

	resultA := interpreter.Eval(operatorAST, data)
	assert.True(t, resultA)

	resultB := interpreter.Eval(translatedAST, data)
	assert.True(t, resultB)

	assert.Equal(t, resultA, resultB)
}
