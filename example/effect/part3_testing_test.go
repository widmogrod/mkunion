package effect

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Part 3: testing with tapes.
//
// Because every operation and answer is data, a run can be recorded, saved as
// JSON, and replayed with no clock, files, network or mail. The program under
// test from here on is Notify, whose Send must never happen twice.

func TestPart3_theTraceIsTheAssertion(t *testing.T) {
	live, out, mail := newWorld()
	var trace []MyEff

	got, err := Run(context.Background(), Trace(MyEffHandlerFunc(live), &trace), Notify("name.txt", to))

	require.NoError(t, err)
	assert.Equal(t, "receipt-1", got)
	assert.Equal(t, notifyOps, trace, "what the program asked for, in order")
	assert.Equal(t, []Mail{{Key: "", To: to, Msg: greeting}}, mail.Sent, "what the world did about it")
	assert.Equal(t, "sent receipt-1\n", out.String())
}

// --8<-- [start:record-replay]

func TestPart3_recordOnceReplayForever(t *testing.T) {
	// Record: one run against the real world writes a tape of facts.
	live, _, _ := newWorld()
	var tape []Step[MyEff]
	want, err := Run(context.Background(), Record(MyEffHandlerFunc(live), &tape), Notify("name.txt", to))
	require.NoError(t, err)
	assert.Equal(t, []Step[MyEff]{
		{Op: &ReadFile{Path: "name.txt"}, Answer: []byte("Ada\n")},
		{Op: &Now{}, Answer: noon},
		{Op: &Send{To: to, Msg: greeting}, Answer: "receipt-1"},
		{Op: &Log{Msg: "sent receipt-1"}, Answer: Unit{}},
	}, tape)

	// Replay: the tape answers. No world is needed.
	got, err := Run(context.Background(), Replay(tape, nil), Notify("name.txt", to))
	require.NoError(t, err)
	assert.Equal(t, want, got)

	// Drift: a program that asks for something else fails loudly at the step.
	_, err = Run(context.Background(), Replay(tape, nil), Notify("other.txt", to))
	require.ErrorContains(t, err, "replay mismatch at step 1")

	// A short tape ends with a clear error instead of touching the world.
	_, err = Run(context.Background(), Replay(tape[:2], nil), Notify("name.txt", to))
	require.ErrorIs(t, err, ErrTapeEnded)
}

// --8<-- [end:record-replay]

func TestPart3_aTapeIsJSON(t *testing.T) {
	tape := notifyTape // the tape from the test above, as a literal

	data, err := TapeToJSON(tape)
	require.NoError(t, err)
	assert.JSONEq(t, `[
		{"op": {"$type": "effect.ReadFile", "effect.ReadFile": {"Path": "name.txt"}}, "answer": "QWRhCg=="},
		{"op": {"$type": "effect.Now",      "effect.Now": {}},                         "answer": "2026-09-01T12:00:00Z"},
		{"op": {"$type": "effect.Send",     "effect.Send": {"To": "ada@example.com", "Msg": "Hello Ada, it is 12:00PM"}}, "answer": "receipt-1"},
		{"op": {"$type": "effect.Log",      "effect.Log": {"Msg": "sent receipt-1"}}, "answer": {}}
	]`, string(data), "operations use the JSON mkunion generates for the union")

	// Ship it, load it, replay it: the JSON alone is enough.
	loaded, err := TapeFromJSON(data)
	require.NoError(t, err)
	assert.Equal(t, tape, loaded, "answers come back with the type each operation declared")
	got, err := Run(context.Background(), Replay(loaded, nil), Notify("name.txt", to))
	require.NoError(t, err)
	assert.Equal(t, "receipt-1", got)
}

func TestPart3_tapeJSONKeepsErrorsAndRejectsGarbage(t *testing.T) {
	data, err := TapeToJSON([]Step[MyEff]{{Op: &Now{}, Err: errors.New("clock down")}})
	require.NoError(t, err)
	assert.JSONEq(t, `[{"op": {"$type": "effect.Now", "effect.Now": {}}, "err": "clock down"}]`, string(data))
	loaded, err := TapeFromJSON(data)
	require.NoError(t, err)
	assert.EqualError(t, loaded[0].Err, "clock down")

	_, err = TapeFromJSON([]byte(`not json`))
	require.Error(t, err)
	_, err = TapeFromJSON([]byte(`[{"op":{"$type":"effect.Nope"}}]`))
	assert.EqualError(t, err, "step 0: effect.MyEffFromJSON: unknown type: effect.Nope")
	_, err = TapeFromJSON([]byte(`[{"op":{"$type":"effect.Random","effect.Random":{"Max":6}},"answer":"six"}]`))
	assert.EqualError(t, err, "step 0: json: cannot unmarshal string into Go value of type int", "an answer of the wrong type is refused")
}
