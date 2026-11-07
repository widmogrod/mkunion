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

	g := NewHandlerGenerator(inferred.RetrieveUnion("Query")).WithUnions(inferred.RetrieveUnions())
	result, err := g.Generate()
	require.NoError(t, err)
	assert.NotContains(t, string(result), "DispatchQuery", "answers of three unrelated types: no answer union, so no RPC layer")

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

// NewQueryHandler builds a QueryHandler from one function per variant: the arms
// of HandleQuery as a value. Exhaustive: every arm must be given. Use it where a
// handler is a value, such as effect.Interpret, or a field of a struct that
// embeds the handlers of several packages.
func NewQueryHandler(
	onGetUser func(ctx context.Context, op *GetUser) (*User, error),
	onCount func(ctx context.Context, op *Count) (int, error),
	onLastSeen func(ctx context.Context, op *LastSeen) (time.Time, error),
	onTouch func(ctx context.Context, op *Touch) error,
) QueryHandler {
	return &queryHandlerFuncs{onGetUser: onGetUser, onCount: onCount, onLastSeen: onLastSeen, onTouch: onTouch}
}

type queryHandlerFuncs struct {
	onGetUser func(ctx context.Context, op *GetUser) (*User, error)
	onCount func(ctx context.Context, op *Count) (int, error)
	onLastSeen func(ctx context.Context, op *LastSeen) (time.Time, error)
	onTouch func(ctx context.Context, op *Touch) error
}

func (h *queryHandlerFuncs) HandleGetUser(ctx context.Context, op *GetUser) (*User, error) {
	return h.onGetUser(ctx, op)
}

func (h *queryHandlerFuncs) HandleCount(ctx context.Context, op *Count) (int, error) {
	return h.onCount(ctx, op)
}

func (h *queryHandlerFuncs) HandleLastSeen(ctx context.Context, op *LastSeen) (time.Time, error) {
	return h.onLastSeen(ctx, op)
}

func (h *queryHandlerFuncs) HandleTouch(ctx context.Context, op *Touch) error {
	return h.onTouch(ctx, op)
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

// AroundQuery decorates every method of next with around. around gets the
// operation as the Query union and a call that runs it on next, and returns the
// answer untyped; AroundQuery casts it back to the variant's declared type. One
// function wraps all variants: logging, metrics, deadlines.
func AroundQuery(next QueryHandler, around func(ctx context.Context, op Query, call func(ctx context.Context) (any, error)) (any, error)) QueryHandler {
	return NewQueryHandler(
		func(ctx context.Context, op *GetUser) (*User, error) {
			out, err := around(ctx, op, func(ctx context.Context) (any, error) { return next.HandleGetUser(ctx, op) })
			answer, _ := out.(*User)
			return answer, err
		},
		func(ctx context.Context, op *Count) (int, error) {
			out, err := around(ctx, op, func(ctx context.Context) (any, error) { return next.HandleCount(ctx, op) })
			answer, _ := out.(int)
			return answer, err
		},
		func(ctx context.Context, op *LastSeen) (time.Time, error) {
			out, err := around(ctx, op, func(ctx context.Context) (any, error) { return next.HandleLastSeen(ctx, op) })
			answer, _ := out.(time.Time)
			return answer, err
		},
		func(ctx context.Context, op *Touch) error {
			_, err := around(ctx, op, func(ctx context.Context) (any, error) { return nil, next.HandleTouch(ctx, op) })
			return err
		},
	)
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

func TestHandlerGenerator_rpcLayerWhenEveryAnswerIsAVariantOfOneUnion(t *testing.T) {
	inferred, err := shape.InferFromFile("testutils/rpc.go")
	require.NoError(t, err)

	tagged, err := NewHandlerGenerator(inferred.RetrieveUnion("Inquiry")).WithUnions(inferred.RetrieveUnions()).Generate()
	require.NoError(t, err)
	for _, want := range []string{
		"func DispatchInquiry(ctx context.Context, h InquiryHandler, op Inquiry) (Reply, error) {",
		"\treturn MatchInquiryR2[Reply, error](op,",
		"func NewInquiryClient(send func(ctx context.Context, op Inquiry) (Reply, error)) InquiryHandler {",
		"func (c *inquiryClient) HandleFetch(ctx context.Context, op *Fetch) (*Page, error) {",
		"\treturn zero, &InquiryUnexpectedAnswer{Inquiry: op, Reply: res}",
		"type InquiryUnexpectedAnswer struct {\n\tInquiry Inquiry\n\tReply Reply\n}",
		"func (e *InquiryUnexpectedAnswer) Unwrap() error {",
		"func AroundInquiry(next InquiryHandler, around func(ctx context.Context, op Inquiry, call func(ctx context.Context) (any, error)) (any, error)) InquiryHandler {",
	} {
		assert.Contains(t, string(tagged), want)
	}

	inferredOnly, err := NewHandlerGenerator(inferred.RetrieveUnion("Probe")).WithUnions(inferred.RetrieveUnions()).Generate()
	require.NoError(t, err)
	assert.Contains(t, string(inferredOnly), "func DispatchProbe(ctx context.Context, h ProbeHandler, op Probe) (Reading, error) {",
		"no answers= option: Reading is inferred because the only answer is *Celsius")

	_, err = NewHandlerGenerator(inferred.RetrieveUnion("Inquiry")).Generate()
	assert.EqualError(t, err, "generators.HandlerGenerator.Generate: union Inquiry: answers=Reply names no union in package testutils; the answer union must be declared in the same package",
		"without WithUnions the option cannot be honoured, and says so")
}

func TestHandlerGenerator_answersOptionRefusesADriftedVariant(t *testing.T) {
	inferred, err := shape.InferFromFile("testutils/rpc.go")
	require.NoError(t, err)
	unions := inferred.RetrieveUnions()
	inquiry := inferred.RetrieveUnion("Inquiry")
	probe := inferred.RetrieveUnion("Probe")

	drifted := *inquiry
	drifted.Variant = append(append([]shape.Shape{}, inquiry.Variant...), probe.Variant[0]) // ReadTemp answers *Celsius
	_, err = NewHandlerGenerator(&drifted).WithUnions(unions).Generate()
	assert.EqualError(t, err, "generators.HandlerGenerator.Generate: union Inquiry: variant ReadTemp answers with *Celsius, which is not a variant of Reply")

	silent := *inquiry
	silent.Variant = append(append([]shape.Shape{}, inquiry.Variant...), &shape.StructLike{Name: "Nudge", PkgName: inquiry.PkgName, PkgImportName: inquiry.PkgImportName})
	_, err = NewHandlerGenerator(&silent).WithUnions(unions).Generate()
	assert.EqualError(t, err, "generators.HandlerGenerator.Generate: union Inquiry: variant Nudge has no f.Returns, but answers=Reply needs every variant to answer with a Reply")

	// The same drift without the option is not an error: the RPC layer is
	// simply not generated, and the code that called DispatchInquiry stops compiling.
	untagged := drifted
	untagged.Tags = map[string]shape.Tag{shape.TagUnionName: {Value: "Inquiry"}}
	out, err := NewHandlerGenerator(&untagged).WithUnions(unions).Generate()
	require.NoError(t, err)
	assert.NotContains(t, string(out), "DispatchInquiry")
}
