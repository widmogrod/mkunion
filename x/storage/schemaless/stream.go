package schemaless

import (
	"container/list"
	"context"
	"errors"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/shape"
	"github.com/widmogrod/mkunion/x/storage/predicate"
	"sync"
)

type AppendLoger[T any] interface {
	Close()
	Change(from, to *Record[T]) error
	Delete(data Record[T]) error
	Push(x Change[T])
	Append(b *AppendLog[T])
	Subscribe(ctx context.Context, fromOffset int, filter *predicate.WherePredicates, f func(Change[T])) error
}

type Change[T any] struct {
	Before  *Record[T]
	After   *Record[T]
	Deleted bool
	Offset  int
}

func NewAppendLog[T any](shapeDef shape.Shape) *AppendLog[T] {
	mux := &sync.RWMutex{}
	return &AppendLog[T]{
		log:      list.List{},
		mux:      mux,
		cond:     sync.NewCond(mux.RLocker()),
		shapeDef: shapeDef,
	}
}

var _ AppendLoger[any] = (*AppendLog[any])(nil)

// AppendLog is a stream of events, and in context of schemaless, it is a stream of changes to records, or deleted record with past state
type AppendLog[T any] struct {
	log        list.List
	mux        *sync.RWMutex
	cond       *sync.Cond
	closed     bool
	nextOffset int
	shapeDef   shape.Shape
}

func (a *AppendLog[T]) Close() {
	a.mux.Lock()
	defer a.mux.Unlock()

	a.closed = true
	a.cond.Broadcast()
}

func (a *AppendLog[T]) Change(from, to *Record[T]) error {
	a.mux.Lock()
	defer a.mux.Unlock()

	if a.closed {
		panic("cannot append to closed log")
	}

	a.pushBack(Change[T]{
		Before:  from,
		After:   to,
		Deleted: false,
	})
	a.cond.Broadcast()
	return nil
}

// pushBack stamps the change with the next offset; callers hold the write lock.
func (a *AppendLog[T]) pushBack(x Change[T]) {
	x.Offset = a.nextOffset
	a.nextOffset++
	a.log.PushBack(x)
}

func (a *AppendLog[T]) Delete(data Record[T]) error {
	a.mux.Lock()
	defer a.mux.Unlock()

	if a.closed {
		panic("cannot append to closed log")
	}

	a.pushBack(Change[T]{
		Before:  &data,
		Deleted: true,
	})
	a.cond.Broadcast()
	return nil
}

func (a *AppendLog[T]) Push(x Change[T]) {
	a.mux.Lock()
	defer a.mux.Unlock()

	if a.closed {
		panic("cannot append to closed log")
	}

	a.pushBack(x)
	a.cond.Broadcast()
}

func (a *AppendLog[T]) Append(b *AppendLog[T]) {
	a.mux.Lock()
	defer a.mux.Unlock()

	b.mux.Lock()
	defer b.mux.Unlock()

	if b.closed {
		panic("cannot append to closed log")
	}

	for e := b.log.Front(); e != nil; e = e.Next() {
		// re-stamp offsets so they stay monotonic within this log
		a.pushBack(e.Value.(Change[T]))
	}
	a.cond.Broadcast()
}

func (a *AppendLog[T]) Subscribe(ctx context.Context, fromOffset int, filter *predicate.WherePredicates, f func(Change[T])) error {
	// wake parked waiters when the context is cancelled;
	// cond.Wait cannot observe ctx on its own
	stopWatch := context.AfterFunc(ctx, func() {
		a.cond.Broadcast()
	})
	defer stopWatch()

	var prev *list.Element = nil

	// Until there are messages, wait; Close and ctx cancellation unblock
	a.cond.L.Lock()
	for a.log.Len() == 0 && !a.closed && ctx.Err() == nil {
		a.cond.Wait()
	}
	if err := ctx.Err(); err != nil {
		a.cond.L.Unlock()
		return err
	}
	if a.log.Len() == 0 {
		// closed with nothing to consume
		a.cond.L.Unlock()
		return nil
	}

	// Select the offset to start reading messages from
	switch fromOffset {
	case 0:
		prev = a.log.Front()
	case -1:
		prev = a.log.Back()
	default:
		found := false
		for e := a.log.Front(); e != nil; e = e.Next() {
			if e.Value.(Change[T]).Offset == fromOffset {
				prev = e
				found = true
				break
			}
		}

		if !found {
			a.cond.L.Unlock()
			return errors.New("offset not found")
		}
	}
	a.cond.L.Unlock()

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		msg := prev.Value.(Change[T])

		validCondition := true
		if filter != nil && msg.After != nil {
			validCondition = predicate.EvaluateSchema(filter.Predicate, schema.FromGo[Record[T]](*msg.After), filter.Params)
		}

		if validCondition {
			f(msg)
		}

		// Wait for new changes to be available
		a.cond.L.Lock()
		for prev.Next() == nil && !a.closed && ctx.Err() == nil {
			a.cond.Wait()
		}

		if err := ctx.Err(); err != nil {
			a.cond.L.Unlock()
			return err
		}

		// If the stream is closed, and there are no more messages, return
		// this guarantees that multiple can consume the log, even if it's closed
		if prev.Next() == nil && a.closed {
			a.cond.L.Unlock()
			return nil
		}

		prev = prev.Next()
		a.cond.L.Unlock()
	}
}
