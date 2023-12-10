package generators

import (
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/shape"
	"os"
	"testing"
)

func TestDeSerJSONGenerator(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	inferred, err := shape.InferFromFile("testutils/tree.go")
	assert.NoError(t, err)

	g := NewDeSerJSONGenerator(
		inferred.RetrieveUnion("Tree"),
		NewHelper(WithPackageName("testutils")),
	)

	result, err := g.Generate()
	assert.NoError(t, err)

	reference, err := os.ReadFile("deser_json_generator_test_tree.go.asset")
	assert.NoError(t, err)
	assert.Equal(t, string(reference), string(result))
}
