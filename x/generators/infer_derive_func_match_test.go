package generators

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInferDeriveFuncMatchFromFile(t *testing.T) {
	out, err := InferDeriveFuncMatchFromFile("testutils/derive_func_match.go")
	assert.NoError(t, err)
	assert.Equal(t, "testutils", out.PackageName)

	spec, err := out.MatchSpec("MatchAlphabetNumberTuple")
	assert.NoError(t, err)

	assert.Equal(t, "MatchAlphabetNumberTuple", spec.Name)
	assert.Equal(t, []string{"Alphabet", "Number"}, spec.Inputs)
	assert.Equal(t, []string{"Match1", "Match2", "Match3"}, spec.Names)
	assert.Equal(t, [][]string{
		{"*A1", "*N0"},
		{"*C3", "any"},
		{"any", "any"},
	}, spec.Cases)
}
