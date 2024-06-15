package taskqueue

import (
	"context"
	log "github.com/sirupsen/logrus"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/predicate"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
	"time"
)

func NewTaskQueue[T any](
	desc *Description,
	queue Queuer[schemaless.Record[T]],
	find schemaless.Repository[T],
	stream schemaless.AppendLoger[T],
	proc Processor[schemaless.Record[T]],
) *TaskQueue[T] {
	return &TaskQueue[T]{
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
	Delete(ctx context.Context, tasks []Task[T]) error
}

type Processor[T any] interface {
	Process(task Task[T]) error
}

type TaskQueue[T any] struct {
	desc   *Description
	queue  Queuer[schemaless.Record[T]]
	find   schemaless.Repository[T]
	stream schemaless.AppendLoger[T]
	proc   Processor[schemaless.Record[T]]
}

func (q *TaskQueue[T]) RunCDC(ctx context.Context) error {
	filter := predicate.MustWhere(q.desc.Filter, q.params(), nil)

	return q.stream.Subscribe(ctx, 0, filter, func(change schemaless.Change[T]) {
		err := q.queue.Push(ctx, Task[schemaless.Record[T]]{
			ID:   change.After.ID,
			Data: *change.After,
		})
		if err != nil {
			panic(err)
		}
	})
}

func (q *TaskQueue[T]) RunSelector(ctx context.Context) error {
	var timeDelta = time.Second * 1
	var startTime time.Time
	for {
		startTime = time.Now()

		var after = &schemaless.FindingRecords[schemaless.Record[T]]{
			RecordType: q.desc.Entity,
			Where:      predicate.MustWhere(q.desc.Filter, q.params(), nil),
			Limit:      10,
		}

		log := log.WithField("where", q.desc.Filter)

		for {
			records, err := q.find.FindingRecords(*after)
			if err != nil {
				panic(err)
				return err
			}

			log.Infof("taskqueue: FindingRecords(): %d", len(records.Items))

			for _, record := range records.Items {
				err := q.queue.Push(ctx, Task[schemaless.Record[T]]{
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

func (q *TaskQueue[T]) RunProcessor(ctx context.Context) error {
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

func (q *TaskQueue[T]) params() predicate.ParamBinds {
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
	F func(task Task[T])
}

func (proc *FunctionProcessor[T]) Process(task Task[T]) error {
	//t, err := schemaless.RecordAs[T](task.Data)
	//if err != nil {
	//	panic(err)
	//}
	//
	//proc.F(Task[schemaless.Record[T]]{
	//	ID:   task.ID,
	//	Data: t,
	//})

	proc.F(task)
	return nil
}

//var _ Processor[schemaless.Record[schema.Schema]] = &FunctionProcessor[schemaless.Record[schema.Schema]]{}
