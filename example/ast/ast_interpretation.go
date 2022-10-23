package ast

import (
	"reflect"
)

func NewInterpreter() *IntrprateOperatorAST {
	return &IntrprateOperatorAST{
		valueExtractor: &InterpretValueAST{},
	}
}

type MapAny = map[string]any

var noResult = &struct{}{}

var _ OperatorVisitor = (*IntrprateOperatorAST)(nil)
var _ ValueVisitor = (*InterpretValueAST)(nil)

type InterpretValueAST struct {
	V MapAny
}

func (e *InterpretValueAST) VisitALit(v *ALit) any {
	return v.Value
}

func (e *InterpretValueAST) VisitAAccessor(v *AAccessor) any {
	var val any = e.V
	for _, p := range v.Path {
		m, ok := val.(MapAny)
		if !ok {
			return noResult
		}
		val, ok = m[p]
		if !ok {
			return noResult
		}
	}

	return val
}

type IntrprateOperatorAST struct {
	valueExtractor *InterpretValueAST
}

func (e *IntrprateOperatorAST) VisitAEq(v *AEq) any {
	l := v.L.Accept(e.valueExtractor)
	r := v.R.Accept(e.valueExtractor)
	if l == noResult || r == noResult {
		return false
	}

	return reflect.DeepEqual(l, r)
}

func (e *IntrprateOperatorAST) VisitAGt(v *AGt) any {
	l := v.L.Accept(e.valueExtractor)
	r := v.R.Accept(e.valueExtractor)
	if l == noResult || r == noResult {
		return false
	}

	// uglinest of golang
	switch x := l.(type) {
	case int:
		switch y := r.(type) {
		case int:
			return x > y
		}
		return false
	case float64:
		switch y := r.(type) {
		case float64:
			return x > y
		}
		return false
	}

	return false
}

func (e *IntrprateOperatorAST) VisitAOr(v *AOr) any {
	for _, p := range *v {
		if p.Accept(e).(bool) {
			return true
		}
	}
	return false
}

func (e *IntrprateOperatorAST) Eval(ast Operator, data MapAny) bool {
	e.valueExtractor.V = data
	return ast.Accept(e).(bool)
}
