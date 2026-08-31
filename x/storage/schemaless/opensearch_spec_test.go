package schemaless_test

import (
	"os"
	"testing"

	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"github.com/widmogrod/mkunion/x/storage/schemaless/spec"
	"github.com/widmogrod/mkunion/x/storage/schemaless/spec/specdata"
)

func openSearchClient(t *testing.T) *opensearch.Client {
	t.Helper()
	address := os.Getenv("OPENSEARCH_ADDRESS")
	if address == "" {
		t.Skip(`Skipping test because:
- OPENSEARCH_ADDRESS is not set.
- Assuming OpenSearch is not running.

To run this test, please set OPENSEARCH_ADDRESS to the address of your OpenSearch instance, like:
	export OPENSEARCH_ADDRESS=http://localhost:9200
`)
	}

	client, err := opensearch.NewClient(opensearch.Config{
		Addresses: []string{address},
	})
	require.NoError(t, err)
	return client
}

// openSearchCapabilities: OpenSearch downgrades three capabilities.
//   - BackwardPagination: search_after cursors only move forward.
//   - AtomicBatch: each record is written in its own request, so a failing
//     batch can leave earlier records durably written.
//   - MonotonicOverwriteVersion: an overwrite writes the stale writer's
//     version+1, so the stored version can move backward.
func openSearchCapabilities() spec.Capabilities {
	return spec.FullCapabilities().
		WithoutBackwardPagination().
		WithoutAtomicBatch().
		WithoutMonotonicOverwriteVersion()
}

func TestOpenSearchRepositorySpec(t *testing.T) {
	client := openSearchClient(t)

	const indexName = "spec-test-records-index"

	// searching an index that was never written to is an error, not an empty
	// result; one overwrite-policy write makes sure the index exists
	warmup := schemaless.NewOpenSearchRepository[schemaless.ExampleRecord](client, indexName)
	command := schemaless.Save(schemaless.Record[schemaless.ExampleRecord]{
		ID: "warmup", Type: "spec-warmup", Data: schemaless.ExampleRecord{Name: "Warmup", Age: 1},
	})
	command.UpdatingPolicy = schemaless.PolicyOverwriteServerChanges
	_, err := warmup.UpdateRecords(command)
	require.NoError(t, err, "while creating the shared test index")

	spec.RunRepositorySpec(t,
		func(t *testing.T) schemaless.Repository[schemaless.ExampleRecord] {
			// the suite namespaces record types, so one shared index is fine
			return schemaless.NewOpenSearchRepository[schemaless.ExampleRecord](client, indexName)
		},
		openSearchCapabilities(),
	)
}

func TestOpenSearchComplexQuerySpec(t *testing.T) {
	client := openSearchClient(t)

	// a dedicated index: every payload type gets its own namespace, so an
	// unfiltered query never decodes foreign payloads
	const indexName = "spec-test-vehicles-index"

	warmup := schemaless.NewOpenSearchRepository[specdata.Vehicle](client, indexName)
	command := schemaless.Save(schemaless.Record[specdata.Vehicle]{
		ID: "warmup", Type: "spec-warmup",
		Data: specdata.Vehicle{Name: "warmup", Wheels: 1, Engine: &specdata.Petrol{Brand: "warmup", Cylinders: 1}},
	})
	command.UpdatingPolicy = schemaless.PolicyOverwriteServerChanges
	_, err := warmup.UpdateRecords(command)
	require.NoError(t, err, "while creating the shared vehicle index")

	spec.RunComplexQuerySpec(t,
		func(t *testing.T) schemaless.Repository[specdata.Vehicle] {
			return schemaless.NewOpenSearchRepository[specdata.Vehicle](client, indexName)
		},
		openSearchCapabilities(),
	)
}
