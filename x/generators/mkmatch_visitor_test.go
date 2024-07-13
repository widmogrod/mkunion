package generators

import (
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/shape"
	"testing"
)

func TestInferDeriveFuncMatchFromFile(t *testing.T) {
	inferred, err := shape.InferFromFile("testutils/derive_func_match.go")
	assert.NoError(t, err)

	out := NewMkMatchTaggedNodeVisitor()
	out.FromInferredInfo(inferred)

	assert.NoError(t, err)
	assert.Equal(t, "testutils", inferred.PackageName())

	specs := out.Specs()
	assert.Len(t, specs, 1)

	spec := specs[0]

	assert.Equal(t, "MatchAlphabetNumberTuple", spec.Name)
	assert.Equal(t, []string{"Alphabet", "Number"}, spec.Inputs)
	assert.Equal(t, []string{"Match1", "Match2", "Match3", "Finally"}, spec.Names)
	assert.Equal(t, [][]string{
		{"*A1", "*N0"},
		{"*C3", "*time.Duration"},
		{"map[Some[*strings.Replacer]]*bytes.Buffer", "*time.Duration"},
		{"any", "any"},
	}, spec.Cases)

	assert.Equal(t, map[string]string{"bytes": "bytes", "strings": "strings", "time": "time"}, spec.UsedPackMap)
}
