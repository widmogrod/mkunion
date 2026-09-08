package effect

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The core is generic over Op. These tests use the smallest Op that works:
// a struct with a name. example/welcome runs the same core over a real union.

type op struct{ Name string }

// echo answers every operation with its name.
func echo(_ context.Context, o op) (any, error) { return o.Name, nil }

// ask is a one-step program that expects a string answer.
func ask(name string) Eff[op, string] { return PerformAs[op, string](op{name}) }

func TestRun_walksEveryNode(t *testing.T) {
	ctx := context.Background()

	got, err := Run(ctx, echo, Return[op]("done"))
	require.NoError(t, err)
	assert.Equal(t, "done", got)

	boom := errors.New("boom")
	_, err = Run(ctx, echo, Throw[op, string](boom))
	assert.ErrorIs(t, err, boom)

	got, err = Run(ctx, echo, ask("a"))
	require.NoError(t, err)
	assert.Equal(t, "a", got)

	suspended := &Suspend[op, string]{Resume: func() Eff[op, string] { return ask("late") }}
	got, err = Run(ctx, echo, suspended)
	require.NoError(t, err)
	assert.Equal(t, "late", got)
}

func TestRun_handlerErrorAndCancelledContextStopTheProgram(t *testing.T) {
	failing := func(context.Context, op) (any, error) { return nil, errors.New("down") }
	_, err := Run(context.Background(), failing, ask("a"))
	assert.EqualError(t, err, "down")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	spy := func(context.Context, op) (any, error) { called = true; return nil, nil }
	_, err = Run(ctx, spy, ask("a"))
	assert.ErrorIs(t, err, context.Canceled)
	assert.False(t, called, "a cancelled context is not performed")
}

func TestPerformAs_wrongAnswerTypeIsAnError(t *testing.T) {
	lying := func(context.Context, op) (any, error) { return 42, nil }
	_, err := Run(context.Background(), lying, ask("a"))
	assert.EqualError(t, err, "effect: handler answered int to effect.op, want string")
}

// bogus satisfies Eff without being one of its variants.
type bogus struct{}

func (bogus) AcceptEff(EffVisitor[op, string]) any { return nil }

func TestRun_refusesAnUnknownNode(t *testing.T) {
	_, err := Run[op, string](context.Background(), echo, bogus{})
	assert.ErrorContains(t, err, "unknown program node")
}

func TestThenAndMap_composeOverEveryVariant(t *testing.T) {
	ctx := context.Background()
	twice := func(s string) Eff[op, string] { return Map(ask(s), func(x string) string { return x + x }) }

	got, err := Run(ctx, echo, Then(ask("a"), twice))
	require.NoError(t, err)
	assert.Equal(t, "aa", got, "Bind then Bind")

	got, err = Run(ctx, echo, Then(Return[op]("b"), twice))
	require.NoError(t, err)
	assert.Equal(t, "bb", got, "Pure then Bind")

	boom := errors.New("boom")
	_, err = Run(ctx, echo, Then(Throw[op, string](boom), twice))
	assert.ErrorIs(t, err, boom, "Fail short-circuits")

	suspended := &Suspend[op, string]{Resume: func() Eff[op, string] { return ask("c") }}
	got, err = Run(ctx, echo, Then(suspended, twice))
	require.NoError(t, err)
	assert.Equal(t, "cc", got, "Suspend then Bind")
}

func TestProc_directStyleBody(t *testing.T) {
	ctx := context.Background()

	prog := Proc(func(e *Env[op]) (string, error) {
		a := DoAs[op, string](e, op{"a"})
		b := DoAs[op, string](e, op{"b"})
		return a + b, nil
	})
	var trace []op
	got, err := Run(ctx, Wrap(echo, Trace(&trace)), prog)
	require.NoError(t, err)
	assert.Equal(t, "ab", got)
	assert.Equal(t, []op{{"a"}, {"b"}}, trace)

	t.Run("a handler error unwinds the body", func(t *testing.T) {
		reached := false
		prog := Proc(func(e *Env[op]) (string, error) {
			DoAs[op, string](e, op{"a"})
			reached = true
			return "", nil
		})
		down := errors.New("down")
		_, err := Run(ctx, func(context.Context, op) (any, error) { return nil, down }, prog)
		assert.ErrorIs(t, err, down)
		assert.False(t, reached)
	})

	t.Run("AttemptAs hands the error to the body", func(t *testing.T) {
		down := errors.New("down")
		prog := Proc(func(e *Env[op]) (string, error) {
			_, err := AttemptAs[op, string](e, op{"a"})
			return "recovered: " + err.Error(), nil
		})
		got, err := Run(ctx, func(context.Context, op) (any, error) { return nil, down }, prog)
		require.NoError(t, err)
		assert.Equal(t, "recovered: down", got)
	})

	t.Run("AttemptAs checks the answer type", func(t *testing.T) {
		prog := Proc(func(e *Env[op]) (string, error) {
			_, err := AttemptAs[op, int](e, op{"a"})
			return "", err
		})
		_, err := Run(ctx, echo, prog)
		assert.EqualError(t, err, "effect: handler answered string to effect.op, want int")
	})

	t.Run("a body error fails the program", func(t *testing.T) {
		bad := errors.New("bad body")
		prog := Proc(func(*Env[op]) (string, error) { return "", bad })
		_, err := Run(ctx, echo, prog)
		assert.ErrorIs(t, err, bad)
	})

	t.Run("a body panic is not swallowed", func(t *testing.T) {
		prog := Proc(func(*Env[op]) (string, error) { panic("boom") })
		assert.PanicsWithValue(t, "boom", func() { _, _ = Run(ctx, echo, prog) })
	})
}

func TestWrap_firstListedIsOutermost(t *testing.T) {
	var order []string
	tag := func(name string) Middleware[op] {
		return func(h Handler[op]) Handler[op] {
			return func(ctx context.Context, o op) (any, error) {
				order = append(order, name+" in")
				a, err := h(ctx, o)
				order = append(order, name+" out")
				return a, err
			}
		}
	}
	_, err := Run(context.Background(), Wrap(echo, tag("A"), tag("B")), ask("x"))
	require.NoError(t, err)
	assert.Equal(t, []string{"A in", "B in", "B out", "A out"}, order)
}

func TestRetryWith(t *testing.T) {
	blip := errors.New("blip")
	failFirst := func(n int) Middleware[op] {
		return func(h Handler[op]) Handler[op] {
			calls := 0
			return func(ctx context.Context, o op) (any, error) {
				calls++
				if calls <= n {
					return nil, blip
				}
				return h(ctx, o)
			}
		}
	}
	always := func(attempts int) func(op) RetryPolicy {
		return func(op) RetryPolicy {
			return RetryPolicy{Attempts: attempts, Backoff: ExponentialBackoff(time.Millisecond)}
		}
	}

	t.Run("retries up to the policy, with backoff", func(t *testing.T) {
		var waits []time.Duration
		sleep := func(d time.Duration) { waits = append(waits, d) }
		got, err := Run(context.Background(), Wrap(echo, RetryWith(always(3), nil, sleep), failFirst(2)), ask("a"))
		require.NoError(t, err)
		assert.Equal(t, "a", got)
		assert.Equal(t, []time.Duration{time.Millisecond, 2 * time.Millisecond}, waits)
	})

	t.Run("gives up after the last attempt, wrapped", func(t *testing.T) {
		_, err := Run(context.Background(), Wrap(echo, RetryWith(always(2), nil, nil), failFirst(5)), ask("a"))
		assert.ErrorIs(t, err, blip)
		assert.EqualError(t, err, "effect: effect.op failed after 2 attempts: blip")
	})

	t.Run("attempts 1 or less means no retry and no wrapping", func(t *testing.T) {
		_, err := Run(context.Background(), Wrap(echo, RetryWith(always(0), nil, nil), failFirst(1)), ask("a"))
		assert.EqualError(t, err, "blip")
	})

	t.Run("a budget caps retries across the run", func(t *testing.T) {
		budget := &RetryBudget{Left: 1}
		_, err := Run(context.Background(), Wrap(echo, RetryWith(always(5), budget, nil), failFirst(3)), ask("a"))
		assert.ErrorIs(t, err, ErrRetryBudget)
		assert.Equal(t, 0, budget.Left)
	})

	t.Run("a cancelled context is not retried", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		h := Wrap(func(context.Context, op) (any, error) { cancel(); return nil, blip }, RetryWith(always(5), nil, nil))
		_, err := h(ctx, op{"a"})
		assert.EqualError(t, err, "effect: effect.op failed after 5 attempts: blip")
	})

	t.Run("Retry is the same for every operation", func(t *testing.T) {
		got, err := Run(context.Background(), Wrap(echo, Retry[op](2), failFirst(1)), ask("a"))
		require.NoError(t, err)
		assert.Equal(t, "a", got)
	})
}

func TestStepKeys_numberEveryStep(t *testing.T) {
	var keys []string
	spy := func(ctx context.Context, o op) (any, error) { keys = append(keys, StepKey(ctx)); return o.Name, nil }
	prog := Then(ask("a"), func(string) Eff[op, string] { return ask("b") })
	_, err := Run(context.Background(), Wrap(spy, StepKeys[op]("run")), prog)
	require.NoError(t, err)
	assert.Equal(t, []string{"run/1", "run/2"}, keys)
	assert.Equal(t, "", StepKey(context.Background()), "no key without StepKeys")
}

func TestFaults(t *testing.T) {
	ctx := context.Background()
	fault := errors.New("fault")
	three := Then(ask("a"), func(string) Eff[op, string] {
		return Then(ask("b"), func(string) Eff[op, string] { return ask("c") })
	})

	t.Run("FailEvery refuses every nth call before performing", func(t *testing.T) {
		var trace []op
		_, err := Run(ctx, Wrap(echo, FailEvery[op](2, fault), Trace(&trace)), three)
		assert.ErrorIs(t, err, fault)
		assert.Equal(t, []op{{"a"}}, trace, "the second call never reached the handler")
	})

	t.Run("LoseAnswerAt performs the nth call and reports an error anyway", func(t *testing.T) {
		var trace []op
		_, err := Run(ctx, Wrap(echo, LoseAnswerAt[op](2, fault), Trace(&trace)), three)
		assert.ErrorIs(t, err, fault)
		assert.Equal(t, []op{{"a"}, {"b"}}, trace, "the second call did reach the handler")
	})

	t.Run("CrashAfter performs n calls and fails every later one", func(t *testing.T) {
		var trace []op
		_, err := Run(ctx, Wrap(echo, CrashAfter[op](2, fault), Trace(&trace)), three)
		assert.ErrorIs(t, err, fault)
		assert.Equal(t, []op{{"a"}, {"b"}}, trace)
	})

	t.Run("Chaos is reproducible from its seed", func(t *testing.T) {
		outcome := func(seed uint64) string {
			cfg := ChaosConfig{Seed: seed, FailRate: 0.3, LoseAnswerRate: 0.3}
			got, err := Run(ctx, Wrap(echo, Chaos[op](cfg)), three)
			if err != nil {
				return err.Error()
			}
			return got
		}
		seen := map[string]bool{}
		for seed := uint64(0); seed < 50; seed++ {
			first := outcome(seed)
			assert.Equal(t, first, outcome(seed), "seed %d", seed)
			seen[first] = true
		}
		assert.True(t, seen["c"], "some seed lets the program through")
		assert.Contains(t, seen, "chaos: refused effect.op")
		assert.Contains(t, seen, "chaos: answer to effect.op lost")
	})
}

func TestRecordAndReplay(t *testing.T) {
	ctx := context.Background()
	two := Then(ask("a"), func(string) Eff[op, string] { return ask("b") })

	var tape []Step[op]
	got, err := Run(ctx, Wrap(echo, Record(&tape)), two)
	require.NoError(t, err)
	assert.Equal(t, "b", got)
	assert.Equal(t, []Step[op]{{Op: op{"a"}, Answer: "a"}, {Op: op{"b"}, Answer: "b"}}, tape)

	got, err = Run(ctx, Replay(tape, nil), two)
	require.NoError(t, err)
	assert.Equal(t, "b", got, "the tape answers, no handler needed")

	_, err = Run(ctx, Replay(tape, nil), ask("x"))
	assert.ErrorContains(t, err, "replay mismatch at step 1")

	_, err = Run(ctx, Replay(tape[:1], nil), two)
	assert.ErrorIs(t, err, ErrTapeEnded)

	got, err = Run(ctx, Replay(tape[:1], echo), two)
	require.NoError(t, err)
	assert.Equal(t, "b", got, "after the tape, rest takes over")

	down := errors.New("down")
	var failed []Step[op]
	_, err = Run(ctx, Wrap(func(context.Context, op) (any, error) { return nil, down }, Record(&failed)), two)
	assert.ErrorIs(t, err, down)
	assert.Equal(t, []Step[op]{{Op: op{"a"}, Err: down}}, failed, "a failure is on the tape too")
}

func TestGuard_refusesBeforePerforming(t *testing.T) {
	var trace []op
	deny := func(o op) error {
		if o.Name == "b" {
			return errors.New("no b")
		}
		return nil
	}
	two := Then(ask("a"), func(string) Eff[op, string] { return ask("b") })
	_, err := Run(context.Background(), Wrap(echo, Guard(deny), Trace(&trace)), two)
	assert.ErrorIs(t, err, ErrDenied)
	assert.EqualError(t, err, "effect: denied by policy: no b")
	assert.Equal(t, []op{{"a"}}, trace)
}

func TestSpans_oneSpanPerOperation(t *testing.T) {
	clock := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	tick := func() time.Time { clock = clock.Add(time.Second); return clock }
	at := func(s int) time.Time { return time.Date(2026, 9, 1, 12, 0, s, 0, time.UTC) }
	var spans []Span

	_, err := Run(context.Background(), Wrap(echo, Spans[op](tick, &spans)), ask("a"))
	require.NoError(t, err)
	assert.Equal(t, []Span{{Name: "effect.op", Attrs: "{Name:a}", Start: at(1), End: at(2)}}, spans)

	spans, clock = nil, at(0)
	_, err = Run(context.Background(), Wrap(func(context.Context, op) (any, error) { return nil, errors.New("down") }, Spans[op](tick, &spans)), ask("a"))
	assert.Error(t, err)
	assert.Equal(t, "down", spans[0].Err)
}

// jsonOp renders as JSON, the way a mkunion union does.
type jsonOp struct{ Name string }

func (j jsonOp) MarshalJSON() ([]byte, error) { return []byte(`{"name":"` + j.Name + `"}`), nil }

func TestDiffTraces(t *testing.T) {
	want := []op{{"read"}, {"now"}, {"send"}}
	got := []op{{"config"}, {"read"}, {"send"}, {"log"}}
	assert.Equal(t, []string{
		"+ effect.op{Name:config}",
		"  effect.op{Name:read}",
		"- effect.op{Name:now}",
		"  effect.op{Name:send}",
		"+ effect.op{Name:log}",
	}, DiffTraces(want, got))

	assert.Empty(t, DiffTraces[op](nil, nil))
	assert.Equal(t, []string{`  effect.jsonOp{"name":"a"}`}, DiffTraces([]jsonOp{{"a"}}, []jsonOp{{"a"}}),
		"an operation with JSON is described through it")
}
