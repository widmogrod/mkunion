package spec

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/shape"
	"github.com/widmogrod/mkunion/x/storage/predicate"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
)

// AppendLogCapabilities enumerates the optional behaviours of an AppendLoger.
// The in-memory append log defines the full contract:
// FullAppendLogCapabilities().
type AppendLogCapabilities struct {
	// Filtering: Subscribe honours a where-predicate filter, and a nil
	// filter delivers every change.
	Filtering bool
	// OffsetResume: Subscribe with fromOffset > 0 resumes from that change.
	OffsetResume bool
	// Replay: a subscriber attaching after changes were appended still
	// receives all of them.
	Replay bool
	// MergeAppend: Append(b) merges another log's changes into this one.
	MergeAppend bool
}

func FullAppendLogCapabilities() AppendLogCapabilities {
	return AppendLogCapabilities{
		Filtering:    true,
		OffsetResume: true,
		Replay:       true,
		MergeAppend:  true,
	}
}

func (c AppendLogCapabilities) WithoutFiltering() AppendLogCapabilities {
	c.Filtering = false
	return c
}

func (c AppendLogCapabilities) WithoutOffsetResume() AppendLogCapabilities {
	c.OffsetResume = false
	return c
}

func (c AppendLogCapabilities) WithoutReplay() AppendLogCapabilities {
	c.Replay = false
	return c
}

func (c AppendLogCapabilities) WithoutMergeAppend() AppendLogCapabilities {
	c.MergeAppend = false
	return c
}

// NewAppendLogFunc returns a fresh, empty append log for one subtest.
type NewAppendLogFunc func(t *testing.T) schemaless.AppendLoger[schemaless.ExampleRecord]

type appendLogChange = schemaless.Change[schemaless.ExampleRecord]
type appendLogRecord = schemaless.Record[schemaless.ExampleRecord]

func appendLogSpecRecord(id, name string) appendLogRecord {
	return appendLogRecord{
		ID: id, Type: "users",
		Data: schemaless.ExampleRecord{Name: name},
	}
}

// appendLogSubscription runs Subscribe in a goroutine and hands its output
// back over channels, so tests can assert on blocking behaviour.
type appendLogSubscription struct {
	changes chan appendLogChange
	done    chan error
	cancel  context.CancelFunc
}

func subscribeAppendLog(log schemaless.AppendLoger[schemaless.ExampleRecord], fromOffset int, filter *predicate.WherePredicates) *appendLogSubscription {
	ctx, cancel := context.WithCancel(context.Background())
	s := &appendLogSubscription{
		changes: make(chan appendLogChange, 64),
		done:    make(chan error, 1),
		cancel:  cancel,
	}
	go func() {
		s.done <- log.Subscribe(ctx, fromOffset, filter, func(c appendLogChange) {
			s.changes <- c
		})
	}()
	// give the subscriber a moment to attach; implementations without
	// replay would otherwise miss changes pushed right after this call
	time.Sleep(50 * time.Millisecond)
	return s
}

func (s *appendLogSubscription) collect(t *testing.T, n int) []appendLogChange {
	t.Helper()
	var result []appendLogChange
	for i := 0; i < n; i++ {
		select {
		case c := <-s.changes:
			result = append(result, c)
		case <-time.After(2 * time.Second):
			t.Fatalf("expected %d changes, got %d before timing out", n, len(result))
		}
	}
	return result
}

func (s *appendLogSubscription) expectNoMore(t *testing.T) {
	t.Helper()
	select {
	case c := <-s.changes:
		t.Fatalf("expected no more changes, got %+v", c)
	case <-time.After(200 * time.Millisecond):
		// quiet, as expected
	}
}

// end cancels the subscription and waits for Subscribe to return.
// nil is accepted alongside context.Canceled: a log closed mid-test makes
// Subscribe return before the cancellation is observed.
func (s *appendLogSubscription) end(t *testing.T) {
	t.Helper()
	s.cancel()
	select {
	case err := <-s.done:
		if err != nil {
			assert.ErrorIs(t, err, context.Canceled)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Subscribe did not return after cancellation")
	}
}

func changeNames(changes []appendLogChange) []string {
	result := make([]string, len(changes))
	for i, c := range changes {
		record := c.After
		if record == nil {
			record = c.Before
		}
		result[i] = record.Data.Name
	}
	return result
}

// newSourceAppendLog builds a concrete in-memory log, used as the argument
// of the MergeAppend behaviour.
func newSourceAppendLog(t *testing.T) *schemaless.AppendLog[schemaless.ExampleRecord] {
	t.Helper()
	shapeDef, found := shape.LookupShapeReflectAndIndex[schemaless.Change[schemaless.ExampleRecord]]()
	require.True(t, found)
	return schemaless.NewAppendLog[schemaless.ExampleRecord](shapeDef)
}

// RunAppendLogSpec runs the behavioural specification against an AppendLoger
// implementation. Behaviours excluded by caps are reported as skipped
// subtests. name is one of the AppendLog* constants; it names the capability
// report the run writes to the report/ directory.
func RunAppendLogSpec(t *testing.T, name string, newLog NewAppendLogFunc, caps AppendLogCapabilities) {
	r := newRunner(t, appendLogReportPrefix+name, suiteAppendLog, caps)

	r.run("pushed changes reach a subscriber in order with increasing offsets", func(t *testing.T) {
		log := newLog(t)
		sub := subscribeAppendLog(log, 0, nil)

		log.Push(schemaless.Change[schemaless.ExampleRecord]{After: ptr(appendLogSpecRecord("1", "alice"))})
		log.Push(schemaless.Change[schemaless.ExampleRecord]{After: ptr(appendLogSpecRecord("2", "bob"))})
		log.Push(schemaless.Change[schemaless.ExampleRecord]{After: ptr(appendLogSpecRecord("3", "carol"))})

		changes := sub.collect(t, 3)
		assert.Equal(t, []string{"alice", "bob", "carol"}, changeNames(changes),
			"changes must arrive in append order")
		assert.Less(t, changes[0].Offset, changes[1].Offset)
		assert.Less(t, changes[1].Offset, changes[2].Offset)

		sub.end(t)
	})

	r.run("change and delete emit corresponding events", func(t *testing.T) {
		log := newLog(t)
		sub := subscribeAppendLog(log, 0, nil)

		record := appendLogSpecRecord("1", "alice")
		require.NoError(t, log.Change(nil, ptr(record)))
		require.NoError(t, log.Delete(record))

		changes := sub.collect(t, 2)

		require.NotNil(t, changes[0].After, "a change event carries the new state")
		assert.False(t, changes[0].Deleted)
		assert.Equal(t, "alice", changes[0].After.Data.Name)

		require.NotNil(t, changes[1].Before, "a delete event carries the past state")
		assert.True(t, changes[1].Deleted)
		assert.Equal(t, "alice", changes[1].Before.Data.Name)

		sub.end(t)
	})

	r.run("every subscriber receives every change", func(t *testing.T) {
		log := newLog(t)
		first := subscribeAppendLog(log, 0, nil)
		second := subscribeAppendLog(log, 0, nil)

		log.Push(schemaless.Change[schemaless.ExampleRecord]{After: ptr(appendLogSpecRecord("1", "alice"))})
		log.Push(schemaless.Change[schemaless.ExampleRecord]{After: ptr(appendLogSpecRecord("2", "bob"))})

		assert.Equal(t, []string{"alice", "bob"}, changeNames(first.collect(t, 2)))
		assert.Equal(t, []string{"alice", "bob"}, changeNames(second.collect(t, 2)))

		first.end(t)
		second.end(t)
	})

	r.run("close unblocks a subscriber waiting on an empty log", func(t *testing.T) {
		log := newLog(t)
		sub := subscribeAppendLog(log, 0, nil)

		log.Close()

		select {
		case err := <-sub.done:
			assert.NoError(t, err, "closing the log must end the subscription cleanly")
		case <-time.After(2 * time.Second):
			t.Fatal("Close must unblock a subscriber waiting on an empty log")
		}
	})

	r.run("context cancellation unblocks a waiting subscriber", func(t *testing.T) {
		log := newLog(t)
		sub := subscribeAppendLog(log, 0, nil)

		sub.cancel()

		select {
		case err := <-sub.done:
			assert.ErrorIs(t, err, context.Canceled)
		case <-time.After(2 * time.Second):
			t.Fatal("ctx cancellation must unblock a waiting subscriber")
		}
	})

	r.runGated(caps.Replay, "Replay",
		"a closed log replays every change to a late subscriber", func(t *testing.T) {
			log := newLog(t)
			log.Push(schemaless.Change[schemaless.ExampleRecord]{After: ptr(appendLogSpecRecord("1", "alice"))})
			log.Push(schemaless.Change[schemaless.ExampleRecord]{After: ptr(appendLogSpecRecord("2", "bob"))})
			log.Close()

			var got []appendLogChange
			err := log.Subscribe(context.Background(), 0, nil, func(c appendLogChange) {
				got = append(got, c)
			})
			require.NoError(t, err)
			assert.Equal(t, []string{"alice", "bob"}, changeNames(got))
		})

	r.runGated(caps.OffsetResume, "OffsetResume",
		"subscription resumes from a given offset", func(t *testing.T) {
			log := newLog(t)
			log.Push(schemaless.Change[schemaless.ExampleRecord]{After: ptr(appendLogSpecRecord("1", "alice"))})
			log.Push(schemaless.Change[schemaless.ExampleRecord]{After: ptr(appendLogSpecRecord("2", "bob"))})
			log.Push(schemaless.Change[schemaless.ExampleRecord]{After: ptr(appendLogSpecRecord("3", "carol"))})
			log.Close()

			var all []appendLogChange
			err := log.Subscribe(context.Background(), 0, nil, func(c appendLogChange) {
				all = append(all, c)
			})
			require.NoError(t, err)
			require.Len(t, all, 3)

			var resumed []appendLogChange
			err = log.Subscribe(context.Background(), all[1].Offset, nil, func(c appendLogChange) {
				resumed = append(resumed, c)
			})
			require.NoError(t, err)
			assert.Equal(t, []string{"bob", "carol"}, changeNames(resumed),
				"resuming from an offset must deliver that change and everything after it")
		})

	r.runGated(caps.Filtering, "Filtering",
		"filter delivers only matching changes", func(t *testing.T) {
			log := newLog(t)
			sub := subscribeAppendLog(log, 0, predicate.MustWhere(
				"Data.Name = :name",
				predicate.ParamBinds{":name": schema.MkString("alice")},
				nil,
			))

			log.Push(schemaless.Change[schemaless.ExampleRecord]{After: ptr(appendLogSpecRecord("1", "alice"))})
			log.Push(schemaless.Change[schemaless.ExampleRecord]{After: ptr(appendLogSpecRecord("2", "bob"))})
			log.Push(schemaless.Change[schemaless.ExampleRecord]{After: ptr(appendLogSpecRecord("3", "alice"))})

			changes := sub.collect(t, 2)
			assert.Equal(t, []string{"alice", "alice"}, changeNames(changes))
			sub.expectNoMore(t)

			sub.end(t)
		})

	r.runGated(caps.MergeAppend, "MergeAppend",
		"append merges another log's changes", func(t *testing.T) {
			source := newSourceAppendLog(t)
			source.Push(schemaless.Change[schemaless.ExampleRecord]{After: ptr(appendLogSpecRecord("1", "alice"))})
			source.Push(schemaless.Change[schemaless.ExampleRecord]{After: ptr(appendLogSpecRecord("2", "bob"))})

			log := newLog(t)
			sub := subscribeAppendLog(log, 0, nil)

			log.Append(source)

			assert.Equal(t, []string{"alice", "bob"}, changeNames(sub.collect(t, 2)))
			sub.end(t)
		})
}

func ptr[T any](x T) *T {
	return &x
}
