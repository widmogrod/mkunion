package effect

import (
	"context"
	"fmt"
)

// This file is the reusable core. Nothing here knows about a concrete
// operation. Eff[Op, A] is the program, Handler[Op] answers one operation,
// Run walks the program and calls the handler until it reaches Pure or Fail.

// --8<-- [start:eff-def]

// Eff is a program that performs operations of type Op and yields a value of type A.
//
// Op is the union of operations the program may ask for (see MyEff in ops.go).
// The continuation in Bind receives the handler's answer as `any`; the typed
// layer a union generates (the `handler` option) hides that cast from user code.
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
	// plain Go body does not start before Run.
	Suspend[Op, A any] struct {
		Resume func() Eff[Op, A]
	}
)

// --8<-- [end:eff-def]

// --8<-- [start:handler]

// Handler performs one operation and returns its answer.
// The answer type is `any` because Go interfaces cannot carry generic methods,
// even on Go 1.27. Typed wrappers live at the edges (see the `handler` union option).
type Handler[Op any] func(ctx context.Context, op Op) (any, error)

// Middleware wraps a handler and returns a handler. Because a handler is one
// function, one middleware sees every operation of every program.
type Middleware[Op any] func(Handler[Op]) Handler[Op]

// Wrap applies middleware to a handler, first listed outermost:
// Wrap(h, A, B) is A(B(h)), so A sees an operation first and its answer last.
func Wrap[Op any](h Handler[Op], middleware ...Middleware[Op]) Handler[Op] {
	for i := len(middleware) - 1; i >= 0; i-- {
		h = middleware[i](h)
	}
	return h
}

// --8<-- [end:handler]

// Return lifts a value into a finished program.
func Return[Op, A any](value A) Eff[Op, A] {
	return &Pure[Op, A]{Value: value}
}

// Throw lifts an error into a finished program.
func Throw[Op, A any](err error) Eff[Op, A] {
	return &Fail[Op, A]{Err: err}
}

// PerformAs asks the handler to perform op and expects an answer of type R.
// A wrong R becomes an error in Run. Prefer a typed Perform built on XOf[R],
// which ties R to the operation at compile time (see example/welcome/ops.go).
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
// It is a pattern match over the variants, so a new variant cannot be forgotten.
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

// --8<-- [start:trace]

// Trace records every operation a handler performs, in order.
// It is the smallest example of middleware (see middleware.go for more).
func Trace[Op any](sink *[]Op) Middleware[Op] {
	return func(h Handler[Op]) Handler[Op] {
		return func(ctx context.Context, op Op) (any, error) {
			*sink = append(*sink, op)
			return h(ctx, op)
		}
	}
}

// --8<-- [end:trace]
