package effect

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Two hand-written "unions", the way the handler option would generate them:
// each operation performs itself against any handler that has its method.

type greet struct{ Name string }

type greeter interface {
	HandleGreet(ctx context.Context, op *greet) (string, error)
}

func (g *greet) Answer(_ context.Context, h any) (string, error) {
	typed, ok := h.(greeter)
	if !ok {
		return "", errors.New("a: handler does not implement greeter")
	}
	return typed.HandleGreet(context.Background(), g)
}
func (g *greet) Perform(ctx context.Context, h any) (any, error) { return g.Answer(ctx, h) }

type count struct{}

type counter interface {
	HandleCount(ctx context.Context, op *count) (int, error)
}

func (c *count) Answer(_ context.Context, h any) (int, error) {
	typed, ok := h.(counter)
	if !ok {
		return 0, errors.New("b: handler does not implement counter")
	}
	return typed.HandleCount(context.Background(), c)
}
func (c *count) Perform(ctx context.Context, h any) (any, error) { return c.Answer(ctx, h) }

// both is one handler value for both unions.
type both struct{ n int }

func (b *both) HandleGreet(_ context.Context, op *greet) (string, error) { return "hi " + op.Name, nil }
func (b *both) HandleCount(context.Context, *count) (int, error)         { b.n++; return b.n, nil }

func TestFx_operationsFromTwoUnionsInOneBody(t *testing.T) {
	ctx := context.Background()
	program := Prog(func(fx Fx) (string, error) {
		hello := fx.Do(&greet{Name: "ada"}) // string
		n := fx.Do(&count{})                // int
		n = fx.Do(&count{})
		return hello + " x" + string(rune('0'+n)), nil
	})

	var trace []Op
	got, err := Interpret(ctx, program, &both{}, Trace(&trace))
	require.NoError(t, err)
	assert.Equal(t, "hi ada x2", got)
	assert.Equal(t, []Op{&greet{Name: "ada"}, &count{}, &count{}}, trace, "one trace, both unions")
}

func TestFx_runsAProgramFromAnotherPackageInPlace(t *testing.T) {
	// A "library" program over Op: Perform then Map, no body.
	twice := Then(Perform(&count{}), func(int) Eff[Op, int] { return Perform(&count{}) })

	program := Prog(func(fx Fx) (int, error) {
		fx.Do(&greet{Name: "x"})
		return fx.Run(twice) + 10, nil
	})

	var trace []Op
	got, err := Interpret(context.Background(), program, &both{}, Trace(&trace))
	require.NoError(t, err)
	assert.Equal(t, 12, got)
	assert.Equal(t, []Op{&greet{Name: "x"}, &count{}, &count{}}, trace)

	t.Run("TryRun hands the error to the body, Run stops it", func(t *testing.T) {
		partial := struct{ *both }{} // has HandleGreet and HandleCount through embedding
		partial.both = &both{}
		onlyGreet := struct{ greeter }{greeter: partial}

		saw := Prog(func(fx Fx) (string, error) {
			_, err := fx.TryRun(twice)
			return "saw: " + err.Error(), nil
		})
		got, err := Interpret(context.Background(), saw, onlyGreet)
		require.NoError(t, err)
		assert.Equal(t, "saw: b: handler does not implement counter", got)

		stops := Prog(func(fx Fx) (string, error) {
			fx.Run(twice)
			return "unreachable", nil
		})
		_, err = Interpret(context.Background(), stops, onlyGreet)
		assert.EqualError(t, err, "b: handler does not implement counter")
	})
}

func TestFx_attemptAndAMissingHandler(t *testing.T) {
	program := Prog(func(fx Fx) (string, error) {
		_, err := fx.Attempt(&count{})
		return "attempt: " + err.Error(), nil
	})
	got, err := Interpret(context.Background(), program, struct{}{})
	require.NoError(t, err)
	assert.Equal(t, "attempt: b: handler does not implement counter", got)

	_, err = Run[Op, string](context.Background(), HandlerOf(struct{}{}), PerformAs[Op, string](nil))
	assert.EqualError(t, err, "effect: nil operation")
}
