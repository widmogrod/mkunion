// Package billing is a library that owns one effect union: money.
//
// Its one operation, Charge, can be refused in four ways. Every refusal is
// an answer, a value the program must look at, not a Go error. A Go error
// from Charge means the line to the bank broke, and that is for Retry
// middleware, never for a program.
//
// The package also shows programs built from primitives instead of a body:
// ChargeWithPatience is Then, Return and a match, and it composes clock's
// Sleep with its own Charge, across packages, with no glue.
package billing

import (
	"strconv"
	"time"

	"github.com/widmogrod/mkunion/example/compose/clock"
	"github.com/widmogrod/mkunion/f"
	"github.com/widmogrod/mkunion/x/effect"
)

// --8<-- [start:ops]

// Receipt is the proof of a charge that went through.
type Receipt struct{ ID string }

// ChargeError is every way a charge can be refused. The bank did its job
// and said no. A program decides what to do with each one; retry middleware
// never sees them, because they are values.
//
//go:tag mkunion:"ChargeError"
type (
	// InvalidAmount is a validation failure: the request itself is wrong.
	// Trying again with the same request cannot help.
	InvalidAmount struct{ Reason string }
	// RateLimited means: too fast. Try again after RetryAfter.
	RateLimited struct{ RetryAfter time.Duration }
	// QuotaExceeded means: too much today. Try again after ResetAt.
	QuotaExceeded struct{ ResetAt time.Time }
	// OutOfCredits means the account is short by Missing. Only money helps.
	OutOfCredits struct{ Missing int }
)

// Outcome is what a Charge answers with.
type Outcome = f.Result[Receipt, ChargeError]

// Effect is what this package can ask for.
//
//go:tag mkunion:"Effect,handler"
type (
	// Charge asks to take Amount from the account.
	Charge struct {
		f.Returns[Outcome]
		Amount int
	}
)

// --8<-- [end:ops]

// --8<-- [start:patience]

// ChargeWithPatience charges, and when the bank says "too fast", waits as
// long as it asks and tries again, up to patience times. Every other
// refusal is returned as it is: the caller decides.
//
// This is a program built from primitives, with no body: Then continues
// on the answer, Return ends, and clock.Sleep is a step like any other.
// Because it is a value, it can be traced, replayed and diffed like Remind.
func ChargeWithPatience(amount, patience int) effect.Eff[effect.Op, Outcome] {
	return effect.Then(effect.Perform(&Charge{Amount: amount}), func(outcome Outcome) effect.Eff[effect.Op, Outcome] {
		return f.MatchResultR1(outcome,
			func(*f.Ok[Receipt, ChargeError]) effect.Eff[effect.Op, Outcome] {
				return effect.Return[effect.Op](outcome)
			},
			func(refused *f.Err[Receipt, ChargeError]) effect.Eff[effect.Op, Outcome] {
				limited, ok := refused.Error.(*RateLimited)
				if !ok || patience == 0 {
					return effect.Return[effect.Op](outcome)
				}
				return effect.Then(effect.Perform(&clock.Sleep{For: limited.RetryAfter}), func(effect.Unit) effect.Eff[effect.Op, Outcome] {
					return ChargeWithPatience(amount, patience-1)
				})
			},
		)
	})
}

// --8<-- [end:patience]

// Refused is a ChargeError as a Go error, for the moment a program gives up
// on an outcome and hands it to a caller that speaks error.
type Refused struct{ Reason ChargeError }

func (r *Refused) Error() string {
	return "billing: refused: " + MatchChargeErrorR1(r.Reason,
		func(x *InvalidAmount) string { return "invalid amount: " + x.Reason },
		func(x *RateLimited) string { return "rate limited, retry after " + x.RetryAfter.String() },
		func(x *QuotaExceeded) string { return "quota exceeded until " + x.ResetAt.Format(time.Kitchen) },
		func(x *OutOfCredits) string { return "out of credits, missing " + strconv.Itoa(x.Missing) },
	)
}
