// Package clock is a library that owns one effect union: time.
//
// It knows nothing about the applications that use it. It ships the union,
// a typed Perform, one program built from its own operations (WaitUntil),
// and a handler for the real clock. An application composes it with other
// packages; see example/compose.
package clock

import (
	"context"
	"time"

	"github.com/widmogrod/mkunion/f"
	"github.com/widmogrod/mkunion/x/effect"
)

// --8<-- [start:ops]

// Effect is what this package can ask for.
//
//go:tag mkunion:"Effect,handler"
type (
	// Now asks for the current time.
	Now struct{ f.Returns[time.Time] }
	// Sleep asks to wait.
	Sleep struct {
		f.Returns[effect.Unit]
		For time.Duration
	}
)

// Perform asks for one clock operation as a program. R comes from f.Returns.
func Perform[R any](op EffectOf[R]) effect.Eff[Effect, R] {
	return effect.PerformAs[Effect, R](op)
}

// WaitUntil is a program this package ships: read the clock, then sleep the
// difference. An application lifts it into its own union (see example/compose).
func WaitUntil(t time.Time) effect.Eff[Effect, effect.Unit] {
	return effect.Then(Perform(&Now{}), func(now time.Time) effect.Eff[Effect, effect.Unit] {
		if !t.After(now) {
			return effect.Return[Effect](effect.Unit{})
		}
		return Perform(&Sleep{For: t.Sub(now)})
	})
}

// --8<-- [end:ops]

// System answers from the wall clock.
type System struct{}

var _ EffectHandler = System{}

func (System) HandleNow(context.Context, *Now) (time.Time, error) { return time.Now(), nil }

func (System) HandleSleep(ctx context.Context, op *Sleep) (effect.Unit, error) {
	select {
	case <-ctx.Done():
		return effect.Unit{}, ctx.Err()
	case <-time.After(op.For):
		return effect.Unit{}, nil
	}
}
