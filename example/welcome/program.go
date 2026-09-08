package welcome

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/widmogrod/mkunion/f"
)

// A program is plain Go written against Fx. Calling Greet or Notify performs
// nothing; it returns a Program value. Run gives the value meaning with a
// handler of your choice (see handlers.go).

// --8<-- [start:fx-api]

// Program is a program over the MyEff operations that yields A.
type Program[A any] = Eff[MyEff, A]

// Fx is the handle a program body uses to ask for operations.
type Fx struct{ env *Env[MyEff] }

// Prog turns a plain Go body into a Program. The body runs later, inside Run,
// as a coroutine: every operation pauses it and the handler's answer resumes it.
func Prog[A any](body func(fx Fx) (A, error)) Program[A] {
	return Proc(func(e *Env[MyEff]) (A, error) { return body(Fx{env: e}) })
}

// Do asks for any operation and returns its typed answer. R is inferred from
// the operation's f.Returns. On error the body stops, and the program
// fails with that error.
func (fx Fx) Do[R any](op MyEffOf[R]) R { return DoAs[MyEff, R](fx.env, op) }

// Attempt is Do that returns the error instead of stopping the body.
func (fx Fx) Attempt[R any](op MyEffOf[R]) (R, error) { return AttemptAs[MyEff, R](fx.env, op) }

// One method per operation. Mechanical, like the typed layer in ops.go.

func (fx Fx) Log(msg string)              { fx.Do(&Log{Msg: msg}) }
func (fx Fx) Now() time.Time              { return fx.Do(&Now{}) }
func (fx Fx) ReadFile(path string) []byte { return fx.Do(&ReadFile{Path: path}) }
func (fx Fx) Random(max int) int          { return fx.Do(&Random{Max: max}) }
func (fx Fx) Send(to, msg string) string  { return fx.Do(&Send{To: to, Msg: msg}) }

// --8<-- [end:fx-api]

// --8<-- [start:interpret]

// Interpret runs a program against a handler and returns its value.
//
// The program was built earlier and performed nothing. Interpret is where it
// gets meaning: h answers every operation, in order. Middleware is optional
// and listed outermost first, so Interpret(ctx, p, h, Retry(3), Record(&tape))
// retries around a recorder: the tape sees every attempt.
func Interpret[A any](ctx context.Context, program Program[A], h MyEffHandler, middleware ...Middleware[MyEff]) (A, error) {
	return Run(ctx, Wrap(MyEffHandlerFunc(h), middleware...), program)
}

// --8<-- [end:interpret]

// --8<-- [start:greet]

// Greet reads a name from a file, asks for the time, logs a greeting and returns it.
// Operations, in order: ReadFile, Now, Log.
func Greet(path string) Program[string] {
	return Prog(func(fx Fx) (string, error) {
		name := strings.TrimSpace(string(fx.ReadFile(path)))
		msg := fmt.Sprintf("Hello %s, it is %s", name, fx.Now().Format(time.Kitchen))
		fx.Log(msg)
		return msg, nil
	})
}

// --8<-- [end:greet]

// --8<-- [start:notify]

// Notify reads a name, mails a greeting, and logs the receipt.
// Operations, in order: ReadFile, Now, Send, Log. Send is the one that must
// not happen twice; parts 3 to 5 are built around that.
func Notify(path, to string) Program[string] {
	return Prog(func(fx Fx) (string, error) {
		name := strings.TrimSpace(string(fx.ReadFile(path)))
		receipt := fx.Send(to, "Hello "+name+", it is "+fx.Now().Format(time.Kitchen))
		fx.Log("sent " + receipt)
		return receipt, nil
	})
}

// --8<-- [end:notify]

// --8<-- [start:greet-or-guest]

// GreetOrGuest handles a missing file in place: a guest gets a greeting too.
func GreetOrGuest(path string) Program[string] {
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

// --8<-- [end:greet-or-guest]

// --8<-- [start:roll]

// RollUntil rolls a die until it shows want and returns how many rolls it took.
// A loop is a loop. A run of a million rolls is a million steps in Run, not a
// million stack frames; part 1 has a test for that.
func RollUntil(want, maxRolls int) Program[int] {
	return Prog(func(fx Fx) (int, error) {
		for roll := 1; roll <= maxRolls; roll++ {
			if fx.Random(6)+1 == want {
				return roll, nil
			}
		}
		return 0, fmt.Errorf("no %d in %d rolls", want, maxRolls)
	})
}

// --8<-- [end:roll]

// --8<-- [start:pay]

// Pay charges the customer and logs the receipt. A refusal is an answer, so
// the body matches on it, exhaustively, and decides. A timeout would be a Go
// error: fx.Do would stop the body, and Retry (part 4) would have a say first.
func Pay(amount int) Program[string] {
	return Prog(func(fx Fx) (string, error) {
		outcome := fx.Do(&Charge{Amount: amount}) // f.Result[Receipt, ChargeError], no cast
		return f.MatchResultR2(outcome,
			func(ok *f.Ok[Receipt, ChargeError]) (string, error) {
				fx.Log("charged " + ok.Value.ID)
				return ok.Value.ID, nil
			},
			func(refused *f.Err[Receipt, ChargeError]) (string, error) {
				return "", MatchChargeErrorR1(refused.Error,
					func(x *OutOfBudget) error { return fmt.Errorf("charge refused: short by %d", x.Missing) },
					func(x *QuotaExceeded) error {
						return fmt.Errorf("charge refused: quota resets at %s", x.ResetAt.Format(time.Kitchen))
					},
				)
			},
		)
	})
}

// --8<-- [end:pay]
