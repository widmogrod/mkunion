package effect

import (
	"context"
	"fmt"
)

// One Op type for every union.
//
// A program is over one Op. With the `handler` option, mkunion gives every
// variant of every union two methods, Perform and Answer, that hand the
// variant to any handler value carrying that union's Handle methods. So all
// those variants satisfy Op, programs from different packages are the same
// type, Eff[Op, A], and one handler value that embeds each package's handler
// answers them all. No wrapper union, no lifting. See example/compose.
//
// The price: the compiler no longer proves that a handler covers every
// package. Ask it to, with an intersection interface at the edge:
//
//	type Handlers interface{ clock.EffectHandler; mailer.EffectHandler }
//	func Interpret(ctx, p, h Handlers)
//
// A handler that lacks a package's methods fails at the first operation of
// that package, with an error that names both.

// --8<-- [start:op]

// Op is an operation from any union generated with the `handler` option.
type Op interface {
	// Perform hands the operation to h, which must carry the Handle methods
	// of the operation's union, and returns the answer untyped.
	Perform(ctx context.Context, h any) (any, error)
}

// OpOf is an Op that answers with R. Every generated variant satisfies
// exactly one instantiation, so R is inferred from the operation.
type OpOf[R any] interface {
	Op
	Answer(ctx context.Context, h any) (R, error)
}

// Perform asks for one operation as a program.
func Perform[R any](op OpOf[R]) Eff[Op, R] {
	return PerformAs[Op, R](op)
}

// HandlerOf turns a handler value into the function Run uses: each operation
// performs itself against h.
func HandlerOf(h any) Handler[Op] {
	return func(ctx context.Context, op Op) (any, error) {
		if op == nil {
			return nil, fmt.Errorf("effect: nil operation")
		}
		return op.Perform(ctx, h)
	}
}

// Interpret runs a program against a handler value, with optional middleware
// listed outermost first.
func Interpret[A any](ctx context.Context, program Eff[Op, A], h any, middleware ...Middleware[Op]) (A, error) {
	return Run(ctx, Wrap(HandlerOf(h), middleware...), program)
}

// --8<-- [end:op]

// --8<-- [start:fx]

// Fx is the handle a direct-style body uses to perform operations from any
// union, and to run programs from any package, in place.
type Fx struct{ env *Env[Op] }

// Prog turns a plain Go body into a program over Op.
func Prog[A any](body func(fx Fx) (A, error)) Eff[Op, A] {
	return Proc(func(e *Env[Op]) (A, error) { return body(Fx{env: e}) })
}

// Do performs op and returns its answer, typed by the operation. On error
// the body stops and the program fails with that error.
func (fx Fx) Do[R any](op OpOf[R]) R { return DoAs[Op, R](fx.env, op) }

// Attempt is Do that returns the error instead of stopping the body.
func (fx Fx) Attempt[R any](op OpOf[R]) (R, error) { return AttemptAs[Op, R](fx.env, op) }

// Run runs a program in place: each of its steps goes through this body's
// handler and middleware. On error the body stops.
func (fx Fx) Run[A any](program Eff[Op, A]) A {
	value, err := fx.TryRun(program)
	if err != nil {
		fx.env.Unwind(err)
	}
	return value
}

// TryRun is Run that returns the error instead of stopping the body.
func (fx Fx) TryRun[A any](program Eff[Op, A]) (A, error) {
	return Embed(fx.env, program, func(op Op) Op { return op })
}

// --8<-- [end:fx]
