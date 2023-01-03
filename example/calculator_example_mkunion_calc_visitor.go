// Code generated by mkunion. DO NOT EDIT.
package example

import (
	"github.com/widmogrod/mkunion/f"
)

type CalcVisitor interface {
	VisitLit(v *Lit) any
	VisitSum(v *Sum) any
	VisitMul(v *Mul) any
}

type Calc interface {
	Accept(g CalcVisitor) any
}

func (r *Lit) Accept(v CalcVisitor) any { return v.VisitLit(r) }
func (r *Sum) Accept(v CalcVisitor) any { return v.VisitSum(r) }
func (r *Mul) Accept(v CalcVisitor) any { return v.VisitMul(r) }

var (
	_ Calc = (*Lit)(nil)
	_ Calc = (*Sum)(nil)
	_ Calc = (*Mul)(nil)
)

type CalcOneOf struct {
	Lit *Lit `json:",omitempty"`
	Sum *Sum `json:",omitempty"`
	Mul *Mul `json:",omitempty"`
}

func (r *CalcOneOf) Accept(v CalcVisitor) any {
	switch {
	case r.Lit != nil:
		return v.VisitLit(r.Lit)
	case r.Sum != nil:
		return v.VisitSum(r.Sum)
	case r.Mul != nil:
		return v.VisitMul(r.Mul)
	default:
		panic("unexpected")
	}
}

var _ Calc = (*CalcOneOf)(nil)

type mapCalcToOneOf struct{}

func (t *mapCalcToOneOf) VisitLit(v *Lit) any { return &CalcOneOf{Lit: v} }
func (t *mapCalcToOneOf) VisitSum(v *Sum) any { return &CalcOneOf{Sum: v} }
func (t *mapCalcToOneOf) VisitMul(v *Mul) any { return &CalcOneOf{Mul: v} }

var defaultMapCalcToOneOf CalcVisitor = &mapCalcToOneOf{}

func MapCalcToOneOf(v Calc) *CalcOneOf {
	return v.Accept(defaultMapCalcToOneOf).(*CalcOneOf)
}

func MustMatchCalc[TOut any](
	x Calc,
	f1 func(x *Lit) TOut,
	f2 func(x *Sum) TOut,
	f3 func(x *Mul) TOut,
) TOut {
	return f.MustMatch3(x, f1, f2, f3)
}