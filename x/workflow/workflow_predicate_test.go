package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/schema"
)

func predicateContext() BaseState {
	return BaseState{
		Variables: map[string]schema.Schema{
			"a": schema.MkInt(1),
			"b": schema.MkInt(2),
			"s": schema.MkString("x"),
		},
	}
}

func cmpPred(left Reshaper, op string, right Reshaper) *Compare {
	return &Compare{Operation: op, Left: left, Right: right}
}

func varRef(name string) *GetValue  { return &GetValue{Path: name} }
func lit(x schema.Schema) *SetValue { return &SetValue{Value: x} }
func intLit(v int64) *SetValue      { return lit(schema.MkInt(v)) }
func boolCmp(op string, v int64) *Compare {
	return cmpPred(varRef("a"), op, intLit(v))
}

func TestExecutePredicateCompare(t *testing.T) {
	ctx := predicateContext()

	useCases := map[string]struct {
		pred Predicate
		want bool
	}{
		"equal true":      {boolCmp("=", 1), true},
		"equal false":     {boolCmp("=", 2), false},
		"not equal true":  {boolCmp("!=", 2), true},
		"not equal false": {boolCmp("!=", 1), false},
		"less true":       {boolCmp("<", 2), true},
		"less false":      {boolCmp("<", 1), false},
		"lte equal":       {boolCmp("<=", 1), true},
		"lte false":       {boolCmp("<=", 0), false},
		"greater true":    {boolCmp(">", 0), true},
		"greater false":   {boolCmp(">", 1), false},
		"gte equal":       {boolCmp(">=", 1), true},
		"gte false":       {boolCmp(">=", 2), false},
		"variable to variable": {
			cmpPred(varRef("a"), "<", varRef("b")), true,
		},
		"string comparison": {
			cmpPred(varRef("s"), "=", lit(schema.MkString("x"))), true,
		},
	}
	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			got, err := ExecutePredicate(ctx, uc.pred, nil)
			require.NoError(t, err)
			assert.Equal(t, uc.want, got)
		})
	}

	t.Run("invalid operator errors", func(t *testing.T) {
		_, err := ExecutePredicate(ctx, boolCmp("~~", 1), nil)
		assert.ErrorContains(t, err, "invalid compare operator")
	})

	t.Run("unknown variable on the left errors", func(t *testing.T) {
		_, err := ExecutePredicate(ctx, cmpPred(varRef("nope"), "=", intLit(1)), nil)
		assert.ErrorContains(t, err, "left comapre failed")
	})

	t.Run("unknown variable on the right errors", func(t *testing.T) {
		_, err := ExecutePredicate(ctx, cmpPred(intLit(1), "=", varRef("nope")), nil)
		assert.ErrorContains(t, err, "right comapre failed")
	})
}

func TestExecutePredicateCombinators(t *testing.T) {
	ctx := predicateContext()
	yes := boolCmp("=", 1)
	no := boolCmp("=", 2)
	bad := cmpPred(varRef("nope"), "=", intLit(1))

	useCases := map[string]struct {
		pred Predicate
		want bool
	}{
		"and all true":        {&And{L: []Predicate{yes, yes}}, true},
		"and short-circuits":  {&And{L: []Predicate{no, yes}}, false},
		"empty and is true":   {&And{}, true},
		"or any true":         {&Or{L: []Predicate{no, yes}}, true},
		"or all false":        {&Or{L: []Predicate{no, no}}, false},
		"empty or is false":   {&Or{}, false},
		"not inverts":         {&Not{P: no}, true},
		"not inverts back":    {&Not{P: yes}, false},
		"nested combinations": {&And{L: []Predicate{yes, &Or{L: []Predicate{no, &Not{P: no}}}}}, true},
	}
	for name, uc := range useCases {
		t.Run(name, func(t *testing.T) {
			got, err := ExecutePredicate(ctx, uc.pred, nil)
			require.NoError(t, err)
			assert.Equal(t, uc.want, got)
		})
	}

	t.Run("errors propagate through combinators", func(t *testing.T) {
		for _, p := range []Predicate{
			&And{L: []Predicate{bad}},
			&Or{L: []Predicate{bad}},
			&Not{P: bad},
		} {
			_, err := ExecutePredicate(ctx, p, nil)
			assert.Error(t, err, "%T", p)
		}
	})

	t.Run("and short-circuit skips the error after a false", func(t *testing.T) {
		got, err := ExecutePredicate(ctx, &And{L: []Predicate{no, bad}}, nil)
		require.NoError(t, err, "documents short-circuit: the failing branch is never evaluated")
		assert.False(t, got)
	})
}
