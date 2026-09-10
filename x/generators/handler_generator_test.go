package generators

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/widmogrod/mkunion/x/shape"
)

func TestHandlerGenerator_Query(t *testing.T) {
	inferred, err := shape.InferFromFile("testutils/handler.go")
	require.NoError(t, err)

	g := NewHandlerGenerator(inferred.RetrieveUnion("Query"))
	result, err := g.Generate()
	require.NoError(t, err)

	assert.Equal(t, `// One handler interface per Query variant, so a handler can be assembled from
// parts. A variant with f.Returns answers with that type; one without answers
// with an error only.
type (
	QueryGetUserHandler interface {
		HandleGetUser(ctx context.Context, op *GetUser) (*User, error)
	}
	QueryCountHandler interface {
		HandleCount(ctx context.Context, op *Count) (int, error)
	}
	QueryLastSeenHandler interface {
		HandleLastSeen(ctx context.Context, op *LastSeen) (time.Time, error)
	}
	QueryTouchHandler interface {
		HandleTouch(ctx context.Context, op *Touch) error
	}
)

// QueryHandler handles every Query. Adding a variant to Query breaks every
// QueryHandler at compile time.
type QueryHandler interface {
	QueryGetUserHandler
	QueryCountHandler
	QueryLastSeenHandler
	QueryTouchHandler
}

// HandleQuery hands op to the arm for its variant. Each arm is typed by the
// variant's f.Returns; the answer comes back untyped because the arms do not
// share a type. Exhaustive: every arm must be given.
func HandleQuery(
	ctx context.Context,
	op Query,
	onGetUser func(ctx context.Context, op *GetUser) (*User, error),
	onCount func(ctx context.Context, op *Count) (int, error),
	onLastSeen func(ctx context.Context, op *LastSeen) (time.Time, error),
	onTouch func(ctx context.Context, op *Touch) error,
) (any, error) {
	return MatchQueryR2(op,
		func(x *GetUser) (any, error) { return onGetUser(ctx, x) },
		func(x *Count) (any, error) { return onCount(ctx, x) },
		func(x *LastSeen) (any, error) { return onLastSeen(ctx, x) },
		func(x *Touch) (any, error) { return nil, onTouch(ctx, x) },
	)
}

// Perform lets a variant with f.Returns be performed by any handler value that
// has its Handle method, so operations from several unions can share one
// program and one handler (see x/effect: Op, OpOf, Fx). The answer's static
// type is carried by f.Returns.Ret, not by Perform.
func (r *GetUser) Perform(ctx context.Context, h any) (any, error) {
	typed, ok := h.(QueryGetUserHandler)
	if !ok {
		return nil, fmt.Errorf("testutils: handler %T does not implement QueryGetUserHandler", h)
	}
	return typed.HandleGetUser(ctx, r)
}

func (r *Count) Perform(ctx context.Context, h any) (any, error) {
	typed, ok := h.(QueryCountHandler)
	if !ok {
		return nil, fmt.Errorf("testutils: handler %T does not implement QueryCountHandler", h)
	}
	return typed.HandleCount(ctx, r)
}

func (r *LastSeen) Perform(ctx context.Context, h any) (any, error) {
	typed, ok := h.(QueryLastSeenHandler)
	if !ok {
		return nil, fmt.Errorf("testutils: handler %T does not implement QueryLastSeenHandler", h)
	}
	return typed.HandleLastSeen(ctx, r)
}

`, string(result))

	assert.Equal(t, PkgMap{"context": "context", "fmt": "fmt", "time": "time"}, g.ExtractImports(),
		"context and fmt plus the packages of the answer types; never the union's own package or f")
}

func TestHandlerGenerator_rejectsGenericUnion(t *testing.T) {
	inferred, err := shape.InferFromFile("testutils/generic.go")
	require.NoError(t, err)

	_, err = NewHandlerGenerator(inferred.RetrieveUnion("Record")).Generate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "has type parameters")
}
