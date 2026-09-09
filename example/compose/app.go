// Package compose is an application that uses two effect packages, clock and
// mailer, in one program with one handler and one trace.
//
// There is no glue. Every operation generated with the `handler` option is an
// effect.Op, so programs from both packages are the same type as this one,
// and a handler is any value that has both packages' Handle methods. The one
// line this package adds, Handlers, asks the compiler to check that.
package compose

import (
	"context"
	"time"

	"github.com/widmogrod/mkunion/example/compose/clock"
	"github.com/widmogrod/mkunion/example/compose/mailer"
	"github.com/widmogrod/mkunion/x/effect"
)

// --8<-- [start:handlers]

// Handlers is what this application needs from the world: both packages,
// at compile time. A value that misses one method does not get past here.
type Handlers interface {
	clock.EffectHandler
	mailer.EffectHandler
}

// Interpret is effect.Interpret with that check in front.
func Interpret[A any](ctx context.Context, program effect.Eff[effect.Op, A], h Handlers, middleware ...effect.Middleware[effect.Op]) (A, error) {
	return effect.Interpret(ctx, program, h, middleware...)
}

// --8<-- [end:handlers]

// --8<-- [start:remind]

// Remind waits until at, mails name, and says when it did.
// Three steps, from two packages, in one plain Go body.
func Remind(name, msg string, at time.Time) effect.Eff[effect.Op, string] {
	return effect.Prog(func(fx effect.Fx) (string, error) {
		fx.Run(clock.WaitUntil(at))                 // a program clock ships
		receipt := fx.Run(mailer.Notify(name, msg)) // a program mailer ships
		sent := fx.Do(&clock.Now{})                 // one operation, a time.Time
		return receipt + " at " + sent.Format(time.Kitchen), nil
	})
}

// --8<-- [end:remind]
