package effect

import (
	"time"

	"github.com/widmogrod/mkunion/f"
)

// --8<-- [start:ops-def]

// Unit is the answer of an operation that has nothing to return.
type Unit struct{}

// Effect is the set of operations our programs may ask for.
//
// Each variant embeds f.Returns[R] to declare the type of its answer. The
// `handler` option makes mkunion generate the typed layer from it:
// EffectHandler (one method per operation), EffectOf[R] (an Effect that
// answers with R), EffectHandlerFunc (the adapter Run uses) and
// EffectDefaults (zero answers, for tests). See ops_union_gen.go.
//
//go:tag mkunion:"Effect,handler"
type (
	// Log writes a line somewhere.
	Log struct {
		f.Returns[Unit]
		Msg string
	}
	// Now asks for the current time.
	Now struct{ f.Returns[time.Time] }
	// ReadFile asks for the content of a file.
	ReadFile struct {
		f.Returns[[]byte]
		Path string
	}
	// Random asks for a number in [0, Max).
	Random struct {
		f.Returns[int]
		Max int
	}
	// Send asks for a message to be delivered. It is not idempotent: sending twice sends two.
	Send struct {
		f.Returns[string]
		To, Msg string
	}
)

// --8<-- [end:ops-def]

// --8<-- [start:typed-layer]

// Perform asks for one operation as a program. R is inferred from the
// operation's f.Returns, so `Perform(&Now{})` is an Eff[Effect, time.Time].
func Perform[R any](op EffectOf[R]) Eff[Effect, R] {
	return PerformAs[Effect, R](op)
}

// --8<-- [end:typed-layer]
