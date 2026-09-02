package effect

import (
	"fmt"
	"strings"
	"time"
)

// --8<-- [start:greet]

// Greet reads a name from a file, asks for the time, logs a greeting and returns it.
//
// Nothing happens when you call Greet. It only builds a value of type
// Eff[Effect, string]. Run gives it meaning with a handler of your choice.
func Greet(path string) Eff[Effect, string] {
	return Then(Perform(&ReadFile{Path: path}), func(name []byte) Eff[Effect, string] {
		return Then(Perform(&Now{}), func(now time.Time) Eff[Effect, string] {
			msg := fmt.Sprintf("Hello %s, it is %s", strings.TrimSpace(string(name)), now.Format(time.Kitchen))
			return Then(Perform(&Log{Msg: msg}), func(Unit) Eff[Effect, string] {
				return Return[Effect](msg)
			})
		})
	})
}

// --8<-- [end:greet]

// --8<-- [start:roll]

// RollUntil rolls a die until it shows want, and returns how many rolls it took.
// It gives up with an error after maxRolls.
//
// The recursion builds the program lazily inside continuations, so a run of a
// million rolls is a loop in Run, not a million stack frames.
func RollUntil(want, maxRolls int) Eff[Effect, int] {
	return rollFrom(1, want, maxRolls)
}

func rollFrom(roll, want, maxRolls int) Eff[Effect, int] {
	if roll > maxRolls {
		return Throw[Effect, int](fmt.Errorf("no %d in %d rolls", want, maxRolls))
	}
	return Then(Perform(&Random{Max: 6}), func(got int) Eff[Effect, int] {
		if got+1 == want {
			return Return[Effect](roll)
		}
		return rollFrom(roll+1, want, maxRolls)
	})
}

// --8<-- [end:roll]

// --8<-- [start:greet-direct]

// GreetDirect is Greet in direct style: plain Go, early returns, no continuations.
// The price: there is no program value to inspect, trace ahead of time, or replay.
func GreetDirect(d Direct, path string) (string, error) {
	name, err := d.Perform(&ReadFile{Path: path})
	if err != nil {
		return "", err
	}
	now, err := d.Perform(&Now{})
	if err != nil {
		return "", err
	}
	msg := fmt.Sprintf("Hello %s, it is %s", strings.TrimSpace(string(name)), now.Format(time.Kitchen))
	if _, err := d.Perform(&Log{Msg: msg}); err != nil {
		return "", err
	}
	return msg, nil
}

// --8<-- [end:greet-direct]

// --8<-- [start:greet-chained]

// GreetChained is Greet with method chaining. Steps that only transform a value
// chain flat. Steps that need two earlier values still nest, as in any language
// without do-notation.
func GreetChained(path string) Eff[Effect, string] {
	return Start(Perform(&ReadFile{Path: path})).
		Map(func(raw []byte) string { return strings.TrimSpace(string(raw)) }).
		Then(func(name string) Eff[Effect, string] {
			return Then(Perform(&Now{}), func(now time.Time) Eff[Effect, string] {
				msg := fmt.Sprintf("Hello %s, it is %s", name, now.Format(time.Kitchen))
				return Map(Perform(&Log{Msg: msg}), func(Unit) string { return msg })
			})
		}).Eff
}

// --8<-- [end:greet-chained]
