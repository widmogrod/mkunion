package effect

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGreetFx_matchesGreet(t *testing.T) {
	files := map[string]string{"name.txt": "Ada\n"}

	var traceA, traceB []Effect
	a, errA := Run(context.Background(), Trace(HandlerOf(&Fake{Clock: noon, Files: files}), &traceA), Greet("name.txt"))
	b, errB := Run(context.Background(), Trace(HandlerOf(&Fake{Clock: noon, Files: files}), &traceB), GreetFx("name.txt"))

	require.NoError(t, errA)
	require.NoError(t, errB)
	assert.Equal(t, "Hello Ada, it is 12:00PM", b)
	assert.Equal(t, a, b)
	assert.Equal(t, traceA, traceB)
}

func TestGreetFx_handlerErrorUnwindsBody(t *testing.T) {
	fake := &Fake{Clock: noon}
	var trace []Effect

	_, err := Run(context.Background(), Trace(HandlerOf(fake), &trace), GreetFx("missing.txt"))

	require.ErrorContains(t, err, `no file "missing.txt"`)
	assert.Len(t, trace, 1)
	assert.Empty(t, fake.Logs)
}

func TestGreetOrGuestFx_attemptHandlesErrorInPlace(t *testing.T) {
	fake := &Fake{}

	got, err := Run(context.Background(), HandlerOf(fake), GreetOrGuestFx("missing.txt"))

	require.NoError(t, err)
	assert.Equal(t, "Hello guest", got)
	assert.Equal(t, []string{"Hello guest"}, fake.Logs)
}

func TestRollUntilFx_isStackSafe(t *testing.T) {
	const rolls = 1_000_000
	_, err := Run(context.Background(), HandlerOf(&Fake{Rolls: []int{0}}), RollUntilFx(6, rolls))
	require.ErrorContains(t, err, "no 6 in 1000000 rolls")

	got, err := Run(context.Background(), HandlerOf(&Fake{Rolls: []int{0, 0, 5}}), RollUntilFx(6, rolls))
	require.NoError(t, err)
	assert.Equal(t, 3, got)
}

// clockOnly overrides one method; Defaults supplies the other three.
type clockOnly struct {
	Defaults
	at time.Time
}

func (c clockOnly) HandleNow(context.Context, *Now) (time.Time, error) { return c.at, nil }

func TestDefaults_embedAndOverrideOneMethod(t *testing.T) {
	prog := Prog(func(fx Fx) (string, error) {
		fx.Log("ignored by Defaults")
		return fx.Now().Format(time.Kitchen) + " and rolled " + itoa(fx.Random(6)), nil
	})

	got, err := Run(context.Background(), HandlerOf(clockOnly{at: noon}), prog)

	require.NoError(t, err)
	assert.Equal(t, "12:00PM and rolled 0", got)

	_, err = Run(context.Background(), HandlerOf(clockOnly{}), GreetFx("name.txt"))
	require.ErrorContains(t, err, `defaults: no file "name.txt"`)
}

func TestFx_genericDoInfersAnswerType(t *testing.T) {
	prog := Prog(func(fx Fx) (time.Time, error) {
		return fx.Do(&Now{}), nil // no type annotation needed
	})

	got, err := Run(context.Background(), HandlerOf(&Fake{Clock: noon}), prog)

	require.NoError(t, err)
	assert.Equal(t, noon, got)
}

func TestProgram_aliasIsTheSameTypeAsEff(t *testing.T) {
	var p Program[string] = Greet("name.txt")
	var e Eff[Effect, string] = p
	assert.NotNil(t, e)
}

func itoa(n int) string { return string(rune('0' + n)) }
