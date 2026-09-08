package effect

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
	live, _, mail := newWorld()
	blip := errors.New("network blip")
	var attempts []string

	// Every second call fails. Retry (3 tries each) and journal wrap the handler once.
	h := Retry(journal(FailEvery(HandlerOf(live), 2, blip), &attempts), 3)
	got, err := Run(context.Background(), h, Notify("name.txt", to))

	require.NoError(t, err)
	assert.Equal(t, "receipt-1", got)
	assert.Equal(t, []string{
		"*effect.ReadFile ok",
		"*effect.Now err: network blip",
		"*effect.Now ok",
		"*effect.Send err: network blip",
		"*effect.Send ok",
		"*effect.Log err: network blip",
		"*effect.Log ok",
	}, attempts, "every operation was retried by the same ten lines")
	assert.Equal(t, []Mail{{Key: "", To: to, Msg: greeting}}, mail.Sent,
		"FailEvery refuses before performing, so this retry was harmless. The next tests show when it is not.")
}

// strictPolicy is an exhaustive match. Add an operation to Effect and this
// function stops compiling until someone decides whether it may be retried.
func strictPolicy(op Effect) RetryPolicy {
	return MatchEffectR1(op,
		func(*Log) RetryPolicy { return RetryPolicy{Attempts: 2} },
		func(*Now) RetryPolicy { return RetryPolicy{Attempts: 2} },
		func(*ReadFile) RetryPolicy {
			return RetryPolicy{Attempts: 3, Backoff: ExponentialBackoff(10 * time.Millisecond)}
		},
		func(*Random) RetryPolicy { return RetryPolicy{Attempts: 2} },
		// Send is not idempotent. Without an idempotency key it must not be retried.
		func(*Send) RetryPolicy { return RetryPolicy{Attempts: 1} },
	)
}

// keyedPolicy may retry everything, because StepKeys gives every step an
// idempotency key and the mail server dedups on it.
func keyedPolicy(Effect) RetryPolicy { return RetryPolicy{Attempts: 3} }

func TestPart4_retryIsAPolicyPerOperation(t *testing.T) {
	blip := errors.New("network blip")

	t.Run("ReadFile is retried with backoff, Send is not", func(t *testing.T) {
		live, _, mail := newWorld()
		var attempts []string
		var waits []time.Duration
		fakeSleep := func(d time.Duration) { waits = append(waits, d) }

		// Fault: the 1st and 2nd call fail (ReadFile twice), then the 5th (Send).
		failing := flakyAt(HandlerOf(live), map[int]error{1: blip, 2: blip, 5: blip})
		h := RetryWith(journal(failing, &attempts), strictPolicy, nil, fakeSleep)

		_, err := Run(context.Background(), h, Notify("name.txt", to))

		require.ErrorIs(t, err, blip)
		assert.EqualError(t, err, "network blip", "Send was not retried, so its error passes through unwrapped")
		assert.Equal(t, []string{
			"*effect.ReadFile err: network blip",
			"*effect.ReadFile err: network blip",
			"*effect.ReadFile ok",
			"*effect.Now ok",
			"*effect.Send err: network blip",
		}, attempts)
		assert.Equal(t, []time.Duration{10 * time.Millisecond, 20 * time.Millisecond}, waits, "exponential backoff before each ReadFile retry")
		assert.Empty(t, mail.Sent, "the failed Send delivered nothing")
	})

	t.Run("a budget caps retries across the whole run", func(t *testing.T) {
		live, _, _ := newWorld()
		var attempts []string
		budget := &RetryBudget{Left: 1}

		// ReadFile fails twice. Policy allows 3 attempts, but the run may retry once.
		failing := flakyAt(HandlerOf(live), map[int]error{1: blip, 2: blip})
		h := RetryWith(journal(failing, &attempts), strictPolicy, budget, nil)

		_, err := Run(context.Background(), h, Notify("name.txt", to))

		require.ErrorIs(t, err, ErrRetryBudget)
		assert.Equal(t, []string{
			"*effect.ReadFile err: network blip",
			"*effect.ReadFile err: network blip",
		}, attempts, "one retry was spent; the second was refused by the budget")
		assert.Equal(t, 0, budget.Left)
	})

	t.Run("middleware order decides what the tape records", func(t *testing.T) {
		// Record outside Retry: the tape holds one committed answer per step.
		live, _, _ := newWorld()
		var committed []Step[Effect]
		failing := flakyAt(HandlerOf(live), map[int]error{1: blip})
		_, err := Run(context.Background(), Record(RetryWith(failing, strictPolicy, nil, nil), &committed), Notify("name.txt", to))
		require.NoError(t, err)
		assert.Equal(t, []Step[Effect]{
			{Op: &ReadFile{Path: "name.txt"}, Answer: []byte("Ada\n")},
			{Op: &Now{}, Answer: noon},
			{Op: &Send{To: to, Msg: greeting}, Answer: "receipt-1"},
			{Op: &Log{Msg: "sent receipt-1"}, Answer: Unit{}},
		}, committed)

		// Record inside Retry: the tape holds every attempt, failures included.
		live, _, _ = newWorld()
		var attempts []Step[Effect]
		failing = flakyAt(HandlerOf(live), map[int]error{1: blip})
		_, err = Run(context.Background(), RetryWith(Record(failing, &attempts), strictPolicy, nil, nil), Notify("name.txt", to))
		require.NoError(t, err)
		assert.Equal(t, []Step[Effect]{
			{Op: &ReadFile{Path: "name.txt"}, Err: blip},
			{Op: &ReadFile{Path: "name.txt"}, Answer: []byte("Ada\n")},
			{Op: &Now{}, Answer: noon},
			{Op: &Send{To: to, Msg: greeting}, Answer: "receipt-1"},
			{Op: &Log{Msg: "sent receipt-1"}, Answer: Unit{}},
		}, attempts)

		// Only the committed tape replays cleanly. The attempt tape replays the failure.
		got, err := Run(context.Background(), Replay(committed, nil), Notify("name.txt", to))
		require.NoError(t, err)
		assert.Equal(t, "receipt-1", got)

		_, err = Run(context.Background(), Replay(attempts, nil), Notify("name.txt", to))
		require.ErrorIs(t, err, blip, "a tape of attempts is not a tape of facts")
	})
}

func TestPart4_atLeastOnceAndIdempotencyKeys(t *testing.T) {
	lost := errors.New("connection reset after write")

	t.Run("without keys the mail is sent twice", func(t *testing.T) {
		live, _, mail := newWorld()
		// Call 3 is Send: the mail server delivers, then the answer is lost.
		h := RetryWith(LoseAnswerAt(HandlerOf(live), 3, lost), keyedPolicy, nil, nil)

		got, err := Run(context.Background(), h, Notify("name.txt", to))

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
		h := StepKeys(RetryWith(LoseAnswerAt(HandlerOf(live), 3, lost), keyedPolicy, nil, nil), "run-1")

		got, err := Run(context.Background(), h, Notify("name.txt", to))

		require.NoError(t, err)
		assert.Equal(t, "receipt-1", got, "the retry got the receipt of the first delivery")
		assert.Equal(t, []Mail{
			{Key: "run-1/3", To: to, Msg: greeting},
		}, mail.Sent, "delivered once")
	})
}

func TestPart4_crashAtEveryStepThenResume(t *testing.T) {
	// Reference run: no faults. Every resumed run below must reproduce it.
	live, out, mail := newWorld()
	want, err := Run(context.Background(), StepKeys(HandlerOf(live), "run-1"), Notify("name.txt", to))
	require.NoError(t, err)
	assert.Equal(t, "receipt-1", want)
	assert.Equal(t, []Mail{{Key: "run-1/3", To: to, Msg: greeting}}, mail.Sent)
	assert.Equal(t, "sent receipt-1\n", out.String())

	crash := errors.New("power cut")
	for k := 0; k <= len(notifyOps); k++ {
		t.Run(fmt.Sprintf("crash after step %d", k), func(t *testing.T) {
			// One world survives the crash: the mail server. Files and clock are the same.
			live, out, mail := newWorld()
			var tape []Step[Effect]

			// First life: perform k steps, then die.
			first := StepKeys(Record(CrashAfter(HandlerOf(live), k, crash), &tape), "run-1")
			_, err := Run(context.Background(), first, Notify("name.txt", to))
			if k == len(notifyOps) {
				require.NoError(t, err, "no crash point left")
			} else {
				require.ErrorIs(t, err, crash)
				assert.Equal(t, notifyOps[:k], opsOf(tape[:k]), "the tape holds exactly the steps that happened")
				assert.Equal(t, crash, tape[k].Err, "the crash itself is on the tape")
			}
			facts := tape[:k] // drop the crashed step; it is not a fact about the world

			// Second life: replay the facts, then continue live. Same key prefix, so
			// step 3 is still "run-1/3" even when it is replayed.
			second := StepKeys(Replay(facts, HandlerOf(live)), "run-1")
			got, err := Run(context.Background(), second, Notify("name.txt", to))

			require.NoError(t, err)
			assert.Equal(t, want, got, "same receipt as the reference run")
			assert.Equal(t, []Mail{{Key: "run-1/3", To: to, Msg: greeting}}, mail.Sent, "exactly one mail, whatever the crash point")
			// Log has no key, so it is at-least-once by choice. Here the crash is
			// always before a step, so the log line is never duplicated.
			assert.Equal(t, "sent receipt-1\n", out.String())
		})
	}
}

func TestPart4_seededChaos(t *testing.T) {
	const seeds = 500
	cfg := func(seed uint64) ChaosConfig { return ChaosConfig{Seed: seed, FailRate: 0.15, LoseAnswerRate: 0.15} }

	t.Run("without keys, chaos finds double delivery", func(t *testing.T) {
		duplicates := map[uint64][]Mail{}
		for seed := uint64(0); seed < seeds; seed++ {
			live, _, mail := newWorld()
			h := RetryWith(Chaos(HandlerOf(live), cfg(seed)), keyedPolicy, nil, nil)
			_, _ = Run(context.Background(), h, Notify("name.txt", to))
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
			h := StepKeys(RetryWith(Chaos(HandlerOf(live), cfg(seed)), keyedPolicy, nil, nil), "run")
			got, err := Run(context.Background(), h, Notify("name.txt", to))

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
		h := StepKeys(RetryWith(Chaos(HandlerOf(live), cfg(failingSeed)), keyedPolicy, nil, nil), "run")
		_, again := Run(context.Background(), h, Notify("name.txt", to))
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
