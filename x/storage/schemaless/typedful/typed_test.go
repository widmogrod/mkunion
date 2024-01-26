package typedful

import (
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/predicate"
	. "github.com/widmogrod/mkunion/x/storage/schemaless"
	"testing"
)

func TestNewRepository2Typed(t *testing.T) {
	storage := NewInMemoryRepository[schema.Schema]()
	r := NewTypedRepository[User](storage)

	updated, err := r.UpdateRecords(exampleUserRecords)
	assert.NoError(t, err)
	assert.Len(t, updated.Saved, 3)
	assert.Len(t, updated.Deleted, 0)

	result, err := r.FindingRecords(FindingRecords[Record[User]]{
		Where: predicate.MustWhere(
			`Data.Age > :age`,
			predicate.ParamBinds{
				":age": schema.MkInt(20),
			},
		),
		Sort: []SortField{
			{
				Field:      `Data.Name`,
				Descending: false,
			},
		},
	})
	assert.NoError(t, err)

	if assert.Len(t, result.Items, 2) {
		assert.Equal(t, "Alice", result.Items[0].Data.Name)
		assert.Equal(t, "Jane", result.Items[1].Data.Name)
	}
}
