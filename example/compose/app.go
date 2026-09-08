// Package compose is an application that composes two effect packages,
// clock and mailer, into one program with one handler and one trace.
//
// Go has no effect rows, so the application declares the union of unions
// itself: one variant per package. Everything else is x/effect: Lift and
// Embed carry the packages' programs into that union, and the generated
// handlers of each package answer their own operations.
package compose

import (
	"context"
	"time"

	"github.com/widmogrod/mkunion/example/compose/clock"
	"github.com/widmogrod/mkunion/example/compose/mailer"
	"github.com/widmogrod/mkunion/x/effect"
)

// --8<-- [start:union]

// AppEff is the application's union: one variant per package it composes.
// Adding a package is adding a variant; the compiler then points at Handler.
//
//go:tag mkunion:"AppEff,noserde"
type (
	ClockOp struct{ Op clock.Effect }
	MailOp  struct{ Op mailer.Effect }
)

// fromClock and fromMail say how a package's operation is spelled in AppEff.
func fromClock(op clock.Effect) AppEff { return &ClockOp{Op: op} }
func fromMail(op mailer.Effect) AppEff { return &MailOp{Op: op} }

// Handler routes every operation to the handler of the package that owns it.
// Each package's handler is the typed one mkunion generated for its union.
func Handler(c clock.EffectHandler, m mailer.EffectHandler) effect.Handler[AppEff] {
	answerClock, answerMail := clock.EffectHandlerFunc(c), mailer.EffectHandlerFunc(m)
	return func(ctx context.Context, op AppEff) (any, error) {
		return MatchAppEffR2(op,
			func(x *ClockOp) (any, error) { return answerClock(ctx, x.Op) },
			func(x *MailOp) (any, error) { return answerMail(ctx, x.Op) },
		)
	}
}

// --8<-- [end:union]

// --8<-- [start:fx]

// Program is a program over both packages' operations.
type Program[A any] = effect.Eff[AppEff, A]

// Fx is the handle a program body uses. One generic method per package
// performs any of that package's operations, typed by its f.Returns. Two
// more methods embed the programs the packages ship.
type Fx struct{ env *effect.Env[AppEff] }

// Prog turns a plain Go body into a Program.
func Prog[A any](body func(fx Fx) (A, error)) Program[A] {
	return effect.Proc(func(e *effect.Env[AppEff]) (A, error) { return body(Fx{env: e}) })
}

// Clock performs one clock operation: fx.Clock(&clock.Now{}) is a time.Time.
func (fx Fx) Clock[R any](op clock.EffectOf[R]) R {
	return effect.DoAs[AppEff, R](fx.env, fromClock(op))
}

// Mail performs one mail operation: fx.Mail(&mailer.Send{...}) is a receipt.
func (fx Fx) Mail[R any](op mailer.EffectOf[R]) R {
	return effect.DoAs[AppEff, R](fx.env, fromMail(op))
}

// WaitUntil embeds clock.WaitUntil: its steps go through this body's handler.
func (fx Fx) WaitUntil(t time.Time) {
	embed(fx, clock.WaitUntil(t), fromClock)
}

// Notify embeds mailer.Notify.
func (fx Fx) Notify(name, msg string) string {
	return embed(fx, mailer.Notify(name, msg), fromMail)
}

// embed runs a package's program inside the body. An error unwinds the body,
// the same as a failed fx.Clock or fx.Mail.
func embed[Sub, A any](fx Fx, program effect.Eff[Sub, A], inject func(Sub) AppEff) A {
	value, err := effect.Embed(fx.env, program, inject)
	if err != nil {
		fx.env.Unwind(err)
	}
	return value
}

// Interpret runs a program with one handler per package, and middleware
// over both. Middleware is listed outermost first.
func Interpret[A any](ctx context.Context, program Program[A], c clock.EffectHandler, m mailer.EffectHandler, middleware ...effect.Middleware[AppEff]) (A, error) {
	return effect.Run(ctx, effect.Wrap(Handler(c, m), middleware...), program)
}

// --8<-- [end:fx]

// --8<-- [start:remind]

// Remind waits until at, mails name, and says when it did.
// Three steps, from two packages, in one plain Go body.
func Remind(name, msg string, at time.Time) Program[string] {
	return Prog(func(fx Fx) (string, error) {
		fx.WaitUntil(at)
		receipt := fx.Notify(name, msg)
		sent := fx.Clock(&clock.Now{})
		return receipt + " at " + sent.Format(time.Kitchen), nil
	})
}

// --8<-- [end:remind]
