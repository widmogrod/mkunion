package effect

import "fmt"

// Composing effects from more than one union.
//
// A program is over one Op. When two packages each define their own union,
// an application defines a third union that wraps both, one variant per
// package, and lifts each package's programs into it. Go cannot build that
// union on the fly (no effect rows), but the wrapper is a few lines of
// mkunion, and Lift and Embed do the rest. See example/compose.

// --8<-- [start:lift]

// Lift turns a program over Sub into a program over Super.
// inject says how a Sub operation is spelled in Super, usually by wrapping it
// in the variant the application reserved for that package.
func Lift[Sub, Super, A any](e Eff[Sub, A], inject func(Sub) Super) Eff[Super, A] {
	return MatchEffR1(e,
		func(x *Pure[Sub, A]) Eff[Super, A] { return Return[Super](x.Value) },
		func(x *Fail[Sub, A]) Eff[Super, A] { return Throw[Super, A](x.Err) },
		func(x *Bind[Sub, A]) Eff[Super, A] {
			return &Bind[Super, A]{
				Op:   inject(x.Op),
				Cont: func(answer any, err error) Eff[Super, A] { return Lift(x.Cont(answer, err), inject) },
			}
		},
		func(x *Suspend[Sub, A]) Eff[Super, A] {
			return &Suspend[Super, A]{Resume: func() Eff[Super, A] { return Lift(x.Resume(), inject) }}
		},
	)
}

// Embed runs a program over Sub inside a direct-style body over Super.
// Every step of e goes through env, injected into Super, so the body's
// handler and middleware see it like any other operation.
func Embed[Super, Sub, A any](env *Env[Super], e Eff[Sub, A], inject func(Sub) Super) (A, error) {
	var zero A
	for {
		switch x := e.(type) {
		case *Pure[Sub, A]:
			return x.Value, nil
		case *Fail[Sub, A]:
			return zero, x.Err
		case *Bind[Sub, A]:
			e = x.Cont(env.perform(inject(x.Op)))
		case *Suspend[Sub, A]:
			e = x.Resume()
		default:
			return zero, fmt.Errorf("effect: unknown program node %T", e)
		}
	}
}

// --8<-- [end:lift]

// Unit is the answer of an operation that has nothing to return.
type Unit struct{}
