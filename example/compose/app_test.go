package compose

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/example/compose/billing"
	"github.com/widmogrod/mkunion/example/compose/clock"
	"github.com/widmogrod/mkunion/example/compose/mailer"
	"github.com/widmogrod/mkunion/f"
	"github.com/widmogrod/mkunion/x/effect"
)

var noon = time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

// --8<-- [start:world]

// Three aliases, so the generated EffectFuncs can be embedded side by side.
type (
	clockFuncs   = clock.EffectFuncs
	mailFuncs    = mailer.EffectFuncs
	billingFuncs = billing.EffectFuncs
)

// world is one handler for all three packages: it embeds each package's
// generated EffectFuncs, filled inline. No package knows about the others.
type world struct {
	clockFuncs
	mailFuncs
	billingFuncs
	now      time.Time
	slept    []time.Duration
	sent     []string
	charged  []int
	outcomes []billing.Outcome // the bank's script; the last one repeats
}

var _ Handlers = (*world)(nil) // the compiler checks that both packages are covered

func newWorld() *world {
	w := &world{now: noon}
	w.clockFuncs = clock.EffectFuncs{
		Now: func(context.Context, *clock.Now) (time.Time, error) { return w.now, nil },
		Sleep: func(_ context.Context, op *clock.Sleep) (effect.Unit, error) {
			w.slept = append(w.slept, op.For)
			w.now = w.now.Add(op.For)
			return effect.Unit{}, nil
		},
	}
	w.mailFuncs = mailer.EffectFuncs{
		Resolve: func(_ context.Context, op *mailer.Resolve) (string, error) { return op.Name + "@example.com", nil },
		Send: func(_ context.Context, op *mailer.Send) (string, error) {
			w.sent = append(w.sent, op.To+": "+op.Msg)
			return "receipt-1", nil
		},
	}
	w.billingFuncs = billing.EffectFuncs{
		Charge: func(_ context.Context, op *billing.Charge) (billing.Outcome, error) {
			w.charged = append(w.charged, op.Amount)
			next := w.outcomes[0]
			if len(w.outcomes) > 1 {
				w.outcomes = w.outcomes[1:]
			}
			return next, nil
		},
	}
	w.outcomes = []billing.Outcome{charged("c1")}
	return w
}

func charged(id string) billing.Outcome                  { return f.MkOk[billing.ChargeError](billing.Receipt{ID: id}) }
func refused(reason billing.ChargeError) billing.Outcome { return f.MkErr[billing.Receipt](reason) }

// --8<-- [end:world]

// --8<-- [start:one-trace]

func TestRemind_twoPackagesOneProgramOneTrace(t *testing.T) {
	ctx := context.Background()
	program := Remind("ada", "stand-up", noon.Add(time.Hour))

	w := newWorld()
	var trace []effect.Op
	got, err := Interpret(ctx, program, w, effect.Trace(&trace))

	require.NoError(t, err)
	assert.Equal(t, "receipt-1 at 1:00PM", got)
	assert.Equal(t, []time.Duration{time.Hour}, w.slept)
	assert.Equal(t, []string{"ada@example.com: stand-up"}, w.sent)
	assert.Equal(t, []effect.Op{
		&clock.Now{},                                         // clock.WaitUntil, step 1
		&clock.Sleep{For: time.Hour},                         // clock.WaitUntil, step 2
		&mailer.Resolve{Name: "ada"},                         // mailer.Notify, step 1
		&mailer.Send{To: "ada@example.com", Msg: "stand-up"}, // mailer.Notify, step 2
		&clock.Now{},                                         // the body's own fx.Do
	}, trace, "one trace over both packages, in the order the body asked")
}

// --8<-- [end:one-trace]

func TestRemind_oneMiddlewareCoversBothPackages(t *testing.T) {
	program := Remind("ada", "stand-up", noon)

	w := newWorld()
	blip := errors.New("smtp: connection reset")
	realSend := w.mailFuncs.Send
	calls := 0
	w.mailFuncs.Send = func(ctx context.Context, op *mailer.Send) (string, error) {
		if calls++; calls == 1 {
			return "", blip
		}
		return realSend(ctx, op)
	}

	var attempts []effect.Op
	got, err := Interpret(context.Background(), program, w,
		effect.Retry[effect.Op](3), effect.Trace(&attempts))

	require.NoError(t, err)
	assert.Equal(t, "receipt-1 at 12:00PM", got)
	assert.Equal(t, []string{"ada@example.com: stand-up"}, w.sent)
	assert.Equal(t, []effect.Op{
		&clock.Now{},
		&mailer.Resolve{Name: "ada"},
		&mailer.Send{To: "ada@example.com", Msg: "stand-up"}, // failed
		&mailer.Send{To: "ada@example.com", Msg: "stand-up"}, // retried, by the same Retry that covers clock
		&clock.Now{},
	}, attempts)
}

func TestRemind_aPackageErrorUnwindsTheBody(t *testing.T) {
	program := Remind("ada", "stand-up", noon)
	down := errors.New("directory down")

	w := newWorld()
	w.mailFuncs.Resolve = func(context.Context, *mailer.Resolve) (string, error) { return "", down }
	var trace []effect.Op
	_, err := Interpret(context.Background(), program, w, effect.Trace(&trace))

	assert.ErrorIs(t, err, down)
	assert.Equal(t, []effect.Op{
		&clock.Now{},
		&mailer.Resolve{Name: "ada"},
	}, trace, "nothing after the failing step ran, in either package")
}

// --8<-- [start:as-is]

// A package's program is already an application program. Nothing to lift.
func TestAPackageProgramRunsAsIs(t *testing.T) {
	w := newWorld()
	got, err := Interpret(context.Background(), mailer.Notify("ada", "hi"), w)

	require.NoError(t, err)
	assert.Equal(t, "receipt-1", got)
	assert.Equal(t, []string{"ada@example.com: hi"}, w.sent)
}

// A handler that covers one package only is refused by the compiler through
// Handlers. Bypass that check, and the first foreign operation says what is missing.
func TestAHandlerThatMissesAPackageFailsAtItsFirstOperation(t *testing.T) {
	onlyClock := newWorld().clockFuncs

	_, err := effect.Interpret(context.Background(), Remind("ada", "hi", noon), onlyClock)
	assert.EqualError(t, err, "mailer: handler clock.EffectFuncs does not implement EffectHandler")
}

// --8<-- [end:as-is]

// --8<-- [start:paid-test]

func TestSendPaid_everyRefusalIsDecidedByTheBody(t *testing.T) {
	program := SendPaid("ada", "invoice", 10)
	resetAt := noon.Add(time.Hour)

	cases := []struct {
		name    string
		bank    []billing.Outcome
		want    string
		refusal billing.ChargeError // nil when sent
		trace   []effect.Op
	}{
		{
			name:  "charged, then mailed",
			bank:  []billing.Outcome{charged("c1")},
			want:  "receipt-1 paid by c1",
			trace: []effect.Op{&billing.Charge{Amount: 10}, &mailer.Resolve{Name: "ada"}, &mailer.Send{To: "ada@example.com", Msg: "invoice"}},
		},
		{
			name:  "too fast: the library program waited, the body never saw it",
			bank:  []billing.Outcome{refused(&billing.RateLimited{RetryAfter: time.Second}), charged("c2")},
			want:  "receipt-1 paid by c2",
			trace: []effect.Op{&billing.Charge{Amount: 10}, &clock.Sleep{For: time.Second}, &billing.Charge{Amount: 10}, &mailer.Resolve{Name: "ada"}, &mailer.Send{To: "ada@example.com", Msg: "invoice"}},
		},
		{
			name:  "quota exceeded: the body waits for the reset, once",
			bank:  []billing.Outcome{refused(&billing.QuotaExceeded{ResetAt: resetAt}), charged("c3")},
			want:  "receipt-1 paid by c3",
			trace: []effect.Op{&billing.Charge{Amount: 10}, &clock.Now{}, &clock.Sleep{For: time.Hour}, &billing.Charge{Amount: 10}, &mailer.Resolve{Name: "ada"}, &mailer.Send{To: "ada@example.com", Msg: "invoice"}},
		},
		{
			name:    "quota exceeded twice: the second refusal stands",
			bank:    []billing.Outcome{refused(&billing.QuotaExceeded{ResetAt: resetAt})},
			refusal: &billing.QuotaExceeded{ResetAt: resetAt},
			trace:   []effect.Op{&billing.Charge{Amount: 10}, &clock.Now{}, &clock.Sleep{For: time.Hour}, &billing.Charge{Amount: 10}},
		},
		{
			name:    "invalid amount: nothing can help, nothing else runs",
			bank:    []billing.Outcome{refused(&billing.InvalidAmount{Reason: "currency mismatch"})},
			refusal: &billing.InvalidAmount{Reason: "currency mismatch"},
			trace:   []effect.Op{&billing.Charge{Amount: 10}},
		},
		{
			name:    "out of credits: give up",
			bank:    []billing.Outcome{refused(&billing.OutOfCredits{Missing: 4})},
			refusal: &billing.OutOfCredits{Missing: 4},
			trace:   []effect.Op{&billing.Charge{Amount: 10}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := newWorld()
			w.outcomes = tc.bank
			var trace []effect.Op

			got, err := Interpret(context.Background(), program, w, effect.Trace(&trace))

			assert.Equal(t, tc.trace, trace)
			if tc.refusal == nil {
				require.NoError(t, err)
				assert.Equal(t, tc.want, got)
				assert.Len(t, w.sent, 1)
				return
			}
			var notSent *ErrNotSent
			require.ErrorAs(t, err, &notSent, "the body gave up with a typed error")
			assert.Equal(t, tc.refusal, notSent.Refused.Reason, "the reason is the union value, not a string")
			assert.Empty(t, w.sent, "nothing was mailed")
		})
	}
}

// --8<-- [end:paid-test]

func TestSendPaid_aRefusalIsNeverRetriedABrokenLineIs(t *testing.T) {
	t.Run("out of credits under Retry(3): one charge", func(t *testing.T) {
		w := newWorld()
		w.outcomes = []billing.Outcome{refused(&billing.OutOfCredits{Missing: 4})}
		var attempts []effect.Op

		_, err := Interpret(context.Background(), SendPaid("ada", "invoice", 10), w, effect.Retry[effect.Op](3), effect.Trace(&attempts))

		var notSent *ErrNotSent
		require.ErrorAs(t, err, &notSent)
		assert.Equal(t, []effect.Op{&billing.Charge{Amount: 10}}, attempts, "an answer is not an error; Retry never saw it")
	})

	t.Run("connection reset under Retry(3): charged once, on the second try", func(t *testing.T) {
		w := newWorld()
		blip := errors.New("bank: connection reset")
		charge := w.billingFuncs.Charge
		calls := 0
		w.billingFuncs.Charge = func(ctx context.Context, op *billing.Charge) (billing.Outcome, error) {
			if calls++; calls == 1 {
				return nil, blip
			}
			return charge(ctx, op)
		}
		var attempts []effect.Op

		got, err := Interpret(context.Background(), SendPaid("ada", "invoice", 10), w, effect.Retry[effect.Op](3), effect.Trace(&attempts))

		require.NoError(t, err)
		assert.Equal(t, "receipt-1 paid by c1", got)
		assert.Equal(t, []effect.Op{&billing.Charge{Amount: 10}, &billing.Charge{Amount: 10}, &mailer.Resolve{Name: "ada"}, &mailer.Send{To: "ada@example.com", Msg: "invoice"}}, attempts)
		assert.Equal(t, []int{10}, w.charged, "the failed attempt never reached the bank")
	})
}

// --8<-- [start:recomposed-test]

func TestSendPaidOrQueue_aFallbackAroundAFinishedProgram(t *testing.T) {
	w := newWorld()
	w.outcomes = []billing.Outcome{refused(&billing.OutOfCredits{Missing: 4})}
	var trace []effect.Op

	got, err := Interpret(context.Background(), SendPaidOrQueue("ada", "invoice", 10), w, effect.Trace(&trace))

	require.NoError(t, err)
	assert.Equal(t, "queued for ada@example.com", got)
	assert.Equal(t, []effect.Op{&billing.Charge{Amount: 10}, &mailer.Resolve{Name: "ada"}}, trace,
		"the refused charge, then the fallback's own step; SendPaid was not changed to get this")
}

// --8<-- [end:recomposed-test]
