package taskqueue

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
)

type rec = schemaless.Record[schema.Schema]

// scriptedQueue replays Pop batches and records pushes and deletions.
type scriptedQueue struct {
	pushed  []Task[rec]
	pushErr error

	pops    [][]Task[rec]
	popErr  error // returned after the scripted pops run out
	popIdx  int
	deleted [][]Task[rec]
}

func (q *scriptedQueue) Push(_ context.Context, task Task[rec]) error {
	if q.pushErr != nil {
		return q.pushErr
	}
	q.pushed = append(q.pushed, task)
	return nil
}

func (q *scriptedQueue) Pop(_ context.Context) ([]Task[rec], error) {
	if q.popIdx >= len(q.pops) {
		if q.popErr != nil {
			return nil, q.popErr
		}
		return nil, errors.New("scriptedQueue: out of pops")
	}
	batch := q.pops[q.popIdx]
	q.popIdx++
	return batch, nil
}

func (q *scriptedQueue) Delete(_ context.Context, tasks []Task[rec]) error {
	q.deleted = append(q.deleted, tasks)
	return nil
}

// scriptedRepo pages through scripted FindingRecords results.
type scriptedRepo struct {
	pages   []schemaless.PageResult[rec]
	pageIdx int
	findErr error // returned after the scripted pages run out
}

func (r *scriptedRepo) Get(string, schemaless.RecordType) (rec, error) {
	panic("not used")
}

func (r *scriptedRepo) UpdateRecords(schemaless.UpdateRecords[rec]) (*schemaless.UpdateRecordsResult[rec], error) {
	panic("not used")
}

func (r *scriptedRepo) FindingRecords(schemaless.FindingRecords[rec]) (schemaless.PageResult[rec], error) {
	if r.pageIdx >= len(r.pages) {
		return schemaless.PageResult[rec]{}, r.findErr
	}
	page := r.pages[r.pageIdx]
	r.pageIdx++
	return page, nil
}

type collectingProcessor struct {
	processed []Task[rec]
	procErr   error
}

func (p *collectingProcessor) Process(task Task[rec]) error {
	if p.procErr != nil {
		return p.procErr
	}
	p.processed = append(p.processed, task)
	return nil
}

func testDescription() *Description {
	return &Description{
		Entity: "process",
		// the queue binds :now automatically; the filter must use it
		Filter: "Data.Timestamp <= :now",
	}
}

// runUntilPanic runs the endless loop and reports the panic that ends it.
func runUntilPanic(t *testing.T, fn func() error) (panicked any) {
	t.Helper()
	done := make(chan any, 1)
	go func() {
		defer func() { done <- recover() }()
		_ = fn()
	}()
	select {
	case p := <-done:
		require.NotNil(t, p, "the loop can only end by panicking")
		return p
	case <-time.After(10 * time.Second):
		t.Fatal("loop did not terminate")
		return nil
	}
}

func TestRunProcessor(t *testing.T) {
	t.Run("processes and deletes batches in order", func(t *testing.T) {
		task1 := Task[rec]{ID: "1"}
		task2 := Task[rec]{ID: "2"}
		queue := &scriptedQueue{
			pops:   [][]Task[rec]{{task1}, {task2}},
			popErr: errors.New("stop"),
		}
		proc := &collectingProcessor{}
		q := NewTaskQueue[schema.Schema](testDescription(), queue, &scriptedRepo{}, nil, proc)

		p := runUntilPanic(t, func() error { return q.RunProcessor(context.Background()) })
		assert.ErrorContains(t, p.(error), "stop")

		require.Len(t, proc.processed, 2)
		assert.Equal(t, "1", proc.processed[0].ID)
		assert.Equal(t, "2", proc.processed[1].ID)
		require.Len(t, queue.deleted, 2, "every batch must be deleted after processing")
	})

	t.Run("processor failure ends the loop", func(t *testing.T) {
		queue := &scriptedQueue{pops: [][]Task[rec]{{{ID: "1"}}}}
		proc := &collectingProcessor{procErr: errors.New("cannot process")}
		q := NewTaskQueue[schema.Schema](testDescription(), queue, &scriptedRepo{}, nil, proc)

		p := runUntilPanic(t, func() error { return q.RunProcessor(context.Background()) })
		assert.ErrorContains(t, p.(error), "cannot process")
		assert.Empty(t, queue.deleted, "failed batches must not be deleted")
	})
}

func TestRunSelector(t *testing.T) {
	record := func(id string) rec {
		return rec{ID: id, Type: "process", Data: schema.MkString("x")}
	}

	t.Run("pushes every record across pages", func(t *testing.T) {
		next := &schemaless.FindingRecords[rec]{}
		repo := &scriptedRepo{
			pages: []schemaless.PageResult[rec]{
				{Items: []rec{record("1")}, Next: next},
				{Items: []rec{record("2")}},
			},
			findErr: errors.New("stop"),
		}
		queue := &scriptedQueue{}
		q := NewTaskQueue[schema.Schema](testDescription(), queue, repo, nil, &collectingProcessor{})

		p := runUntilPanic(t, func() error { return q.RunSelector(context.Background()) })
		assert.ErrorContains(t, p.(error), "stop")

		require.Len(t, queue.pushed, 2)
		assert.Equal(t, "1", queue.pushed[0].ID)
		assert.Equal(t, "2", queue.pushed[1].ID)
		require.NotNil(t, queue.pushed[0].Data)
	})

	t.Run("push failure ends the loop", func(t *testing.T) {
		repo := &scriptedRepo{
			pages: []schemaless.PageResult[rec]{{Items: []rec{record("1")}}},
		}
		queue := &scriptedQueue{pushErr: errors.New("queue full")}
		q := NewTaskQueue[schema.Schema](testDescription(), queue, repo, nil, &collectingProcessor{})

		p := runUntilPanic(t, func() error { return q.RunSelector(context.Background()) })
		assert.ErrorContains(t, p.(error), "queue full")
	})
}
