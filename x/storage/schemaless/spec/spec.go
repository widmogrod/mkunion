// Package spec defines a behavioural specification for schemaless.Repository
// implementations. Every storage backend (in-memory, DynamoDB, OpenSearch, ...)
// is expected to pass the same suite, modulo an explicit set of Capabilities.
//
// The in-memory repository defines the full contract: FullCapabilities().
// A backend that cannot provide a behaviour declares the downgrade explicitly:
//
//	spec.RunRepositorySpec(t, newRepo,
//		spec.FullCapabilities().
//			WithoutSortByDataField().
//			WithoutBackwardPagination(),
//	)
//
// Downgraded behaviours show up as skipped subtests, so the capability matrix
// stays visible in test output instead of silently shrinking the contract.
package spec

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/predicate"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
)

// Capabilities enumerates the optional behaviours of a Repository.
// The zero value means "no optional behaviour"; FullCapabilities() is the
// complete contract as implemented by the in-memory repository.
type Capabilities struct {
	// SortByDataField: FindingRecords honours Sort on arbitrary record fields
	// (e.g. "Data.Name"). DynamoDB cannot sort on non-key attributes.
	SortByDataField bool
	// BackwardPagination: FindingRecords supports the Before cursor and
	// returns Prev links, allowing paging backward through results.
	// Requires SortByDataField (stable order) to be meaningful.
	BackwardPagination bool
	// AtomicBatch: UpdateRecords batches are all-or-nothing; on error nothing
	// is written. OpenSearch is atomic per record only.
	AtomicBatch bool
}

// FullCapabilities is the complete Repository contract, as implemented by the
// in-memory repository.
func FullCapabilities() Capabilities {
	return Capabilities{
		SortByDataField:    true,
		BackwardPagination: true,
		AtomicBatch:        true,
	}
}

func (c Capabilities) WithoutSortByDataField() Capabilities {
	c.SortByDataField = false
	return c
}

func (c Capabilities) WithoutBackwardPagination() Capabilities {
	c.BackwardPagination = false
	return c
}

func (c Capabilities) WithoutAtomicBatch() Capabilities {
	c.AtomicBatch = false
	return c
}

func (c Capabilities) skipUnless(t *testing.T, enabled bool, capability string) {
	t.Helper()
	if !enabled {
		t.Skipf("capability downgrade: this storage does not support %s", capability)
	}
}

type (
	// Data is the record payload the spec suite stores and queries.
	Data = schemaless.ExampleRecord
	// Repo is the repository type the spec suite exercises.
	Repo = schemaless.Repository[Data]
	// NewRepoFunc returns a repository backed by a store that contains no
	// records for the record types the suite generates. Backends with shared
	// state (a shared index or table) are fine: the suite namespaces every
	// subtest with a unique record type.
	NewRepoFunc func(t *testing.T) Repo
)

var recordTypeCounter atomic.Uint64

// uniqueRecordType namespaces a subtest so suites can share one index/table.
func uniqueRecordType() string {
	return fmt.Sprintf("spec-%d-%d", time.Now().UnixNano(), recordTypeCounter.Add(1))
}

func seedRecords(recordType string) []schemaless.Record[Data] {
	return []schemaless.Record[Data]{
		{ID: "1", Type: recordType, Data: Data{Name: "Alice", Age: 39}},
		{ID: "2", Type: recordType, Data: Data{Name: "Bob", Age: 40}},
		{ID: "3", Type: recordType, Data: Data{Name: "Jane", Age: 30}},
		{ID: "4", Type: recordType, Data: Data{Name: "John", Age: 20}},
		{ID: "5", Type: recordType, Data: Data{Name: "Zarlie", Age: 39}},
	}
}

func mustSave(t *testing.T, repo Repo, records ...schemaless.Record[Data]) {
	t.Helper()
	result, err := repo.UpdateRecords(schemaless.Save(records...))
	require.NoError(t, err, "seeding records must succeed")
	require.Len(t, result.Saved, len(records))
}

// findAllPages follows Next cursors and returns every item, and the pages.
func findAllPages(t *testing.T, repo Repo, query schemaless.FindingRecords[schemaless.Record[Data]]) ([]schemaless.Record[Data], [][]schemaless.Record[Data]) {
	t.Helper()
	var items []schemaless.Record[Data]
	var pages [][]schemaless.Record[Data]
	for i := 0; ; i++ {
		require.Less(t, i, 100, "pagination must terminate")
		page, err := repo.FindingRecords(query)
		require.NoError(t, err, "finding records must succeed")
		items = append(items, page.Items...)
		pages = append(pages, page.Items)
		if !page.HasNext() {
			return items, pages
		}
		query = *page.Next
	}
}

func names(items []schemaless.Record[Data]) []string {
	result := make([]string, len(items))
	for i, item := range items {
		result[i] = item.Data.Name
	}
	return result
}

func ids(items []schemaless.Record[Data]) []string {
	result := make([]string, len(items))
	for i, item := range items {
		result[i] = item.ID
	}
	return result
}

func sortByName(descending bool) []schemaless.SortField {
	return []schemaless.SortField{{Field: "Data.Name", Descending: descending}}
}

// RunRepositorySpec runs the behavioural specification against a repository.
// Behaviours excluded by caps are reported as skipped subtests.
func RunRepositorySpec(t *testing.T, newRepo NewRepoFunc, caps Capabilities) {
	t.Run("get of a missing record returns ErrNotFound", func(t *testing.T) {
		repo := newRepo(t)
		_, err := repo.Get("does-not-exist", uniqueRecordType())
		assert.ErrorIs(t, err, schemaless.ErrNotFound)
	})

	t.Run("empty update command returns ErrEmptyCommand", func(t *testing.T) {
		repo := newRepo(t)
		_, err := repo.UpdateRecords(schemaless.UpdateRecords[schemaless.Record[Data]]{})
		assert.ErrorIs(t, err, schemaless.ErrEmptyCommand)
	})

	t.Run("saved record can be read back with its data", func(t *testing.T) {
		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSave(t, repo, schemaless.Record[Data]{
			ID: "1", Type: recordType, Data: Data{Name: "Alice", Age: 39},
		})

		got, err := repo.Get("1", recordType)
		require.NoError(t, err)
		assert.Equal(t, "1", got.ID)
		assert.Equal(t, recordType, got.Type)
		assert.Equal(t, Data{Name: "Alice", Age: 39}, got.Data)
		assert.GreaterOrEqual(t, got.Version, uint16(1), "a stored record has a version")
	})

	t.Run("update with the current version succeeds and bumps the version", func(t *testing.T) {
		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSave(t, repo, schemaless.Record[Data]{
			ID: "1", Type: recordType, Data: Data{Name: "Alice", Age: 39},
		})

		current, err := repo.Get("1", recordType)
		require.NoError(t, err)

		current.Data.Age = 40
		mustSave(t, repo, current)

		got, err := repo.Get("1", recordType)
		require.NoError(t, err)
		assert.Equal(t, 40, got.Data.Age)
		assert.Greater(t, got.Version, current.Version, "version must increase on update")
	})

	t.Run("write with a stale version fails with ErrVersionConflict", func(t *testing.T) {
		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSave(t, repo, schemaless.Record[Data]{
			ID: "1", Type: recordType, Data: Data{Name: "Alice", Age: 39},
		})

		current, err := repo.Get("1", recordType)
		require.NoError(t, err)

		// move the server ahead of the client's copy
		mustSave(t, repo, current)

		stale := current
		stale.Data.Age = 100
		_, err = repo.UpdateRecords(schemaless.Save(stale))
		assert.ErrorIs(t, err, schemaless.ErrVersionConflict)

		got, err := repo.Get("1", recordType)
		require.NoError(t, err)
		assert.Equal(t, 39, got.Data.Age, "a stale write must not change the record")
	})

	t.Run("PolicyOverwriteServerChanges wins over a stale version", func(t *testing.T) {
		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSave(t, repo, schemaless.Record[Data]{
			ID: "1", Type: recordType, Data: Data{Name: "Alice", Age: 39},
		})

		current, err := repo.Get("1", recordType)
		require.NoError(t, err)

		// move the server ahead of the client's copy
		mustSave(t, repo, current)

		stale := current
		stale.Data.Age = 100
		command := schemaless.Save(stale)
		command.UpdatingPolicy = schemaless.PolicyOverwriteServerChanges
		_, err = repo.UpdateRecords(command)
		require.NoError(t, err, "overwrite policy must accept a stale version")

		got, err := repo.Get("1", recordType)
		require.NoError(t, err)
		assert.Equal(t, 100, got.Data.Age)
	})

	t.Run("deleted record is gone", func(t *testing.T) {
		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSave(t, repo, schemaless.Record[Data]{
			ID: "1", Type: recordType, Data: Data{Name: "Alice", Age: 39},
		})

		got, err := repo.Get("1", recordType)
		require.NoError(t, err)

		result, err := repo.UpdateRecords(schemaless.Delete(got))
		require.NoError(t, err)
		assert.Len(t, result.Deleted, 1)

		_, err = repo.Get("1", recordType)
		assert.ErrorIs(t, err, schemaless.ErrNotFound)
	})

	t.Run("where predicate filters records", func(t *testing.T) {
		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSave(t, repo, seedRecords(recordType)...)

		// a page may hold fewer matches than the limit (DynamoDB filters after
		// scanning a page), so follow Next cursors to collect every match
		items, _ := findAllPages(t, repo, schemaless.FindingRecords[schemaless.Record[Data]]{
			RecordType: recordType,
			Where: predicate.MustWhere(
				"Data.Age > :age AND Data.Age < :maxAge",
				predicate.ParamBinds{
					":age":    schema.MkInt(20),
					":maxAge": schema.MkInt(40),
				},
				nil,
			),
			Limit: 10,
		})
		assert.ElementsMatch(t, []string{"Alice", "Jane", "Zarlie"}, names(items))
	})

	t.Run("record type separates records", func(t *testing.T) {
		repo := newRepo(t)
		recordTypeA := uniqueRecordType()
		recordTypeB := uniqueRecordType()
		// distinct IDs across types: DynamoDB and OpenSearch key result.Saved
		// by ID alone, so a shared ID would collide in the result map
		mustSave(t, repo,
			schemaless.Record[Data]{ID: "1", Type: recordTypeA, Data: Data{Name: "Alice", Age: 39}},
			schemaless.Record[Data]{ID: "2", Type: recordTypeA, Data: Data{Name: "Bob", Age: 40}},
			schemaless.Record[Data]{ID: "3", Type: recordTypeB, Data: Data{Name: "Jane", Age: 30}},
		)

		items, _ := findAllPages(t, repo, schemaless.FindingRecords[schemaless.Record[Data]]{
			RecordType: recordTypeA,
			Limit:      10,
		})
		assert.ElementsMatch(t, []string{"Alice", "Bob"}, names(items))
	})

	t.Run("forward pagination visits every record exactly once", func(t *testing.T) {
		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSave(t, repo, seedRecords(recordType)...)

		query := schemaless.FindingRecords[schemaless.Record[Data]]{
			RecordType: recordType,
			Limit:      2,
		}
		if caps.SortByDataField {
			// backends without stable natural order need a sort to paginate
			// deterministically; backends without sorting have a stable scan order
			query.Sort = sortByName(false)
		}

		items, pages := findAllPages(t, repo, query)
		assert.ElementsMatch(t, []string{"1", "2", "3", "4", "5"}, ids(items),
			"no record may be lost or duplicated across pages")
		for _, page := range pages {
			assert.LessOrEqual(t, len(page), 2, "no page may exceed the limit")
		}
	})

	t.Run("sorting orders records by a data field", func(t *testing.T) {
		caps.skipUnless(t, caps.SortByDataField, "sorting by data fields (SortByDataField)")

		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSave(t, repo, seedRecords(recordType)...)

		ascending, err := repo.FindingRecords(schemaless.FindingRecords[schemaless.Record[Data]]{
			RecordType: recordType,
			Sort:       sortByName(false),
			Limit:      10,
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"Alice", "Bob", "Jane", "John", "Zarlie"}, names(ascending.Items))

		descending, err := repo.FindingRecords(schemaless.FindingRecords[schemaless.Record[Data]]{
			RecordType: recordType,
			Sort:       sortByName(true),
			Limit:      10,
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"Zarlie", "John", "Jane", "Bob", "Alice"}, names(descending.Items))
	})

	t.Run("sorted pagination keeps order across pages", func(t *testing.T) {
		caps.skipUnless(t, caps.SortByDataField, "sorting by data fields (SortByDataField)")

		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSave(t, repo, seedRecords(recordType)...)

		items, _ := findAllPages(t, repo, schemaless.FindingRecords[schemaless.Record[Data]]{
			RecordType: recordType,
			Sort:       sortByName(false),
			Limit:      2,
		})
		assert.Equal(t, []string{"Alice", "Bob", "Jane", "John", "Zarlie"}, names(items))
	})

	t.Run("prev cursor pages backward", func(t *testing.T) {
		caps.skipUnless(t, caps.BackwardPagination, "backward pagination (BackwardPagination)")

		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSave(t, repo, seedRecords(recordType)...)

		firstPage, err := repo.FindingRecords(schemaless.FindingRecords[schemaless.Record[Data]]{
			RecordType: recordType,
			Sort:       sortByName(false),
			Limit:      2,
		})
		require.NoError(t, err)
		require.Equal(t, []string{"Alice", "Bob"}, names(firstPage.Items))
		require.True(t, firstPage.HasNext())

		secondPage, err := repo.FindingRecords(*firstPage.Next)
		require.NoError(t, err)
		require.Equal(t, []string{"Jane", "John"}, names(secondPage.Items))
		require.True(t, secondPage.HasPrev(), "a non-first page must link backward")

		backToFirst, err := repo.FindingRecords(*secondPage.Prev)
		require.NoError(t, err)
		assert.Equal(t, []string{"Alice", "Bob"}, names(backToFirst.Items))
	})

	t.Run("batch with a conflict writes nothing", func(t *testing.T) {
		caps.skipUnless(t, caps.AtomicBatch, "all-or-nothing batches (AtomicBatch)")

		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSave(t, repo, schemaless.Record[Data]{
			ID: "1", Type: recordType, Data: Data{Name: "Alice", Age: 39},
		})

		current, err := repo.Get("1", recordType)
		require.NoError(t, err)

		// move the server ahead so `current` is stale
		mustSave(t, repo, current)

		stale := current
		stale.Data.Age = 100
		fresh := schemaless.Record[Data]{ID: "2", Type: recordType, Data: Data{Name: "Bob", Age: 40}}

		_, err = repo.UpdateRecords(schemaless.Save(stale, fresh))
		assert.ErrorIs(t, err, schemaless.ErrVersionConflict)

		_, err = repo.Get("2", recordType)
		assert.ErrorIs(t, err, schemaless.ErrNotFound,
			"the conflict-free record must not be written when the batch fails")

		got, err := repo.Get("1", recordType)
		require.NoError(t, err)
		assert.Equal(t, 39, got.Data.Age, "the stale record must keep its server state")
	})
}
