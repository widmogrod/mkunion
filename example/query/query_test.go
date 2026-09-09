package query

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var directory = InMemory{Users: []User{{ID: "1", Name: "Ada"}, {ID: "2", Name: "Alan"}, {ID: "3", Name: "Grace"}}}

// --8<-- [start:typed-answers]

func TestAsk_answersWithTheTypeEachQueryDeclares(t *testing.T) {
	ctx := context.Background()

	user, err := Ask(ctx, directory, &GetUser{ID: "1"}) // user is *User, no cast
	require.NoError(t, err)
	assert.Equal(t, &User{ID: "1", Name: "Ada"}, user)

	found, err := Ask(ctx, directory, &FindUsers{Prefix: "A"}) // found is []User
	require.NoError(t, err)
	assert.Equal(t, []User{{ID: "1", Name: "Ada"}, {ID: "2", Name: "Alan"}}, found)

	count, err := Ask(ctx, directory, &CountUsers{}) // count is int
	require.NoError(t, err)
	assert.Equal(t, 3, count)
}

// --8<-- [end:typed-answers]

// --8<-- [start:test-handlers]

// onlyCount is a test handler: it embeds the generated QueryDefaults, so it
// only has to answer the one query the test cares about.
type onlyCount struct{ QueryDefaults }

func (onlyCount) HandleCountUsers(context.Context, *CountUsers) (int, error) { return 42, nil }

func TestHandlers_forTests(t *testing.T) {
	ctx := context.Background()

	count, err := Ask(ctx, onlyCount{}, &CountUsers{})
	require.NoError(t, err)
	assert.Equal(t, 42, count)

	user, err := Ask(ctx, onlyCount{}, &GetUser{ID: "1"})
	require.NoError(t, err)
	assert.Nil(t, user, "QueryDefaults answers with the zero value")

	// QueryFuncs is the same idea as closures: fill in what matters, leave the rest nil.
	down := errors.New("directory down")
	flaky := QueryFuncs{
		GetUser: func(context.Context, *GetUser) (*User, error) { return nil, down },
	}
	_, err = Ask(ctx, flaky, &GetUser{ID: "1"})
	assert.ErrorIs(t, err, down)
}

// --8<-- [end:test-handlers]

// --8<-- [start:over-the-wire]

// A query is data, so it can arrive as JSON. The union's generated JSON
// decodes it into the right variant, and QueryHandlerFunc dispatches it.
// This is a query endpoint in five lines.
func serve(h QueryHandler, request []byte) ([]byte, error) {
	q, err := QueryFromJSON(request)
	if err != nil {
		return nil, err
	}
	answer, err := QueryHandlerFunc(h)(context.Background(), q)
	if err != nil {
		return nil, err
	}
	return json.Marshal(answer)
}

func TestQueries_overTheWire(t *testing.T) {
	answer, err := serve(directory, []byte(`{"$type":"query.FindUsers","query.FindUsers":{"Prefix":"G"}}`))
	require.NoError(t, err)
	assert.JSONEq(t, `[{"ID":"3","Name":"Grace"}]`, string(answer))

	answer, err = serve(directory, []byte(`{"$type":"query.CountUsers","query.CountUsers":{}}`))
	require.NoError(t, err)
	assert.Equal(t, `3`, string(answer))

	_, err = serve(directory, []byte(`{"$type":"query.DeleteUser","query.DeleteUser":{}}`))
	assert.ErrorContains(t, err, "unknown type: query.DeleteUser", "a query the union does not have is refused before any handler runs")
}

// --8<-- [end:over-the-wire]
