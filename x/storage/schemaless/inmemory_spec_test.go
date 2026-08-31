package schemaless_test

import (
	"testing"

	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"github.com/widmogrod/mkunion/x/storage/schemaless/spec"
	"github.com/widmogrod/mkunion/x/storage/schemaless/spec/specdata"
)

// TestInMemoryRepositorySpec: the in-memory repository defines the full
// contract, so it runs with FullCapabilities and skips nothing.
func TestInMemoryRepositorySpec(t *testing.T) {
	spec.RunRepositorySpec(t,
		func(t *testing.T) schemaless.Repository[schemaless.ExampleRecord] {
			return schemaless.NewInMemoryRepository[schemaless.ExampleRecord]()
		},
		spec.FullCapabilities(),
	)
}

// TestInMemoryComplexQuerySpec: nested-union payloads and composed
// predicates, full contract.
func TestInMemoryComplexQuerySpec(t *testing.T) {
	spec.RunComplexQuerySpec(t,
		func(t *testing.T) schemaless.Repository[specdata.Vehicle] {
			return schemaless.NewInMemoryRepository[specdata.Vehicle]()
		},
		spec.FullCapabilities(),
	)
}
