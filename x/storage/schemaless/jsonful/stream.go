package jsonful

import (
	"context"

	log "github.com/sirupsen/logrus"
	"github.com/widmogrod/mkunion/x/storage/predicate"
	"github.com/widmogrod/mkunion/x/storage/schemaless"
)

var _ schemaless.AppendLoger[any] = (*AppendLog[any])(nil)

// AppendLog adapts schemaless.AppendLog so subscription filters are
// evaluated with the shape-driven JSON evaluator, the same query language
// the repository uses, instead of the reflective schema.FromGo path.
type AppendLog[T any] struct {
	inner     *schemaless.AppendLog[T]
	evaluator *predicate.JSONEvaluator
}

func (a *AppendLog[T]) Close() {
	a.inner.Close()
}

func (a *AppendLog[T]) Change(from, to *schemaless.Record[T]) error {
	return a.inner.Change(from, to)
}

func (a *AppendLog[T]) Delete(data schemaless.Record[T]) error {
	return a.inner.Delete(data)
}

func (a *AppendLog[T]) Push(x schemaless.Change[T]) {
	a.inner.Push(x)
}

func (a *AppendLog[T]) Append(b *schemaless.AppendLog[T]) {
	a.inner.Append(b)
}

func (a *AppendLog[T]) Subscribe(ctx context.Context, fromOffset int, filter *predicate.WherePredicates, f func(schemaless.Change[T])) error {
	// Validate filter locations before consuming the stream, so a typo
	// fails the subscription instead of silently dropping every change.
	if filter != nil {
		if err := validateLocations(a.evaluator, filter.Predicate); err != nil {
			return err
		}
	}

	return a.inner.Subscribe(ctx, fromOffset, nil, func(change schemaless.Change[T]) {
		if filter != nil && change.After != nil {
			doc, err := recordDocument(*change.After)
			if err != nil {
				log.Warnf("jsonful.AppendLog.Subscribe: cannot encode change ID=%s: %s", change.After.ID, err)
				return
			}
			ok, err := a.evaluator.Evaluate(filter.Predicate, doc, filter.Params)
			if err != nil {
				log.Warnf("jsonful.AppendLog.Subscribe: cannot evaluate filter on ID=%s: %s", change.After.ID, err)
				return
			}
			if !ok {
				return
			}
		}
		f(change)
	})
}

func validateLocations(evaluator *predicate.JSONEvaluator, p predicate.Predicate) error {
	return predicate.MatchPredicateR1(
		p,
		func(x *predicate.And) error {
			for _, sub := range x.L {
				if err := validateLocations(evaluator, sub); err != nil {
					return err
				}
			}
			return nil
		},
		func(x *predicate.Or) error {
			for _, sub := range x.L {
				if err := validateLocations(evaluator, sub); err != nil {
					return err
				}
			}
			return nil
		},
		func(x *predicate.Not) error {
			return validateLocations(evaluator, x.P)
		},
		func(x *predicate.Compare) error {
			_, err := evaluator.ResolvePaths(x.Location)
			if err != nil {
				return err
			}
			if loc, ok := x.BindValue.(*predicate.Locatable); ok {
				_, err = evaluator.ResolvePaths(loc.Location)
			}
			return err
		},
	)
}
