package typedful

import (
	"context"
	"fmt"
	"github.com/widmogrod/mkunion/x/schema"
	"github.com/widmogrod/mkunion/x/shape"
	"github.com/widmogrod/mkunion/x/storage/predicate"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
)

func NewTypedAppendLog[T any](log schemaless.AppendLoger[schema.Schema]) *TypedAppendLog[T] {
	encodedAs, found := shape.LookupShapeReflectAndIndex[schemaless.Record[schema.Schema]]()
	if !found {
		panic(fmt.Errorf("typedful.NewTypedRepoWithAggregator: shape not found %w", shape.ErrShapeNotFound))
	}

	location, err := schema.NewTypedLocationWithEncoded[schemaless.Record[T]](encodedAs)
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
	t.log.Close()
}

func (t *TypedAppendLog[T]) Change(from, to *schemaless.Record[T]) error {
	return t.log.Change(toSchemaRecordPtr(from), toSchemaRecordPtr(to))
}

func (t *TypedAppendLog[T]) Delete(data schemaless.Record[T]) error {
	return t.log.Delete(toSchemaRecord(data))
}

func (t *TypedAppendLog[T]) Push(x schemaless.Change[T]) {
	t.log.Push(schemaless.Change[schema.Schema]{
		Before:  toSchemaRecordPtr(x.Before),
		After:   toSchemaRecordPtr(x.After),
		Deleted: x.Deleted,
		Offset:  x.Offset,
	})
}

// Append merges a typed concrete log, but the backing log is
// schema-typed; converting between the two concrete list types is not
// supported. Use Change/Delete/Push instead.
func (t *TypedAppendLog[T]) Append(b *schemaless.AppendLog[T]) {
	panic("typedful.TypedAppendLog.Append: not supported; use Change/Delete/Push")
}

func toSchemaRecord[T any](r schemaless.Record[T]) schemaless.Record[schema.Schema] {
	return schemaless.Record[schema.Schema]{
		ID:      r.ID,
		Type:    r.Type,
		Data:    schema.FromGo(r.Data),
		Version: r.Version,
	}
}

func toSchemaRecordPtr[T any](r *schemaless.Record[T]) *schemaless.Record[schema.Schema] {
	if r == nil {
		return nil
	}
	result := toSchemaRecord(*r)
	return &result
}

func (t *TypedAppendLog[T]) Subscribe(ctx context.Context, fromOffset int, filter *predicate.WherePredicates, f func(schemaless.Change[T])) error {
	// nil filter means "every change"
	var filterw *predicate.WherePredicates
	if filter != nil {
		filterw = &predicate.WherePredicates{
			Predicate: WrapPredicate(filter.Predicate, t.loc),
			Params:    filter.Params,
			Shape:     t.loc.ShapeDef(),
		}
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
