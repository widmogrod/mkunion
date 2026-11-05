package welcome

import (
	"context"
	"errors"
	"fmt"
	"github.com/widmogrod/mkunion/x/effect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Part 2: under the hood.
//
// A Program is an Eff: a union of Pure, Fail, Bind and Suspend. Fx.Do builds
// Bind nodes for you, from a coroutine. You can also build them by hand with
// Perform and Then. Both styles produce the same data, and the tests prove it.

// --8<-- [start:greet-then]

// greetWithThen is Greet built by hand: Perform makes a one-step program, Then
// chains the next step onto its answer. This is what Fx.Do does for you.
func greetWithThen(path string) Program[string] {
	return effect.Then(Perform(&ReadFile{Path: path}), func(raw []byte) Program[string] {
		return effect.Then(Perform(&Now{}), func(now time.Time) Program[string] {
			msg := fmt.Sprintf("Hello %s, it is %s", strings.TrimSpace(string(raw)), now.Format(time.Kitchen))
			return effect.Map(Perform(&Log{Msg: msg}), func(Unit) string { return msg })
		})
	})
}

// --8<-- [end:greet-then]

// styles is the option selector for this part: the same program, two ways of
// writing it. Every test below runs against both.
var styles = []struct {
	name  string
	greet func(path string) Program[string]
}{
	{"fx (plain Go body, default)", Greet},
	{"then (explicit continuations)", greetWithThen},
}

func TestPart2_bothStylesLeaveTheSameTrace(t *testing.T) {
	for _, style := range styles {
		t.Run(style.name, func(t *testing.T) {
			fake := &Fake{Clock: noon, Files: map[string]string{"name.txt": "Ada\n"}}
			var trace []MyEff

			got, err := effect.Run(context.Background(), effect.Wrap(HandlerOf(fake), effect.Trace(&trace)), style.greet("name.txt"))

			require.NoError(t, err)
			assert.Equal(t, "Hello Ada, it is 12:00PM", got)
			assert.Equal(t, []MyEff{
				&ReadFile{Path: "name.txt"},
				&Now{},
				&Log{Msg: "Hello Ada, it is 12:00PM"},
			}, trace)
		})
	}
}

func TestPart2_bothStylesStopOnTheFirstError(t *testing.T) {
	for _, style := range styles {
		t.Run(style.name, func(t *testing.T) {
			fake := &Fake{Clock: noon}
			var trace []MyEff

			_, err := effect.Run(context.Background(), effect.Wrap(HandlerOf(fake), effect.Trace(&trace)), style.greet("missing.txt"))

			require.ErrorContains(t, err, `no file "missing.txt"`)
			assert.Equal(t, []MyEff{&ReadFile{Path: "missing.txt"}}, trace)
		})
	}
}

func TestPart2_thenAndMapAreOrdinaryValues(t *testing.T) {
	// Return and Map need no handler at all: nothing is performed.
	prog := effect.Map(effect.Return[MyEff](20), func(n int) int { return n + 1 })
	got, err := effect.Run(context.Background(), HandlerOf(&Fake{}), prog)
	require.NoError(t, err)
	assert.Equal(t, 21, got)

	// Throw short-circuits: the continuation is never built, the handler never called.
	boom := errors.New("boom")
	called := false
	h := func(context.Context, MyEff) (any, error) { called = true; return nil, nil }
	failed := effect.Then(effect.Throw[MyEff, int](boom), func(int) Program[Unit] { return Perform(&Log{Msg: "unreachable"}) })
	_, err = effect.Run(context.Background(), h, failed)
	require.ErrorIs(t, err, boom)
	assert.False(t, called)

	// Then composes a plain-Go program with a hand-built one: one Bind chain.
	mixed := effect.Then(RollUntil(6, 10), func(int) Program[time.Time] { return Perform(&Now{}) })
	var trace []MyEff
	when, err := effect.Run(context.Background(), effect.Wrap(HandlerOf(&Fake{Clock: noon, Rolls: []int{0, 5}}), effect.Trace(&trace)), mixed)
	require.NoError(t, err)
	assert.Equal(t, noon, when)
	assert.Equal(t, []MyEff{&Random{Max: 6}, &Random{Max: 6}, &Now{}}, trace, "two rolls from the body, then the hand-built step")
}

func TestPart2_aWrongAnswerTypeIsAnErrorNotAPanic(t *testing.T) {
	lying := func(context.Context, MyEff) (any, error) { return "not bytes", nil }

	for _, style := range styles {
		t.Run(style.name, func(t *testing.T) {
			_, err := effect.Run(context.Background(), lying, style.greet("name.txt"))
			require.ErrorContains(t, err, "handler answered string to *welcome.ReadFile, want []uint8")
		})
	}
}

// bogusEff satisfies the Eff interface without being a generated variant.
type bogusEff struct{}

func (bogusEff) AcceptEff(effect.EffVisitor[MyEff, int]) any { return nil }

func TestPart2_runRefusesAnUnknownNode(t *testing.T) {
	_, err := effect.Run[MyEff, int](context.Background(), HandlerOf(&Fake{}), bogusEff{})
	require.ErrorContains(t, err, "unknown program node")
}

func TestPart2_procIsACoroutine(t *testing.T) {
	t.Run("a body panic is not swallowed", func(t *testing.T) {
		boom := effect.Proc(func(*effect.Env[MyEff]) (int, error) { panic("boom") })
		assert.PanicsWithValue(t, "boom", func() {
			_, _ = effect.Run(context.Background(), HandlerOf(&Fake{}), boom)
		})
	})

	t.Run("a handler error unwinds the body and frees the coroutine", func(t *testing.T) {
		before := runtime.NumGoroutine()
		_, err := effect.Run(context.Background(), HandlerOf(&Fake{}), Greet("missing.txt"))
		require.Error(t, err)
		assertNoLeak(t, before)
	})

	t.Run("a cancelled context unwinds the body too", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		before := runtime.NumGoroutine()
		_, err := effect.Run(ctx, HandlerOf(&Fake{}), Greet("name.txt"))
		require.ErrorIs(t, err, context.Canceled)
		assertNoLeak(t, before)
	})

	t.Run("Attempt sees the context error", func(t *testing.T) {
		var seen error
		prog := effect.Proc(func(e *effect.Env[MyEff]) (int, error) {
			_, seen = effect.AttemptAs[MyEff, time.Time](e, &Now{})
			return 0, seen
		})
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := effect.Run(ctx, HandlerOf(&Fake{}), prog)
		require.ErrorIs(t, err, context.Canceled)
		assert.ErrorIs(t, seen, context.Canceled)
	})
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
