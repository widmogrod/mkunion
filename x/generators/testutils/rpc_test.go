package testutils

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// desk answers every Inquiry. It is a complete InquiryHandler.
type desk struct{ pages map[string]string }

func (desk) HandlePing(context.Context, *Ping) (*Pong, error) { return &Pong{}, nil }

func (d desk) HandleFetch(_ context.Context, op *Fetch) (*Page, error) {
	body, ok := d.pages[op.URL]
	if !ok {
		return nil, errors.New("no such page")
	}
	return &Page{Body: body}, nil
}

var _ InquiryHandler = desk{}

// send is an in-process transport: DispatchInquiry on the desk, with the one
// error mapping an endpoint would apply. It is what rpc.Endpoint.Send does.
func send(h InquiryHandler) func(ctx context.Context, op Inquiry) (Reply, error) {
	return func(ctx context.Context, op Inquiry) (Reply, error) {
		res, err := DispatchInquiry(ctx, h, op)
		if err != nil {
			return &Refused{Reason: err.Error()}, nil
		}
		return res, nil
	}
}

func TestDispatchInquiry_answersWithTheReplyUnion(t *testing.T) {
	ctx := context.Background()
	res, err := DispatchInquiry(ctx, desk{pages: map[string]string{"a": "A"}}, &Fetch{URL: "a"})
	require.NoError(t, err)
	assert.Equal(t, &Page{Body: "A"}, res, "the answer is a Reply holding the *Page")

	_, err = DispatchInquiry(ctx, desk{}, &Fetch{URL: "b"})
	assert.EqualError(t, err, "no such page")
}

func TestNewInquiryClient_typesTheAnswerAndFlagsTheRest(t *testing.T) {
	ctx := context.Background()
	client := NewInquiryClient(send(desk{pages: map[string]string{"a": "A"}}))

	page, err := client.HandleFetch(ctx, &Fetch{URL: "a"}) // page is *Page, no cast
	require.NoError(t, err)
	assert.Equal(t, "A", page.Body)

	_, err = client.HandleFetch(ctx, &Fetch{URL: "b"})
	var unexpected *InquiryUnexpectedAnswer
	require.ErrorAs(t, err, &unexpected)
	assert.Equal(t, &Fetch{URL: "b"}, unexpected.Inquiry)
	assert.Equal(t, &Refused{Reason: "no such page"}, unexpected.Reply)
	assert.EqualError(t, err, "testutils: *testutils.Fetch answered with *testutils.Refused: no such page")

	var refused *Refused
	require.ErrorAs(t, err, &refused, "Unwrap hands errors.As the Reply when it is an error")
	assert.Equal(t, "no such page", refused.Reason)

	wrong := NewInquiryClient(func(context.Context, Inquiry) (Reply, error) { return &Pong{}, nil })
	_, err = wrong.HandleFetch(ctx, &Fetch{URL: "a"})
	assert.EqualError(t, err, "testutils: *testutils.Fetch answered with *testutils.Pong")
	assert.False(t, errors.As(err, &refused), "a Pong is not an error: nothing to unwrap")

	transport := NewInquiryClient(func(context.Context, Inquiry) (Reply, error) { return nil, errors.New("connection refused") })
	_, err = transport.HandlePing(ctx, &Ping{})
	assert.EqualError(t, err, "connection refused", "a transport error passes through untouched")
}

func TestAroundInquiry_decoratesEveryVariantOnce(t *testing.T) {
	ctx := context.Background()
	var seen []string
	logged := AroundInquiry(desk{pages: map[string]string{"a": "A"}}, func(ctx context.Context, op Inquiry, call func(ctx context.Context) (any, error)) (any, error) {
		seen = append(seen, MatchInquiryR1(op, func(*Ping) string { return "ping" }, func(*Fetch) string { return "fetch" }))
		return call(ctx)
	})
	pong, err := logged.HandlePing(ctx, &Ping{})
	require.NoError(t, err)
	assert.Equal(t, &Pong{}, pong)
	page, err := logged.HandleFetch(ctx, &Fetch{URL: "a"})
	require.NoError(t, err)
	assert.Equal(t, "A", page.Body)
	assert.Equal(t, []string{"ping", "fetch"}, seen)

	short := AroundInquiry(desk{}, func(context.Context, Inquiry, func(context.Context) (any, error)) (any, error) {
		return nil, errors.New("circuit open")
	})
	page, err = short.HandleFetch(ctx, &Fetch{URL: "a"})
	assert.Nil(t, page, "a decorator that does not call through answers with the zero value")
	assert.EqualError(t, err, "circuit open")
}
