package mkunion

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExtractInferenceForTree(t *testing.T) {
	out, err := InferFromFile("example/tree_example_test.go")
	assert.NoError(t, err)
	assert.Equal(t, "example", out.PackageName)
	assert.Equal(t,
		map[string][]Branching{
			"Branch": {
				{Lit: PtrStr("L")},
				{Lit: PtrStr("R")},
			},
			"Leaf": nil,
		},
		out.ForVariantType("Tree", []string{"Branch", "Leaf"}))

}

func TestExtractInferenceForWherePredicate(t *testing.T) {
	out, err := InferFromFile("example/where_predicate_example_test.go")
	assert.NoError(t, err)
	assert.Equal(t, "example", out.PackageName)
	assert.Equal(t,
		map[string][]Branching{
			"Eq":  nil,
			"And": nil,
			"Or":  nil,
			"Path": {
				{Lit: PtrStr("Condition")},
				{List: PtrStr("Then")},
			},
		},
		out.ForVariantType("WherePredicate", []string{"Eq", "And", "Or", "Path"}))
}
