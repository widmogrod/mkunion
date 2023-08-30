package schemaless

import (
	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/predicate"
	"os"
	"testing"
)

func TestNewOpenSearchRepository(t *testing.T) {
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
		Addresses: []string{
			address,
		},
	})

	repo := NewOpenSearchRepository(client, "test-records-index")

	// clean database
	err = repo.UpdateRecords(UpdateRecords[Record[schema.Schema]]{
		Deleting: exampleUpdateRecords.Saving,
	})
	assert.NoError(t, err, "while deleting records")

	err = repo.UpdateRecords(exampleUpdateRecords)
	assert.NoError(t, err, "while saving records")

	result, err := repo.FindingRecords(FindingRecords[Record[schema.Schema]]{
		RecordType: "exampleRecord",
		Where: predicate.MustWhere(
			"Data.Age > :age AND Data.Age < :maxAge",
			predicate.ParamBinds{
				":age":    schema.MkInt(20),
				":maxAge": schema.MkInt(40),
			}),
		Sort: []SortField{
			{
				Field:      "Data.Name",
				Descending: false,
			},
		},
		Limit: 2,
	})
	assert.NoError(t, err, "while finding records")

	if assert.Len(t, result.Items, 2, "first page should have 2 items") {
		assert.Equal(t, "Alice", schema.AsDefault[string](schema.Get(result.Items[0].Data, "Name"), "no-name"))
		assert.Equal(t, "Jane", schema.AsDefault[string](schema.Get(result.Items[1].Data, "Name"), "no-name"))

		if assert.True(t, result.HasNext(), "should have next page of results") {
			nextResult, err := repo.FindingRecords(*result.Next)

			assert.NoError(t, err)
			if assert.Len(t, nextResult.Items, 1, "second page should have 1 item") {
				assert.Equal(t, "Zarlie", schema.AsDefault[string](schema.Get(nextResult.Items[0].Data, "Name"), "no-name"))
			}
		}

		result, err := repo.Get(result.Items[0].ID, result.Items[0].Type)
		assert.NoError(t, err)
		assert.Equal(t, "Alice", schema.AsDefault[string](schema.Get(result.Data, "Name"), "no-name"))
	}
}
