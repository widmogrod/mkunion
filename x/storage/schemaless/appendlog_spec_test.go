package schemaless_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/shape"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"github.com/widmogrod/mkunion/x/storage/schemaless/spec"
)

func TestInMemoryAppendLogSpec(t *testing.T) {
	spec.RunAppendLogSpec(t, spec.AppendLogInMemory,
		func(t *testing.T) schemaless.AppendLoger[schemaless.ExampleRecord] {
			shapeDef, found := shape.LookupShapeReflectAndIndex[schemaless.Change[schemaless.ExampleRecord]]()
			require.True(t, found)
			return schemaless.NewAppendLog[schemaless.ExampleRecord](shapeDef)
		},
		spec.FullAppendLogCapabilities(),
	)
}
