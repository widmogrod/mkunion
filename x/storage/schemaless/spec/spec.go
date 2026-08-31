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
	// MonotonicOverwriteVersion: PolicyOverwriteServerChanges bumps the
	// version from the server's copy, so versions keep increasing no matter
	// how stale the writer was. OpenSearch writes the stale writer's
	// version+1 instead, so an overwrite can move the version backward.
	MonotonicOverwriteVersion bool
}

// FullCapabilities is the complete Repository contract, as implemented by the
// in-memory repository.
func FullCapabilities() Capabilities {
	return Capabilities{
		SortByDataField:           true,
		BackwardPagination:        true,
		AtomicBatch:               true,
		MonotonicOverwriteVersion: true,
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

func (c Capabilities) WithoutMonotonicOverwriteVersion() Capabilities {
	c.MonotonicOverwriteVersion = false
	return c
}

func (c Capabilities) skipUnless(t *testing.T, enabled bool, capability string) {
	t.Helper()
	if !enabled {
		t.Skipf("capability downgrade: this storage does not support %s", capability)
	}
}

// NewRepoFunc returns a repository backed by a store that contains no
// records for the record types the suite generates. Backends with shared
// state (a shared index or table) are fine: the suite namespaces every
// subtest with a unique record type.
//
// The signature names schemaless.ExampleRecord directly on purpose: a
// spec-owned alias here would make the backend package's generated type
// registry import this package back, creating an import cycle.
type NewRepoFunc func(t *testing.T) schemaless.Repository[schemaless.ExampleRecord]

var recordTypeCounter atomic.Uint64

// uniqueRecordType namespaces a subtest so suites can share one index/table.
func uniqueRecordType() string {
	return fmt.Sprintf("spec-%d-%d", time.Now().UnixNano(), recordTypeCounter.Add(1))
}

func seedRecords(recordType string) []schemaless.Record[schemaless.ExampleRecord] {
	return []schemaless.Record[schemaless.ExampleRecord]{
		{ID: "1", Type: recordType, Data: schemaless.ExampleRecord{Name: "Alice", Age: 39}},
		{ID: "2", Type: recordType, Data: schemaless.ExampleRecord{Name: "Bob", Age: 40}},
		{ID: "3", Type: recordType, Data: schemaless.ExampleRecord{Name: "Jane", Age: 30}},
		{ID: "4", Type: recordType, Data: schemaless.ExampleRecord{Name: "John", Age: 20}},
		{ID: "5", Type: recordType, Data: schemaless.ExampleRecord{Name: "Zarlie", Age: 39}},
	}
}

func mustSave(t *testing.T, repo schemaless.Repository[schemaless.ExampleRecord], records ...schemaless.Record[schemaless.ExampleRecord]) {
	t.Helper()
	result, err := repo.UpdateRecords(schemaless.Save(records...))
	require.NoError(t, err, "seeding records must succeed")
	require.Len(t, result.Saved, len(records))
}

// findAllPages follows Next cursors and returns every item, and the pages.
func findAllPages(t *testing.T, repo schemaless.Repository[schemaless.ExampleRecord], query schemaless.FindingRecords[schemaless.Record[schemaless.ExampleRecord]]) ([]schemaless.Record[schemaless.ExampleRecord], [][]schemaless.Record[schemaless.ExampleRecord]) {
	t.Helper()
	var items []schemaless.Record[schemaless.ExampleRecord]
	var pages [][]schemaless.Record[schemaless.ExampleRecord]
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

func names(items []schemaless.Record[schemaless.ExampleRecord]) []string {
	result := make([]string, len(items))
	for i, item := range items {
		result[i] = item.Data.Name
	}
	return result
}

func ids(items []schemaless.Record[schemaless.ExampleRecord]) []string {
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

	t.Run("get with the wrong record type returns ErrNotFound", func(t *testing.T) {
		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSave(t, repo, schemaless.Record[schemaless.ExampleRecord]{
			ID: "1", Type: recordType, Data: schemaless.ExampleRecord{Name: "Alice", Age: 39},
		})

		_, err := repo.Get("1", uniqueRecordType())
		assert.ErrorIs(t, err, schemaless.ErrNotFound,
			"a record is addressed by ID and type together")
	})

	t.Run("delete of a missing record is not an error", func(t *testing.T) {
		repo := newRepo(t)
		recordType := uniqueRecordType()

		_, err := repo.UpdateRecords(schemaless.Delete(schemaless.Record[schemaless.ExampleRecord]{
			ID: "ghost", Type: recordType,
		}))
		assert.NoError(t, err, "deleting what is already gone is a no-op")
	})

	t.Run("empty update command returns ErrEmptyCommand", func(t *testing.T) {
		repo := newRepo(t)
		_, err := repo.UpdateRecords(schemaless.UpdateRecords[schemaless.Record[schemaless.ExampleRecord]]{})
		assert.ErrorIs(t, err, schemaless.ErrEmptyCommand)
	})

	t.Run("saved record can be read back with its data", func(t *testing.T) {
		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSave(t, repo, schemaless.Record[schemaless.ExampleRecord]{
			ID: "1", Type: recordType, Data: schemaless.ExampleRecord{Name: "Alice", Age: 39},
		})

		got, err := repo.Get("1", recordType)
		require.NoError(t, err)
		assert.Equal(t, "1", got.ID)
		assert.Equal(t, recordType, got.Type)
		assert.Equal(t, schemaless.ExampleRecord{Name: "Alice", Age: 39}, got.Data)
		assert.GreaterOrEqual(t, got.Version, uint16(1), "a stored record has a version")
	})

	t.Run("update with the current version succeeds and bumps the version", func(t *testing.T) {
		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSave(t, repo, schemaless.Record[schemaless.ExampleRecord]{
			ID: "1", Type: recordType, Data: schemaless.ExampleRecord{Name: "Alice", Age: 39},
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
		mustSave(t, repo, schemaless.Record[schemaless.ExampleRecord]{
			ID: "1", Type: recordType, Data: schemaless.ExampleRecord{Name: "Alice", Age: 39},
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
		mustSave(t, repo, schemaless.Record[schemaless.ExampleRecord]{
			ID: "1", Type: recordType, Data: schemaless.ExampleRecord{Name: "Alice", Age: 39},
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

	t.Run("overwrites keep the version increasing", func(t *testing.T) {
		caps.skipUnless(t, caps.MonotonicOverwriteVersion,
			"monotonic versions under overwrites (MonotonicOverwriteVersion)")

		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSave(t, repo, schemaless.Record[schemaless.ExampleRecord]{
			ID: "1", Type: recordType, Data: schemaless.ExampleRecord{Name: "Alice", Age: 39},
		})

		stale, err := repo.Get("1", recordType)
		require.NoError(t, err)

		// a concurrent writer moves the server ahead; `stale` stays behind
		mustSave(t, repo, stale)
		server, err := repo.Get("1", recordType)
		require.NoError(t, err)

		overwrite := func(age int) schemaless.Record[schemaless.ExampleRecord] {
			record := stale
			record.Data.Age = age
			command := schemaless.Save(record)
			command.UpdatingPolicy = schemaless.PolicyOverwriteServerChanges
			_, err := repo.UpdateRecords(command)
			require.NoError(t, err)
			got, err := repo.Get("1", recordType)
			require.NoError(t, err)
			return got
		}

		first := overwrite(50)
		assert.Equal(t, server.Version+1, first.Version,
			"an overwrite bumps the server version by one, however stale the writer")

		second := overwrite(60)
		assert.Equal(t, server.Version+2, second.Version,
			"repeated stale overwrites keep the version growing")
		assert.Equal(t, 60, second.Data.Age)
	})

	t.Run("deleted record is gone", func(t *testing.T) {
		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSave(t, repo, schemaless.Record[schemaless.ExampleRecord]{
			ID: "1", Type: recordType, Data: schemaless.ExampleRecord{Name: "Alice", Age: 39},
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
		items, _ := findAllPages(t, repo, schemaless.FindingRecords[schemaless.Record[schemaless.ExampleRecord]]{
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

	t.Run("OR predicate matches either branch", func(t *testing.T) {
		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSave(t, repo, seedRecords(recordType)...)

		items, _ := findAllPages(t, repo, schemaless.FindingRecords[schemaless.Record[schemaless.ExampleRecord]]{
			RecordType: recordType,
			Where: predicate.MustWhere(
				"Data.Name = :a OR Data.Age > :age",
				predicate.ParamBinds{
					":a":   schema.MkString("John"),
					":age": schema.MkInt(39),
				},
				nil,
			),
			Limit: 10,
		})
		assert.ElementsMatch(t, []string{"John", "Bob"}, names(items))
	})

	t.Run("NOT over OR excludes both branches", func(t *testing.T) {
		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSave(t, repo, seedRecords(recordType)...)

		// the string syntax has no parentheses, so NOT (a OR b) is built
		// from predicate values directly
		items, _ := findAllPages(t, repo, schemaless.FindingRecords[schemaless.Record[schemaless.ExampleRecord]]{
			RecordType: recordType,
			Where: &predicate.WherePredicates{
				Predicate: &predicate.Not{
					P: &predicate.Or{L: []predicate.Predicate{
						&predicate.Compare{Location: "ID", Operation: "=", BindValue: &predicate.BindValue{BindName: ":a"}},
						&predicate.Compare{Location: "ID", Operation: "=", BindValue: &predicate.BindValue{BindName: ":b"}},
					}},
				},
				Params: predicate.ParamBinds{
					":a": schema.MkString("1"),
					":b": schema.MkString("2"),
				},
			},
			Limit: 10,
		})
		assert.ElementsMatch(t, []string{"3", "4", "5"}, ids(items),
			"NOT (a OR b) must exclude records matching either branch")
	})

	t.Run("query without a record type is not an error", func(t *testing.T) {
		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSave(t, repo, seedRecords(recordType)...)

		// the backing store may be shared, so only a subset check is safe:
		// every seeded record must come back among the results
		items, _ := findAllPages(t, repo, schemaless.FindingRecords[schemaless.Record[schemaless.ExampleRecord]]{
			Limit: 100,
		})
		seen := map[string]bool{}
		for _, item := range items {
			if item.Type == recordType {
				seen[item.ID] = true
			}
		}
		assert.Len(t, seen, 5, "an unfiltered query must include records of every type")
	})

	t.Run("record type separates records", func(t *testing.T) {
		repo := newRepo(t)
		recordTypeA := uniqueRecordType()
		recordTypeB := uniqueRecordType()
		// distinct IDs across types: DynamoDB and OpenSearch key result.Saved
		// by ID alone, so a shared ID would collide in the result map
		mustSave(t, repo,
			schemaless.Record[schemaless.ExampleRecord]{ID: "1", Type: recordTypeA, Data: schemaless.ExampleRecord{Name: "Alice", Age: 39}},
			schemaless.Record[schemaless.ExampleRecord]{ID: "2", Type: recordTypeA, Data: schemaless.ExampleRecord{Name: "Bob", Age: 40}},
			schemaless.Record[schemaless.ExampleRecord]{ID: "3", Type: recordTypeB, Data: schemaless.ExampleRecord{Name: "Jane", Age: 30}},
		)

		items, _ := findAllPages(t, repo, schemaless.FindingRecords[schemaless.Record[schemaless.ExampleRecord]]{
			RecordType: recordTypeA,
			Limit:      10,
		})
		assert.ElementsMatch(t, []string{"Alice", "Bob"}, names(items))
	})

	t.Run("forward pagination visits every record exactly once", func(t *testing.T) {
		repo := newRepo(t)
		recordType := uniqueRecordType()
		mustSave(t, repo, seedRecords(recordType)...)

		query := schemaless.FindingRecords[schemaless.Record[schemaless.ExampleRecord]]{
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

		ascending, err := repo.FindingRecords(schemaless.FindingRecords[schemaless.Record[schemaless.ExampleRecord]]{
			RecordType: recordType,
			Sort:       sortByName(false),
			Limit:      10,
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"Alice", "Bob", "Jane", "John", "Zarlie"}, names(ascending.Items))

		descending, err := repo.FindingRecords(schemaless.FindingRecords[schemaless.Record[schemaless.ExampleRecord]]{
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

		items, _ := findAllPages(t, repo, schemaless.FindingRecords[schemaless.Record[schemaless.ExampleRecord]]{
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

		firstPage, err := repo.FindingRecords(schemaless.FindingRecords[schemaless.Record[schemaless.ExampleRecord]]{
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
		mustSave(t, repo, schemaless.Record[schemaless.ExampleRecord]{
			ID: "1", Type: recordType, Data: schemaless.ExampleRecord{Name: "Alice", Age: 39},
		})

		current, err := repo.Get("1", recordType)
		require.NoError(t, err)

		// move the server ahead so `current` is stale
		mustSave(t, repo, current)

		stale := current
		stale.Data.Age = 100
		fresh := schemaless.Record[schemaless.ExampleRecord]{ID: "2", Type: recordType, Data: schemaless.ExampleRecord{Name: "Bob", Age: 40}}

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
