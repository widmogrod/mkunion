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

// Part 5: seeing what happened.
//
// A trace of typed operations can do what log lines and spans cannot: be
// diffed against another run, be checked against a policy before anything
// runs, and be turned into spans from one place.

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

func TestPart5_traceDiffFindsABehaviourRegression(t *testing.T) {
	withConfig := fstest.MapFS{"name.txt": {Data: []byte("Ada\n")}, "config.txt": {Data: []byte("{}")}}

	live, _, _ := newWorld()
	live.FS = withConfig
	var v1 []Effect
	out1, err := Run(context.Background(), Trace(HandlerOf(live), &v1), Notify("name.txt", to))
	require.NoError(t, err)

	live, _, _ = newWorld()
	live.FS = withConfig
	var v2 []Effect
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

func TestPart5_policyGuardIsAuthorizationOnEffects(t *testing.T) {
	live, out, mail := newWorld()
	var trace []Effect

	_, err := Run(context.Background(), Guard(Trace(HandlerOf(live), &trace), dryRun), Notify("name.txt", to))

	require.ErrorIs(t, err, ErrDenied)
	assert.EqualError(t, err, `effect: denied by policy: dry run: would send "Hello Ada, it is 12:00PM" to ada@example.com`)
	assert.Equal(t, []Effect{&ReadFile{Path: "name.txt"}, &Now{}}, trace, "the reads happened; the Send never reached the handler")
	assert.Empty(t, mail.Sent)
	assert.Empty(t, out.String())

	_, err = Run(context.Background(), Guard(HandlerOf(live), dryRun), Prog(func(fx Fx) (string, error) {
		return string(fx.ReadFile("prod.env")), nil
	}))
	assert.EqualError(t, err, `effect: denied by policy: dry run: refuse to read secrets from "prod.env"`)
}

func TestPart5_spansForEveryOperationFromOnePlace(t *testing.T) {
	live, _, _ := newWorld()
	clock := noon
	tick := func() time.Time { clock = clock.Add(10 * time.Millisecond); return clock }
	at := func(ms int) time.Time { return noon.Add(time.Duration(ms) * time.Millisecond) }
	var spans []Span

	_, err := Run(context.Background(), Spans(HandlerOf(live), tick, &spans), Notify("name.txt", to))

	require.NoError(t, err)
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
