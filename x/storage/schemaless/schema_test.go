package schemaless

import (
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/predicate"
	"testing"
)

func TestNewRepository2WithSchema(t *testing.T) {
	repo := NewInMemoryRepository[ExampleRecord]()
	assert.NotNil(t, repo)

	updated, err := repo.UpdateRecords(exampleUpdateRecords)
	assert.NoError(t, err)
	assert.Len(t, updated.Saved, 5, "should have 5 saved records")
	assert.Len(t, updated.Deleted, 0, "should have 0 deleted records")

	result, err := repo.FindingRecords(FindingRecords[Record[ExampleRecord]]{
		Where: predicate.MustWhere(`Data.Age > :age AND Data.Age < :maxAge`, predicate.ParamBinds{
			":age":    schema.MkInt(20),
			":maxAge": schema.MkInt(40),
		}, nil),
		Sort: []SortField{
			{
				Field:      `Data.Name`,
				Descending: false,
			},
		},
		Limit: 2,
	})
	assert.NoError(t, err)

	if assert.Len(t, result.Items, 2, "first page should have 2 items") {
		assert.Equal(t, "Alice", result.Items[0].Data.Name, "no-name")
		assert.Equal(t, "Jane", result.Items[1].Data.Name, "no-name")
	}

	if assert.True(t, result.HasNext(), "should have next page of results") {
		nextResult, err := repo.FindingRecords(*result.Next)

		assert.NoError(t, err)
		if assert.Len(t, nextResult.Items, 1, "second page should have 1 item") {
			assert.Equal(t, "Zarlie", nextResult.Items[0].Data.Name, "no-name")

			// find last before
			if assert.True(t, nextResult.HasPrev(), "should have previous page of results") {
				beforeResult, err := repo.FindingRecords(*nextResult.Prev)
				assert.NoError(t, err)

				if assert.Len(t, beforeResult.Items, 2, "before page should have 1 item") {
					assert.Equal(t, "Jane", beforeResult.Items[1].Data.Name, "no-name")
				}
			}
		}
	}
}

func TestRepositoryWithSchema_UpdateRecords_Deletion(t *testing.T) {
	repo := NewInMemoryRepository[ExampleRecord]()
	assert.NotNil(t, repo)

	updated, err := repo.UpdateRecords(exampleUpdateRecords)
	assert.NoError(t, err)

	assert.Len(t, updated.Saved, 5, "should have 5 saved records")
	assert.Len(t, updated.Deleted, 0, "should have 0 deleted records")

	result, err := repo.FindingRecords(FindingRecords[Record[ExampleRecord]]{})
	assert.NoError(t, err)
	assert.Len(t, result.Items, 5, "should have 5 records")
	assert.False(t, result.HasNext(), "should not have next page of results")

	deleting := map[string]Record[ExampleRecord]{}
	for _, item := range result.Items {
		deleting[item.ID] = item
	}

	updated, err = repo.UpdateRecords(UpdateRecords[Record[ExampleRecord]]{
		Deleting: deleting,
	})

	assert.NoError(t, err)
	assert.Len(t, updated.Saved, 0, "should have 0 saved records")
	assert.Len(t, updated.Deleted, 5, "should have 5 deleted records")

	result, err = repo.FindingRecords(FindingRecords[Record[ExampleRecord]]{})
	assert.NoError(t, err)
	for _, item := range result.Items {
		t.Log(item)
	}
	assert.Len(t, result.Items, 0, "should have 0 records")
}
