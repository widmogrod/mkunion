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
// declared. R is inferred from the argument, so a caller never spells it.
func Ask[R any](ctx context.Context, h QueryHandler, q QueryOf[R]) (R, error) {
	return q.HandleQuery(ctx, h)
}

// users answers from a map. It embeds QueryDefaults so only GetUser and
// Count need a body; LastSeen falls back to the zero time.
type users struct {
	QueryDefaults
	byID map[string]*User
}

func (u users) HandleGetUser(_ context.Context, op *GetUser) (*User, error) {
	user, ok := u.byID[op.ID]
	if !ok {
		return nil, errors.New("no such user")
	}
	return user, nil
}

func (u users) HandleCount(context.Context, *Count) (int, error) {
	return len(u.byID), nil
}

func TestQueryHandler_answersWithDeclaredType(t *testing.T) {
	ctx := context.Background()
	h := users{byID: map[string]*User{"1": {ID: "1", Name: "Ada"}}}

	user, err := Ask(ctx, h, &GetUser{ID: "1"}) // user is *User, no cast
	require.NoError(t, err)
	assert.Equal(t, "Ada", user.Name)

	count, err := Ask(ctx, h, &Count{}) // count is int
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	seen, err := Ask(ctx, h, &LastSeen{ID: "1"}) // seen is time.Time, from QueryDefaults
	require.NoError(t, err)
	assert.Equal(t, time.Time{}, seen)

	_, err = Ask(ctx, h, &GetUser{ID: "missing"})
	assert.EqualError(t, err, "no such user")
}

func TestQueryHandlerFunc_dispatchesEveryVariant(t *testing.T) {
	ctx := context.Background()
	run := QueryHandlerFunc(users{byID: map[string]*User{"1": {ID: "1"}}})

	answer, err := run(ctx, &Count{})
	require.NoError(t, err)
	assert.Equal(t, 1, answer, "the untyped adapter carries the declared answer as any")

	answer, err = run(ctx, &GetUser{ID: "1"})
	require.NoError(t, err)
	assert.Equal(t, &User{ID: "1"}, answer)
}
