// Code generated by mkunion. DO NOT EDIT.
package example

type CalcDefaultVisitor[A any] struct {
	Default A
	OnLit   func(x *Lit) A
	OnSum   func(x *Sum) A
	OnMul   func(x *Mul) A
}

func (t *CalcDefaultVisitor[A]) VisitLit(v *Lit) any {
	if t.OnLit != nil {
		return t.OnLit(v)
	}
	return t.Default
}
func (t *CalcDefaultVisitor[A]) VisitSum(v *Sum) any {
	if t.OnSum != nil {
		return t.OnSum(v)
	}
	return t.Default
}
func (t *CalcDefaultVisitor[A]) VisitMul(v *Mul) any {
	if t.OnMul != nil {
		return t.OnMul(v)
	}
	return t.Default
}
