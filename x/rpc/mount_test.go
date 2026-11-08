package rpc

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tu "github.com/widmogrod/mkunion/x/generators/testutils"
)

// thermometer answers Probe: the Command endpoint of the mount tests, with a
// counter so a replayed command is seen not to run twice.
type thermometer struct{ reads atomic.Int32 }

func (t *thermometer) HandleReadTemp(ctx context.Context, _ *tu.ReadTemp) (*tu.Celsius, error) {
	t.reads.Add(1)
	// The request is in the context: a handler can read a header.
	if HeadersFrom(ctx).Get("X-Unit") == "warm" {
		return &tu.Celsius{Value: 30}, nil
	}
	return &tu.Celsius{Value: 20}, nil
}

func mounted(t *testing.T) (*httptest.Server, *thermometer) {
	t.Helper()
	therm := &thermometer{}
	mux := http.NewServeMux()
	Mount(mux, "/v1/desk",
		&Endpoint[tu.Inquiry, tu.Reply]{
			Kind: Query,
			Dispatch: func(ctx context.Context, op tu.Inquiry) (tu.Reply, error) {
				return tu.DispatchInquiry(ctx, desk{pages: map[string]string{"a": "A"}}, op)
			},
			DecodeError: func(err error) tu.Reply { return &tu.Refused{Reason: "decode: " + err.Error()} },
			Error:       func(err error) tu.Reply { return &tu.Refused{Reason: err.Error()} },
			Names:       tu.InquiryNames(),
			ByName:      tu.NewInquiryByName,
		},
	)
	Mount(mux, "/v1/probe",
		&Endpoint[tu.Probe, tu.Reading]{
			Kind:        Command,
			Dispatch:    func(ctx context.Context, op tu.Probe) (tu.Reading, error) { return tu.DispatchProbe(ctx, therm, op) },
			DecodeError: func(err error) tu.Reading { return &tu.Celsius{Value: -1} },
			Error:       func(err error) tu.Reading { return &tu.Celsius{Value: -2} },
			Names:       tu.ProbeNames(),
			ByName:      tu.NewProbeByName,
		},
	)
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts, therm
}

func do(t *testing.T, method, url, body string, headers ...string) (*http.Response, string) {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	require.NoError(t, err)
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()
	var sb strings.Builder
	_, _ = io.Copy(&sb, res.Body)
	return res, sb.String()
}

func TestMountServesBothModesWithOneHandler(t *testing.T) {
	ts, _ := mounted(t)
	res, body := do(t, "POST", ts.URL+"/v1/desk", `{"$type":"testutils.Fetch","testutils.Fetch":{"URL":"a"}}`)
	assert.Equal(t, 200, res.StatusCode)
	assert.Contains(t, body, `"Body":"A"`)

	res, body = do(t, "POST", ts.URL+"/v1/desk/Fetch", `{"URL":"a"}`)
	assert.Equal(t, 200, res.StatusCode, "bare payload, variant from the URL")
	assert.Contains(t, body, `"$type":"testutils.Page"`, "the answer is still the union")

	res, body = do(t, "POST", ts.URL+"/v1/desk/testutils.Fetch", `{"URL":"a"}`)
	assert.Equal(t, 200, res.StatusCode, "the full $type name works in the URL too")

	res, body = do(t, "POST", ts.URL+"/v1/desk/Ping", ``)
	assert.Equal(t, 200, res.StatusCode, "an empty body is the zero variant")
	assert.Contains(t, body, `"$type":"testutils.Pong"`)

	res, body = do(t, "POST", ts.URL+"/v1/desk/Nope", `{}`)
	assert.Equal(t, 404, res.StatusCode)
	assert.Contains(t, body, "unknown type: Nope")

	res, body = do(t, "POST", ts.URL+"/v1/desk/Fetch", `{"URL":`)
	assert.Equal(t, 400, res.StatusCode)
	assert.Contains(t, body, "decode:")

	res, body = do(t, "POST", ts.URL+"/v1/desk", `{"$type":"testutils.ReadTemp","testutils.ReadTemp":{}}`)
	assert.Equal(t, 400, res.StatusCode, "a $type of another resource is unknown here")
	assert.Contains(t, body, "unknown type")
}

func TestMountQueriesMayBeGetCommandsMayNot(t *testing.T) {
	ts, therm := mounted(t)
	res, body := do(t, "GET", ts.URL+"/v1/desk/Fetch?q="+url.QueryEscape(`{"URL":"a"}`), "")
	assert.Equal(t, 200, res.StatusCode)
	assert.Contains(t, body, `"Body":"A"`)

	res, _ = do(t, "GET", ts.URL+"/v1/probe/ReadTemp", "")
	assert.Equal(t, 405, res.StatusCode, "a command is POST only")
	assert.Equal(t, "POST", res.Header.Get("Allow"))
	assert.Equal(t, int32(0), therm.reads.Load())

	res, _ = do(t, "DELETE", ts.URL+"/v1/desk/Fetch", "")
	assert.Equal(t, 405, res.StatusCode)
}

func TestMountReplaysACommandWithTheSameIdempotencyKey(t *testing.T) {
	ts, therm := mounted(t)
	res, first := do(t, "POST", ts.URL+"/v1/probe/ReadTemp", `{}`, "Idempotency-Key", "k1", "X-Unit", "warm")
	assert.Equal(t, 200, res.StatusCode)
	assert.Contains(t, first, `"Value":30`, "the handler read the header through the context")
	assert.Empty(t, res.Header.Get("Idempotent-Replayed"))

	res, again := do(t, "POST", ts.URL+"/v1/probe/ReadTemp", `{}`, "Idempotency-Key", "k1")
	assert.Equal(t, 200, res.StatusCode)
	assert.Equal(t, first, again, "the first answer, byte for byte, header or not")
	assert.Equal(t, "true", res.Header.Get("Idempotent-Replayed"))
	assert.Equal(t, int32(1), therm.reads.Load(), "the handler ran once")

	do(t, "POST", ts.URL+"/v1/probe/ReadTemp", `{}`, "Idempotency-Key", "k2")
	do(t, "POST", ts.URL+"/v1/probe/ReadTemp", `{}`)
	assert.Equal(t, int32(3), therm.reads.Load(), "a new key or no key runs")

	// A query ignores the header: it is not a command.
	do(t, "POST", ts.URL+"/v1/desk/Ping", ``, "Idempotency-Key", "k1")
	res, _ = do(t, "POST", ts.URL+"/v1/desk/Ping", ``, "Idempotency-Key", "k1")
	assert.Empty(t, res.Header.Get("Idempotent-Replayed"))
}

func TestVariantClientIsTheSameClientOnThePerVariantRoutes(t *testing.T) {
	ts, _ := mounted(t)
	assertClient(t, tu.NewInquiryClient(VariantClient[tu.Inquiry, tu.Reply](ts.Client(), ts.URL+"/v1/desk")))
	assertClient(t, tu.NewInquiryClient(Client[tu.Inquiry, tu.Reply](ts.Client(), ts.URL+"/v1/desk")))
}

func TestNamesIsTheCatalogueOfAMount(t *testing.T) {
	eps := []Mounted{
		&Endpoint[tu.Inquiry, tu.Reply]{Names: tu.InquiryNames()},
		&Endpoint[tu.Probe, tu.Reading]{Names: tu.ProbeNames()},
	}
	assert.Equal(t, []string{"testutils.Ping", "testutils.Fetch", "testutils.Lookup", "testutils.Scan", "testutils.ReadTemp"}, Names(eps...))
	assert.Nil(t, RequestFrom(context.Background()), "off HTTP there is no request")
}
