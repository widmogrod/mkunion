package taskqueue

import (
	"context"
)

func NewInMemoryQueue[T any]() *Queue[T] {
	return &Queue[T]{
		queue: make(chan Task[T], 100),
	}
}

var _ Queuer[any] = (*Queue[any])(nil)

type Queue[T any] struct {
	queue chan Task[T]
}

func (q *Queue[T]) Push(ctx context.Context, task Task[T]) error {
	q.queue <- task
	return nil
}

func (q *Queue[T]) Pop(ctx context.Context) ([]Task[T], error) {
	return []Task[T]{<-q.queue}, nil
}

func (*Queue[T]) Delete(ctx context.Context, tasks []Task[T]) error {
	return nil
}

func (q *Queue[T]) Close() error {
	close(q.queue)
	return nil
}
