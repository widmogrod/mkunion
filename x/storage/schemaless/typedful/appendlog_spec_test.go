package typedful

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/shape"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"github.com/widmogrod/mkunion/x/storage/schemaless/spec"
)

func TestTypedAppendLogSpec(t *testing.T) {
	spec.RunAppendLogSpec(t, spec.AppendLogTypedful,
		func(t *testing.T) schemaless.AppendLoger[schemaless.ExampleRecord] {
			shapeDef, found := shape.LookupShapeReflectAndIndex[schemaless.Change[schema.Schema]]()
			require.True(t, found)
			return NewTypedAppendLog[schemaless.ExampleRecord](
				schemaless.NewAppendLog[schema.Schema](shapeDef),
			)
		},
		// Append takes a concrete *AppendLog[T]; the typed wrapper cannot
		// merge it into its schema-typed backing log
		spec.FullAppendLogCapabilities().WithoutMergeAppend(),
	)
}

// TestMain regenerates the capability docs after a green run, so the
// append-log report written by this package lands in capabilities.md and
// the README, the same way the schemaless package does it.
func TestMain(m *testing.M) {
	code := m.Run()
	if code == 0 {
		if err := spec.GenerateDocs(); err != nil {
			fmt.Fprintf(os.Stderr, "spec: could not generate capability docs: %s\n", err)
			code = 1
		}
	}
	os.Exit(code)
}
