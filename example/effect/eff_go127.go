package effect

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// This file uses Go 1.27 generic methods: a method may declare its own type
// parameters. Then, Map and Perform become methods, so programs read left to
// right instead of inside out.
//
// What generic methods do not buy us: Eff is an interface, and Go 1.27 still
// forbids type parameters on interface methods, so the union itself cannot carry
// Then. A concrete wrapper (Program) must, and the handler boundary stays `any`.

// --8<-- [start:program-127]

// Program wraps an Eff so that Then and Map can be methods.
type Program[Op, A any] struct{ Eff Eff[Op, A] }

// Start begins a chain.
func Start[Op, A any](e Eff[Op, A]) Program[Op, A] {
	return Program[Op, A]{Eff: e}
}

// Then is the method form of the package-level Then. B is a method type parameter.
func (p Program[Op, A]) Then[B any](k func(A) Eff[Op, B]) Program[Op, B] {
	return Program[Op, B]{Eff: Then(p.Eff, k)}
}

// Map is the method form of the package-level Map.
func (p Program[Op, A]) Map[B any](f func(A) B) Program[Op, B] {
	return Program[Op, B]{Eff: Map(p.Eff, f)}
}

// --8<-- [end:program-127]

// --8<-- [start:direct-127]

// Direct performs operations right now against one handler.
// Perform is one generic method that serves every operation: R is inferred from
// the operation's Result method, so `d.Perform(&Now{})` returns (time.Time, error).
type Direct struct {
	Ctx     context.Context
	Handler Handler[Effect]
}

// Perform runs one operation and returns its typed answer.
func (d Direct) Perform[R any](op EffectOf[R]) (R, error) {
	return PerformDirect(d.Ctx, d.Handler, op)
}

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

// --8<-- [end:direct-127]

// --8<-- [start:chained-127]

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

// --8<-- [end:chained-127]
