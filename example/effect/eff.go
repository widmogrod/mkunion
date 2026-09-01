// Package effect explores an algebraic effect system built from mkunion unions.
//
// The idea in one line: a program is data, and a handler gives that data meaning.
//
//   - Eff[Op, A] is the program. It is a union: Pure, Fail or Bind.
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
	Bind[Op, A any] struct {
		Op   Op
		Cont func(answer any) Eff[Op, A]
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
		Cont: func(answer any) Eff[Op, R] {
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
// On Go 1.27 the same operation is available as a method, see Program.Then in eff_go127.go.
func Then[Op, A, B any](e Eff[Op, A], k func(A) Eff[Op, B]) Eff[Op, B] {
	return MatchEffR1(e,
		func(x *Pure[Op, A]) Eff[Op, B] { return k(x.Value) },
		func(x *Fail[Op, A]) Eff[Op, B] { return Throw[Op, B](x.Err) },
		func(x *Bind[Op, A]) Eff[Op, B] {
			return &Bind[Op, B]{
				Op:   x.Op,
				Cont: func(answer any) Eff[Op, B] { return Then(x.Cont(answer), k) },
			}
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
// does not grow the Go stack. Cancelled contexts stop the loop before the
// next operation.
func Run[Op, A any](ctx context.Context, h Handler[Op], e Eff[Op, A]) (A, error) {
	var zero A
	for {
		switch x := e.(type) {
		case *Pure[Op, A]:
			return x.Value, nil
		case *Fail[Op, A]:
			return zero, x.Err
		case *Bind[Op, A]:
			if err := ctx.Err(); err != nil {
				return zero, err
			}
			answer, err := h(ctx, x.Op)
			if err != nil {
				return zero, err
			}
			e = x.Cont(answer)
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
