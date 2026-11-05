package query

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func directory() *InMemory {
	return &InMemory{Users: []User{{ID: "1", Name: "Ada"}, {ID: "2", Name: "Alan"}, {ID: "3", Name: "Grace"}}}
}

// --8<-- [start:typed-answers]

func TestAsk_answersWithTheTypeEachQueryDeclares(t *testing.T) {
	ctx := context.Background()
	h := directory()

	user, err := Ask(ctx, h, &GetUser{ID: "1"}) // user is *User, no cast
	require.NoError(t, err)
	assert.Equal(t, &User{ID: "1", Name: "Ada"}, user)

	found, err := Ask(ctx, h, &FindUsers{Prefix: "A"}) // found is []User
	require.NoError(t, err)
	assert.Equal(t, []User{{ID: "1", Name: "Ada"}, {ID: "2", Name: "Alan"}}, found)

	count, err := Ask(ctx, h, &CountUsers{}) // count is int
	require.NoError(t, err)
	assert.Equal(t, 3, count)

	// DeleteUser has no f.Returns, so it has no Perform and Ask cannot take it.
	// It is handled directly, with an error only.
	require.NoError(t, h.HandleDeleteUser(ctx, &DeleteUser{ID: "2"}))
	count, err = Ask(ctx, h, &CountUsers{})
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

// --8<-- [end:typed-answers]

// --8<-- [start:test-handlers]

// onlyCount is a test handler with one method. It is a QueryCountUsersHandler,
// not a QueryHandler: the compiler will not let Ask take it, because Ask
// wants every question answered. Perform asks only for the method it needs.
type onlyCount struct{}

func (onlyCount) HandleCountUsers(context.Context, *CountUsers) (int, error) { return 42, nil }

func TestHandlers_forTests(t *testing.T) {
	ctx := context.Background()

	answer, err := (&CountUsers{}).Perform(ctx, onlyCount{})
	require.NoError(t, err)
	assert.Equal(t, 42, answer)

	_, err = (&GetUser{ID: "1"}).Perform(ctx, onlyCount{})
	assert.EqualError(t, err, "query: handler query.onlyCount does not implement QueryGetUserHandler",
		"a partial handler fails at the first question it cannot answer, by name")

	// HandleQuery is the function form: one typed arm per question, all of
	// them required. Closures make an inline handler with no struct at all.
	down := errors.New("directory down")
	answer, err = HandleQuery(ctx, &GetUser{ID: "1"},
		func(context.Context, *GetUser) (*User, error) { return nil, down },
		func(context.Context, *FindUsers) ([]User, error) { return nil, nil },
		func(context.Context, *CountUsers) (int, error) { return 0, nil },
		func(context.Context, *DeleteUser) error { return nil },
	)
	assert.ErrorIs(t, err, down)
	assert.Nil(t, answer)
}

// --8<-- [end:test-handlers]

// --8<-- [start:over-the-wire]

// A query is data, so it can arrive as JSON. The union's generated JSON
// decodes it into the right variant, and HandleQuery dispatches it to the
// handler's methods. This is a query endpoint in a few lines.
func serve(h QueryHandler, request []byte) ([]byte, error) {
	q, err := QueryFromJSON(request)
	if err != nil {
		return nil, err
	}
	answer, err := HandleQuery(context.Background(), q,
		h.HandleGetUser, h.HandleFindUsers, h.HandleCountUsers, h.HandleDeleteUser)
	if err != nil {
		return nil, err
	}
	return json.Marshal(answer)
}

func TestQueries_overTheWire(t *testing.T) {
	h := directory()

	answer, err := serve(h, []byte(`{"$type":"query.FindUsers","query.FindUsers":{"Prefix":"G"}}`))
	require.NoError(t, err)
	assert.JSONEq(t, `[{"ID":"3","Name":"Grace"}]`, string(answer))

	answer, err = serve(h, []byte(`{"$type":"query.DeleteUser","query.DeleteUser":{"ID":"3"}}`))
	require.NoError(t, err)
	assert.Equal(t, `null`, string(answer), "a question with no answer type answers with nothing")

	answer, err = serve(h, []byte(`{"$type":"query.CountUsers","query.CountUsers":{}}`))
	require.NoError(t, err)
	assert.Equal(t, `2`, string(answer))

	_, err = serve(h, []byte(`{"$type":"query.DropUsers","query.DropUsers":{}}`))
	assert.ErrorContains(t, err, "unknown type: query.DropUsers", "a query the union does not have is refused before any handler runs")
}

// --8<-- [end:over-the-wire]
