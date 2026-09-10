package testutils

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Ask performs one query and returns its answer with the type the variant
// declared. R is inferred from the f.Returns marker, so a caller never spells
// it. The cast is safe: Perform hands the variant to its own typed method.
func Ask[R any](ctx context.Context, h QueryHandler, q interface {
	Query
	Ret() R
	Perform(ctx context.Context, h any) (any, error)
}) (R, error) {
	answer, err := q.Perform(ctx, h)
	if err != nil {
		var zero R
		return zero, err
	}
	return answer.(R), nil
}

// users answers from a map. It is a complete QueryHandler: every variant has
// a method, nothing is defaulted.
type users struct {
	byID    map[string]*User
	touched []string
}

func (u *users) HandleGetUser(_ context.Context, op *GetUser) (*User, error) {
	user, ok := u.byID[op.ID]
	if !ok {
		return nil, errors.New("no such user")
	}
	return user, nil
}

func (u *users) HandleCount(context.Context, *Count) (int, error) {
	return len(u.byID), nil
}

func (u *users) HandleLastSeen(context.Context, *LastSeen) (time.Time, error) {
	return time.Time{}, nil
}

func (u *users) HandleTouch(_ context.Context, op *Touch) error {
	u.touched = append(u.touched, op.ID)
	return nil
}

var _ QueryHandler = (*users)(nil)

func TestQueryHandler_answersWithDeclaredType(t *testing.T) {
	ctx := context.Background()
	h := &users{byID: map[string]*User{"1": {ID: "1", Name: "Ada"}}}

	user, err := Ask(ctx, h, &GetUser{ID: "1"}) // user is *User, no cast
	require.NoError(t, err)
	assert.Equal(t, "Ada", user.Name)

	count, err := Ask(ctx, h, &Count{}) // count is int
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	seen, err := Ask(ctx, h, &LastSeen{ID: "1"}) // seen is time.Time
	require.NoError(t, err)
	assert.Equal(t, time.Time{}, seen)

	_, err = Ask(ctx, h, &GetUser{ID: "missing"})
	assert.EqualError(t, err, "no such user")
}

func TestHandleQuery_functionForm(t *testing.T) {
	ctx := context.Background()
	var touched []string
	run := func(op Query) (any, error) {
		return HandleQuery(ctx, op,
			func(context.Context, *GetUser) (*User, error) { return &User{ID: "1"}, nil },
			func(context.Context, *Count) (int, error) { return 7, nil },
			func(context.Context, *LastSeen) (time.Time, error) { return time.Time{}, nil },
			func(_ context.Context, op *Touch) error { touched = append(touched, op.ID); return nil },
		)
	}

	answer, err := run(&Count{})
	require.NoError(t, err)
	assert.Equal(t, 7, answer, "the answer comes back as any, with the declared value inside")

	answer, err = run(&Touch{ID: "x"})
	require.NoError(t, err)
	assert.Nil(t, answer, "a variant without f.Returns answers with nil")
	assert.Equal(t, []string{"x"}, touched)
}

// countOnly has one method. It is a QueryCountHandler, not a QueryHandler.
type countOnly struct{}

func (countOnly) HandleCount(context.Context, *Count) (int, error) { return 3, nil }

func TestPerform_asksOnlyForTheVariantsMethod(t *testing.T) {
	ctx := context.Background()

	answer, err := (&Count{}).Perform(ctx, countOnly{})
	require.NoError(t, err)
	assert.Equal(t, 3, answer, "a handler with just HandleCount performs Count")

	_, err = (&GetUser{ID: "1"}).Perform(ctx, countOnly{})
	assert.EqualError(t, err, "testutils: handler testutils.countOnly does not implement QueryGetUserHandler")
}
