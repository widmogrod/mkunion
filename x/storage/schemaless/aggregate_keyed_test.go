package schemaless

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/schema"
)

func newCountByNameAggregate(t *testing.T) *KeyedAggregate[ExampleRecord, ExampleRecord] {
	t.Helper()
	storage := NewInMemoryRepository[schema.Schema]()
	return NewKeyedAggregate[ExampleRecord, ExampleRecord](
		"count-by-name",
		[]string{"users"},
		func(data ExampleRecord) (string, ExampleRecord) {
			return data.Name, ExampleRecord{Age: 1}
		},
		func(a, b ExampleRecord) (ExampleRecord, error) {
			return ExampleRecord{Age: a.Age + b.Age}, nil
		},
		func(a, b ExampleRecord) (ExampleRecord, error) {
			return ExampleRecord{Age: a.Age - b.Age}, nil
		},
		storage,
	)
}

func TestKeyedAggregateDeleteOnUnseenIndexDoesNotSeedIt(t *testing.T) {
	agg := newCountByNameAggregate(t)

	// deleting a record whose index was never built and is not in storage
	// must not create the index, and especially must not credit it with
	// the deleted record's contribution as if it was appended
	err := agg.Delete(Record[ExampleRecord]{
		ID: "1", Type: "users", Data: ExampleRecord{Name: "alice"},
	})
	assert.NoError(t, err)

	indices := agg.GetVersionedIndices()
	assert.Empty(t, indices,
		"a delete of an untracked index must not seed that index")
}

func TestKeyedAggregateDeleteUncombines(t *testing.T) {
	agg := newCountByNameAggregate(t)

	first := Record[ExampleRecord]{ID: "1", Type: "users", Data: ExampleRecord{Name: "alice"}}
	second := Record[ExampleRecord]{ID: "2", Type: "users", Data: ExampleRecord{Name: "alice"}}

	assert.NoError(t, agg.Append(first))
	assert.NoError(t, agg.Append(second))
	assert.Equal(t, 2, agg.GetIndexByKey("alice").Age)

	assert.NoError(t, agg.Delete(second))
	assert.Equal(t, 1, agg.GetIndexByKey("alice").Age,
		"deleting a record must subtract its contribution from the index")
}

func TestInMemoryPaginationIsDeterministicWithoutSort(t *testing.T) {
	repo := NewInMemoryRepository[ExampleRecord]()

	var want []string
	saving := UpdateRecords[Record[ExampleRecord]]{
		Saving:         map[string]Record[ExampleRecord]{},
		UpdatingPolicy: PolicyIfServerNotChanged,
	}
	for i := 0; i < 20; i++ {
		id := string(rune('a' + i))
		want = append(want, id)
		saving.Saving[id] = Record[ExampleRecord]{
			ID: id, Type: "users", Data: ExampleRecord{Name: id, Age: i},
		}
	}
	_, err := repo.UpdateRecords(saving)
	assert.NoError(t, err)

	// no Sort given: cursor pagination must still visit every record
	// exactly once, which requires a stable scan order
	var got []string
	query := FindingRecords[Record[ExampleRecord]]{
		RecordType: "users",
		Limit:      3,
	}
	for i := 0; ; i++ {
		if !assert.Less(t, i, 100, "pagination must terminate") {
			return
		}
		page, err := repo.FindingRecords(query)
		assert.NoError(t, err)
		for _, item := range page.Items {
			got = append(got, item.ID)
		}
		if !page.HasNext() {
			break
		}
		query = *page.Next
	}

	assert.Equal(t, want, got,
		"without an explicit sort, records must come back in stable ID order, none lost or duplicated")
}
