package effect

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCatch_handlesFailureLikeThenHandlesSuccess(t *testing.T) {
	ctx := context.Background()
	down := errors.New("down")
	failing := func(context.Context, op) (any, error) { return nil, down }
	recover := func(err error) Eff[op, string] { return Return[op]("recovered from " + err.Error()) }

	got, err := Run(ctx, failing, Catch(ask("a"), recover))
	require.NoError(t, err)
	assert.Equal(t, "recovered from down", got, "a failed Bind is caught")

	got, err = Run(ctx, echo, Catch(ask("a"), recover))
	require.NoError(t, err)
	assert.Equal(t, "a", got, "a success passes through")

	got, err = Run(ctx, echo, Catch(Throw[op, string](down), recover))
	require.NoError(t, err)
	assert.Equal(t, "recovered from down", got, "a Fail is caught")

	got, err = Run(ctx, echo, Catch(Return[op]("pure"), recover))
	require.NoError(t, err)
	assert.Equal(t, "pure", got)

	suspended := &Suspend[op, string]{Resume: func() Eff[op, string] { return Throw[op, string](down) }}
	got, err = Run(ctx, echo, Catch[op, string](suspended, recover))
	require.NoError(t, err)
	assert.Equal(t, "recovered from down", got, "a Suspend is caught after it resumes")

	// The handler may fail too, and that failure is the program's.
	other := errors.New("other")
	_, err = Run(ctx, failing, Catch(ask("a"), func(error) Eff[op, string] { return Throw[op, string](other) }))
	assert.ErrorIs(t, err, other)
}

func TestOrElse_fallsBackAndKeepsTheTrace(t *testing.T) {
	flaky := func(_ context.Context, o op) (any, error) {
		if o.Name == "primary" {
			return nil, errors.New("primary down")
		}
		return o.Name, nil
	}
	var trace []op
	got, err := Run(context.Background(), Wrap(flaky, Trace(&trace)), OrElse(ask("primary"), ask("backup")))
	require.NoError(t, err)
	assert.Equal(t, "backup", got)
	assert.Equal(t, []op{{"primary"}, {"backup"}}, trace, "both attempts are on the trace")
}
