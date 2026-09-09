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

// --8<-- [start:world]

// Two aliases, so both generated EffectFuncs can be embedded side by side.
type (
	clockFuncs = clock.EffectFuncs
	mailFuncs  = mailer.EffectFuncs
)

// world is one handler for both packages: it embeds each package's generated
// EffectFuncs, filled inline. Neither package knows about the other.
type world struct {
	clockFuncs
	mailFuncs
	now   time.Time
	slept []time.Duration
	sent  []string
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
	return w
}

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
