package effect

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGreetDo_matchesGreet(t *testing.T) {
	files := map[string]string{"name.txt": "Ada\n"}

	var traceA, traceB []Effect
	a, errA := Run(context.Background(), Trace(HandlerOf(&Fake{Clock: noon, Files: files}), &traceA), Greet("name.txt"))
	b, errB := Run(context.Background(), Trace(HandlerOf(&Fake{Clock: noon, Files: files}), &traceB), GreetDo("name.txt"))

	require.NoError(t, errA)
	require.NoError(t, errB)
	assert.Equal(t, "Hello Ada, it is 12:00PM", b)
	assert.Equal(t, a, b)
	assert.Equal(t, traceA, traceB)
}

func TestGreetDo_isLazy(t *testing.T) {
	var trace []Effect
	prog := GreetDo("name.txt")
	assert.Empty(t, trace, "building the program performs nothing")

	_, err := Run(context.Background(), Trace(HandlerOf(&Fake{Clock: noon, Files: map[string]string{"name.txt": "Ada"}}), &trace), prog)
	require.NoError(t, err)
	assert.Len(t, trace, 3)
}

func TestGreetDo_handlerErrorUnwindsBody(t *testing.T) {
	fake := &Fake{Clock: noon}
	var trace []Effect
	before := runtime.NumGoroutine()

	_, err := Run(context.Background(), Trace(HandlerOf(fake), &trace), GreetDo("missing.txt"))

	require.ErrorContains(t, err, `no file "missing.txt"`)
	assert.Len(t, trace, 1, "nothing after the failing operation runs")
	assert.Empty(t, fake.Logs)
	assertNoLeak(t, before)
}

func TestGreetDo_cancelledContextUnwindsBody(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var trace []Effect
	before := runtime.NumGoroutine()

	_, err := Run(ctx, Trace(HandlerOf(&Fake{}), &trace), GreetDo("name.txt"))

	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, trace)
	assertNoLeak(t, before)
}

func TestGreetOrGuest_attemptHandlesErrorInPlace(t *testing.T) {
	fake := &Fake{}

	got, err := Run(context.Background(), HandlerOf(fake), GreetOrGuest("missing.txt"))

	require.NoError(t, err)
	assert.Equal(t, "Hello guest", got)
	assert.Equal(t, []string{"Hello guest"}, fake.Logs)
}

func TestRollUntilDo_isStackSafe(t *testing.T) {
	const rolls = 1_000_000
	got, err := Run(context.Background(), HandlerOf(&Fake{Rolls: []int{0}}), RollUntilDo(6, rolls))
	require.ErrorContains(t, err, "no 6 in 1000000 rolls")
	assert.Zero(t, got)

	got, err = Run(context.Background(), HandlerOf(&Fake{Rolls: []int{0, 0, 5}}), RollUntilDo(6, rolls))
	require.NoError(t, err)
	assert.Equal(t, 3, got)
}

func TestProc_composesWithThenAndMap(t *testing.T) {
	prog := Map(
		Then(RollUntilDo(6, 10), func(roll int) Eff[Effect, time.Time] { return Perform(&Now{}) }),
		func(now time.Time) string { return now.Format(time.Kitchen) },
	)

	got, err := Run(context.Background(), HandlerOf(&Fake{Clock: noon, Rolls: []int{5}}), prog)

	require.NoError(t, err)
	assert.Equal(t, "12:00PM", got)
}

func TestProc_wrongAnswerTypeUnwindsBody(t *testing.T) {
	lying := func(context.Context, Effect) (any, error) { return "not a time", nil }

	_, err := Run(context.Background(), lying, GreetDo("name.txt"))

	require.ErrorContains(t, err, "handler answered string to *effect.ReadFile, want []uint8")
}

func TestProc_bodyPanicIsNotSwallowed(t *testing.T) {
	boom := Proc(func(*Env[Effect]) (int, error) { panic("boom") })
	assert.PanicsWithValue(t, "boom", func() {
		_, _ = Run(context.Background(), HandlerOf(&Fake{}), boom)
	})
}

func TestProc_attemptSeesContextError(t *testing.T) {
	// A cancelled context reaches the body as an error from Attempt.
	var seen error
	prog := Proc(func(e *Env[Effect]) (int, error) {
		_, seen = Attempt(e, &Now{})
		return 0, seen
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := Run(ctx, HandlerOf(&Fake{}), prog)

	require.ErrorIs(t, err, context.Canceled)
	assert.True(t, errors.Is(seen, context.Canceled))
}

// assertNoLeak waits briefly for the coroutine behind a Proc to be gone.
func assertNoLeak(t *testing.T, before int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= before {
			return
		}
		runtime.Gosched()
	}
	assert.LessOrEqual(t, runtime.NumGoroutine(), before, "coroutine leaked")
}
