package welcome

import (
	"github.com/widmogrod/mkunion/x/effect"
	"time"

	"github.com/widmogrod/mkunion/f"
)

// --8<-- [start:ops-def]

// Unit is the answer of an operation that has nothing to return.
type Unit struct{}

// MyEff is the set of operations our programs may ask for.
//
// Each variant embeds f.Returns[R] to declare the type of its answer. The
// `handler` option makes mkunion generate the typed layer from it:
// MyEffHandler (one method per operation), MyEffOf[R] (an MyEff that
// answers with R), MyEffHandlerFunc (the adapter Run uses) and
// MyEffDefaults (zero answers, for tests). See ops_union_gen.go.
//
//go:tag mkunion:"MyEff,handler"
type (
	// Log writes a line somewhere.
	Log struct {
		f.Returns[Unit]
		Msg string
	}
	// Now asks for the current time.
	Now struct{ f.Returns[time.Time] }
	// ReadFile asks for the content of a file.
	ReadFile struct {
		f.Returns[[]byte]
		Path string
	}
	// Random asks for a number in [0, Max).
	Random struct {
		f.Returns[int]
		Max int
	}
	// Send asks for a message to be delivered. It is not idempotent: sending twice sends two.
	Send struct {
		f.Returns[string]
		To, Msg string
	}
	// Charge asks to take Amount from the customer's account. Its answer is a
	// Result: a Receipt, or a ChargeError that says why the charge was refused.
	Charge struct {
		f.Returns[f.Result[Receipt, ChargeError]]
		Amount int
	}
)

// --8<-- [end:ops-def]

// --8<-- [start:charge-error]

// Receipt is the proof of a charge that went through.
type Receipt struct{ ID string }

// ChargeError is every way a charge can be refused. These are answers, not
// failures: the bank did its job and said no. A program must decide what to
// do with each one, and retry middleware never sees them, because they are
// values, not Go errors. A timeout or a lost connection is a Go error.
//
//go:tag mkunion:"ChargeError"
type (
	// OutOfBudget means the account is short by Missing.
	OutOfBudget struct{ Missing int }
	// QuotaExceeded means the account may charge again after ResetAt.
	QuotaExceeded struct{ ResetAt time.Time }
)

// --8<-- [end:charge-error]

// --8<-- [start:typed-layer]

// Perform asks for one operation as a program. R is inferred from the
// operation's f.Returns, so `Perform(&Now{})` is an Eff[MyEff, time.Time].
func Perform[R any](op MyEffOf[R]) effect.Eff[MyEff, R] {
	return effect.PerformAs[MyEff, R](op)
}

// --8<-- [end:typed-layer]
