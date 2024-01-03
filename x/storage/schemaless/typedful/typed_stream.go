package typedful

import (
	"context"
	"fmt"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
)

func NewTypedAppendLog[T any](log schemaless.AppendLoger[schema.Schema]) *TypedAppendLog[T] {
	location, err := NewTypedLocation[schemaless.Record[T]]()
	if err != nil {
		panic(fmt.Errorf("typedful.NewTypedRepoWithAggregator: %w", err))
	}

	return &TypedAppendLog[T]{
		log: log,
		loc: location,
	}
}

type TypedAppendLog[T any] struct {
	log schemaless.AppendLoger[schema.Schema]
	loc *TypedLocation
}

func (t *TypedAppendLog[T]) Close() {
	//TODO implement me
	panic("implement me")
}

func (t *TypedAppendLog[T]) Change(from, to schemaless.Record[T]) error {
	//TODO implement me
	panic("implement me")
}

func (t *TypedAppendLog[T]) Delete(data schemaless.Record[T]) error {
	//TODO implement me
	panic("implement me")
}

func (t *TypedAppendLog[T]) Push(x schemaless.Change[T]) {
	//TODO implement me
	panic("implement me")
}

func (t *TypedAppendLog[T]) Append(b *schemaless.AppendLog[T]) {
	//TODO implement me
	panic("implement me")
}

func (t *TypedAppendLog[T]) Subscribe(ctx context.Context, fromOffset int, f func(schemaless.Change[T])) error {
	//TODO implement me
	panic("implement me")
}

var _ schemaless.AppendLoger[any] = &TypedAppendLog[any]{}
