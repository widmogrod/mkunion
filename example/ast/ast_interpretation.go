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

func (e *InterpretValueAST) VisitLit(v *Lit) any {
	return v.Value
}

func (e *InterpretValueAST) VisitAccessor(v *Accessor) any {
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

func (e *IntrprateOperatorAST) VisitEq(v *Eq) any {
	l := v.L.Accept(e.valueExtractor)
	r := v.R.Accept(e.valueExtractor)
	if l == noResult || r == noResult {
		return false
	}

	return reflect.DeepEqual(l, r)
}

func (e *IntrprateOperatorAST) VisitGt(v *Gt) any {
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
		case float64:
			return float64(x) > y
		}
		return false
	case float64:
		switch y := r.(type) {
		case int:
			return x > float64(y)
		case float64:
			return x > y
		}
		return false
	}

	return false
}

func (e *IntrprateOperatorAST) VisitOr(v *Or) any {
	for _, p := range v.List {
		if p.Accept(e).(bool) {
			return true
		}
	}
	return false
}

func (e *IntrprateOperatorAST) VisitAnd(v *And) any {
	for _, p := range v.List {
		if !p.Accept(e).(bool) {
			return false
		}
	}
	return true
}

func (e *IntrprateOperatorAST) VisitNot(v *Not) any {
	return !v.Operator.Accept(e).(bool)
}

func (e *IntrprateOperatorAST) Eval(ast Operator, data MapAny) bool {
	e.valueExtractor.V = data
	return ast.Accept(e).(bool)
}

func (e *IntrprateOperatorAST) Value(v Value, data MapAny) (value interface{}, found bool) {
	e.valueExtractor.V = data
	val := v.Accept(e.valueExtractor)
	return val, val != noResult
}
