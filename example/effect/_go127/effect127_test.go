//go:build go1.27

package effect127

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/example/effect"
)

var noon = time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

func TestGreetDirect(t *testing.T) {
	fake := &effect.Fake{Clock: noon, Files: map[string]string{"name.txt": "Ada"}}
	var trace []effect.Effect
	d := Direct{Ctx: context.Background(), Handler: effect.Trace(effect.HandlerOf(fake), &trace)}

	got, err := GreetDirect(d, "name.txt")

	require.NoError(t, err)
	assert.Equal(t, "Hello Ada, it is 12:00PM", got)
	assert.Equal(t, []effect.Effect{&effect.ReadFile{Path: "name.txt"}, &effect.Now{}, &effect.Log{Msg: got}}, trace)

	_, err = GreetDirect(d, "missing.txt")
	require.ErrorContains(t, err, `no file "missing.txt"`)
}

func TestGreetChained_matchesGreet(t *testing.T) {
	files := map[string]string{"name.txt": "Ada\n"}

	var traceA, traceB []effect.Effect
	a, errA := effect.Run(context.Background(), effect.Trace(effect.HandlerOf(&effect.Fake{Clock: noon, Files: files}), &traceA), effect.Greet("name.txt"))
	b, errB := effect.Run(context.Background(), effect.Trace(effect.HandlerOf(&effect.Fake{Clock: noon, Files: files}), &traceB), GreetChained("name.txt"))

	require.NoError(t, errA)
	require.NoError(t, errB)
	assert.Equal(t, a, b)
	assert.Equal(t, traceA, traceB)
}

func TestProgram_methodTypeParameterIsInferred(t *testing.T) {
	p := Start(effect.Return[effect.Effect](20)).
		Map(func(n int) int { return n + 1 }).
		Map(func(n int) string { return "n=" + strconv.Itoa(n) })

	got, err := effect.Run(context.Background(), effect.HandlerOf(&effect.Fake{}), p.Eff)

	require.NoError(t, err)
	assert.Equal(t, "n=21", got)
}
