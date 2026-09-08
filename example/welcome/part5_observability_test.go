package welcome

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
	var v1 []MyEff
	out1, err := Interpret(context.Background(), Notify("name.txt", to), live, Trace(&v1))
	require.NoError(t, err)

	live, _, _ = newWorld()
	live.FS = withConfig
	var v2 []MyEff
	out2, err := Interpret(context.Background(), notifyV2("name.txt", to), live, Trace(&v2))
	require.NoError(t, err)

	assert.Equal(t, out1, out2, "same output: an output-only test passes")
	assert.Equal(t, []string{
		`+ *welcome.ReadFile{"Path":"config.txt"}`,
		`+ *welcome.Now{}`,
		`  *welcome.ReadFile{"Path":"name.txt"}`,
		`- *welcome.Now{}`,
		`  *welcome.Send{"To":"ada@example.com","Msg":"Hello Ada, it is 12:00PM"}`,
		`  *welcome.Log{"Msg":"sent receipt-1"}`,
	}, DiffTraces(v1, v2), "the behaviour diff shows the new read and the moved clock")
}

// dryRun is an exhaustive policy: reads are fine, nothing leaves the process.
func dryRun(op MyEff) error {
	return MatchMyEffR1(op,
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
		func(x *Charge) error { return fmt.Errorf("dry run: would charge %d", x.Amount) },
	)
}

func TestPart5_policyGuardIsAuthorizationOnEffects(t *testing.T) {
	program := Notify("name.txt", to)

	live, out, mail := newWorld()
	var trace []MyEff
	_, err := Interpret(context.Background(), program, live, Guard(dryRun), Trace(&trace))

	require.ErrorIs(t, err, ErrDenied)
	assert.EqualError(t, err, `effect: denied by policy: dry run: would send "Hello Ada, it is 12:00PM" to ada@example.com`)
	assert.Equal(t, []MyEff{&ReadFile{Path: "name.txt"}, &Now{}}, trace, "the reads happened; the Send never reached the handler")
	assert.Empty(t, mail.Sent)
	assert.Empty(t, out.String())

	readSecret := Prog(func(fx Fx) (string, error) {
		return string(fx.ReadFile("prod.env")), nil
	})
	_, err = Interpret(context.Background(), readSecret, live, Guard(dryRun))
	assert.EqualError(t, err, `effect: denied by policy: dry run: refuse to read secrets from "prod.env"`)
}

func TestPart5_spansForEveryOperationFromOnePlace(t *testing.T) {
	live, _, _ := newWorld()
	clock := noon
	tick := func() time.Time { clock = clock.Add(10 * time.Millisecond); return clock }
	at := func(ms int) time.Time { return noon.Add(time.Duration(ms) * time.Millisecond) }
	var spans []Span
	program := Notify("name.txt", to)

	_, err := Interpret(context.Background(), program, live, Spans[MyEff](tick, &spans))

	require.NoError(t, err)
	assert.Equal(t, []Span{
		{Name: "*welcome.ReadFile", Attrs: `{"Path":"name.txt"}`, Start: at(10), End: at(20)},
		{Name: "*welcome.Now", Attrs: `{}`, Start: at(30), End: at(40)},
		{Name: "*welcome.Send", Attrs: `{"To":"ada@example.com","Msg":"Hello Ada, it is 12:00PM"}`, Start: at(50), End: at(60)},
		{Name: "*welcome.Log", Attrs: `{"Msg":"sent receipt-1"}`, Start: at(70), End: at(80)},
	}, spans)

	// A failure lands on the span too.
	spans, clock = nil, noon
	blip := errors.New("disk hiccup")
	_, err = Interpret(context.Background(), program, live, Spans[MyEff](tick, &spans), FailEvery[MyEff](1, blip))
	require.ErrorIs(t, err, blip)
	assert.Equal(t, []Span{
		{Name: "*welcome.ReadFile", Attrs: `{"Path":"name.txt"}`, Start: at(10), End: at(20), Err: "disk hiccup"},
	}, spans)
}
