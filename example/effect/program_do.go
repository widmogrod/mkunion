package effect

import (
	"fmt"
	"strings"
	"time"
)

// This file mirrors program.go in direct style. Same programs, same handlers,
// same traces; the bodies are plain Go instead of nested continuations.

// --8<-- [start:do-helpers]

// Do performs one operation inside a Proc body and returns its typed answer.
// R is inferred from the operation's Result method.
func Do[R any](e *Env[Effect], op EffectOf[R]) R {
	return DoAs[Effect, R](e, op)
}

// Attempt is Do that returns the error instead of unwinding the body.
func Attempt[R any](e *Env[Effect], op EffectOf[R]) (R, error) {
	return AttemptAs[Effect, R](e, op)
}

// One helper per operation. They are mechanical, like the typed layer in
// ops.go, and are what a generator would emit. Hand-written for now.

func DoLog(e *Env[Effect], msg string) { Do(e, &Log{Msg: msg}) }

func DoNow(e *Env[Effect]) time.Time { return Do(e, &Now{}) }

func DoReadFile(e *Env[Effect], path string) []byte { return Do(e, &ReadFile{Path: path}) }

func DoRandom(e *Env[Effect], max int) int { return Do(e, &Random{Max: max}) }

// --8<-- [end:do-helpers]

// --8<-- [start:greet-do]

// GreetDo is Greet written in direct style. Compare with Greet in program.go.
func GreetDo(path string) Eff[Effect, string] {
	return Proc(func(e *Env[Effect]) (string, error) {
		name := DoReadFile(e, path)
		now := DoNow(e)
		msg := fmt.Sprintf("Hello %s, it is %s", strings.TrimSpace(string(name)), now.Format(time.Kitchen))
		DoLog(e, msg)
		return msg, nil
	})
}

// --8<-- [end:greet-do]

// --8<-- [start:roll-do]

// RollUntilDo is RollUntil written in direct style. A loop is a loop.
func RollUntilDo(want, maxRolls int) Eff[Effect, int] {
	return Proc(func(e *Env[Effect]) (int, error) {
		for roll := 1; roll <= maxRolls; roll++ {
			if DoRandom(e, 6)+1 == want {
				return roll, nil
			}
		}
		return 0, fmt.Errorf("no %d in %d rolls", want, maxRolls)
	})
}

// --8<-- [end:roll-do]

// --8<-- [start:greet-or-guest]

// GreetOrGuest shows Attempt: a missing file is not fatal, the guest gets a greeting too.
func GreetOrGuest(path string) Eff[Effect, string] {
	return Proc(func(e *Env[Effect]) (string, error) {
		name, err := Attempt(e, &ReadFile{Path: path})
		if err != nil {
			name = []byte("guest")
		}
		msg := "Hello " + strings.TrimSpace(string(name))
		DoLog(e, msg)
		return msg, nil
	})
}

// --8<-- [end:greet-or-guest]
