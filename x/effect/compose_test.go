package effect

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// wide is an application union: it wraps op and, say, another package's op2.
type wide struct {
	Small *op
	Other string
}

func inject(o op) wide { return wide{Small: &o} }

func wideEcho(_ context.Context, w wide) (any, error) {
	if w.Small != nil {
		return w.Small.Name, nil
	}
	return w.Other, nil
}

func TestLift_programOverSubRunsOverSuper(t *testing.T) {
	ctx := context.Background()
	two := Then(ask("a"), func(a string) Eff[op, string] { return Map(ask("b"), func(b string) string { return a + b }) })

	var trace []wide
	got, err := Run(ctx, Wrap(wideEcho, Trace(&trace)), Lift(two, inject))
	require.NoError(t, err)
	assert.Equal(t, "ab", got)
	assert.Equal(t, []wide{inject(op{"a"}), inject(op{"b"})}, trace, "every step is spelled in the wider union")

	_, err = Run(ctx, wideEcho, Lift(Throw[op, string](errors.New("boom")), inject))
	assert.EqualError(t, err, "boom")

	suspended := &Suspend[op, string]{Resume: func() Eff[op, string] { return ask("late") }}
	got, err = Run(ctx, wideEcho, Lift[op, wide, string](suspended, inject))
	require.NoError(t, err)
	assert.Equal(t, "late", got)

	direct := Proc(func(e *Env[op]) (string, error) { return DoAs[op, string](e, op{"d"}), nil })
	got, err = Run(ctx, wideEcho, Lift(direct, inject))
	require.NoError(t, err)
	assert.Equal(t, "d", got, "a direct-style program lifts too")
}

func TestEmbed_subProgramInsideADirectStyleBody(t *testing.T) {
	ctx := context.Background()
	two := Then(ask("a"), func(a string) Eff[op, string] { return Map(ask("b"), func(b string) string { return a + b }) })

	body := Proc(func(e *Env[wide]) (string, error) {
		other := DoAs[wide, string](e, wide{Other: "x"})
		inner, err := Embed(e, two, inject)
		if err != nil {
			return "", err
		}
		return other + inner, nil
	})

	var trace []wide
	got, err := Run(ctx, Wrap(wideEcho, Trace(&trace)), body)
	require.NoError(t, err)
	assert.Equal(t, "xab", got)
	assert.Equal(t, []wide{{Other: "x"}, inject(op{"a"}), inject(op{"b"})}, trace, "one trace, both unions, in order")

	t.Run("an error in the embedded program reaches the body", func(t *testing.T) {
		down := errors.New("down")
		body := Proc(func(e *Env[wide]) (string, error) {
			_, err := Embed(e, ask("a"), inject)
			return "saw: " + err.Error(), nil
		})
		got, err := Run(ctx, func(context.Context, wide) (any, error) { return nil, down }, body)
		require.NoError(t, err)
		assert.Equal(t, "saw: down", got)

		_, err = Embed[wide](nil, Throw[op, string](down), inject)
		assert.ErrorIs(t, err, down, "a failed program needs no env")
	})

	t.Run("a suspended and an unknown node", func(t *testing.T) {
		suspended := &Suspend[op, string]{Resume: func() Eff[op, string] { return Return[op]("s") }}
		got, err := Embed[wide](nil, suspended, inject)
		require.NoError(t, err)
		assert.Equal(t, "s", got)

		_, err = Embed[wide, op, string](nil, bogus{}, inject)
		assert.ErrorContains(t, err, "unknown program node")
	})
}
