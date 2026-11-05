// Package compose is an application that uses three effect packages, clock,
// mailer and billing, in one program with one handler and one trace.
//
// There is no glue. Every union variant that embeds f.Returns is an
// effect.Op, so programs from every package are the same type as this one,
// and a handler is any value that has every package's Handle methods. The one
// line this package adds, Handlers, asks the compiler to check that.
package compose

import (
	"context"
	"time"

	"github.com/widmogrod/mkunion/example/compose/billing"
	"github.com/widmogrod/mkunion/example/compose/clock"
	"github.com/widmogrod/mkunion/example/compose/mailer"
	"github.com/widmogrod/mkunion/f"
	"github.com/widmogrod/mkunion/x/effect"
)

// --8<-- [start:handlers]

// Handlers is what this application needs from the world: three packages,
// at compile time. A value that misses one method does not get past here.
type Handlers interface {
	clock.EffectHandler
	mailer.EffectHandler
	billing.EffectHandler
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

// --8<-- [start:paid]

// ErrNotSent says a paid message was not sent, and why, as a billing refusal.
// A caller that only speaks error can still match on the reason.
type ErrNotSent struct{ Refused *billing.Refused }

func (e *ErrNotSent) Error() string { return "not sent: " + e.Refused.Error() }
func (e *ErrNotSent) Unwrap() error { return e.Refused }

// SendPaid charges price, then mails. Every refusal is an answer, and the
// body decides each one differently:
//
//   - too fast: the library program already waited and tried again
//   - quota exceeded: this application waits for the reset, once
//   - invalid amount, out of credits: give up, as an ErrNotSent
//
// A Go error from any step, a broken line, is none of this body's business.
// It stops the body, and Retry middleware around Interpret gets its say.
func SendPaid(name, msg string, price int) effect.Eff[effect.Op, string] {
	return effect.Prog(func(fx effect.Fx) (string, error) {
		outcome := fx.Run(billing.ChargeWithPatience(price, 2))

		if refused, isRefused := outcome.(*f.Err[billing.Receipt, billing.ChargeError]); isRefused {
			if quota, ok := refused.Error.(*billing.QuotaExceeded); ok {
				fx.Run(clock.WaitUntil(quota.ResetAt))
				outcome = fx.Do(&billing.Charge{Amount: price}) // once; a second refusal stands
			}
		}

		return f.MatchResultR2(outcome,
			func(ok *f.Ok[billing.Receipt, billing.ChargeError]) (string, error) {
				receipt := fx.Run(mailer.Notify(name, msg))
				return receipt + " paid by " + ok.Value.ID, nil
			},
			func(refused *f.Err[billing.Receipt, billing.ChargeError]) (string, error) {
				return "", &ErrNotSent{Refused: &billing.Refused{Reason: refused.Error}}
			},
		)
	})
}

// --8<-- [end:paid]

// --8<-- [start:recomposed]

// SendPaidOrQueue is SendPaid with a fallback, built from primitives around
// the finished program: when it fails for any reason, queue the message
// instead. OrElse is Catch, which is Then for the failure side. No body was
// opened, and SendPaid was not changed.
func SendPaidOrQueue(name, msg string, price int) effect.Eff[effect.Op, string] {
	return effect.OrElse(
		SendPaid(name, msg, price),
		effect.Map(effect.Perform(&mailer.Resolve{Name: name}), func(to string) string {
			return "queued for " + to
		}),
	)
}

// --8<-- [end:recomposed]
