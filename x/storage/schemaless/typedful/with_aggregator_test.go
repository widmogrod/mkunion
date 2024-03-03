package typedful

import (
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/predicate"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"testing"
)

func TestNewRepositoryInMemory(t *testing.T) {
	storage := schemaless.NewInMemoryRepository[schema.Schema]()
	aggregate := func() schemaless.Aggregator[User, UsersCountByAge] {
		return schemaless.NewKeyedAggregate[User, UsersCountByAge](
			"byAge",
			[]string{"user"},
			func(data User) (string, UsersCountByAge) {
				return AgeRangeKey(data.Age), UsersCountByAge{
					Count: 1,
				}
			},
			func(a, b UsersCountByAge) (UsersCountByAge, error) {
				return UsersCountByAge{
					Count: a.Count + b.Count,
				}, nil
			},
			storage,
		)
	}
	r := NewTypedRepoWithAggregator[User, UsersCountByAge](
		storage,
		aggregate,
	)

	updated, err := r.UpdateRecords(exampleUserRecords)
	assert.NoError(t, err)
	assert.Len(t, updated.Saved, 5)
	assert.Len(t, updated.Deleted, 0)

	result, err := r.FindingRecords(schemaless.FindingRecords[schemaless.Record[User]]{
		Where: predicate.MustWhere(
			`Data.Age > :age`,
			predicate.ParamBinds{
				":age": schema.MkInt(20),
			},
		),
		Sort: []schemaless.SortField{
			{
				Field:      "Data.Name",
				Descending: false,
			},
		},
	})
	assert.NoError(t, err)

	if assert.Len(t, result.Items, 2) {
		assert.Equal(t, "Alice", result.Items[0].Data.Name)
		assert.Equal(t, "Jane", result.Items[1].Data.Name)
	}

	results, err := storage.FindingRecords(schemaless.FindingRecords[schemaless.Record[schema.Schema]]{
		RecordType: "byAge",
		Sort: []schemaless.SortField{
			{
				Field:      "Data.Count",
				Descending: false,
			},
		},
	})
	assert.NoError(t, err)

	if assert.Len(t, results.Items, 2) {
		r, err := schemaless.RecordAs[UsersCountByAge](results.Items[0])
		assert.NoError(t, err)
		assert.Equal(t, "byAge:20-30", r.ID)
		assert.Equal(t, 1, r.Data.Count)

		r, err = schemaless.RecordAs[UsersCountByAge](results.Items[1])
		assert.NoError(t, err)
		assert.Equal(t, "byAge:30-40", r.ID)
		assert.Equal(t, 2, r.Data.Count)
	}
}
