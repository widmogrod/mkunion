package schemaless

import (
	"container/list"
	"context"
	"errors"
	"sync"
)

type Change[T any] struct {
	Before  *Record[T]
	After   *Record[T]
	Deleted bool
	Offset  int
}

func NewAppendLog[T any]() *AppendLog[T] {
	mux := &sync.RWMutex{}
	return &AppendLog[T]{
		log:  list.List{},
		mux:  mux,
		cond: sync.NewCond(mux.RLocker()),
	}
}

// AppendLog is a stream of events, and in context of schemaless, it is a stream of changes to records, or deleted record with past state
type AppendLog[T any] struct {
	log    list.List
	mux    *sync.RWMutex
	cond   *sync.Cond
	closed bool
}

func (a *AppendLog[T]) Close() {
	a.mux.RLock()
	defer a.mux.RUnlock()

	a.closed = true
	a.cond.Broadcast()
}

func (a *AppendLog[T]) Change(from, to Record[T]) error {
	a.mux.Lock()
	defer a.mux.Unlock()

	if a.closed {
		panic("cannot append to closed log")
	}

	a.log.PushBack(Change[T]{
		Before:  &from,
		After:   &to,
		Deleted: false,
	})
	a.cond.Broadcast()
	return nil
}

func (a *AppendLog[T]) Delete(data Record[T]) error {
	a.mux.Lock()
	defer a.mux.Unlock()

	if a.closed {
		panic("cannot append to closed log")
	}

	a.log.PushBack(Change[T]{
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

	a.log.PushBack(x)
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
		a.log.PushBack(e.Value)
	}
	a.cond.Broadcast()
}

func (a *AppendLog[T]) Subscribe(ctx context.Context, fromOffset int, f func(Change[T])) error {
	var prev *list.Element = nil

	// Until, there is no messages, wait
	a.cond.L.Lock()
	for a.log.Len() == 0 {
		a.cond.Wait()
	}

	// Select the offset to start reading messages from
	switch fromOffset {
	case 0:
		prev = a.log.Front()
	case -1:
		prev = a.log.Back()
	default:
		for e := a.log.Front(); e != nil; e = e.Next() {
			prev = e
			if e.Value.(Change[T]).Offset == fromOffset {
				break
			}
		}

		if prev == a.log.Back() {
			a.cond.L.Unlock()
			return errors.New("offset not found")
		}
	}
	a.cond.L.Unlock()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		default:
			msg := prev.Value.(Change[T])

			f(msg)

			// Wait for new changes to be available
			a.cond.L.Lock()
			for prev.Next() == nil && !a.closed {
				a.cond.Wait()
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
}
