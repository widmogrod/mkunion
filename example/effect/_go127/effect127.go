//go:build go1.27

// Package effect127 explores what Go 1.27 generic methods add to the effect system
// in example/effect. A method may now declare its own type parameters, so Then,
// Map and Perform become methods and programs read left to right.
//
// The directory name starts with an underscore, so `go test ./...` and
// `mkunion watch ./...` skip it. The module stays on an older Go version and the
// mkunion parser (built with the module's Go version) never sees generic methods.
// Run this package explicitly:
//
//	GOTOOLCHAIN=go1.27.0 go test ./example/effect/_go127/
//
// What generic methods do not buy us: an Eff is an interface, and Go 1.27 still
// forbids type parameters on interface methods, so the union itself cannot carry
// Then. A concrete wrapper (Program) must, and the handler boundary stays `any`.
package effect127

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/widmogrod/mkunion/example/effect"
)

// --8<-- [start:program-127]

// Program wraps an Eff so that Then and Map can be methods.
type Program[Op, A any] struct{ Eff effect.Eff[Op, A] }

// Start begins a chain.
func Start[Op, A any](e effect.Eff[Op, A]) Program[Op, A] {
	return Program[Op, A]{Eff: e}
}

// Then is the method form of effect.Then. B is a method type parameter.
func (p Program[Op, A]) Then[B any](k func(A) effect.Eff[Op, B]) Program[Op, B] {
	return Program[Op, B]{Eff: effect.Then(p.Eff, k)}
}

// Map is the method form of effect.Map.
func (p Program[Op, A]) Map[B any](f func(A) B) Program[Op, B] {
	return Program[Op, B]{Eff: effect.Map(p.Eff, f)}
}

// --8<-- [end:program-127]

// --8<-- [start:direct-127]

// Direct performs operations right now against one handler.
// Perform is one generic method that serves every operation: R is inferred from
// the operation's Result method, so `d.Perform(&effect.Now{})` returns (time.Time, error).
type Direct struct {
	Ctx     context.Context
	Handler effect.Handler[effect.Effect]
}

// Perform runs one operation and returns its typed answer.
func (d Direct) Perform[R any](op effect.EffectOf[R]) (R, error) {
	return effect.PerformDirect(d.Ctx, d.Handler, op)
}

// GreetDirect is effect.Greet in direct style: plain Go, early returns, no continuations.
// The price: there is no program value to inspect, trace ahead of time, or replay.
func GreetDirect(d Direct, path string) (string, error) {
	name, err := d.Perform(&effect.ReadFile{Path: path})
	if err != nil {
		return "", err
	}
	now, err := d.Perform(&effect.Now{})
	if err != nil {
		return "", err
	}
	msg := fmt.Sprintf("Hello %s, it is %s", strings.TrimSpace(string(name)), now.Format(time.Kitchen))
	if _, err := d.Perform(&effect.Log{Msg: msg}); err != nil {
		return "", err
	}
	return msg, nil
}

// --8<-- [end:direct-127]

// --8<-- [start:chained-127]

// GreetChained is effect.Greet with method chaining. Steps that only transform a
// value chain flat. Steps that need two earlier values still nest, as in any
// language without do-notation.
func GreetChained(path string) effect.Eff[effect.Effect, string] {
	return Start(effect.Perform(&effect.ReadFile{Path: path})).
		Map(func(raw []byte) string { return strings.TrimSpace(string(raw)) }).
		Then(func(name string) effect.Eff[effect.Effect, string] {
			return effect.Then(effect.Perform(&effect.Now{}), func(now time.Time) effect.Eff[effect.Effect, string] {
				msg := fmt.Sprintf("Hello %s, it is %s", name, now.Format(time.Kitchen))
				return effect.Map(effect.Perform(&effect.Log{Msg: msg}), func(effect.Unit) string { return msg })
			})
		}).Eff
}

// --8<-- [end:chained-127]
