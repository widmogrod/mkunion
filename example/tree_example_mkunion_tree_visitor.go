// Code generated by mkunion. DO NOT EDIT.
package example

import (
	"github.com/widmogrod/mkunion/f"
)

type TreeVisitor interface {
	VisitBranch(v *Branch) any
	VisitLeaf(v *Leaf) any
}

type Tree interface {
	Accept(g TreeVisitor) any
}

func (r *Branch) Accept(v TreeVisitor) any { return v.VisitBranch(r) }
func (r *Leaf) Accept(v TreeVisitor) any   { return v.VisitLeaf(r) }

var (
	_ Tree = (*Branch)(nil)
	_ Tree = (*Leaf)(nil)
)

type TreeOneOf struct {
	Branch *Branch `json:",omitempty"`
	Leaf   *Leaf   `json:",omitempty"`
}

func (r *TreeOneOf) Accept(v TreeVisitor) any {
	switch {
	case r.Branch != nil:
		return v.VisitBranch(r.Branch)
	case r.Leaf != nil:
		return v.VisitLeaf(r.Leaf)
	default:
		panic("unexpected")
	}
}

func (r *TreeOneOf) Unwrap() Tree {
	switch {
	case r.Branch != nil:
		return r.Branch
	case r.Leaf != nil:
		return r.Leaf
	}

	return nil
}

var _ Tree = (*TreeOneOf)(nil)

type mapTreeToOneOf struct{}

func (t *mapTreeToOneOf) VisitBranch(v *Branch) any { return &TreeOneOf{Branch: v} }
func (t *mapTreeToOneOf) VisitLeaf(v *Leaf) any     { return &TreeOneOf{Leaf: v} }

var defaultMapTreeToOneOf TreeVisitor = &mapTreeToOneOf{}

func MapTreeToOneOf(v Tree) *TreeOneOf {
	return v.Accept(defaultMapTreeToOneOf).(*TreeOneOf)
}

func MustMatchTree[TOut any](
	x Tree,
	f1 func(x *Branch) TOut,
	f2 func(x *Leaf) TOut,
) TOut {
	return f.MustMatch2(x, f1, f2)
}
