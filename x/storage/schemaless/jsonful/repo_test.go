package jsonful

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/predicate"
	"github.com/widmogrod/mkunion/x/storage/predicate/testutil"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
)

func record(id string, age int, tree testutil.Treeish) schemaless.Record[testutil.SampleStruct] {
	return schemaless.Record[testutil.SampleStruct]{
		ID:   id,
		Type: "sample",
		Data: testutil.SampleStruct{
			ID:   id,
			Age:  age,
			Tree: tree,
		},
	}
}

func newRepo(t *testing.T) *InMemoryRepository[testutil.SampleStruct] {
	t.Helper()
	repo, err := NewInMemoryRepository[testutil.SampleStruct]()
	require.NoError(t, err)

	_, err = repo.UpdateRecords(schemaless.Save(
		record("1", 20, &testutil.Branch{Name: "b1"}),
		record("2", 30, &testutil.Leaf{Value: schema.MkString("v2")}),
		record("3", 39, &testutil.Branch{Name: "b3"}),
		record("4", 40, &testutil.Leaf{Value: schema.MkString("v4")}),
		record("5", 39, &testutil.Branch{Name: "b5"}),
	))
	require.NoError(t, err)
	return repo
}

func find(t *testing.T, repo *InMemoryRepository[testutil.SampleStruct], where string, params predicate.ParamBinds) []string {
	t.Helper()
	query := schemaless.FindingRecords[schemaless.Record[testutil.SampleStruct]]{
		RecordType: "sample",
	}
	if where != "" {
		query.Where = predicate.MustWhere(where, params, nil)
	}
	result, err := repo.FindingRecords(query)
	require.NoError(t, err)

	ids := make([]string, 0, len(result.Items))
	for _, item := range result.Items {
		ids = append(ids, item.ID)
	}
	return ids
}

func TestInMemoryRepository_GetIsDirect(t *testing.T) {
	repo := newRepo(t)

	found, err := repo.Get("3", "sample")
	require.NoError(t, err)
	assert.Equal(t, "3", found.Data.ID)
	assert.Equal(t, uint16(1), found.Version)

	_, err = repo.Get("nope", "sample")
	assert.ErrorIs(t, err, schemaless.ErrNotFound)
}

func TestInMemoryRepository_Queries(t *testing.T) {
	repo := newRepo(t)

	t.Run("struct field", func(t *testing.T) {
		ids := find(t, repo, "Data.Age > :n", predicate.ParamBinds{":n": schema.MkInt(35)})
		assert.ElementsMatch(t, []string{"3", "4", "5"}, ids)
	})

	t.Run("fractional compare is exact", func(t *testing.T) {
		ids := find(t, repo, "Data.Age > :n", predicate.ParamBinds{":n": schema.MkFloat(39.5)})
		assert.ElementsMatch(t, []string{"4"}, ids)
	})

	t.Run("bare union field", func(t *testing.T) {
		ids := find(t, repo, "Data.Tree.Name = :n", predicate.ParamBinds{":n": schema.MkString("b3")})
		assert.ElementsMatch(t, []string{"3"}, ids)
	})

	t.Run("union discriminator", func(t *testing.T) {
		ids := find(t, repo, `Data.Tree["$type"] = :t`, predicate.ParamBinds{":t": schema.MkString("testutil.Leaf")})
		assert.ElementsMatch(t, []string{"2", "4"}, ids)
	})

	t.Run("record metadata", func(t *testing.T) {
		ids := find(t, repo, "ID = :id", predicate.ParamBinds{":id": schema.MkString("2")})
		assert.ElementsMatch(t, []string{"2"}, ids)
	})

	t.Run("typo returns error", func(t *testing.T) {
		query := schemaless.FindingRecords[schemaless.Record[testutil.SampleStruct]]{
			RecordType: "sample",
			Where: predicate.MustWhere("Data.Agee = :n", predicate.ParamBinds{
				":n": schema.MkInt(1),
			}, nil),
		}
		_, err := repo.FindingRecords(query)
		assert.Error(t, err)
	})
}

func TestInMemoryRepository_SortAndPagination(t *testing.T) {
	repo := newRepo(t)

	query := schemaless.FindingRecords[schemaless.Record[testutil.SampleStruct]]{
		RecordType: "sample",
		Sort: []schemaless.SortField{
			{Field: "Data.Age", Descending: true},
		},
		Limit: 2,
	}

	page1, err := repo.FindingRecords(query)
	require.NoError(t, err)
	require.Len(t, page1.Items, 2)
	assert.Equal(t, "4", page1.Items[0].ID)
	assert.Equal(t, "3", page1.Items[1].ID) // ties on Age=39 break by ID
	require.True(t, page1.HasNext())
	assert.Equal(t, "sample", page1.Next.RecordType)

	page2, err := repo.FindingRecords(*page1.Next)
	require.NoError(t, err)
	require.Len(t, page2.Items, 2)
	assert.Equal(t, "5", page2.Items[0].ID)
	assert.Equal(t, "2", page2.Items[1].ID)

	page3, err := repo.FindingRecords(*page2.Next)
	require.NoError(t, err)
	require.Len(t, page3.Items, 1)
	assert.Equal(t, "1", page3.Items[0].ID)
	assert.False(t, page3.HasNext())
}

func TestInMemoryRepository_Versioning(t *testing.T) {
	repo := newRepo(t)

	t.Run("new record must start at version zero", func(t *testing.T) {
		bad := record("100", 1, &testutil.Leaf{Value: schema.MkString("x")})
		bad.Version = 900
		_, err := repo.UpdateRecords(schemaless.Save(bad))
		assert.ErrorIs(t, err, schemaless.ErrVersionConflict)
	})

	t.Run("stale update conflicts", func(t *testing.T) {
		current, err := repo.Get("1", "sample")
		require.NoError(t, err)

		stale := current
		stale.Version = current.Version - 1
		_, err = repo.UpdateRecords(schemaless.Save(stale))
		assert.ErrorIs(t, err, schemaless.ErrVersionConflict)

		// and the failed command must not have touched the store
		after, err := repo.Get("1", "sample")
		require.NoError(t, err)
		assert.Equal(t, current.Version, after.Version)
	})

	t.Run("fresh update succeeds and bumps version", func(t *testing.T) {
		current, err := repo.Get("1", "sample")
		require.NoError(t, err)

		current.Data.Age = 21
		result, err := repo.UpdateRecords(schemaless.Save(current))
		require.NoError(t, err)
		saved := result.Saved["1:sample"]
		assert.Equal(t, current.Version+1, saved.Version)

		ids := find(t, repo, "Data.Age = :n", predicate.ParamBinds{":n": schema.MkInt(21)})
		assert.ElementsMatch(t, []string{"1"}, ids)
	})

	t.Run("overwrite policy wins", func(t *testing.T) {
		current, err := repo.Get("2", "sample")
		require.NoError(t, err)

		blind := current
		blind.Version = 0
		blind.Data.Age = 99
		_, err = repo.UpdateRecords(schemaless.UpdateRecords[schemaless.Record[testutil.SampleStruct]]{
			UpdatingPolicy: schemaless.PolicyOverwriteServerChanges,
			Saving:         map[string]schemaless.Record[testutil.SampleStruct]{"2:sample": blind},
		})
		require.NoError(t, err)

		after, err := repo.Get("2", "sample")
		require.NoError(t, err)
		assert.Equal(t, 99, after.Data.Age)
		assert.Equal(t, current.Version+1, after.Version)
	})
}

func TestInMemoryRepository_BeforeCursor(t *testing.T) {
	repo := newRepo(t)

	query := schemaless.FindingRecords[schemaless.Record[testutil.SampleStruct]]{
		RecordType: "sample",
		Sort:       []schemaless.SortField{{Field: "Data.Age", Descending: false}},
		Limit:      2,
	}

	page1, err := repo.FindingRecords(query)
	require.NoError(t, err)
	require.Len(t, page1.Items, 2)

	page2, err := repo.FindingRecords(*page1.Next)
	require.NoError(t, err)
	require.Len(t, page2.Items, 2)
	require.NotNil(t, page2.Prev)

	// paging backward from page2 gives page1 again
	back, err := repo.FindingRecords(*page2.Prev)
	require.NoError(t, err)
	require.Len(t, back.Items, 2)
	assert.Equal(t, page1.Items[0].ID, back.Items[0].ID)
	assert.Equal(t, page1.Items[1].ID, back.Items[1].ID)
}

func TestInMemoryRepository_BadSortFieldErrors(t *testing.T) {
	repo := newRepo(t)

	_, err := repo.FindingRecords(schemaless.FindingRecords[schemaless.Record[testutil.SampleStruct]]{
		RecordType: "sample",
		Sort:       []schemaless.SortField{{Field: "Data.Agee"}},
	})
	assert.Error(t, err)
}

func TestInMemoryRepository_Delete(t *testing.T) {
	repo := newRepo(t)

	current, err := repo.Get("2", "sample")
	require.NoError(t, err)

	_, err = repo.UpdateRecords(schemaless.Delete(current))
	require.NoError(t, err)

	_, err = repo.Get("2", "sample")
	assert.ErrorIs(t, err, schemaless.ErrNotFound)
}
