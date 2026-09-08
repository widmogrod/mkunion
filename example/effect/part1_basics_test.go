package effect

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Part 1: the basics.
//
// A program is plain Go against Fx. It performs nothing until Run gives it a
// handler. The same program runs against Fake in tests and Live in production.

func TestPart1_sameProgramFakeHandler(t *testing.T) {
	fake := &Fake{Clock: noon, Files: map[string]string{"name.txt": "Ada\n"}}
	var trace []Effect

	got, err := Run(context.Background(), Trace(HandlerOf(fake), &trace), Greet("name.txt"))

	require.NoError(t, err)
	assert.Equal(t, "Hello Ada, it is 12:00PM", got)
	assert.Equal(t, []string{"Hello Ada, it is 12:00PM"}, fake.Logs, "the Fake remembers what was logged")
	assert.Equal(t, []Effect{
		&ReadFile{Path: "name.txt"},
		&Now{},
		&Log{Msg: "Hello Ada, it is 12:00PM"},
	}, trace, "every operation, in order, as data")
}

func TestPart1_sameProgramLiveHandler(t *testing.T) {
	live, out, _ := newWorld()

	got, err := Run(context.Background(), HandlerOf(live), Greet("name.txt"))

	require.NoError(t, err)
	assert.Equal(t, "Hello Ada, it is 12:00PM", got)
	assert.Equal(t, "Hello Ada, it is 12:00PM\n", out.String(), "Live wrote the log line for real")
}

func TestPart1_buildingAProgramPerformsNothing(t *testing.T) {
	fake := &Fake{Clock: noon, Files: map[string]string{"name.txt": "Ada"}}

	prog := Greet("name.txt")

	assert.Empty(t, fake.Logs, "nothing ran yet")
	_, err := Run(context.Background(), HandlerOf(fake), prog)
	require.NoError(t, err)
	assert.Len(t, fake.Logs, 1, "now it did")
}

func TestPart1_handlerErrorStopsTheProgram(t *testing.T) {
	fake := &Fake{Clock: noon} // no files
	var trace []Effect

	_, err := Run(context.Background(), Trace(HandlerOf(fake), &trace), Greet("missing.txt"))

	require.ErrorContains(t, err, `no file "missing.txt"`)
	assert.Equal(t, []Effect{&ReadFile{Path: "missing.txt"}}, trace, "nothing after the failing operation runs")
	assert.Empty(t, fake.Logs)
}

func TestPart1_cancelledContextStopsBeforeTheNextOperation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var trace []Effect

	_, err := Run(ctx, Trace(HandlerOf(&Fake{}), &trace), Greet("name.txt"))

	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, trace)
}

func TestPart1_attemptHandlesAnErrorInPlace(t *testing.T) {
	fake := &Fake{} // no files: ReadFile fails, the body carries on

	got, err := Run(context.Background(), HandlerOf(fake), GreetOrGuest("missing.txt"))

	require.NoError(t, err)
	assert.Equal(t, "Hello guest", got)
	assert.Equal(t, []string{"Hello guest"}, fake.Logs)
}

// clockOnly overrides one method; Defaults supplies the other four.
type clockOnly struct {
	Defaults
	at time.Time
}

func (c clockOnly) HandleNow(context.Context, *Now) (time.Time, error) { return c.at, nil }

func TestPart1_defaultsLetATestOverrideOneMethod(t *testing.T) {
	prog := Prog(func(fx Fx) (string, error) {
		fx.Log("ignored by Defaults")
		return fx.Now().Format(time.Kitchen) + " and rolled " + strconv.Itoa(fx.Random(6)), nil
	})

	got, err := Run(context.Background(), HandlerOf(clockOnly{at: noon}), prog)

	require.NoError(t, err)
	assert.Equal(t, "12:00PM and rolled 0", got)

	_, err = Run(context.Background(), HandlerOf(clockOnly{}), Greet("name.txt"))
	assert.EqualError(t, err, `defaults: no file "name.txt"`, "Defaults refuses reads, so a test cannot depend on one by accident")
}

func TestPart1_aLoopIsALoop(t *testing.T) {
	const rolls = 1_000_000
	_, err := Run(context.Background(), HandlerOf(&Fake{Rolls: []int{0}}), RollUntil(6, rolls))
	require.ErrorContains(t, err, "no 6 in 1000000 rolls", "a million operations, no stack growth")

	got, err := Run(context.Background(), HandlerOf(&Fake{Rolls: []int{0, 0, 5}}), RollUntil(6, rolls))
	require.NoError(t, err)
	assert.Equal(t, 3, got, "the third roll was a six")

	_, err = Run(context.Background(), HandlerOf(&Fake{}), RollUntil(6, 1))
	require.ErrorContains(t, err, "no rolls configured", "a Fake with no rolls says so")
}

func TestPart1_fxDoInfersTheAnswerType(t *testing.T) {
	prog := Prog(func(fx Fx) (time.Time, error) {
		return fx.Do(&Now{}), nil // no type annotation: R comes from Now's Result method
	})

	got, err := Run(context.Background(), HandlerOf(&Fake{Clock: noon}), prog)

	require.NoError(t, err)
	assert.Equal(t, noon, got)
}

func TestPart1_programIsAnAliasForEff(t *testing.T) {
	var p Program[string] = Greet("name.txt")
	var e Eff[Effect, string] = p
	assert.NotNil(t, e, "same type, shorter name")
}
