// Package effect explores an algebraic effect system built from mkunion unions.
//
// The idea in one line: a program is data, and a handler gives that data meaning.
//
//   - Eff[Op, A] is the program. It is a union: Pure, Fail, Bind or Suspend.
//   - Op is any union of operations (see Effect in ops.go).
//   - Handler[Op] answers one operation at a time.
//   - Run walks the program and calls the handler until it reaches Pure or Fail.
//
// This file is the reusable core. Nothing here knows about a concrete operation.
//
// The type registry is off for this package. mkunion's registry generator
// mistakes the type parameter Op of Trace for a package type, and it ignores
// the noserde option on Eff. Both are generator bugs, not effect-system limits.
//
//go:tag mkunion:",no-type-registry"
package effect

import (
	"context"
	"fmt"
)

// --8<-- [start:eff-def]

// Eff is a program that performs operations of type Op and yields a value of type A.
//
// Op is the union of operations the program may ask for (see Effect).
// The continuation in Bind receives the handler's answer as `any`; the typed
// constructors in ops.go hide that cast from user code.
//
//go:tag mkunion:"Eff[Op, A],noserde"
type (
	// Pure is a finished program holding its result.
	Pure[Op, A any] struct{ Value A }
	// Fail is a finished program holding an error.
	Fail[Op, A any] struct{ Err error }
	// Bind asks the handler to perform Op, then continues with the answer.
	// When the handler fails, or the context is cancelled, Cont receives the
	// error instead of an answer, so the program can react or unwind.
	Bind[Op, A any] struct {
		Op   Op
		Cont func(answer any, err error) Eff[Op, A]
	}
	// Suspend is a program that is built on demand. Proc uses it so that a
	// direct-style body does not start before Run.
	Suspend[Op, A any] struct {
		Resume func() Eff[Op, A]
	}
)

// --8<-- [end:eff-def]

// Handler performs one operation and returns its answer.
// The answer type is `any` because Go interfaces cannot carry generic methods,
// even on Go 1.27. Typed wrappers live at the edges (see HandlerOf and Perform).
type Handler[Op any] func(ctx context.Context, op Op) (any, error)

// Return lifts a value into a finished program.
func Return[Op, A any](value A) Eff[Op, A] {
	return &Pure[Op, A]{Value: value}
}

// Throw lifts an error into a finished program.
func Throw[Op, A any](err error) Eff[Op, A] {
	return &Fail[Op, A]{Err: err}
}

// PerformAs asks the handler to perform op and expects an answer of type R.
// A wrong R becomes an error in Run. Prefer the typed Perform in ops.go, which
// ties R to the operation at compile time.
func PerformAs[Op, R any](op Op) Eff[Op, R] {
	return &Bind[Op, R]{
		Op: op,
		Cont: func(answer any, err error) Eff[Op, R] {
			if err != nil {
				return Throw[Op, R](err)
			}
			value, ok := answer.(R)
			if !ok {
				var want R
				return Throw[Op, R](fmt.Errorf("effect: handler answered %T to %T, want %T", answer, op, want))
			}
			return Return[Op](value)
		},
	}
}

// --8<-- [start:then]

// Then sequences two programs: run e, feed its value to k, run what k returns.
// The method form is Chain.Then below.
func Then[Op, A, B any](e Eff[Op, A], k func(A) Eff[Op, B]) Eff[Op, B] {
	return MatchEffR1(e,
		func(x *Pure[Op, A]) Eff[Op, B] { return k(x.Value) },
		func(x *Fail[Op, A]) Eff[Op, B] { return Throw[Op, B](x.Err) },
		func(x *Bind[Op, A]) Eff[Op, B] {
			return &Bind[Op, B]{
				Op:   x.Op,
				Cont: func(answer any, err error) Eff[Op, B] { return Then(x.Cont(answer, err), k) },
			}
		},
		func(x *Suspend[Op, A]) Eff[Op, B] {
			return &Suspend[Op, B]{Resume: func() Eff[Op, B] { return Then(x.Resume(), k) }}
		},
	)
}

// --8<-- [end:then]

// Map transforms the value of a program without performing anything new.
func Map[Op, A, B any](e Eff[Op, A], f func(A) B) Eff[Op, B] {
	return Then(e, func(a A) Eff[Op, B] { return Return[Op](f(a)) })
}

// --8<-- [start:run]

// Run interprets a program with a handler.
//
// It loops instead of recursing, so a program built from a million Binds
// does not grow the Go stack. A cancelled context is not performed; its error
// goes to the continuation like a handler error, so the program unwinds.
func Run[Op, A any](ctx context.Context, h Handler[Op], e Eff[Op, A]) (A, error) {
	var zero A
	for {
		switch x := e.(type) {
		case *Pure[Op, A]:
			return x.Value, nil
		case *Fail[Op, A]:
			return zero, x.Err
		case *Bind[Op, A]:
			var answer any
			err := ctx.Err()
			if err == nil {
				answer, err = h(ctx, x.Op)
			}
			e = x.Cont(answer, err)
		case *Suspend[Op, A]:
			e = x.Resume()
		default:
			return zero, fmt.Errorf("effect: unknown program node %T", e)
		}
	}
}

// --8<-- [end:run]

// Trace wraps a handler and records every operation it performs, in order.
// It is the smallest example of handler composition: handlers are functions,
// so middleware is a function that returns a function.
func Trace[Op any](h Handler[Op], sink *[]Op) Handler[Op] {
	return func(ctx context.Context, op Op) (any, error) {
		*sink = append(*sink, op)
		return h(ctx, op)
	}
}

// --8<-- [start:program-127]

// Chain wraps an Eff so that Then and Map can be methods.
//
// Go 1.27 lets a method declare its own type parameters, so p.Then(k) can
// introduce B. Eff itself is an interface, and interface methods still cannot
// have type parameters, so the union cannot carry Then; this wrapper does.
type Chain[Op, A any] struct{ Eff Eff[Op, A] }

// Start begins a chain.
func Start[Op, A any](e Eff[Op, A]) Chain[Op, A] {
	return Chain[Op, A]{Eff: e}
}

// Then is the method form of the package-level Then. B is a method type parameter.
func (p Chain[Op, A]) Then[B any](k func(A) Eff[Op, B]) Chain[Op, B] {
	return Chain[Op, B]{Eff: Then(p.Eff, k)}
}

// Map is the method form of the package-level Map.
func (p Chain[Op, A]) Map[B any](f func(A) B) Chain[Op, B] {
	return Chain[Op, B]{Eff: Map(p.Eff, f)}
}

// --8<-- [end:program-127]
