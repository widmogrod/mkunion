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

// One test per benefit of "every effect is data that passes through one door".

func newLive() (*Live, *bytes.Buffer) {
	var out bytes.Buffer
	return &Live{
		Out:  &out,
		FS:   fstest.MapFS{"name.txt": {Data: []byte("Ada\n")}},
		Rand: rand.New(rand.NewPCG(1, 2)),
		Now:  func() time.Time { return noon },
	}, &out
}

// 1. One trace of everything: bodies, nested calls, all styles, one ordered list.
func TestBenefit1_oneTraceOfEverything(t *testing.T) {
	live, _ := newLive()
	var trace []Effect

	_, err := Run(context.Background(), Trace(HandlerOf(live), &trace), GreetFx("name.txt"))

	require.NoError(t, err)
	assert.Equal(t, []Effect{
		&ReadFile{Path: "name.txt"},
		&Now{},
		&Log{Msg: "Hello Ada, it is 12:00PM"},
	}, trace)
}

// 2. Middleware for all effects at once: retry and metrics wrap the handler once,
// and every operation of every program gets them.
func TestBenefit2_middlewareForAllEffects(t *testing.T) {
	live, _ := newLive()
	counts := map[string]int{}
	flaky := Count(FailEvery(HandlerOf(live), 2, errors.New("network blip")), counts)

	got, err := Run(context.Background(), Retry(flaky, 3), GreetFx("name.txt"))

	require.NoError(t, err)
	assert.Equal(t, "Hello Ada, it is 12:00PM", got)
	assert.Equal(t, map[string]int{"*effect.ReadFile": 1, "*effect.Now": 2, "*effect.Log": 2}, counts,
		"every second call failed and was retried; Count saw every attempt")
}

// 3. Record and replay: run once for real, then replay with no clock, files or
// network and get the same answer. A golden test for any body, for free.
func TestBenefit3_recordAndReplay(t *testing.T) {
	live, _ := newLive()
	var tape []Step[Effect]
	want, err := Run(context.Background(), Record(HandlerOf(live), &tape), GreetFx("name.txt"))
	require.NoError(t, err)
	require.Len(t, tape, 3)

	got, err := Run(context.Background(), Replay(tape, nil), GreetFx("name.txt"))

	require.NoError(t, err)
	assert.Equal(t, want, got)

	// The replay also guards against drift: a program that asks for something else fails loudly.
	_, err = Run(context.Background(), Replay(tape, nil), GreetFx("other.txt"))
	require.ErrorContains(t, err, "replay mismatch at step 1")

	_, err = Run(context.Background(), Replay(tape, nil), RollUntilFx(6, 1))
	require.ErrorContains(t, err, "replay mismatch")

	_, err = Run(context.Background(), Replay(tape[:2], nil), GreetFx("name.txt"))
	require.ErrorIs(t, err, ErrTapeEnded)
}

// 4. Pause and resume: the program crashes after two steps. The tape has those two
// steps. Resume replays them and continues live from step three. The body runs
// from the top again, but the real world is only asked for what is new.
func TestBenefit4_pauseAndResume(t *testing.T) {
	crash := errors.New("power cut")
	live, _ := newLive()
	var tape []Step[Effect]

	_, err := Run(context.Background(), Record(FailEvery(HandlerOf(live), 3, crash), &tape), GreetFx("name.txt"))
	require.ErrorIs(t, err, crash)
	tape = tape[:2] // the failed step is not a fact about the world; drop it

	live2, out := newLive()
	counts := map[string]int{}
	got, err := Run(context.Background(), Replay(tape, Count(HandlerOf(live2), counts)), GreetFx("name.txt"))

	require.NoError(t, err)
	assert.Equal(t, "Hello Ada, it is 12:00PM", got)
	assert.Equal(t, map[string]int{"*effect.Log": 1}, counts, "only the third step touched the world")
	assert.Equal(t, "Hello Ada, it is 12:00PM\n", out.String())
}

// 5. Fault injection: bad weather for the whole program, deterministic and in-process.
func TestBenefit5_faultInjection(t *testing.T) {
	blip := errors.New("disk hiccup")

	// Without retry the third operation fails, every time.
	live, _ := newLive()
	var trace []Effect
	_, err := Run(context.Background(), Trace(FailEvery(HandlerOf(live), 3, blip), &trace), GreetFx("name.txt"))
	require.ErrorIs(t, err, blip)
	assert.Len(t, trace, 3)

	// A jumping clock is a handler too: embed Defaults, override one method.
	got, err := Run(context.Background(), HandlerOf(&jumpingClock{}), Prog(func(fx Fx) (int, error) {
		return int(fx.Now().Sub(fx.Now()).Hours()), nil
	}))
	require.NoError(t, err)
	assert.Equal(t, -24, got, "time went backwards by a day between two reads")
}

type jumpingClock struct {
	Defaults
	calls int
}

func (c *jumpingClock) HandleNow(context.Context, *Now) (time.Time, error) {
	c.calls++
	return noon.AddDate(0, 0, c.calls), nil
}

// 6. The algebra is a mkunion union: operations have JSON. A recording can be
// saved, shipped, and replayed on another machine from the JSON alone.
func TestBenefit6_recordingIsJSON(t *testing.T) {
	live, _ := newLive()
	var tape []Step[Effect]
	want, err := Run(context.Background(), Record(HandlerOf(live), &tape), GreetFx("name.txt"))
	require.NoError(t, err)

	data, err := TapeToJSON(tape)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"effect.ReadFile"`)
	assert.Contains(t, string(data), `"Path":"name.txt"`)

	loaded, err := TapeFromJSON(data)
	require.NoError(t, err)
	got, err := Run(context.Background(), Replay(loaded, nil), GreetFx("name.txt"))

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestTapeJSON_errorsAndBadInput(t *testing.T) {
	boom := errors.New("boom")
	data, err := TapeToJSON([]Step[Effect]{{Op: &Now{}, Err: boom}})
	require.NoError(t, err)
	loaded, err := TapeFromJSON(data)
	require.NoError(t, err)
	require.Len(t, loaded, 1)
	assert.EqualError(t, loaded[0].Err, "boom")

	_, err = TapeFromJSON([]byte(`not json`))
	require.Error(t, err)
	_, err = TapeFromJSON([]byte(`[{"op":{"$type":"effect.Nope"}}]`))
	require.ErrorContains(t, err, "step 0")
	_, err = TapeFromJSON([]byte(`[{"op":{"$type":"effect.Random","effect.Random":{"Max":6}},"answer":"six"}]`))
	require.ErrorContains(t, err, "step 0")
}
