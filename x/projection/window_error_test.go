package projection

import (
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"github.com/widmogrod/mkunion/x/stream"
)

// loadInts publishes n int records under alternating keys plus a final
// end-of-stream watermark, mirroring the happy-path setup.
func loadInts(t *testing.T, dataStream stream.Stream[schema.Schema], topic string, n int) {
	t.Helper()
	ctx := NewPushAndPullInMemoryContext[any, int](&PullPushContextState{
		PushTopic: topic,
	}, dataStream)

	err := DoLoad(ctx, func(push func(data Data[int]) error) error {
		for i := 0; i < n; i++ {
			if err := push(&Record[int]{
				Key:  fmt.Sprintf("key-%d", i%2),
				Data: i,
			}); err != nil {
				return err
			}
		}
		return push(&Watermark[int]{EventTime: math.MaxInt64})
	})
	require.NoError(t, err)
}

func windowCtx(dataStream stream.Stream[schema.Schema], pull, push string) *PushAndPullInMemoryContext[int, string] {
	return NewPushAndPullInMemoryContext[int, string](&PullPushContextState{
		PullTopic: pull,
		PushTopic: push,
	}, dataStream)
}

// failingWindowRepo fails selected operations; when real is set, the
// other operations delegate to it.
type failingWindowRepo struct {
	real      schemaless.Repository[*WindowRecord[string]]
	getErr    error
	findErr   error
	updateErr error
	deleteErr error
}

func (f *failingWindowRepo) Get(recordID string, recordType schemaless.RecordType) (schemaless.Record[*WindowRecord[string]], error) {
	if f.getErr != nil {
		return schemaless.Record[*WindowRecord[string]]{}, f.getErr
	}
	if f.real != nil {
		return f.real.Get(recordID, recordType)
	}
	return schemaless.Record[*WindowRecord[string]]{}, schemaless.ErrNotFound
}

func (f *failingWindowRepo) UpdateRecords(cmd schemaless.UpdateRecords[schemaless.Record[*WindowRecord[string]]]) (*schemaless.UpdateRecordsResult[schemaless.Record[*WindowRecord[string]]], error) {
	if len(cmd.Deleting) > 0 && f.deleteErr != nil {
		return nil, f.deleteErr
	}
	if len(cmd.Saving) > 0 && f.updateErr != nil {
		return nil, f.updateErr
	}
	if f.real != nil {
		return f.real.UpdateRecords(cmd)
	}
	return &schemaless.UpdateRecordsResult[schemaless.Record[*WindowRecord[string]]]{}, nil
}

func (f *failingWindowRepo) FindingRecords(query schemaless.FindingRecords[schemaless.Record[*WindowRecord[string]]]) (schemaless.PageResult[schemaless.Record[*WindowRecord[string]]], error) {
	if f.findErr != nil {
		return schemaless.PageResult[schemaless.Record[*WindowRecord[string]]]{}, f.findErr
	}
	if f.real != nil {
		return f.real.FindingRecords(query)
	}
	return schemaless.PageResult[schemaless.Record[*WindowRecord[string]]]{}, nil
}

func TestDoWindowErrors(t *testing.T) {
	wd := &FixedWindow{Width: math.MaxInt64}
	fm := &Discard{}
	td := &AtWatermark{}

	t.Run("merge failure on the first record of a window propagates", func(t *testing.T) {
		dataStream := stream.NewInMemoryStream[schema.Schema](stream.WithSystemTime)
		loadInts(t, dataStream, "in", 2)

		err := DoWindow[int, string](
			windowCtx(dataStream, "in", "out"),
			NewWindowInMemoryStore[string]("w"),
			wd, fm, td, "",
			func(x int, agg string) (string, error) {
				return "", errors.New("merge boom")
			}, NoSnapshot{})
		assert.ErrorContains(t, err, "merge boom")
	})

	t.Run("merge failure on a later record of the same window propagates", func(t *testing.T) {
		dataStream := stream.NewInMemoryStream[schema.Schema](stream.WithSystemTime)
		loadInts(t, dataStream, "in", 4)

		calls := map[string]int{}
		err := DoWindow[int, string](
			windowCtx(dataStream, "in", "out"),
			NewWindowInMemoryStore[string]("w"),
			wd, fm, td, "",
			func(x int, agg string) (string, error) {
				calls["all"]++
				if agg != "" {
					return "", errors.New("second merge boom")
				}
				return fmt.Sprintf("%d", x), nil
			}, NoSnapshot{})
		assert.ErrorContains(t, err, "second merge boom")
		assert.GreaterOrEqual(t, calls["all"], 2)
	})

	t.Run("window store load failure propagates", func(t *testing.T) {
		dataStream := stream.NewInMemoryStream[schema.Schema](stream.WithSystemTime)
		loadInts(t, dataStream, "in", 1)

		err := DoWindow[int, string](
			windowCtx(dataStream, "in", "out"),
			&WindowInMemoryStore[string]{store: &failingWindowRepo{getErr: errors.New("db down")}, recordType: "w"},
			wd, fm, td, "",
			func(x int, agg string) (string, error) { return agg, nil }, NoSnapshot{})
		assert.ErrorContains(t, err, "load key=")
	})

	t.Run("window store save failure propagates", func(t *testing.T) {
		dataStream := stream.NewInMemoryStream[schema.Schema](stream.WithSystemTime)
		loadInts(t, dataStream, "in", 1)

		err := DoWindow[int, string](
			windowCtx(dataStream, "in", "out"),
			&WindowInMemoryStore[string]{store: &failingWindowRepo{updateErr: errors.New("db down")}, recordType: "w"},
			wd, fm, td, "",
			func(x int, agg string) (string, error) { return agg, nil }, NoSnapshot{})
		assert.ErrorContains(t, err, "save key=")
	})

	t.Run("flush find failure propagates", func(t *testing.T) {
		dataStream := stream.NewInMemoryStream[schema.Schema](stream.WithSystemTime)
		loadInts(t, dataStream, "in", 0) // only the watermark arrives

		err := DoWindow[int, string](
			windowCtx(dataStream, "in", "out"),
			&WindowInMemoryStore[string]{store: &failingWindowRepo{findErr: errors.New("db down")}, recordType: "w"},
			wd, fm, td, "",
			func(x int, agg string) (string, error) { return agg, nil }, NoSnapshot{})
		assert.ErrorContains(t, err, "flush find")
	})

	t.Run("flush push failure propagates", func(t *testing.T) {
		dataStream := stream.NewInMemoryStream[schema.Schema](stream.WithSystemTime)
		loadInts(t, dataStream, "in", 1)

		// the record merges fine; pushing the flushed window to an empty
		// topic fails
		err := DoWindow[int, string](
			windowCtx(dataStream, "in", ""),
			NewWindowInMemoryStore[string]("w"),
			wd, fm, td, "",
			func(x int, agg string) (string, error) { return fmt.Sprintf("%d", x), nil }, NoSnapshot{})
		assert.ErrorContains(t, err, "flush push")
	})

	t.Run("flush delete failure propagates", func(t *testing.T) {
		dataStream := stream.NewInMemoryStream[schema.Schema](stream.WithSystemTime)
		loadInts(t, dataStream, "in", 1)

		repo := &failingWindowRepo{real: schemaless.NewInMemoryRepository[*WindowRecord[string]](), deleteErr: errors.New("db down")}
		err := DoWindow[int, string](
			windowCtx(dataStream, "in", "out"),
			&WindowInMemoryStore[string]{store: repo, recordType: "w"},
			wd, fm, td, "",
			func(x int, agg string) (string, error) { return fmt.Sprintf("%d", x), nil }, NoSnapshot{})
		assert.ErrorContains(t, err, "flush delete")
	})

	t.Run("pull failure propagates", func(t *testing.T) {
		dataStream := stream.NewInMemoryStream[schema.Schema](stream.WithSystemTime)

		err := DoWindow[int, string](
			windowCtx(dataStream, "", "out"), // empty pull topic cannot be pulled
			NewWindowInMemoryStore[string]("w"),
			wd, fm, td, "",
			func(x int, agg string) (string, error) { return agg, nil }, NoSnapshot{})
		assert.ErrorContains(t, err, "pull")
	})
}
