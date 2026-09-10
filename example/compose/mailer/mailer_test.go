package mailer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/effect"
)

// --8<-- [start:alone]

// fake answers both operations of this package. Two methods, nothing generated.
type fake struct{}

func (fake) HandleResolve(_ context.Context, op *Resolve) (string, error) {
	return op.Name + "@example.com", nil
}
func (fake) HandleSend(context.Context, *Send) (string, error) { return "receipt-1", nil }

var _ EffectHandler = fake{} // the compiler checks that every operation is answered

// The package is tested on its own, with no application around it.
func TestNotify_resolvesThenSends(t *testing.T) {
	program := Notify("ada", "hi")

	var trace []effect.Op
	got, err := effect.Interpret(context.Background(), program, fake{}, effect.Trace(&trace))

	require.NoError(t, err)
	assert.Equal(t, "receipt-1", got)
	assert.Equal(t, []effect.Op{
		&Resolve{Name: "ada"},
		&Send{To: "ada@example.com", Msg: "hi"},
	}, trace)
}

// --8<-- [end:alone]
