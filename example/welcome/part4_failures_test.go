package welcome

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Part 4: failure is normal.
//
// Middleware wraps the handler once and sees every operation. That is where
// retries, idempotency keys and fault injection live. The nasty case is not
// "the call failed" but "the mail went out, then the answer was lost".

func TestPart4_oneMiddlewareCoversEveryOperation(t *testing.T) {
	program := Notify("name.txt", to)

	live, _, mail := newWorld()
	blip := errors.New("network blip")
	var attempts []string

	// Every second call fails. Retry (3 tries each) and journal wrap the handler once.
	// Middleware is listed outermost first: Retry sees journal, journal sees FailEvery.
	got, err := Interpret(context.Background(), program, live,
		Retry[MyEff](3), journal[MyEff](&attempts), FailEvery[MyEff](2, blip))

	require.NoError(t, err)
	assert.Equal(t, "receipt-1", got)
	assert.Equal(t, []string{
		"*welcome.ReadFile ok",
		"*welcome.Now err: network blip",
		"*welcome.Now ok",
		"*welcome.Send err: network blip",
		"*welcome.Send ok",
		"*welcome.Log err: network blip",
		"*welcome.Log ok",
	}, attempts, "every operation was retried by the same wrapper")
	assert.Equal(t, []Mail{{Key: "", To: to, Msg: greeting}}, mail.Sent,
		"FailEvery refuses before performing, so this retry was harmless. The next tests show when it is not.")
}

// --8<-- [start:strict-policy]

// strictPolicy is an exhaustive match. Add an operation to MyEff and this
// function stops compiling until someone decides whether it may be retried.
func strictPolicy(op MyEff) RetryPolicy {
	return MatchMyEffR1(op,
		func(*Log) RetryPolicy { return RetryPolicy{Attempts: 2} },
		func(*Now) RetryPolicy { return RetryPolicy{Attempts: 2} },
		func(*ReadFile) RetryPolicy {
			return RetryPolicy{Attempts: 3, Backoff: ExponentialBackoff(10 * time.Millisecond)}
		},
		func(*Random) RetryPolicy { return RetryPolicy{Attempts: 2} },
		// Send is not idempotent. Without an idempotency key it must not be retried.
		func(*Send) RetryPolicy { return RetryPolicy{Attempts: 1} },
		// Charge may be retried on a broken line. A refusal is an answer, not a
		// failure, so no policy can retry it (TestPart4_aRefusalIsAnAnswerNotAFailure).
		func(*Charge) RetryPolicy { return RetryPolicy{Attempts: 3} },
	)
}

// --8<-- [end:strict-policy]

// keyedPolicy may retry everything, because StepKeys gives every step an
// idempotency key and the mail server dedups on it.
func keyedPolicy(MyEff) RetryPolicy { return RetryPolicy{Attempts: 3} }

func TestPart4_retryIsAPolicyPerOperation(t *testing.T) {
	blip := errors.New("network blip")

	t.Run("ReadFile is retried with backoff, Send is not", func(t *testing.T) {
		live, _, mail := newWorld()
		var attempts []string
		var waits []time.Duration
		fakeSleep := func(d time.Duration) { waits = append(waits, d) }

		// Fault: the 1st and 2nd call fail (ReadFile twice), then the 5th (Send).
		_, err := Interpret(context.Background(), Notify("name.txt", to), live,
			RetryWith(strictPolicy, nil, fakeSleep),
			journal[MyEff](&attempts),
			flakyAt[MyEff](map[int]error{1: blip, 2: blip, 5: blip}))

		require.ErrorIs(t, err, blip)
		assert.EqualError(t, err, "network blip", "Send was not retried, so its error passes through unwrapped")
		assert.Equal(t, []string{
			"*welcome.ReadFile err: network blip",
			"*welcome.ReadFile err: network blip",
			"*welcome.ReadFile ok",
			"*welcome.Now ok",
			"*welcome.Send err: network blip",
		}, attempts)
		assert.Equal(t, []time.Duration{10 * time.Millisecond, 20 * time.Millisecond}, waits, "exponential backoff before each ReadFile retry")
		assert.Empty(t, mail.Sent, "the failed Send delivered nothing")
	})

	t.Run("a budget caps retries across the whole run", func(t *testing.T) {
		live, _, _ := newWorld()
		var attempts []string
		budget := &RetryBudget{Left: 1}

		// ReadFile fails twice. Policy allows 3 attempts, but the run may retry once.
		_, err := Interpret(context.Background(), Notify("name.txt", to), live,
			RetryWith(strictPolicy, budget, nil),
			journal[MyEff](&attempts),
			flakyAt[MyEff](map[int]error{1: blip, 2: blip}))

		require.ErrorIs(t, err, ErrRetryBudget)
		assert.Equal(t, []string{
			"*welcome.ReadFile err: network blip",
			"*welcome.ReadFile err: network blip",
		}, attempts, "one retry was spent; the second was refused by the budget")
		assert.Equal(t, 0, budget.Left)
	})

	t.Run("middleware order decides what the tape records", func(t *testing.T) {
		program := Notify("name.txt", to)

		// Record outside Retry: the tape holds one committed answer per step.
		live, _, _ := newWorld()
		var committed []Step[MyEff]
		_, err := Interpret(context.Background(), program, live,
			Record(&committed), RetryWith(strictPolicy, nil, nil), flakyAt[MyEff](map[int]error{1: blip}))
		require.NoError(t, err)
		assert.Equal(t, []Step[MyEff]{
			{Op: &ReadFile{Path: "name.txt"}, Answer: []byte("Ada\n")},
			{Op: &Now{}, Answer: noon},
			{Op: &Send{To: to, Msg: greeting}, Answer: "receipt-1"},
			{Op: &Log{Msg: "sent receipt-1"}, Answer: Unit{}},
		}, committed)

		// Record inside Retry: the tape holds every attempt, failures included.
		live, _, _ = newWorld()
		var attempts []Step[MyEff]
		_, err = Interpret(context.Background(), program, live,
			RetryWith(strictPolicy, nil, nil), Record(&attempts), flakyAt[MyEff](map[int]error{1: blip}))
		require.NoError(t, err)
		assert.Equal(t, []Step[MyEff]{
			{Op: &ReadFile{Path: "name.txt"}, Err: blip},
			{Op: &ReadFile{Path: "name.txt"}, Answer: []byte("Ada\n")},
			{Op: &Now{}, Answer: noon},
			{Op: &Send{To: to, Msg: greeting}, Answer: "receipt-1"},
			{Op: &Log{Msg: "sent receipt-1"}, Answer: Unit{}},
		}, attempts)

		// Only the committed tape replays cleanly. The attempt tape replays the failure.
		got, err := Run(context.Background(), Replay(committed, nil), program)
		require.NoError(t, err)
		assert.Equal(t, "receipt-1", got)

		_, err = Run(context.Background(), Replay(attempts, nil), program)
		require.ErrorIs(t, err, blip, "a tape of attempts is not a tape of facts")
	})
}

func TestPart4_atLeastOnceAndIdempotencyKeys(t *testing.T) {
	lost := errors.New("connection reset after write")

	t.Run("without keys the mail is sent twice", func(t *testing.T) {
		live, _, mail := newWorld()
		// Call 3 is Send: the mail server delivers, then the answer is lost.
		got, err := Interpret(context.Background(), Notify("name.txt", to), live,
			RetryWith(keyedPolicy, nil, nil), LoseAnswerAt[MyEff](3, lost))

		require.NoError(t, err)
		assert.Equal(t, "receipt-2", got, "the program only saw the second delivery")
		assert.Equal(t, []Mail{
			{Key: "", To: to, Msg: greeting},
			{Key: "", To: to, Msg: greeting},
		}, mail.Sent, "Ada got the same mail twice")
	})

	t.Run("with keys the mail server drops the duplicate", func(t *testing.T) {
		live, _, mail := newWorld()
		// StepKeys is outermost, so both attempts of step 3 carry "run-1/3".
		got, err := Interpret(context.Background(), Notify("name.txt", to), live,
			StepKeys[MyEff]("run-1"), RetryWith(keyedPolicy, nil, nil), LoseAnswerAt[MyEff](3, lost))

		require.NoError(t, err)
		assert.Equal(t, "receipt-1", got, "the retry got the receipt of the first delivery")
		assert.Equal(t, []Mail{
			{Key: "run-1/3", To: to, Msg: greeting},
		}, mail.Sent, "delivered once")
	})
}

func TestPart4_crashAtEveryStepThenResume(t *testing.T) {
	program := Notify("name.txt", to)

	// Reference run: no faults. Every resumed run below must reproduce it.
	live, out, mail := newWorld()
	want, err := Interpret(context.Background(), program, live, StepKeys[MyEff]("run-1"))
	require.NoError(t, err)
	assert.Equal(t, "receipt-1", want)
	assert.Equal(t, []Mail{{Key: "run-1/3", To: to, Msg: greeting}}, mail.Sent)
	assert.Equal(t, "sent receipt-1\n", out.String())

	// --8<-- [start:crash-resume]
	crash := errors.New("power cut")
	for k := 0; k <= len(notifyOps); k++ {
		t.Run(fmt.Sprintf("crash after step %d", k), func(t *testing.T) {
			// One world survives the crash: the mail server. Files and clock are the same.
			live, out, mail := newWorld()
			var tape []Step[MyEff]

			// First life: perform k steps, then die.
			_, err := Interpret(context.Background(), program, live,
				StepKeys[MyEff]("run-1"), Record(&tape), CrashAfter[MyEff](k, crash))
			if k == len(notifyOps) {
				require.NoError(t, err, "no crash point left")
			} else {
				require.ErrorIs(t, err, crash)
				assert.Equal(t, notifyOps[:k], opsOf(tape[:k]), "the tape holds exactly the steps that happened")
				assert.Equal(t, crash, tape[k].Err, "the crash itself is on the tape")
			}
			facts := tape[:k] // drop the crashed step; it is not a fact about the world

			// Second life: replay the facts, then continue live. Same key prefix, so
			// step 3 is still "run-1/3" even when it is replayed. Replay is a handler
			// of its own, so this goes through Run and Wrap, the pieces under Interpret.
			second := Wrap(Replay(facts, MyEffHandlerFunc(live)), StepKeys[MyEff]("run-1"))
			got, err := Run(context.Background(), second, program)

			require.NoError(t, err)
			assert.Equal(t, want, got, "same receipt as the reference run")
			assert.Equal(t, []Mail{{Key: "run-1/3", To: to, Msg: greeting}}, mail.Sent, "exactly one mail, whatever the crash point")
			// Log has no key, so it is at-least-once by choice. Here the crash is
			// always before a step, so the log line is never duplicated.
			assert.Equal(t, "sent receipt-1\n", out.String())
		})
	}
	// --8<-- [end:crash-resume]
}

func TestPart4_seededChaos(t *testing.T) {
	const seeds = 500
	cfg := func(seed uint64) ChaosConfig { return ChaosConfig{Seed: seed, FailRate: 0.15, LoseAnswerRate: 0.15} }

	t.Run("without keys, chaos finds double delivery", func(t *testing.T) {
		duplicates := map[uint64][]Mail{}
		for seed := uint64(0); seed < seeds; seed++ {
			live, _, mail := newWorld()
			_, _ = Interpret(context.Background(), Notify("name.txt", to), live,
				RetryWith(keyedPolicy, nil, nil), Chaos[MyEff](cfg(seed)))
			if len(mail.Sent) > 1 {
				duplicates[seed] = mail.Sent
			}
		}
		require.NotEmpty(t, duplicates, "some seed must hit: Send performed, answer lost, retried")
		t.Logf("double delivery in %d of %d seeds", len(duplicates), seeds)

		first := firstSeed(duplicates)
		assert.Equal(t, []Mail{
			{Key: "", To: to, Msg: greeting},
			{Key: "", To: to, Msg: greeting},
		}, duplicates[first], "seed %d: the shape of the bug", first)
	})

	t.Run("with keys, every world is exactly-once or a clean failure", func(t *testing.T) {
		outcomes := map[string]int{}
		var failingSeed uint64
		var failingErr error
		for seed := uint64(0); seed < seeds; seed++ {
			live, _, mail := newWorld()
			got, err := Interpret(context.Background(), Notify("name.txt", to), live,
				StepKeys[MyEff]("run"), RetryWith(keyedPolicy, nil, nil), Chaos[MyEff](cfg(seed)))

			// The invariants. They hold in every world or the test fails with the seed.
			if err == nil {
				outcomes["ok"]++
				require.Equal(t, "receipt-1", got, "seed %d", seed)
				require.Equal(t, []Mail{{Key: "run/3", To: to, Msg: greeting}}, mail.Sent, "seed %d", seed)
			} else {
				outcomes["failed cleanly"]++
				require.ErrorIs(t, err, ErrChaos, "seed %d: only injected faults may surface", seed)
				require.LessOrEqual(t, len(mail.Sent), 1, "seed %d: never more than one mail", seed)
				if failingErr == nil {
					failingSeed, failingErr = seed, err
				}
			}
		}
		t.Logf("outcomes over %d seeds: %v", seeds, outcomes)
		require.Positive(t, outcomes["ok"])
		require.Positive(t, outcomes["failed cleanly"], "chaos strong enough to exhaust retries sometimes")

		// A failing world is reproducible from its seed alone.
		live, _, _ := newWorld()
		_, again := Interpret(context.Background(), Notify("name.txt", to), live,
			StepKeys[MyEff]("run"), RetryWith(keyedPolicy, nil, nil), Chaos[MyEff](cfg(failingSeed)))
		assert.ErrorIs(t, again, ErrChaos)
		assert.Regexp(t, `^effect: \*welcome\.\w+ failed after 3 attempts: chaos: `, again.Error(), "seed %d", failingSeed)
		assert.EqualError(t, again, failingErr.Error(), "seed %d replays the same failure", failingSeed)
	})
}

func firstSeed(m map[uint64][]Mail) uint64 {
	first := ^uint64(0)
	for seed := range m {
		first = min(first, seed)
	}
	return first
}

// --8<-- [start:refusal]

// A refused charge is an answer. A broken line to the bank is a failure.
// The type system keeps them apart, so Retry can only ever see the second.
func TestPart4_aRefusalIsAnAnswerNotAFailure(t *testing.T) {
	t.Run("a refusal is not retried, whatever the policy says", func(t *testing.T) {
		program := Pay(10)

		fake := &Fake{Budget: 5}
		var attempts []string

		// strictPolicy allows 3 attempts for Charge. It never gets to use them.
		_, err := Interpret(context.Background(), program, fake,
			RetryWith(strictPolicy, nil, nil), journal[MyEff](&attempts))

		assert.EqualError(t, err, "charge refused: short by 5", "the program decided, in Pay")
		assert.Equal(t, []string{"*welcome.Charge ok"}, attempts, "one attempt: the bank answered, so there was nothing to retry")
		assert.Empty(t, fake.Charged)
	})

	t.Run("a broken line is retried, and the charge goes through once", func(t *testing.T) {
		program := Pay(10)

		fake := &Fake{Budget: 15}
		blip := errors.New("bank: connection reset")
		var attempts []string

		got, err := Interpret(context.Background(), program, fake,
			RetryWith(strictPolicy, nil, nil), journal[MyEff](&attempts), flakyAt[MyEff](map[int]error{1: blip}))

		require.NoError(t, err)
		assert.Equal(t, "charge-1", got)
		assert.Equal(t, []string{
			"*welcome.Charge err: bank: connection reset",
			"*welcome.Charge ok",
			"*welcome.Log ok",
		}, attempts)
		assert.Equal(t, []int{10}, fake.Charged)
	})

	t.Run("every refusal is a variant, and the program matches all of them", func(t *testing.T) {
		live, _, _ := newWorld() // budget 15, quota 2 charges, resets at 1PM
		run := func(amount int) (string, error) {
			return Interpret(context.Background(), Pay(amount), live)
		}

		got, err := run(10)
		require.NoError(t, err)
		assert.Equal(t, "charge-1", got)

		_, err = run(10)
		assert.EqualError(t, err, "charge refused: short by 5")

		got, err = run(5)
		require.NoError(t, err)
		assert.Equal(t, "charge-2", got)

		_, err = run(1)
		assert.EqualError(t, err, "charge refused: quota resets at 1:00PM")
	})

	t.Run("a refusal on a tape keeps its variant", func(t *testing.T) {
		var tape []Step[MyEff]
		_, err := Interpret(context.Background(), Pay(10), &Fake{Budget: 5}, Record(&tape))
		assert.EqualError(t, err, "charge refused: short by 5")

		data, err := TapeToJSON(tape)
		require.NoError(t, err)
		assert.JSONEq(t, `[
			{"op": {"$type": "welcome.Charge", "welcome.Charge": {"Amount": 10}},
			 "answer": {"$type": "f.Err", "f.Err": {"Error": {"$type": "welcome.OutOfBudget", "welcome.OutOfBudget": {"Missing": 5}}}}}
		]`, string(data), "the answer is a union, so it is written with the JSON mkunion generates for it")

		loaded, err := TapeFromJSON(data)
		require.NoError(t, err)
		assert.Equal(t, tape, loaded)
	})
}

// --8<-- [end:refusal]
