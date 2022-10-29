package ast

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInterpreter(t *testing.T) {
	var data = MapAny{
		"foo": "bar",
		"question": MapAny{
			"thanks": 22,
		},
	}
	useCases := map[string]struct {
		data       MapAny
		expression Operator
		expected   bool
	}{
		"simple equality (foo == 'bar')": {
			data: data,
			expression: &Eq{
				L: &Accessor{[]string{"foo"}},
				R: &Lit{"bar"},
			},
			expected: true,
		},
		"simple equality (foo == 'bar') fails": {
			data: data,
			expression: &Eq{
				L: &Accessor{[]string{"foo"}},
				R: &Lit{"baz"},
			},
			expected: false,
		},
		"simple equality !(foo == 'baz')": {
			data: data,
			expression: &Not{&Eq{
				L: &Accessor{[]string{"foo"}},
				R: &Lit{"baz"},
			}},
			expected: true,
		},
		"simple equality !(foo == 'bar') fails": {
			data: data,
			expression: &Not{&Eq{
				L: &Accessor{[]string{"foo"}},
				R: &Lit{"bar"},
			}},
			expected: false,
		},
		"simple comparison (question.thanks > 10)": {
			data: data,
			expression: &Or{
				&Gt{
					L: &Accessor{[]string{"question", "thanks"}},
					R: &Lit{10},
				},
			},
			expected: true,
		},
		"simple comparison (question.thanks > 100) fails": {
			data: data,
			expression: &Or{
				&Gt{
					L: &Accessor{[]string{"question", "thanks"}},
					R: &Lit{100},
				},
			},
			expected: false,
		},
		"complex (foo == 'baz') or (question.thanks > 100) fails": {
			data: data,
			expression: &Or{
				&Eq{
					L: &Accessor{[]string{"foo"}},
					R: &Lit{"baz"},
				},
				&Gt{
					L: &Accessor{[]string{"question", "thanks"}},
					R: &Lit{100},
				},
			},
			expected: false,
		},
		"complex (foo == 'bar') and (question.thanks > 10) pass": {
			data: data,
			expression: &And{
				&Eq{
					L: &Accessor{[]string{"foo"}},
					R: &Lit{"bar"},
				},
				&Gt{
					L: &Accessor{[]string{"question", "thanks"}},
					R: &Lit{10},
				},
			},
			expected: true,
		},
		"complex (foo == 'baz') and (question.thanks > 100) fails": {
			data: data,
			expression: &And{
				&Eq{
					L: &Accessor{[]string{"foo"}},
					R: &Lit{"baz"},
				},
				&Gt{
					L: &Accessor{[]string{"question", "thanks"}},
					R: &Lit{100},
				},
			},
			expected: false,
		},
	}
	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			interpreter := NewInterpreter()
			result := interpreter.Eval(uc.expression, uc.data)
			assert.Equal(t, uc.expected, result)
		})
	}
}
