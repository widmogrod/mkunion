package taskqueue

import (
	"context"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/predicate"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"time"
)

func NewTaskQueue(
	desc *Description,
	queue Queuer[schemaless.Record[schema.Schema]],
	find Repository,
	stream *schemaless.AppendLog[schema.Schema],
	proc Processor[schemaless.Record[schema.Schema]],
) *TaskQueue {
	return &TaskQueue{
		desc:   desc,
		queue:  queue,
		find:   find,
		stream: stream,
		proc:   proc,
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
	desc   *Description
	queue  Queuer[schemaless.Record[schema.Schema]]
	find   Repository
	stream *schemaless.AppendLog[schema.Schema]
	proc   Processor[schemaless.Record[schema.Schema]]
}

func (q *TaskQueue) RunCDC(ctx context.Context) error {
	return q.stream.Subscribe(ctx, 0, func(change schemaless.Change[schema.Schema]) {
		filter := predicate.MustWhere(q.desc.Filter, q.params())
		if !predicate.Evaluate(filter.Predicate, schema.FromGo(change.After), filter.Params) {
			return
		}

		err := q.queue.Push(ctx, Task[schemaless.Record[schema.Schema]]{
			ID:   change.After.ID,
			Data: *change.After,
		})
		if err != nil {
			panic(err)
		}
	})
}

func (q *TaskQueue) RunSelector(ctx context.Context) error {
	var timeDelta = time.Second * 1
	var startTime time.Time
	for {
		startTime = time.Now()

		var after = &schemaless.FindingRecords[schemaless.Record[schema.Schema]]{
			RecordType: q.desc.Entity,
			Where:      predicate.MustWhere(q.desc.Filter, q.params()),
			Limit:      10,
		}

		for {
			records, err := q.find.FindingRecords(*after)
			if err != nil {
				panic(err)
				return err
			}

			for _, record := range records.Items {
				err := q.queue.Push(ctx, Task[schemaless.Record[schema.Schema]]{
					ID:   record.ID,
					Data: record,
					Meta: nil,
				})
				if err != nil {
					panic(err)
					return err
				}
			}

			after = records.Next
			if !records.HasNext() {
				break
			}
		}

		// don't run too often
		elapsed := time.Now().Sub(startTime)
		if elapsed < timeDelta {
			wait := timeDelta - elapsed
			time.Sleep(wait)
		}
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

func (q *TaskQueue) params() predicate.ParamBinds {
	timeNow := schema.FromGo(time.Now().Unix())
	return predicate.ParamBinds{
		":now": timeNow,
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
		ID:   task.ID,
		Data: t,
	})

	return nil
}

var _ Processor[schemaless.Record[schema.Schema]] = &FunctionProcessor[schemaless.Record[schema.Schema]]{}
