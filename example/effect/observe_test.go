package effect

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// 4. A typed trace can do what spans cannot: diff, enforce, and feed spans.
// ---------------------------------------------------------------------------

// notifyV2 is "version two" of Notify. Someone added a config lookup and moved
// the clock read. The output is identical, so an output test would not notice.
func notifyV2(path, to string) Program[string] {
	return Prog(func(fx Fx) (string, error) {
		_ = fx.ReadFile("config.txt")
		now := fx.Now()
		name := strings.TrimSpace(string(fx.ReadFile(path)))
		receipt := fx.Send(to, "Hello "+name+", it is "+now.Format(time.Kitchen))
		fx.Log("sent " + receipt)
		return receipt, nil
	})
}

func TestAdvanced4a_traceDiffFindsBehaviourRegression(t *testing.T) {
	live, _, _ := newWorld()
	live.FS = fstest.MapFS{"name.txt": {Data: []byte("Ada\n")}, "config.txt": {Data: []byte("{}")}}

	var v1, v2 []Effect
	out1, err := Run(context.Background(), Trace(HandlerOf(live), &v1), Notify("name.txt", to))
	require.NoError(t, err)
	live, _, _ = newWorld()
	live.FS = fstest.MapFS{"name.txt": {Data: []byte("Ada\n")}, "config.txt": {Data: []byte("{}")}}
	out2, err := Run(context.Background(), Trace(HandlerOf(live), &v2), notifyV2("name.txt", to))
	require.NoError(t, err)

	assert.Equal(t, out1, out2, "same output: an output-only test passes")
	assert.Equal(t, []string{
		`+ &effect.ReadFile{Path:"config.txt"}`,
		`+ &effect.Now{}`,
		`  &effect.ReadFile{Path:"name.txt"}`,
		`- &effect.Now{}`,
		`  &effect.Send{To:"ada@example.com", Msg:"Hello Ada, it is 12:00PM"}`,
		`  &effect.Log{Msg:"sent receipt-1"}`,
	}, DiffTraces(v1, v2), "the behaviour diff shows the new read and the moved clock")
}

// dryRun is an exhaustive policy: reads are fine, nothing leaves the process.
func dryRun(op Effect) error {
	return MatchEffectR1(op,
		func(*Log) error { return nil },
		func(*Now) error { return nil },
		func(x *ReadFile) error {
			if strings.HasSuffix(x.Path, ".env") {
				return fmt.Errorf("dry run: refuse to read secrets from %q", x.Path)
			}
			return nil
		},
		func(*Random) error { return nil },
		func(x *Send) error { return fmt.Errorf("dry run: would send %q to %s", x.Msg, x.To) },
	)
}

func TestAdvanced4b_policyGuardIsAuthorizationOnEffects(t *testing.T) {
	live, out, mail := newWorld()
	var trace []Effect

	_, err := Run(context.Background(), Guard(Trace(HandlerOf(live), &trace), dryRun), Notify("name.txt", to))

	require.ErrorIs(t, err, ErrDenied)
	assert.Equal(t, `effect: denied by policy: dry run: would send "Hello Ada, it is 12:00PM" to ada@example.com`, err.Error())
	assert.Equal(t, []Effect{&ReadFile{Path: "name.txt"}, &Now{}}, trace, "the reads happened; the Send never reached the handler")
	assert.Empty(t, mail.Sent)
	assert.Empty(t, out.String())

	_, err = Run(context.Background(), Guard(HandlerOf(live), dryRun), Prog(func(fx Fx) (string, error) {
		return string(fx.ReadFile("prod.env")), nil
	}))
	assert.EqualError(t, err, `effect: denied by policy: dry run: refuse to read secrets from "prod.env"`)
}

func TestAdvanced4c_spansForEveryOperationFromOnePlace(t *testing.T) {
	live, _, _ := newWorld()
	clock := noon
	tick := func() time.Time { clock = clock.Add(10 * time.Millisecond); return clock }
	var spans []Span

	_, err := Run(context.Background(), Spans(HandlerOf(live), tick, &spans), Notify("name.txt", to))

	require.NoError(t, err)
	at := func(ms int) time.Time { return noon.Add(time.Duration(ms) * time.Millisecond) }
	assert.Equal(t, []Span{
		{Name: "*effect.ReadFile", Attrs: "&{Path:name.txt}", Start: at(10), End: at(20)},
		{Name: "*effect.Now", Attrs: "&{}", Start: at(30), End: at(40)},
		{Name: "*effect.Send", Attrs: "&{To:ada@example.com Msg:Hello Ada, it is 12:00PM}", Start: at(50), End: at(60)},
		{Name: "*effect.Log", Attrs: "&{Msg:sent receipt-1}", Start: at(70), End: at(80)},
	}, spans)

	// A failure lands on the span too.
	spans = nil
	blip := errors.New("disk hiccup")
	_, err = Run(context.Background(), Spans(FailEvery(HandlerOf(live), 1, blip), tick, &spans), Notify("name.txt", to))
	require.ErrorIs(t, err, blip)
	assert.Equal(t, []Span{
		{Name: "*effect.ReadFile", Attrs: "&{Path:name.txt}", Start: at(90), End: at(100), Err: "disk hiccup"},
	}, spans)
}

// ---------------------------------------------------------------------------
// 5. Seeded chaos: many worlds, a few invariants, and every failure replays.
// ---------------------------------------------------------------------------

func TestAdvanced5_seededChaosFindsTheDuplicateAndKeysFixIt(t *testing.T) {
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
