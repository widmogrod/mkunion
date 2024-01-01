package generators

import (
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/shape"
	"testing"
)

func TestVisitorGenerator(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	inferred, err := shape.InferFromFile("testutils/tree.go")
	assert.NoError(t, err)

	g := NewVisitorGenerator(
		inferred.RetrieveUnion("Tree"),
		NewHelper(WithPackageName("testutils")),
	)

	result, err := g.Generate()
	assert.NoError(t, err)
	assert.Equal(t, `package testutils

import (
	"github.com/widmogrod/mkunion/f"
)


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

func (r *Branch) AcceptTree(v TreeVisitor) any { return v.VisitBranch(r) }
func (r *Leaf) AcceptTree(v TreeVisitor) any { return v.VisitLeaf(r) }
func (r *K) AcceptTree(v TreeVisitor) any { return v.VisitK(r) }
func (r *P) AcceptTree(v TreeVisitor) any { return v.VisitP(r) }
func (r *Ma) AcceptTree(v TreeVisitor) any { return v.VisitMa(r) }
func (r *La) AcceptTree(v TreeVisitor) any { return v.VisitLa(r) }
func (r *Ka) AcceptTree(v TreeVisitor) any { return v.VisitKa(r) }

var (
	_ Tree = (*Branch)(nil)
	_ Tree = (*Leaf)(nil)
	_ Tree = (*K)(nil)
	_ Tree = (*P)(nil)
	_ Tree = (*Ma)(nil)
	_ Tree = (*La)(nil)
	_ Tree = (*Ka)(nil)
)

func MatchTree[TOut any](
	x Tree,
	f1 func(x *Branch) TOut,
	f2 func(x *Leaf) TOut,
	f3 func(x *K) TOut,
	f4 func(x *P) TOut,
	f5 func(x *Ma) TOut,
	f6 func(x *La) TOut,
	f7 func(x *Ka) TOut,
	df func(x Tree) TOut,
) TOut {
	return f.Match7(x, f1, f2, f3, f4, f5, f6, f7, df)
}

func MatchTreeR2[TOut1, TOut2 any](
	x Tree,
	f1 func(x *Branch) (TOut1, TOut2),
	f2 func(x *Leaf) (TOut1, TOut2),
	f3 func(x *K) (TOut1, TOut2),
	f4 func(x *P) (TOut1, TOut2),
	f5 func(x *Ma) (TOut1, TOut2),
	f6 func(x *La) (TOut1, TOut2),
	f7 func(x *Ka) (TOut1, TOut2),
	df func(x Tree) (TOut1, TOut2),
) (TOut1, TOut2) {
	return f.Match7R2(x, f1, f2, f3, f4, f5, f6, f7, df)
}

func MustMatchTree[TOut any](
	x Tree,
	f1 func(x *Branch) TOut,
	f2 func(x *Leaf) TOut,
	f3 func(x *K) TOut,
	f4 func(x *P) TOut,
	f5 func(x *Ma) TOut,
	f6 func(x *La) TOut,
	f7 func(x *Ka) TOut,
) TOut {
	return f.MustMatch7(x, f1, f2, f3, f4, f5, f6, f7)
}

func MustMatchTreeR0(
	x Tree,
	f1 func(x *Branch),
	f2 func(x *Leaf),
	f3 func(x *K),
	f4 func(x *P),
	f5 func(x *Ma),
	f6 func(x *La),
	f7 func(x *Ka),
) {
	f.MustMatch7R0(x, f1, f2, f3, f4, f5, f6, f7)
}

func MustMatchTreeR2[TOut1, TOut2 any](
	x Tree,
	f1 func(x *Branch) (TOut1, TOut2),
	f2 func(x *Leaf) (TOut1, TOut2),
	f3 func(x *K) (TOut1, TOut2),
	f4 func(x *P) (TOut1, TOut2),
	f5 func(x *Ma) (TOut1, TOut2),
	f6 func(x *La) (TOut1, TOut2),
	f7 func(x *Ka) (TOut1, TOut2),
) (TOut1, TOut2) {
	return f.MustMatch7R2(x, f1, f2, f3, f4, f5, f6, f7)
}`, string(result))
}
