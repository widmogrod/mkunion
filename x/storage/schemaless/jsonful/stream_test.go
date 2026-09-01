package jsonful

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/predicate"
	"github.com/widmogrod/mkunion/x/storage/predicate/testutil"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
)

// collectChanges saves records, closes the log, and returns every change the
// filter lets through; a closed log replays and then ends the subscription.
func collectChanges(t *testing.T, repo *InMemoryRepository[testutil.SampleStruct], filter *predicate.WherePredicates) []schemaless.Change[testutil.SampleStruct] {
	t.Helper()
	log := repo.AppendLog()
	log.Close()

	var got []schemaless.Change[testutil.SampleStruct]
	err := log.Subscribe(context.Background(), 0, filter, func(change schemaless.Change[testutil.SampleStruct]) {
		got = append(got, change)
	})
	require.NoError(t, err)
	return got
}

func TestAppendLog_SubscribeNoFilter(t *testing.T) {
	repo := newRepo(t)
	got := collectChanges(t, repo, nil)
	assert.Len(t, got, 5)
}

func TestAppendLog_SubscribeFilter(t *testing.T) {
	repo := newRepo(t)
	filter := predicate.MustWhere("Data.Age >= :n", predicate.ParamBinds{
		":n": schema.MkInt(39),
	}, nil)

	got := collectChanges(t, repo, filter)
	require.Len(t, got, 3)
	for _, change := range got {
		assert.GreaterOrEqual(t, change.After.Data.Age, 39)
	}
}

func TestAppendLog_SubscribeFilterUnionField(t *testing.T) {
	repo := newRepo(t)
	filter := predicate.MustWhere("Data.Tree.Name = :name", predicate.ParamBinds{
		":name": schema.MkString("b3"),
	}, nil)

	got := collectChanges(t, repo, filter)
	require.Len(t, got, 1)
	assert.Equal(t, "3", got[0].After.ID)
}

func TestAppendLog_SubscribeDeletePassesFilter(t *testing.T) {
	repo := newRepo(t)
	record, err := repo.Get("2", "sample")
	require.NoError(t, err)
	_, err = repo.UpdateRecords(schemaless.Delete(record))
	require.NoError(t, err)

	// a delete change has After == nil; the filter cannot apply, so the
	// change is delivered for the subscriber to decide
	filter := predicate.MustWhere("Data.Age = :n", predicate.ParamBinds{
		":n": schema.MkInt(-1),
	}, nil)
	got := collectChanges(t, repo, filter)
	require.Len(t, got, 1)
	assert.True(t, got[0].Deleted)
}

func TestAppendLog_SubscribeBadFilterFailsUpFront(t *testing.T) {
	repo := newRepo(t)
	log := repo.AppendLog()

	bad := []*predicate.WherePredicates{
		predicate.MustWhere("Data.Agee = :n", predicate.ParamBinds{":n": schema.MkInt(1)}, nil),
		predicate.MustWhere("Data.Age = :n AND Data.Nope = :n", predicate.ParamBinds{":n": schema.MkInt(1)}, nil),
		predicate.MustWhere("Data.Age = :n OR Data.Nope = :n", predicate.ParamBinds{":n": schema.MkInt(1)}, nil),
		predicate.MustWhere("NOT Data.Nope = :n", predicate.ParamBinds{":n": schema.MkInt(1)}, nil),
		{Predicate: &predicate.Compare{
			Location:  "Data.Age",
			Operation: "=",
			BindValue: &predicate.Locatable{Location: "Data.Nope"},
		}},
	}
	for _, filter := range bad {
		err := log.Subscribe(context.Background(), 0, filter, func(schemaless.Change[testutil.SampleStruct]) {
			t.Fatal("must not deliver")
		})
		assert.Error(t, err)
	}
}

func TestAppendLog_SubscribeLocatableFilterValidates(t *testing.T) {
	repo := newRepo(t)
	log := repo.AppendLog()
	log.Close()

	filter := &predicate.WherePredicates{Predicate: &predicate.Compare{
		Location:  "Data.Age",
		Operation: "=",
		BindValue: &predicate.Locatable{Location: "Data.Age"},
	}}
	count := 0
	err := log.Subscribe(context.Background(), 0, filter, func(schemaless.Change[testutil.SampleStruct]) {
		count++
	})
	require.NoError(t, err)
	assert.Equal(t, 5, count)
}

func TestAppendLog_DelegatesWrites(t *testing.T) {
	repo, err := NewInMemoryRepository[testutil.SampleStruct]()
	require.NoError(t, err)
	log := repo.AppendLog()

	rec := record("x1", 10, &testutil.Branch{Name: "n"})
	rec.Version = 1
	require.NoError(t, log.Change(nil, &rec))
	require.NoError(t, log.Delete(rec))
	log.Push(schemaless.Change[testutil.SampleStruct]{After: &rec})

	other := schemaless.NewAppendLog[testutil.SampleStruct](nil)
	require.NoError(t, other.Change(nil, &rec))
	log.Append(other)

	log.Close()
	count := 0
	err = log.Subscribe(context.Background(), 0, nil, func(schemaless.Change[testutil.SampleStruct]) {
		count++
	})
	require.NoError(t, err)
	assert.Equal(t, 4, count)
}
