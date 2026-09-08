package effect

import (
	"context"
	"time"
)

// --8<-- [start:ops-def]

// Unit is the answer of an operation that has nothing to return.
type Unit struct{}

// Effect is the set of operations our programs may ask for.
//
// Each variant declares the type of its answer with a phantom Result method.
// The method is never called. It exists so the compiler can tie an operation
// to its answer type, in Fx.Do and in HandlerOf.
//
//go:tag mkunion:"Effect"
type (
	// Log writes a line somewhere.
	Log struct{ Msg string }
	// Now asks for the current time.
	Now struct{}
	// ReadFile asks for the content of a file.
	ReadFile struct{ Path string }
	// Random asks for a number in [0, Max).
	Random struct{ Max int }
	// Send asks for a message to be delivered. It is not idempotent: sending twice sends two.
	Send struct{ To, Msg string }
)

func (*Log) Result() Unit        { return Unit{} }
func (*Now) Result() time.Time   { return time.Time{} }
func (*ReadFile) Result() []byte { return nil }
func (*Random) Result() int      { return 0 }
func (*Send) Result() string     { return "" }

// --8<-- [end:ops-def]

// --8<-- [start:typed-layer]

// Everything below this line is mechanical. It follows from the Effect union
// and the Result methods, and is what a `//go:tag mkeffect:"Effect"` generator
// would emit. Today it is written by hand.

// EffectOf is an Effect that answers with R.
type EffectOf[R any] interface {
	Effect
	Result() R
}

// Perform asks for one operation as a program. R is inferred from the
// operation's Result method, so `Perform(&Now{})` is an Eff[Effect, time.Time].
func Perform[R any](op EffectOf[R]) Eff[Effect, R] {
	return PerformAs[Effect, R](op)
}

// EffectHandler answers each operation with its declared type.
// Adding a variant to Effect breaks this interface and HandlerOf at compile time,
// which is the point: every handler must cover every operation.
type EffectHandler interface {
	HandleLog(ctx context.Context, op *Log) (Unit, error)
	HandleNow(ctx context.Context, op *Now) (time.Time, error)
	HandleReadFile(ctx context.Context, op *ReadFile) ([]byte, error)
	HandleRandom(ctx context.Context, op *Random) (int, error)
	HandleSend(ctx context.Context, op *Send) (string, error)
}

// HandlerOf adapts a typed EffectHandler to the untyped Handler the core runs.
// MatchEffectR2 is exhaustive, and answer checks that each Handle method returns
// the type the operation declared. Both checks happen at compile time.
func HandlerOf(h EffectHandler) Handler[Effect] {
	return func(ctx context.Context, op Effect) (any, error) {
		return MatchEffectR2(op,
			func(x *Log) (any, error) { r, err := h.HandleLog(ctx, x); return answer(x, r, err) },
			func(x *Now) (any, error) { r, err := h.HandleNow(ctx, x); return answer(x, r, err) },
			func(x *ReadFile) (any, error) { r, err := h.HandleReadFile(ctx, x); return answer(x, r, err) },
			func(x *Random) (any, error) { r, err := h.HandleRandom(ctx, x); return answer(x, r, err) },
			func(x *Send) (any, error) { r, err := h.HandleSend(ctx, x); return answer(x, r, err) },
		)
	}
}

// answer only compiles when r has the type op declared in Result.
func answer[R any](_ EffectOf[R], r R, err error) (any, error) {
	return r, err
}

// --8<-- [end:typed-layer]
