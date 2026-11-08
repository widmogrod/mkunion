package rpc

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tu "github.com/widmogrod/mkunion/x/generators/testutils"
	"github.com/widmogrod/mkunion/x/shared"
)

// desk answers every Inquiry: the server-side handler.
type desk struct{ pages map[string]string }

func (desk) HandlePing(context.Context, *tu.Ping) (*tu.Pong, error) { return &tu.Pong{}, nil }

func (d desk) HandleFetch(_ context.Context, op *tu.Fetch) (*tu.Page, error) {
	body, ok := d.pages[op.URL]
	if !ok {
		return nil, errors.New("no such page")
	}
	return &tu.Page{Body: body}, nil
}

// endpoint is the whole server side: the generated DispatchInquiry plus the
// two answer functions only the application can write.
func endpoint(h tu.InquiryHandler) *Endpoint[tu.Inquiry, tu.Reply] {
	return &Endpoint[tu.Inquiry, tu.Reply]{
		Dispatch:    func(ctx context.Context, op tu.Inquiry) (tu.Reply, error) { return tu.DispatchInquiry(ctx, h, op) },
		DecodeError: func(err error) tu.Reply { return &tu.Refused{Reason: "decode: " + err.Error()} },
		Error:       func(err error) tu.Reply { return &tu.Refused{Reason: err.Error()} },
	}
}

// assertClient is every assertion a client must satisfy, whichever transport
// it rides on.
func assertClient(t *testing.T, client tu.InquiryHandler) {
	t.Helper()
	ctx := context.Background()

	pong, err := Call(ctx, client, &tu.Ping{}) // pong is *tu.Pong
	require.NoError(t, err)
	assert.Equal(t, &tu.Pong{}, pong)

	page, err := Call(ctx, client, &tu.Fetch{URL: "a"}) // page is *tu.Page
	require.NoError(t, err)
	assert.Equal(t, "A", page.Body)

	_, err = Call(ctx, client, &tu.Fetch{URL: "missing"})
	var refused *tu.Refused
	require.ErrorAs(t, err, &refused, "the server's error variant comes back through errors.As")
	assert.Equal(t, "no such page", refused.Reason)
	var unexpected *tu.InquiryUnexpectedAnswer
	require.ErrorAs(t, err, &unexpected)
	assert.Equal(t, &tu.Fetch{URL: "missing"}, unexpected.Inquiry)
}

func TestClientOverHTTPAndClientOverSendAreTheSameClient(t *testing.T) {
	ep := endpoint(desk{pages: map[string]string{"a": "A"}})
	ts := httptest.NewServer(ep)
	defer ts.Close()

	t.Run("http", func(t *testing.T) {
		assertClient(t, tu.NewInquiryClient(Client[tu.Inquiry, tu.Reply](ts.Client(), ts.URL)))
	})
	t.Run("in-process", func(t *testing.T) {
		assertClient(t, tu.NewInquiryClient(ep.Send))
	})
}

func post(t *testing.T, h http.Handler, body string) (int, string, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1", strings.NewReader(body)))
	out, _ := io.ReadAll(rec.Result().Body)
	return rec.Code, rec.Header().Get("Content-Type"), string(out)
}

func TestEndpointStatusRule(t *testing.T) {
	var decodeErrs []error
	ep := endpoint(desk{pages: map[string]string{"a": "A"}})
	ep.DecodeError = func(err error) tu.Reply {
		decodeErrs = append(decodeErrs, err)
		return &tu.Refused{Reason: "decode"}
	}
	ep.MaxBody = 64

	cases := []struct {
		body   string
		status int
		want   string // in the body
		is     error  // what DecodeError gets, when 400
	}{
		{`{"$type":"testutils.Ping","testutils.Ping":{}}`, 200, `"$type":"testutils.Pong"`, nil},
		{`{"$type":"testutils.Fetch","testutils.Fetch":{"URL":"nope"}}`, 200, `"Reason":"no such page"`, nil},
		{`{"$type":"testutils.Fetch"}`, 200, `"Reason":"no such page"`, nil}, // no payload: zero Fetch, the handler decides
		{``, 400, `"Reason":"decode"`, ErrEmptyBody},
		{`null`, 400, `"Reason":"decode"`, ErrEmptyBody},
		{`{}`, 400, `"Reason":"decode"`, shared.ErrNoType},
		{`{"$type":"testutils.Nope"}`, 400, `"Reason":"decode"`, shared.ErrUnknownType},
		{`[1,2]`, 400, `"Reason":"decode"`, nil},
		{`"text"`, 400, `"Reason":"decode"`, nil},
		{`{"$type":"testutils.Fetch","testutils.Fetch":{"URL":"` + strings.Repeat("x", 100) + `"}}`, 413, `"Reason":"decode"`, nil},
	}
	for _, c := range cases {
		decodeErrs = nil
		status, ct, body := post(t, ep, c.body)
		assert.Equal(t, c.status, status, c.body)
		assert.Equal(t, "application/json", ct, c.body)
		assert.Contains(t, body, c.want, c.body)
		if c.status != 200 {
			require.Len(t, decodeErrs, 1, c.body)
			if c.is != nil {
				assert.ErrorIs(t, decodeErrs[0], c.is, c.body)
			}
		}
	}
}

func TestEndpointWithoutErrorAnswersFiveHundred(t *testing.T) {
	ep := endpoint(desk{})
	ep.Error = nil
	status, _, body := post(t, ep, `{"$type":"testutils.Fetch","testutils.Fetch":{"URL":"nope"}}`)
	assert.Equal(t, 500, status)
	assert.Equal(t, "no such page\n", body)
}

func TestClientStatusRule(t *testing.T) {
	ctx := context.Background()
	var status int
	var contentType, body string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer s3cret", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	defer ts.Close()
	send := Client[tu.Inquiry, tu.Reply](nil, ts.URL, Bearer("s3cret"))
	client := tu.NewInquiryClient(send)

	status, contentType, body = 401, "application/json", `{"$type":"testutils.Refused","testutils.Refused":{"Reason":"who are you"}}`
	_, err := Call(ctx, client, &tu.Ping{})
	var se *StatusError
	require.ErrorAs(t, err, &se, "401 is not an answer, whatever the body")
	assert.Equal(t, 401, se.Status)
	assert.EqualError(t, err, `rpc: HTTP 401: {"$type":"testutils.Refused","testutils.Refused":{"Reason":"who are you"}}`)

	status, contentType, body = 400, "application/json", `{"$type":"testutils.Refused","testutils.Refused":{"Reason":"unknown type: testutils.Ping"}}`
	_, err = Call(ctx, client, &tu.Ping{})
	var refused *tu.Refused
	require.ErrorAs(t, err, &refused, "a 400 with a JSON body is the server's decode answer: a Reply")
	assert.Equal(t, "unknown type: testutils.Ping", refused.Reason)

	status, contentType, body = 400, "text/html", `<h1>Bad Request</h1>`
	_, err = Call(ctx, client, &tu.Ping{})
	require.ErrorAs(t, err, &se, "a 400 that is not JSON came from something else: a proxy")
	assert.Equal(t, 400, se.Status)

	status, contentType, body = 200, "application/json", `{"$type":"testutils.Nope","testutils.Nope":{}}`
	_, err = Call(ctx, client, &tu.Ping{})
	assert.ErrorIs(t, err, shared.ErrUnknownType, "a 200 that is not a Reply is a decode error")
	assert.ErrorContains(t, err, "rpc: decode answer to *testutils.Ping")

	status, contentType, body = 200, "application/json", `null`
	_, err = Call(ctx, client, &tu.Ping{})
	require.ErrorAs(t, err, new(*tu.InquiryUnexpectedAnswer), "a null answer is no Pong")
	assert.EqualError(t, err, "testutils: *testutils.Ping answered with <nil>")
}

func TestWriteKeepsTypeOnlyWhenToldTheUnion(t *testing.T) {
	// The fact this package is built around: JSONMarshal picks the serde by
	// the static type A, so a *Pong marshalled as itself has no $type.
	asPointer, err := shared.JSONMarshal(&tu.Pong{})
	require.NoError(t, err)
	assert.Equal(t, `{}`, string(asPointer))
	asUnion, err := shared.JSONMarshal[tu.Reply](&tu.Pong{})
	require.NoError(t, err)
	assert.Equal(t, `{"$type":"testutils.Pong","testutils.Pong":{}}`, string(asUnion))

	rec := httptest.NewRecorder()
	Write[tu.Reply](rec, 401, &tu.Refused{Reason: "no"})
	assert.Equal(t, 401, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.Equal(t, `{"$type":"testutils.Refused","testutils.Refused":{"Reason":"no"}}`, rec.Body.String())

	// A nil union marshals to nothing, not to null, and not to an error.
	var none tu.Reply
	out, err := shared.JSONMarshal[tu.Reply](none)
	require.NoError(t, err)
	assert.Equal(t, "", string(out))
	rec = httptest.NewRecorder()
	Write[tu.Reply](rec, 200, none)
	assert.Equal(t, 200, rec.Code)
	assert.Equal(t, "", rec.Body.String(), "an endpoint whose Dispatch returns (nil, nil) sends an empty 200")
}

func TestCallInfersTheAnswerAndNamesAMissingMethod(t *testing.T) {
	ctx := context.Background()
	pong, err := Call(ctx, desk{}, &tu.Ping{})
	require.NoError(t, err)
	assert.Equal(t, &tu.Pong{}, pong)

	_, err = Call(ctx, struct{}{}, &tu.Ping{})
	assert.EqualError(t, err, "testutils: handler struct {} does not implement InquiryPingHandler")
}
