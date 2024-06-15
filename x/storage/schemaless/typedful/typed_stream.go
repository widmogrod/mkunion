package typedful

import (
	"context"
	"fmt"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/storage/predicate"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
)

func NewTypedAppendLog[T any](log schemaless.AppendLoger[schema.Schema]) *TypedAppendLog[T] {
	location, err := schema.NewTypedLocation[schemaless.Record[T]]()
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
	loc *schema.TypedLocation
}

func (t *TypedAppendLog[T]) Close() {
	//TODO implement me
	panic("implement me")
}

func (t *TypedAppendLog[T]) Change(from, to *schemaless.Record[T]) error {
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

func (t *TypedAppendLog[T]) Subscribe(ctx context.Context, fromOffset int, filter *predicate.WherePredicates, f func(schemaless.Change[T])) error {
	filterw := &predicate.WherePredicates{
		Predicate: WrapPredicate(filter.Predicate, t.loc),
		Params:    filter.Params,
		Shape:     t.loc.ShapeDef(),
	}

	return t.log.Subscribe(ctx, fromOffset, filterw, func(change schemaless.Change[schema.Schema]) {
		typedChange := schemaless.Change[T]{
			Deleted: change.Deleted,
			Offset:  change.Offset,
		}

		if change.After != nil {
			after, err := schemaless.RecordAs[T](*change.After)
			if err != nil {
				panic(err)
			}
			typedChange.After = &after
		}

		if change.Before != nil {
			before, err := schemaless.RecordAs[T](*change.Before)
			if err != nil {
				panic(err)
			}
			typedChange.Before = &before
		}

		f(typedChange)
	})
}

var _ schemaless.AppendLoger[any] = &TypedAppendLog[any]{}
