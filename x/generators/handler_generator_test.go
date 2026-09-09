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

	assert.Equal(t, `// QueryHandler answers every Query with the type it declares in f.Returns.
// Adding a variant to Query breaks every QueryHandler at compile time.
type QueryHandler interface {
	HandleGetUser(ctx context.Context, op *GetUser) (*User, error)
	HandleCount(ctx context.Context, op *Count) (int, error)
	HandleLastSeen(ctx context.Context, op *LastSeen) (time.Time, error)
}

// QueryOf is a Query that answers with R.
type QueryOf[R any] interface {
	Query
	HandleQuery(ctx context.Context, h QueryHandler) (R, error)
}

var (
	_ QueryOf[*User] = (*GetUser)(nil)
	_ QueryOf[int] = (*Count)(nil)
	_ QueryOf[time.Time] = (*LastSeen)(nil)
)

func (r *GetUser) HandleQuery(ctx context.Context, h QueryHandler) (*User, error) {
	return h.HandleGetUser(ctx, r)
}

func (r *Count) HandleQuery(ctx context.Context, h QueryHandler) (int, error) {
	return h.HandleCount(ctx, r)
}

func (r *LastSeen) HandleQuery(ctx context.Context, h QueryHandler) (time.Time, error) {
	return h.HandleLastSeen(ctx, r)
}

// Perform and Answer let a variant be performed by any handler value that has
// this union's Handle methods, so operations from several unions can share one
// program and one handler (see x/effect: Op, OpOf, Fx). Answer keeps the type.
func (r *GetUser) Answer(ctx context.Context, h any) (*User, error) {
	typed, ok := h.(QueryHandler)
	if !ok {
		var zero *User
		return zero, fmt.Errorf("testutils: handler %T does not implement QueryHandler", h)
	}
	return typed.HandleGetUser(ctx, r)
}

func (r *GetUser) Perform(ctx context.Context, h any) (any, error) { return r.Answer(ctx, h) }

func (r *Count) Answer(ctx context.Context, h any) (int, error) {
	typed, ok := h.(QueryHandler)
	if !ok {
		var zero int
		return zero, fmt.Errorf("testutils: handler %T does not implement QueryHandler", h)
	}
	return typed.HandleCount(ctx, r)
}

func (r *Count) Perform(ctx context.Context, h any) (any, error) { return r.Answer(ctx, h) }

func (r *LastSeen) Answer(ctx context.Context, h any) (time.Time, error) {
	typed, ok := h.(QueryHandler)
	if !ok {
		var zero time.Time
		return zero, fmt.Errorf("testutils: handler %T does not implement QueryHandler", h)
	}
	return typed.HandleLastSeen(ctx, r)
}

func (r *LastSeen) Perform(ctx context.Context, h any) (any, error) { return r.Answer(ctx, h) }

// QueryHandlerFunc adapts a typed QueryHandler to a plain function over the union.
// The answer is the type the variant declares; only its static type is lost.
func QueryHandlerFunc(h QueryHandler) func(ctx context.Context, op Query) (any, error) {
	return func(ctx context.Context, op Query) (any, error) {
		return MatchQueryR2(op,
			func(x *GetUser) (any, error) { return x.HandleQuery(ctx, h) },
			func(x *Count) (any, error) { return x.HandleQuery(ctx, h) },
			func(x *LastSeen) (any, error) { return x.HandleQuery(ctx, h) },
		)
	}
}

// QueryDefaults answers every Query with the zero value of its declared type.
// Embed it in a handler and override only the methods you care about.
type QueryDefaults struct{}

var _ QueryHandler = QueryDefaults{}

func (QueryDefaults) HandleGetUser(context.Context, *GetUser) (*User, error) {
	var zero *User
	return zero, nil
}

func (QueryDefaults) HandleCount(context.Context, *Count) (int, error) {
	var zero int
	return zero, nil
}

func (QueryDefaults) HandleLastSeen(context.Context, *LastSeen) (time.Time, error) {
	var zero time.Time
	return zero, nil
}

// QueryFuncs is a QueryHandler made of functions, one per operation, for handlers
// written inline. A nil function answers with the zero value of its declared type.
type QueryFuncs struct {
	GetUser func(ctx context.Context, op *GetUser) (*User, error)
	Count func(ctx context.Context, op *Count) (int, error)
	LastSeen func(ctx context.Context, op *LastSeen) (time.Time, error)
}

var _ QueryHandler = QueryFuncs{}

func (fs QueryFuncs) HandleGetUser(ctx context.Context, op *GetUser) (*User, error) {
	if fs.GetUser == nil {
		var zero *User
		return zero, nil
	}
	return fs.GetUser(ctx, op)
}

func (fs QueryFuncs) HandleCount(ctx context.Context, op *Count) (int, error) {
	if fs.Count == nil {
		var zero int
		return zero, nil
	}
	return fs.Count(ctx, op)
}

func (fs QueryFuncs) HandleLastSeen(ctx context.Context, op *LastSeen) (time.Time, error) {
	if fs.LastSeen == nil {
		var zero time.Time
		return zero, nil
	}
	return fs.LastSeen(ctx, op)
}

`, string(result))

	assert.Equal(t, PkgMap{"context": "context", "fmt": "fmt", "time": "time"}, g.ExtractImports(),
		"context plus the packages of the answer types; never the union's own package or f")
}

func TestHandlerGenerator_rejectsGenericUnion(t *testing.T) {
	inferred, err := shape.InferFromFile("testutils/generic.go")
	require.NoError(t, err)

	_, err = NewHandlerGenerator(inferred.RetrieveUnion("Record")).Generate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "has type parameters")
}

func TestHandlerGenerator_rejectsVariantWithoutReturns(t *testing.T) {
	inferred, err := shape.InferFromFile("testutils/tree.go")
	require.NoError(t, err)

	_, err = NewHandlerGenerator(inferred.RetrieveUnion("Tree")).Generate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not embed f.Returns[R]")
}
