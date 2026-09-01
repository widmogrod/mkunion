package spec

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/shape"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
)

// The spec suites are exercised here in-package against the in-memory
// reference repository, so the suite definitions themselves are covered
// by their own package's tests (backend packages run them too, but
// per-package coverage does not count cross-package calls).
func cleanupBackendReport(t *testing.T, backend string) {
	t.Helper()
	t.Cleanup(func() {
		collector.Lock()
		delete(collector.reports, backend)
		collector.Unlock()
		_ = os.Remove(reportPath(backend))
	})
}

func TestRepositorySpecSelfCheck(t *testing.T) {
	const backend = "unittest-selfcheck"
	cleanupBackendReport(t, backend)

	RunRepositorySpec(t, backend,
		func(t *testing.T) schemaless.Repository[schemaless.ExampleRecord] {
			return schemaless.NewInMemoryRepository[schemaless.ExampleRecord]()
		},
		FullCapabilities(),
	)
}

func TestAppendLogSpecSelfCheck(t *testing.T) {
	const backend = "unittest-selfcheck-appendlog"
	// the append-log runner prefixes its report and collector names
	cleanupBackendReport(t, backend)
	cleanupBackendReport(t, "appendlog-"+backend)

	RunAppendLogSpec(t, backend,
		func(t *testing.T) schemaless.AppendLoger[schemaless.ExampleRecord] {
			shapeDef, found := shape.LookupShapeReflectAndIndex[schemaless.Change[schemaless.ExampleRecord]]()
			require.True(t, found)
			return schemaless.NewAppendLog[schemaless.ExampleRecord](shapeDef)
		},
		FullAppendLogCapabilities(),
	)
}
