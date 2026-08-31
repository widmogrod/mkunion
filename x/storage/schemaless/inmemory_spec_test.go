package schemaless_test

import (
	"testing"

	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"github.com/widmogrod/mkunion/x/storage/schemaless/spec"
)

// TestInMemoryRepositorySpec: the in-memory repository defines the full
// contract, so it runs with FullCapabilities and skips nothing.
func TestInMemoryRepositorySpec(t *testing.T) {
	spec.RunRepositorySpec(t,
		func(t *testing.T) spec.Repo {
			return schemaless.NewInMemoryRepository[spec.Data]()
		},
		spec.FullCapabilities(),
	)
}

// TestInMemoryComplexQuerySpec: nested-union payloads and composed
// predicates, full contract.
func TestInMemoryComplexQuerySpec(t *testing.T) {
	spec.RunComplexQuerySpec(t,
		func(t *testing.T) spec.ComplexRepo {
			return schemaless.NewInMemoryRepository[spec.Vehicle]()
		},
		spec.FullCapabilities(),
	)
}
