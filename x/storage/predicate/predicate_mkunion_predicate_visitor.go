// Code generated by mkunion. DO NOT EDIT.
package predicate

import (
	"github.com/widmogrod/mkunion/f"
)

type PredicateVisitor interface {
	VisitAnd(v *And) any
	VisitOr(v *Or) any
	VisitNot(v *Not) any
	VisitCompare(v *Compare) any
}

type Predicate interface {
	AcceptPredicate(g PredicateVisitor) any
}

func (r *And) AcceptPredicate(v PredicateVisitor) any     { return v.VisitAnd(r) }
func (r *Or) AcceptPredicate(v PredicateVisitor) any      { return v.VisitOr(r) }
func (r *Not) AcceptPredicate(v PredicateVisitor) any     { return v.VisitNot(r) }
func (r *Compare) AcceptPredicate(v PredicateVisitor) any { return v.VisitCompare(r) }

var (
	_ Predicate = (*And)(nil)
	_ Predicate = (*Or)(nil)
	_ Predicate = (*Not)(nil)
	_ Predicate = (*Compare)(nil)
)

func MatchPredicate[TOut any](
	x Predicate,
	f1 func(x *And) TOut,
	f2 func(x *Or) TOut,
	f3 func(x *Not) TOut,
	f4 func(x *Compare) TOut,
	df func(x Predicate) TOut,
) TOut {
	return f.Match4(x, f1, f2, f3, f4, df)
}

func MatchPredicateR2[TOut1, TOut2 any](
	x Predicate,
	f1 func(x *And) (TOut1, TOut2),
	f2 func(x *Or) (TOut1, TOut2),
	f3 func(x *Not) (TOut1, TOut2),
	f4 func(x *Compare) (TOut1, TOut2),
	df func(x Predicate) (TOut1, TOut2),
) (TOut1, TOut2) {
	return f.Match4R2(x, f1, f2, f3, f4, df)
}

func MustMatchPredicate[TOut any](
	x Predicate,
	f1 func(x *And) TOut,
	f2 func(x *Or) TOut,
	f3 func(x *Not) TOut,
	f4 func(x *Compare) TOut,
) TOut {
	return f.MustMatch4(x, f1, f2, f3, f4)
}

func MustMatchPredicateR0(
	x Predicate,
	f1 func(x *And),
	f2 func(x *Or),
	f3 func(x *Not),
	f4 func(x *Compare),
) {
	f.MustMatch4R0(x, f1, f2, f3, f4)
}

func MustMatchPredicateR2[TOut1, TOut2 any](
	x Predicate,
	f1 func(x *And) (TOut1, TOut2),
	f2 func(x *Or) (TOut1, TOut2),
	f3 func(x *Not) (TOut1, TOut2),
	f4 func(x *Compare) (TOut1, TOut2),
) (TOut1, TOut2) {
	return f.MustMatch4R2(x, f1, f2, f3, f4)
}