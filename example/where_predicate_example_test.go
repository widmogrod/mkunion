package example

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestPredicate(t *testing.T) {
	useCases := map[string]struct {
		Predicate WherePredicate
		Value     interface{}
		Expected  bool
	}{
		"simple equality": {
			Predicate: &Eq{V: "bar"},
			Value:     "bar",
			Expected:  true,
		},
		"equality fails": {
			Predicate: &Eq{V: "bar"},
			Value:     "foo",
			Expected:  false,
		},
		"path value equal": {
			Predicate: &Path{
				Parts:     []string{"foo", "bar"},
				Condition: &Eq{"baz"},
			},
			Value: map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": "baz",
				},
			},
			Expected: true,
		},
		"path value equal 2": {
			Predicate: &Path{
				Parts:     []string{"foo"},
				Condition: &Eq{"baz"},
			},
			Value: map[string]interface{}{
				"foo": map[string]interface{}{
					"bar": "baz",
				},
			},
			Expected: false,
		},
		"path don't exists": {
			Predicate: &Path{
				Parts:     []string{"foo", "bar"},
				Condition: &Eq{"baz"},
			},
			Value:    "some string",
			Expected: false,
		},
	}

	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			isTrue := &evaluatePredicate{V: uc.Value}
			result := uc.Predicate.Accept(isTrue).(bool)
			assert.Equal(t, uc.Expected, result)
		})
	}
}

var _ WherePredicateVisitor = (*evaluatePredicate)(nil)

type evaluatePredicate struct {
	V interface{}
}

func (e *evaluatePredicate) VisitEq(v *Eq) any {
	return v.V == e.V
}

func (e *evaluatePredicate) VisitAnd(v *And) any {
	for _, p := range *v {
		if !p.Accept(e).(bool) {
			return false
		}
	}
	return true
}

func (e *evaluatePredicate) VisitOr(v *Or) any {
	for _, p := range *v {
		if p.Accept(e).(bool) {
			return true
		}
	}
	return false

}

func (e *evaluatePredicate) VisitPath(v *Path) any {
	val := e.V
	for _, p := range v.Parts {
		m, ok := val.(map[string]interface{})
		if !ok {
			return false
		}
		val, ok = m[p]
		if !ok {
			return false
		}
	}

	return v.Condition.Accept(&evaluatePredicate{V: val}).(bool)
}
