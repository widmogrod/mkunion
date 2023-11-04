// Code generated by mkunion. DO NOT EDIT.
package schema

import (
	"github.com/widmogrod/mkunion/f"
)

type LocationVisitor interface {
	VisitLocationField(v *LocationField) any
	VisitLocationIndex(v *LocationIndex) any
	VisitLocationAnything(v *LocationAnything) any
}

type Location interface {
	AcceptLocation(g LocationVisitor) any
}

func (r *LocationField) AcceptLocation(v LocationVisitor) any    { return v.VisitLocationField(r) }
func (r *LocationIndex) AcceptLocation(v LocationVisitor) any    { return v.VisitLocationIndex(r) }
func (r *LocationAnything) AcceptLocation(v LocationVisitor) any { return v.VisitLocationAnything(r) }

var (
	_ Location = (*LocationField)(nil)
	_ Location = (*LocationIndex)(nil)
	_ Location = (*LocationAnything)(nil)
)

func MatchLocation[TOut any](
	x Location,
	f1 func(x *LocationField) TOut,
	f2 func(x *LocationIndex) TOut,
	f3 func(x *LocationAnything) TOut,
	df func(x Location) TOut,
) TOut {
	return f.Match3(x, f1, f2, f3, df)
}

func MatchLocationR2[TOut1, TOut2 any](
	x Location,
	f1 func(x *LocationField) (TOut1, TOut2),
	f2 func(x *LocationIndex) (TOut1, TOut2),
	f3 func(x *LocationAnything) (TOut1, TOut2),
	df func(x Location) (TOut1, TOut2),
) (TOut1, TOut2) {
	return f.Match3R2(x, f1, f2, f3, df)
}

func MustMatchLocation[TOut any](
	x Location,
	f1 func(x *LocationField) TOut,
	f2 func(x *LocationIndex) TOut,
	f3 func(x *LocationAnything) TOut,
) TOut {
	return f.MustMatch3(x, f1, f2, f3)
}

func MustMatchLocationR0(
	x Location,
	f1 func(x *LocationField),
	f2 func(x *LocationIndex),
	f3 func(x *LocationAnything),
) {
	f.MustMatch3R0(x, f1, f2, f3)
}

func MustMatchLocationR2[TOut1, TOut2 any](
	x Location,
	f1 func(x *LocationField) (TOut1, TOut2),
	f2 func(x *LocationIndex) (TOut1, TOut2),
	f3 func(x *LocationAnything) (TOut1, TOut2),
) (TOut1, TOut2) {
	return f.MustMatch3R2(x, f1, f2, f3)
}