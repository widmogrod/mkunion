package mailer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/effect"
)

// --8<-- [start:alone]

// The package is tested on its own, with no application around it.
func TestNotify_resolvesThenSends(t *testing.T) {
	program := Notify("ada", "hi")

	var trace []Effect
	fake := EffectFuncs{
		Resolve: func(_ context.Context, op *Resolve) (string, error) { return op.Name + "@example.com", nil },
		Send:    func(context.Context, *Send) (string, error) { return "receipt-1", nil },
	}
	got, err := effect.Run(context.Background(), effect.Wrap(EffectHandlerFunc(fake), effect.Trace(&trace)), program)

	require.NoError(t, err)
	assert.Equal(t, "receipt-1", got)
	assert.Equal(t, []Effect{
		&Resolve{Name: "ada"},
		&Send{To: "ada@example.com", Msg: "hi"},
	}, trace)
}

// --8<-- [end:alone]
