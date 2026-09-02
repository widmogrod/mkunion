package effect

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// This file mirrors program.go and program_do.go a third time, with the most
// ergonomic surface we can build by hand today:
//
//   - Program[A] is a short name for Eff[Effect, A] (a generic alias).
//   - Prog lifts a plain body into a Program.
//   - Fx is the body's handle. It has one method per operation, plus a generic
//     Do and Attempt (Go 1.27 generic methods) for operations without a helper.
//   - Defaults is a handler tests can embed and override one method at a time.
//
// Everything here follows from the Effect union and the Result methods, and is
// what a `//go:tag mkeffect:"Effect"` generator would emit.

// --8<-- [start:fx-api]

// Program is a program over the Effect operations that yields A.
type Program[A any] = Eff[Effect, A]

// Fx is the handle a body uses to perform operations.
type Fx struct{ env *Env[Effect] }

// Prog turns a direct-style body into a Program.
func Prog[A any](body func(fx Fx) (A, error)) Program[A] {
	return Proc(func(e *Env[Effect]) (A, error) { return body(Fx{env: e}) })
}

// Do performs any operation and returns its typed answer. R is inferred.
func (fx Fx) Do[R any](op EffectOf[R]) R { return DoAs[Effect, R](fx.env, op) }

// Attempt is Do that returns the error instead of unwinding the body.
func (fx Fx) Attempt[R any](op EffectOf[R]) (R, error) { return AttemptAs[Effect, R](fx.env, op) }

// One method per operation.

func (fx Fx) Log(msg string)              { fx.Do(&Log{Msg: msg}) }
func (fx Fx) Now() time.Time              { return fx.Do(&Now{}) }
func (fx Fx) ReadFile(path string) []byte { return fx.Do(&ReadFile{Path: path}) }
func (fx Fx) Random(max int) int          { return fx.Do(&Random{Max: max}) }

// --8<-- [end:fx-api]

// --8<-- [start:defaults]

// Defaults is an EffectHandler with harmless answers. Embed it in a test
// handler and override only the methods the test cares about. The compiler
// still checks that the embedding type is a complete EffectHandler.
type Defaults struct{}

var _ EffectHandler = Defaults{}

func (Defaults) HandleLog(context.Context, *Log) (Unit, error)      { return Unit{}, nil }
func (Defaults) HandleNow(context.Context, *Now) (time.Time, error) { return time.Time{}, nil }
func (Defaults) HandleReadFile(_ context.Context, op *ReadFile) ([]byte, error) {
	return nil, fmt.Errorf("defaults: no file %q", op.Path)
}
func (Defaults) HandleRandom(context.Context, *Random) (int, error) { return 0, nil }

// --8<-- [end:defaults]

// --8<-- [start:greet-fx]

// GreetFx is Greet a third time. Compare with program.go and program_do.go.
func GreetFx(path string) Program[string] {
	return Prog(func(fx Fx) (string, error) {
		name := fx.ReadFile(path)
		now := fx.Now()
		msg := fmt.Sprintf("Hello %s, it is %s", strings.TrimSpace(string(name)), now.Format(time.Kitchen))
		fx.Log(msg)
		return msg, nil
	})
}

// --8<-- [end:greet-fx]

// --8<-- [start:roll-fx]

// RollUntilFx is RollUntil in the same style.
func RollUntilFx(want, maxRolls int) Program[int] {
	return Prog(func(fx Fx) (int, error) {
		for roll := 1; roll <= maxRolls; roll++ {
			if fx.Random(6)+1 == want {
				return roll, nil
			}
		}
		return 0, fmt.Errorf("no %d in %d rolls", want, maxRolls)
	})
}

// --8<-- [end:roll-fx]

// --8<-- [start:greet-or-guest-fx]

// GreetOrGuestFx handles a missing file in place with Attempt.
func GreetOrGuestFx(path string) Program[string] {
	return Prog(func(fx Fx) (string, error) {
		name, err := fx.Attempt(&ReadFile{Path: path})
		if err != nil {
			name = []byte("guest")
		}
		msg := "Hello " + strings.TrimSpace(string(name))
		fx.Log(msg)
		return msg, nil
	})
}

// --8<-- [end:greet-or-guest-fx]
