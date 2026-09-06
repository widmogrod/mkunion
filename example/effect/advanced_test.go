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

// Beyond the happy path. Each test states the data it expects in full, so the
// shape of a tape, a trace or a mailbox is visible in the test itself.
//
// The program under test is Notify. Its operations, in order:
//
//	1. ReadFile{Path: "name.txt"}   -> "Ada\n"
//	2. Now{}                        -> noon
//	3. Send{To, Msg}                -> "receipt-1"   (must never happen twice)
//	4. Log{Msg: "sent receipt-1"}   -> Unit{}

const to = "ada@example.com"

var (
	greeting  = "Hello Ada, it is 12:00PM"
	notifyOps = []Effect{
		&ReadFile{Path: "name.txt"},
		&Now{},
		&Send{To: to, Msg: greeting},
		&Log{Msg: "sent receipt-1"},
	}
)

// journal records every attempt the wrapped handler sees, with its outcome.
// It is the "what really happened" view the tests assert on.
func journal[Op any](h Handler[Op], lines *[]string) Handler[Op] {
	return func(ctx context.Context, op Op) (any, error) {
		answer, err := h(ctx, op)
		outcome := "ok"
		if err != nil {
			outcome = "err: " + err.Error()
		}
		*lines = append(*lines, fmt.Sprintf("%T %s", op, outcome))
		return answer, err
	}
}

// ---------------------------------------------------------------------------
// 1. Retries are a policy per operation, not one number.
// ---------------------------------------------------------------------------

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

func TestAdvanced1_retryPolicyPerOperation(t *testing.T) {
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
		assert.Equal(t, "network blip", err.Error(), "Send was not retried, so its error passes through unwrapped")
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

// flakyAt fails the calls listed in errs (1-based call number) before performing them.
func flakyAt[Op any](h Handler[Op], errs map[int]error) Handler[Op] {
	calls := 0
	return func(ctx context.Context, op Op) (any, error) {
		calls++
		if err, ok := errs[calls]; ok {
			return nil, err
		}
		return h(ctx, op)
	}
}

// ---------------------------------------------------------------------------
// 2. At-least-once is the real problem. Idempotency keys minted by the run fix it.
// ---------------------------------------------------------------------------

// keyedPolicy may retry Send, because every step carries an idempotency key.
func keyedPolicy(Effect) RetryPolicy { return RetryPolicy{Attempts: 3} }

func TestAdvanced2_idempotencyKeysMakeSendSafeToRetry(t *testing.T) {
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

// ---------------------------------------------------------------------------
// 3. Crash at every step, resume, and prove exactly-once. A complete search.
// ---------------------------------------------------------------------------

func TestAdvanced3_crashAtEveryStepThenResume(t *testing.T) {
	// Reference run: no faults. This is what every resumed run must reproduce.
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

			// First life: perform k steps, then die. The k-th answer may already be
			// in the world (CrashAfter fails the call after the k-th, so step k committed).
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

			// Log has no key, so it is at-least-once. Crash right after the Log
			// (k == 4 is the clean run; k == 3 crashes before Log) never duplicates
			// here, but a crash after Log with a lost answer would. That is a choice.
			assert.Equal(t, "sent receipt-1\n", out.String())
		})
	}
}

func opsOf(tape []Step[Effect]) []Effect {
	ops := make([]Effect, 0, len(tape))
	for _, step := range tape {
		ops = append(ops, step.Op)
	}
	return ops
}
