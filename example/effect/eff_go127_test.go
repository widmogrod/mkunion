package effect

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGreetDirect(t *testing.T) {
	fake := &Fake{Clock: noon, Files: map[string]string{"name.txt": "Ada"}}
	var trace []Effect
	d := Direct{Ctx: context.Background(), Handler: Trace(HandlerOf(fake), &trace)}

	got, err := GreetDirect(d, "name.txt")

	require.NoError(t, err)
	assert.Equal(t, "Hello Ada, it is 12:00PM", got)
	assert.Equal(t, []Effect{&ReadFile{Path: "name.txt"}, &Now{}, &Log{Msg: got}}, trace)

	_, err = GreetDirect(d, "missing.txt")
	require.ErrorContains(t, err, `no file "missing.txt"`)
}

func TestGreetChained_matchesGreet(t *testing.T) {
	files := map[string]string{"name.txt": "Ada\n"}

	var traceA, traceB []Effect
	a, errA := Run(context.Background(), Trace(HandlerOf(&Fake{Clock: noon, Files: files}), &traceA), Greet("name.txt"))
	b, errB := Run(context.Background(), Trace(HandlerOf(&Fake{Clock: noon, Files: files}), &traceB), GreetChained("name.txt"))

	require.NoError(t, errA)
	require.NoError(t, errB)
	assert.Equal(t, a, b)
	assert.Equal(t, traceA, traceB)
}

func TestProgram_methodTypeParameterIsInferred(t *testing.T) {
	p := Start(Return[Effect](20)).
		Map(func(n int) int { return n + 1 }).
		Map(func(n int) string { return "n=" + strconv.Itoa(n) })

	got, err := Run(context.Background(), HandlerOf(&Fake{}), p.Eff)

	require.NoError(t, err)
	assert.Equal(t, "n=21", got)
}
