package generators

import (
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/shape"
	"testing"
)

func TestVisitorGenerator_Tree(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	inferred, err := shape.InferFromFile("testutils/tree.go")
	assert.NoError(t, err)

	g := NewVisitorGenerator(inferred.RetrieveUnion("Tree"))

	result, err := g.Generate()
	assert.NoError(t, err)
	assert.Equal(t, `package testutils

type TreeVisitor interface {
	VisitBranch(v *Branch) any
	VisitLeaf(v *Leaf) any
	VisitK(v *K) any
	VisitP(v *P) any
	VisitMa(v *Ma) any
	VisitLa(v *La) any
	VisitKa(v *Ka) any
}

type Tree interface {
	AcceptTree(g TreeVisitor) any
}

var (
	_ Tree = (*Branch)(nil)
	_ Tree = (*Leaf)(nil)
	_ Tree = (*K)(nil)
	_ Tree = (*P)(nil)
	_ Tree = (*Ma)(nil)
	_ Tree = (*La)(nil)
	_ Tree = (*Ka)(nil)
)

func (r *Branch) AcceptTree(v TreeVisitor) any { return v.VisitBranch(r) }
func (r *Leaf) AcceptTree(v TreeVisitor) any { return v.VisitLeaf(r) }
func (r *K) AcceptTree(v TreeVisitor) any { return v.VisitK(r) }
func (r *P) AcceptTree(v TreeVisitor) any { return v.VisitP(r) }
func (r *Ma) AcceptTree(v TreeVisitor) any { return v.VisitMa(r) }
func (r *La) AcceptTree(v TreeVisitor) any { return v.VisitLa(r) }
func (r *Ka) AcceptTree(v TreeVisitor) any { return v.VisitKa(r) }

func MatchTreeR3[T0, T1, T2 any](
	x Tree,
	f1 func(x *Branch) (T0, T1, T2),
	f2 func(x *Leaf) (T0, T1, T2),
	f3 func(x *K) (T0, T1, T2),
	f4 func(x *P) (T0, T1, T2),
	f5 func(x *Ma) (T0, T1, T2),
	f6 func(x *La) (T0, T1, T2),
	f7 func(x *Ka) (T0, T1, T2),
) (T0, T1, T2) {
	switch v := x.(type) {
	case *Branch:
		return f1(v)
	case *Leaf:
		return f2(v)
	case *K:
		return f3(v)
	case *P:
		return f4(v)
	case *Ma:
		return f5(v)
	case *La:
		return f6(v)
	case *Ka:
		return f7(v)
	}
	var result1 T0
	var result2 T1
	var result3 T2
	return result1, result2, result3
}

func MatchTreeR2[T0, T1 any](
	x Tree,
	f1 func(x *Branch) (T0, T1),
	f2 func(x *Leaf) (T0, T1),
	f3 func(x *K) (T0, T1),
	f4 func(x *P) (T0, T1),
	f5 func(x *Ma) (T0, T1),
	f6 func(x *La) (T0, T1),
	f7 func(x *Ka) (T0, T1),
) (T0, T1) {
	switch v := x.(type) {
	case *Branch:
		return f1(v)
	case *Leaf:
		return f2(v)
	case *K:
		return f3(v)
	case *P:
		return f4(v)
	case *Ma:
		return f5(v)
	case *La:
		return f6(v)
	case *Ka:
		return f7(v)
	}
	var result1 T0
	var result2 T1
	return result1, result2
}

func MatchTreeR1[T0 any](
	x Tree,
	f1 func(x *Branch) T0,
	f2 func(x *Leaf) T0,
	f3 func(x *K) T0,
	f4 func(x *P) T0,
	f5 func(x *Ma) T0,
	f6 func(x *La) T0,
	f7 func(x *Ka) T0,
) T0 {
	switch v := x.(type) {
	case *Branch:
		return f1(v)
	case *Leaf:
		return f2(v)
	case *K:
		return f3(v)
	case *P:
		return f4(v)
	case *Ma:
		return f5(v)
	case *La:
		return f6(v)
	case *Ka:
		return f7(v)
	}
	var result1 T0
	return result1
}

func MatchTreeR0(
	x Tree,
	f1 func(x *Branch),
	f2 func(x *Leaf),
	f3 func(x *K),
	f4 func(x *P),
	f5 func(x *Ma),
	f6 func(x *La),
	f7 func(x *Ka),
) {
	switch v := x.(type) {
	case *Branch:
		f1(v)
	case *Leaf:
		f2(v)
	case *K:
		f3(v)
	case *P:
		f4(v)
	case *Ma:
		f5(v)
	case *La:
		f6(v)
	case *Ka:
		f7(v)
	}
}
`, string(result))
}

func TestVisitorGenerator_Generic(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	inferred, err := shape.InferFromFile("testutils/generic.go")
	assert.NoError(t, err)

	g := NewVisitorGenerator(inferred.RetrieveUnion("Record"))

	result, err := g.Generate()
	assert.NoError(t, err)
	assert.Equal(t, `package testutils

type RecordVisitor[A any] interface {
	VisitItem(v *Item[A]) any
}

type Record[A any] interface {
	AcceptRecord(g RecordVisitor[A]) any
}

var (
	_ Record[any] = (*Item[any])(nil)
)

func (r *Item[A]) AcceptRecord(v RecordVisitor[A]) any { return v.VisitItem(r) }

func MatchRecordR3[A any, T0, T1, T2 any](
	x Record[A],
	f1 func(x *Item[A]) (T0, T1, T2),
) (T0, T1, T2) {
	switch v := x.(type) {
	case *Item[A]:
		return f1(v)
	}
	var result1 T0
	var result2 T1
	var result3 T2
	return result1, result2, result3
}

func MatchRecordR2[A any, T0, T1 any](
	x Record[A],
	f1 func(x *Item[A]) (T0, T1),
) (T0, T1) {
	switch v := x.(type) {
	case *Item[A]:
		return f1(v)
	}
	var result1 T0
	var result2 T1
	return result1, result2
}

func MatchRecordR1[A any, T0 any](
	x Record[A],
	f1 func(x *Item[A]) T0,
) T0 {
	switch v := x.(type) {
	case *Item[A]:
		return f1(v)
	}
	var result1 T0
	return result1
}

func MatchRecordR0[A any](
	x Record[A],
	f1 func(x *Item[A]),
) {
	switch v := x.(type) {
	case *Item[A]:
		f1(v)
	}
}
`, string(result))
}
