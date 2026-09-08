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

// --8<-- [start:run-fake]

func TestPart1_sameProgramFakeHandler(t *testing.T) {
	fake := &Fake{Clock: noon, Files: map[string]string{"name.txt": "Ada\n"}}
	var trace []MyEff

	got, err := Run(context.Background(), Trace(MyEffHandlerFunc(fake), &trace), Greet("name.txt"))

	require.NoError(t, err)
	assert.Equal(t, "Hello Ada, it is 12:00PM", got)
	assert.Equal(t, []string{"Hello Ada, it is 12:00PM"}, fake.Logs, "the Fake remembers what was logged")
	assert.Equal(t, []MyEff{
		&ReadFile{Path: "name.txt"},
		&Now{},
		&Log{Msg: "Hello Ada, it is 12:00PM"},
	}, trace, "every operation, in order, as data")
}

// --8<-- [end:run-fake]

func TestPart1_sameProgramLiveHandler(t *testing.T) {
	live, out, _ := newWorld()

	got, err := Run(context.Background(), MyEffHandlerFunc(live), Greet("name.txt"))

	require.NoError(t, err)
	assert.Equal(t, "Hello Ada, it is 12:00PM", got)
	assert.Equal(t, "Hello Ada, it is 12:00PM\n", out.String(), "Live wrote the log line for real")
}

func TestPart1_buildingAProgramPerformsNothing(t *testing.T) {
	fake := &Fake{Clock: noon, Files: map[string]string{"name.txt": "Ada"}}

	prog := Greet("name.txt")

	assert.Empty(t, fake.Logs, "nothing ran yet")
	_, err := Run(context.Background(), MyEffHandlerFunc(fake), prog)
	require.NoError(t, err)
	assert.Len(t, fake.Logs, 1, "now it did")
}

func TestPart1_handlerErrorStopsTheProgram(t *testing.T) {
	fake := &Fake{Clock: noon} // no files
	var trace []MyEff

	_, err := Run(context.Background(), Trace(MyEffHandlerFunc(fake), &trace), Greet("missing.txt"))

	require.ErrorContains(t, err, `no file "missing.txt"`)
	assert.Equal(t, []MyEff{&ReadFile{Path: "missing.txt"}}, trace, "nothing after the failing operation runs")
	assert.Empty(t, fake.Logs)
}

func TestPart1_cancelledContextStopsBeforeTheNextOperation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var trace []MyEff

	_, err := Run(ctx, Trace(MyEffHandlerFunc(&Fake{}), &trace), Greet("name.txt"))

	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, trace)
}

func TestPart1_attemptHandlesAnErrorInPlace(t *testing.T) {
	fake := &Fake{} // no files: ReadFile fails, the body carries on

	got, err := Run(context.Background(), MyEffHandlerFunc(fake), GreetOrGuest("missing.txt"))

	require.NoError(t, err)
	assert.Equal(t, "Hello guest", got)
	assert.Equal(t, []string{"Hello guest"}, fake.Logs)
}

// --8<-- [start:clock-only]

// clockOnly overrides one method; Defaults supplies the other four.
type clockOnly struct {
	Defaults
	at time.Time
}

func (c clockOnly) HandleNow(context.Context, *Now) (time.Time, error) { return c.at, nil }

// --8<-- [end:clock-only]

func TestPart1_defaultsLetATestOverrideOneMethod(t *testing.T) {
	prog := Prog(func(fx Fx) (string, error) {
		fx.Log("ignored by Defaults")
		return fx.Now().Format(time.Kitchen) + " and rolled " + strconv.Itoa(fx.Random(6)), nil
	})

	got, err := Run(context.Background(), MyEffHandlerFunc(clockOnly{at: noon}), prog)

	require.NoError(t, err)
	assert.Equal(t, "12:00PM and rolled 0", got)

	_, err = Run(context.Background(), MyEffHandlerFunc(clockOnly{}), Greet("name.txt"))
	assert.EqualError(t, err, `defaults: no file "name.txt"`, "Defaults refuses reads, so a test cannot depend on one by accident")
}

func TestPart1_aLoopIsALoop(t *testing.T) {
	const rolls = 1_000_000
	_, err := Run(context.Background(), MyEffHandlerFunc(&Fake{Rolls: []int{0}}), RollUntil(6, rolls))
	require.ErrorContains(t, err, "no 6 in 1000000 rolls", "a million operations, no stack growth")

	got, err := Run(context.Background(), MyEffHandlerFunc(&Fake{Rolls: []int{0, 0, 5}}), RollUntil(6, rolls))
	require.NoError(t, err)
	assert.Equal(t, 3, got, "the third roll was a six")
}

func TestPart1_aFakeSaysWhenItHasNoAnswer(t *testing.T) {
	_, err := Run(context.Background(), MyEffHandlerFunc(&Fake{}), RollUntil(6, 1))
	assert.EqualError(t, err, "fake: no rolls configured")
}

func TestPart1_fxDoInfersTheAnswerType(t *testing.T) {
	prog := Prog(func(fx Fx) (time.Time, error) {
		return fx.Do(&Now{}), nil // no type annotation: R comes from Now's f.Returns
	})

	got, err := Run(context.Background(), MyEffHandlerFunc(&Fake{Clock: noon}), prog)

	require.NoError(t, err)
	assert.Equal(t, noon, got)
}

// Program[A] is an alias, not a new type: the assignment below is checked by
// the compiler, so there is nothing left to test at run time.
var _ Eff[MyEff, string] = Program[string](nil)
