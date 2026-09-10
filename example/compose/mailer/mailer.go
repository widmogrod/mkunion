// Package mailer is a library that owns one effect union: mail.
//
// Like clock, it knows nothing about the applications that use it. It ships
// the union and one program built from its own operations (Notify). The test
// in this package shows it can be tested on its own.
package mailer

import (
	"github.com/widmogrod/mkunion/f"
	"github.com/widmogrod/mkunion/x/effect"
)

// --8<-- [start:ops]

// Effect is what this package can ask for.
//
//go:tag mkunion:"Effect"
type (
	// Resolve asks for the address behind a name.
	Resolve struct {
		f.Returns[string]
		Name string
	}
	// Send asks for a message to be delivered, and answers with a receipt.
	Send struct {
		f.Returns[string]
		To, Msg string
	}
)

// Notify is a program this package ships: resolve a name, then send to it.
func Notify(name, msg string) effect.Eff[effect.Op, string] {
	return effect.Then(effect.Perform(&Resolve{Name: name}), func(to string) effect.Eff[effect.Op, string] {
		return effect.Perform(&Send{To: to, Msg: msg})
	})
}

// --8<-- [end:ops]
