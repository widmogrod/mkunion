package generators

import (
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/shape"
	"os"
	"testing"

	_ "github.com/widmogrod/mkunion/x/generators/testutils"
)

func TestSerdeJSONUnion_Generate_Tree(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	inferred, err := shape.InferFromFile("testutils/tree.go")
	assert.NoError(t, err)

	g := NewSerdeJSONUnion(inferred.RetrieveUnion("Tree"))

	result, err := g.Generate()
	assert.NoError(t, err)

	reference, err := os.ReadFile("serde_json_union_test.go.asset")
	assert.NoError(t, err)
	assert.Equal(t, string(reference), string(result))
}

func TestSerdeJSONUnion_Generate_Forest(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	inferred, err := shape.InferFromFile("testutils/tree.go")
	assert.NoError(t, err)

	g := NewSerdeJSONUnion(inferred.RetrieveUnion("Forest"))

	result, err := g.Generate()
	assert.NoError(t, err)

	reference, err := os.ReadFile("serde_json_union_alias_test.go.asset")
	assert.NoError(t, err)
	assert.Equal(t, string(reference), string(result))
}
