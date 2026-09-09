package billing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/example/compose/clock"
	"github.com/widmogrod/mkunion/f"
	"github.com/widmogrod/mkunion/x/effect"
)

type clockFuncs = clock.EffectFuncs

// bank answers Charge from a script of outcomes, and sleeps by noting it.
// It is one handler for two packages, so ChargeWithPatience can run alone.
type bank struct {
	EffectFuncs
	clockFuncs
	slept []time.Duration
}

func scripted(outcomes ...Outcome) *bank {
	b := &bank{}
	b.EffectFuncs = EffectFuncs{
		Charge: func(context.Context, *Charge) (Outcome, error) {
			next := outcomes[0]
			if len(outcomes) > 1 {
				outcomes = outcomes[1:]
			}
			return next, nil
		},
	}
	b.clockFuncs = clock.EffectFuncs{
		Sleep: func(_ context.Context, op *clock.Sleep) (effect.Unit, error) {
			b.slept = append(b.slept, op.For)
			return effect.Unit{}, nil
		},
	}
	return b
}

func ok(id string) Outcome          { return f.MkOk[ChargeError](Receipt{ID: id}) }
func refused(e ChargeError) Outcome { return f.MkErr[Receipt](e) }

// --8<-- [start:patience-test]

func TestChargeWithPatience(t *testing.T) {
	t.Run("too fast: wait as long as the bank asks, then it goes through", func(t *testing.T) {
		b := scripted(refused(&RateLimited{RetryAfter: time.Second}), refused(&RateLimited{RetryAfter: 2 * time.Second}), ok("c1"))
		var trace []effect.Op

		outcome, err := effect.Interpret(context.Background(), ChargeWithPatience(10, 3), b, effect.Trace(&trace))

		require.NoError(t, err)
		assert.Equal(t, ok("c1"), outcome)
		assert.Equal(t, []effect.Op{
			&Charge{Amount: 10}, &clock.Sleep{For: time.Second},
			&Charge{Amount: 10}, &clock.Sleep{For: 2 * time.Second},
			&Charge{Amount: 10},
		}, trace, "two packages, one program, no body")
	})

	t.Run("patience runs out: the last refusal is the answer", func(t *testing.T) {
		limited := refused(&RateLimited{RetryAfter: time.Second})
		b := scripted(limited)
		var trace []effect.Op

		outcome, err := effect.Interpret(context.Background(), ChargeWithPatience(10, 1), b, effect.Trace(&trace))

		require.NoError(t, err, "a refusal is not an error")
		assert.Equal(t, limited, outcome)
		assert.Len(t, trace, 3, "charge, sleep, charge: then it stops")
	})

	t.Run("other refusals are returned at once: waiting cannot help", func(t *testing.T) {
		for _, reason := range []ChargeError{
			&InvalidAmount{Reason: "amount must be positive"},
			&QuotaExceeded{ResetAt: time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)},
			&OutOfCredits{Missing: 5},
		} {
			b := scripted(refused(reason))
			var trace []effect.Op
			outcome, err := effect.Interpret(context.Background(), ChargeWithPatience(10, 3), b, effect.Trace(&trace))
			require.NoError(t, err)
			assert.Equal(t, refused(reason), outcome)
			assert.Equal(t, []effect.Op{&Charge{Amount: 10}}, trace, "%T: one charge, no sleep", reason)
		}
	})

	t.Run("a broken line is a Go error, and Retry middleware owns it", func(t *testing.T) {
		b := scripted(ok("c1"))
		blip := errors.New("bank: connection reset")
		charge := b.EffectFuncs.Charge
		calls := 0
		b.EffectFuncs.Charge = func(ctx context.Context, op *Charge) (Outcome, error) {
			if calls++; calls == 1 {
				return nil, blip
			}
			return charge(ctx, op)
		}
		var attempts []effect.Op

		outcome, err := effect.Interpret(context.Background(), ChargeWithPatience(10, 0), b,
			effect.Retry[effect.Op](2), effect.Trace(&attempts))

		require.NoError(t, err)
		assert.Equal(t, ok("c1"), outcome)
		assert.Equal(t, []effect.Op{&Charge{Amount: 10}, &Charge{Amount: 10}}, attempts, "retried by middleware, with no patience needed")
	})
}

// --8<-- [end:patience-test]

func TestRefused_speaksError(t *testing.T) {
	assert.EqualError(t, &Refused{Reason: &InvalidAmount{Reason: "zero"}}, "billing: refused: invalid amount: zero")
	assert.EqualError(t, &Refused{Reason: &RateLimited{RetryAfter: time.Second}}, "billing: refused: rate limited, retry after 1s")
	assert.EqualError(t, &Refused{Reason: &QuotaExceeded{ResetAt: time.Date(2026, 9, 2, 13, 0, 0, 0, time.UTC)}}, "billing: refused: quota exceeded until 1:00PM")
	assert.EqualError(t, &Refused{Reason: &OutOfCredits{Missing: 5}}, "billing: refused: out of credits, missing 5")
}
