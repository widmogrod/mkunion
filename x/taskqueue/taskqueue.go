package taskqueue

import (
	"context"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/predicate"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"time"
)

func NewTaskQueue(desc *Description, queue Queuer[schemaless.Record[schema.Schema]], find Repository, proc Processor[schemaless.Record[schema.Schema]]) *TaskQueue {
	return &TaskQueue{
		desc:  desc,
		queue: queue,
		find:  find,
		proc:  proc,
	}
}

type Queuer[T any] interface {
	Push(ctx context.Context, task Task[T]) error
	Pop(ctx context.Context) ([]Task[T], error)
	Delete(ctx context.Context, tasks []Task[schemaless.Record[schema.Schema]]) error
}

type Repository interface {
	FindingRecords(query schemaless.FindingRecords[schemaless.Record[schema.Schema]]) (schemaless.PageResult[schemaless.Record[schema.Schema]], error)
}

type Processor[T any] interface {
	Process(task Task[T]) error
}

type TaskQueue struct {
	desc  *Description
	queue Queuer[schemaless.Record[schema.Schema]]
	find  Repository
	proc  Processor[schemaless.Record[schema.Schema]]
}

func (q *TaskQueue) RunSelector(ctx context.Context) error {
	for {
		var after = &schemaless.FindingRecords[schemaless.Record[schema.Schema]]{
			RecordType: q.desc.Entity,
			Where:      predicate.MustWhere(q.desc.Filter, predicate.ParamBinds{}),
			Limit:      10,
		}

		for {
			records, err := q.find.FindingRecords(*after)
			if err != nil {
				return err
			}

			for _, record := range records.Items {
				err := q.queue.Push(ctx, Task[schemaless.Record[schema.Schema]]{
					Data: record,
				})
				if err != nil {
					panic(err)
					return err
				}
			}

			if !records.HasNext() {
				break
			}

			after = records.Next
		}

		time.Sleep(1 * time.Second)
	}
}

func (q *TaskQueue) RunProcessor(ctx context.Context) error {
	for {
		tasks, err := q.queue.Pop(ctx)
		if err != nil {
			panic(err)
			return err
		}

		for _, task := range tasks {
			err = q.proc.Process(task)
			if err != nil {
				panic(err)
				return err
			}
		}
		err = q.queue.Delete(ctx, tasks)
		if err != nil {
			panic(err)
			return err
		}
	}
}

type Description struct {
	Change []string
	Entity string
	Filter string
}

type Task[T any] struct {
	ID   string
	Data T
	Meta map[string]string
}

type FunctionProcessor[T any] struct {
	F func(task Task[schemaless.Record[T]])
}

func (proc *FunctionProcessor[T]) Process(task Task[schemaless.Record[schema.Schema]]) error {
	t, err := schemaless.RecordAs[T](task.Data)
	if err != nil {
		panic(err)
	}

	proc.F(Task[schemaless.Record[T]]{
		Data: t,
	})

	return nil
}

var _ Processor[schemaless.Record[schema.Schema]] = &FunctionProcessor[schemaless.Record[schema.Schema]]{}
