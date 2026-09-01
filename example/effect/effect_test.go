package effect

import (
	"bytes"
	"context"
	"errors"
	"math/rand/v2"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var noon = time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

func TestGreet_withFakeHandler(t *testing.T) {
	fake := &Fake{Clock: noon, Files: map[string]string{"name.txt": "Ada\n"}}
	var trace []Effect

	got, err := Run(context.Background(), Trace(HandlerOf(fake), &trace), Greet("name.txt"))

	require.NoError(t, err)
	assert.Equal(t, "Hello Ada, it is 12:00PM", got)
	assert.Equal(t, []string{"Hello Ada, it is 12:00PM"}, fake.Logs)
	assert.Equal(t, []Effect{&ReadFile{Path: "name.txt"}, &Now{}, &Log{Msg: "Hello Ada, it is 12:00PM"}}, trace)
}

func TestGreet_withLiveHandler(t *testing.T) {
	var out bytes.Buffer
	live := &Live{
		Out:  &out,
		FS:   fstest.MapFS{"name.txt": {Data: []byte("Grace")}},
		Rand: rand.New(rand.NewPCG(1, 2)),
		Now:  func() time.Time { return noon },
	}

	got, err := Run(context.Background(), HandlerOf(live), Greet("name.txt"))

	require.NoError(t, err)
	assert.Equal(t, "Hello Grace, it is 12:00PM", got)
	assert.Equal(t, "Hello Grace, it is 12:00PM\n", out.String())
}

func TestRun_handlerErrorStopsProgram(t *testing.T) {
	fake := &Fake{Clock: noon}
	var trace []Effect

	_, err := Run(context.Background(), Trace(HandlerOf(fake), &trace), Greet("missing.txt"))

	require.ErrorContains(t, err, `no file "missing.txt"`)
	assert.Len(t, trace, 1, "nothing after the failing operation runs")
	assert.Empty(t, fake.Logs)
}

func TestRun_cancelledContextStopsBeforeNextOperation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var trace []Effect

	_, err := Run(ctx, Trace(HandlerOf(&Fake{}), &trace), Greet("name.txt"))

	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, trace)
}

func TestRun_wrongAnswerTypeIsAnErrorNotAPanic(t *testing.T) {
	lying := func(context.Context, Effect) (any, error) { return "not a time", nil }

	_, err := Run(context.Background(), lying, Perform(&Now{}))

	require.ErrorContains(t, err, "handler answered string to *effect.Now, want time.Time")
}

func TestRun_unknownNodeIsAnError(t *testing.T) {
	var bogus bogusEff
	_, err := Run[Effect, int](context.Background(), HandlerOf(&Fake{}), bogus)
	require.ErrorContains(t, err, "unknown program node")
}

// bogusEff satisfies the Eff interface without being a generated variant.
type bogusEff struct{}

func (bogusEff) AcceptEff(EffVisitor[Effect, int]) any { return nil }

func TestRollUntil_isStackSafe(t *testing.T) {
	const rolls = 1_000_000
	fake := &Fake{Rolls: []int{0}} // always rolls a 1, never a 6
	got, err := Run(context.Background(), HandlerOf(fake), RollUntil(6, rolls))
	require.ErrorContains(t, err, "no 6 in 1000000 rolls")
	assert.Zero(t, got)

	fake = &Fake{Rolls: []int{0, 0, 5}} // third roll is a 6
	got, err = Run(context.Background(), HandlerOf(fake), RollUntil(6, rolls))
	require.NoError(t, err)
	assert.Equal(t, 3, got)
}

func TestFake_randomNeedsRolls(t *testing.T) {
	_, err := Run(context.Background(), HandlerOf(&Fake{}), Perform(&Random{Max: 6}))
	require.ErrorContains(t, err, "no rolls configured")
}

func TestThen_failShortCircuitsWithoutCallingHandler(t *testing.T) {
	boom := errors.New("boom")
	called := false
	h := func(context.Context, Effect) (any, error) { called = true; return nil, nil }

	prog := Then(Throw[Effect, int](boom), func(int) Eff[Effect, Unit] {
		return Perform(&Log{Msg: "unreachable"}) // would need a handler call
	})
	_, err := Run(context.Background(), h, Map(prog, func(Unit) string { return "x" }))

	require.ErrorIs(t, err, boom)
	assert.False(t, called)
}

func TestMap_transformsPureValue(t *testing.T) {
	prog := Map(Return[Effect](20), func(n int) int { return n + 1 })
	got, err := Run(context.Background(), HandlerOf(&Fake{}), prog)
	require.NoError(t, err)
	assert.Equal(t, 21, got)
}

func TestPerformDirect_runsOneOperation(t *testing.T) {
	fake := &Fake{Clock: noon}
	got, err := PerformDirect(context.Background(), HandlerOf(fake), &Now{})
	require.NoError(t, err)
	assert.Equal(t, noon, got)
}

func TestResultMethodsArePhantom(t *testing.T) {
	// They exist for the type checker only. Calling them returns zero values.
	assert.Equal(t, Unit{}, (&Log{}).Result())
	assert.True(t, (&Now{}).Result().IsZero())
	assert.Nil(t, (&ReadFile{}).Result())
	assert.Zero(t, (&Random{}).Result())
}
