package compose

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/example/compose/clock"
	"github.com/widmogrod/mkunion/example/compose/mailer"
	"github.com/widmogrod/mkunion/x/effect"
)

var noon = time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

// fakeClock and fakeMail are the two packages' generated EffectFuncs, filled
// inline. Neither package knows about the other.
func fakeClock(now *time.Time, slept *[]time.Duration) clock.EffectFuncs {
	return clock.EffectFuncs{
		Now: func(context.Context, *clock.Now) (time.Time, error) { return *now, nil },
		Sleep: func(_ context.Context, op *clock.Sleep) (effect.Unit, error) {
			*slept = append(*slept, op.For)
			*now = now.Add(op.For)
			return effect.Unit{}, nil
		},
	}
}

func fakeMail(sent *[]string) mailer.EffectFuncs {
	return mailer.EffectFuncs{
		Resolve: func(_ context.Context, op *mailer.Resolve) (string, error) { return op.Name + "@example.com", nil },
		Send: func(_ context.Context, op *mailer.Send) (string, error) {
			*sent = append(*sent, op.To+": "+op.Msg)
			return "receipt-1", nil
		},
	}
}

// --8<-- [start:one-trace]

func TestRemind_twoPackagesOneProgramOneTrace(t *testing.T) {
	ctx := context.Background()
	program := Remind("ada", "stand-up", noon.Add(time.Hour))

	now := noon
	var slept []time.Duration
	var sent []string
	var trace []AppEff
	got, err := Interpret(ctx, program, fakeClock(&now, &slept), fakeMail(&sent), effect.Trace(&trace))

	require.NoError(t, err)
	assert.Equal(t, "receipt-1 at 1:00PM", got)
	assert.Equal(t, []time.Duration{time.Hour}, slept)
	assert.Equal(t, []string{"ada@example.com: stand-up"}, sent)
	assert.Equal(t, []AppEff{
		&ClockOp{Op: &clock.Now{}},                 // clock.WaitUntil, step 1
		&ClockOp{Op: &clock.Sleep{For: time.Hour}}, // clock.WaitUntil, step 2
		&MailOp{Op: &mailer.Resolve{Name: "ada"}},  // mailer.Notify, step 1
		&MailOp{Op: &mailer.Send{To: "ada@example.com", Msg: "stand-up"}},
		&ClockOp{Op: &clock.Now{}}, // the body's own fx.Clock
	}, trace, "one trace over both packages, in the order the body asked")
}

// --8<-- [end:one-trace]

func TestRemind_oneMiddlewareCoversBothPackages(t *testing.T) {
	program := Remind("ada", "stand-up", noon)

	now := noon
	var slept []time.Duration
	var sent []string
	blip := errors.New("smtp: connection reset")
	mail := fakeMail(&sent)
	realSend := mail.Send
	calls := 0
	mail.Send = func(ctx context.Context, op *mailer.Send) (string, error) {
		if calls++; calls == 1 {
			return "", blip
		}
		return realSend(ctx, op)
	}

	var attempts []AppEff
	got, err := Interpret(context.Background(), program, fakeClock(&now, &slept), mail,
		effect.Retry[AppEff](3), effect.Trace(&attempts))

	require.NoError(t, err)
	assert.Equal(t, "receipt-1 at 12:00PM", got)
	assert.Equal(t, []string{"ada@example.com: stand-up"}, sent)
	assert.Equal(t, []AppEff{
		&ClockOp{Op: &clock.Now{}},
		&MailOp{Op: &mailer.Resolve{Name: "ada"}},
		&MailOp{Op: &mailer.Send{To: "ada@example.com", Msg: "stand-up"}}, // failed
		&MailOp{Op: &mailer.Send{To: "ada@example.com", Msg: "stand-up"}}, // retried, by the same Retry that covers clock
		&ClockOp{Op: &clock.Now{}},
	}, attempts)
}

func TestRemind_aPackageErrorUnwindsTheBody(t *testing.T) {
	program := Remind("ada", "stand-up", noon)
	down := errors.New("directory down")

	now := noon
	var slept []time.Duration
	mail := mailer.EffectFuncs{
		Resolve: func(context.Context, *mailer.Resolve) (string, error) { return "", down },
	}
	var trace []AppEff
	_, err := Interpret(context.Background(), program, fakeClock(&now, &slept), mail, effect.Trace(&trace))

	assert.ErrorIs(t, err, down)
	assert.Equal(t, []AppEff{
		&ClockOp{Op: &clock.Now{}},
		&MailOp{Op: &mailer.Resolve{Name: "ada"}},
	}, trace, "nothing after the failing step ran, in either package")
}

// --8<-- [start:lift]

// A package's program can also be lifted as a value, with no body around it.
func TestLift_aPackageProgramIsAnAppProgram(t *testing.T) {
	var program Program[string] = effect.Lift(mailer.Notify("ada", "hi"), fromMail)

	var sent []string
	got, err := Interpret(context.Background(), program, clock.EffectDefaults{}, fakeMail(&sent))

	require.NoError(t, err)
	assert.Equal(t, "receipt-1", got)
	assert.Equal(t, []string{"ada@example.com: hi"}, sent)
}

// --8<-- [end:lift]
