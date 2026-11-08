// Package rpc moves one union pair over HTTP: a client posts one Req value as
// JSON, the server answers one Res value as JSON, both in mkunion's form
// {"$type": "pkg.Variant", "pkg.Variant": {...}}. There are no routes: the
// $type is the route. Req is a union whose variants embed f.Returns[*V] for
// variants V of Res, and the generated DispatchReq and NewReqClient are the two
// sides this package connects:
//
//	server: &rpc.Endpoint[Req, Res]{Dispatch: func(ctx, op) { return DispatchReq(ctx, h, op) }, ...}
//	client: NewReqClient(rpc.Client[Req, Res](nil, url, rpc.Bearer(token)))
//	caller: answer, err := rpc.Call(ctx, client, &GetUser{ID: "1"})   // answer is *User
//
// HTTP status rule: 200 for every answer, application errors included, since
// an Error variant is an answer; 400 when the body is not a Req at all (not
// JSON, no $type, a $type the union lacks, empty); 413 when the body is over
// MaxBody; 500 only when the answer cannot be encoded. Authentication is not
// here: it is middleware in front of the Endpoint, and a request decorator on
// the client.
package rpc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/widmogrod/mkunion/x/shared"
)

// DefaultMaxBody is the largest request body an Endpoint reads when MaxBody is 0.
const DefaultMaxBody = 1 << 20

// ErrEmptyBody is the decode error for a body that holds no Req: empty, or
// JSON null. It reaches Endpoint.DecodeError like the errors from
// shared.JSONUnmarshal (shared.ErrNoType, shared.ErrUnknownType).
var ErrEmptyBody = errors.New("rpc: no request in body")

// Endpoint serves one request union. Dispatch runs a decoded Req; DecodeError
// answers a body that is not a Req; Error answers a Dispatch error, so an
// application error reaches the client as a Res variant rather than as HTTP
// plumbing. Both answer functions are the application's: this package does not
// know which Res variant is the error one.
type Endpoint[Req, Res any] struct {
	Dispatch func(ctx context.Context, req Req) (Res, error)
	// DecodeError answers a body that did not decode to a Req. err wraps
	// shared.ErrNoType, shared.ErrUnknownType or ErrEmptyBody when that is why.
	DecodeError func(err error) Res
	// Error answers a Dispatch error. nil means the error is a 500.
	Error func(err error) Res
	// MaxBody caps the request body; 0 means DefaultMaxBody.
	MaxBody int64
}

// Send is the in-process transport: Dispatch with Error applied, no HTTP. A
// client built on it behaves like one built on Client, so a test can swap the
// wire for a handler without touching the code under test.
func (e *Endpoint[Req, Res]) Send(ctx context.Context, req Req) (Res, error) {
	res, err := e.Dispatch(ctx, req)
	if err != nil && e.Error != nil {
		return e.Error(err), nil
	}
	return res, err
}

// ServeHTTP reads one Req from the body and writes one Res.
func (e *Endpoint[Req, Res]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	maxBody := e.MaxBody
	if maxBody <= 0 {
		maxBody = DefaultMaxBody
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBody))
	if err != nil {
		status := http.StatusBadRequest
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			status = http.StatusRequestEntityTooLarge
		}
		Write(w, status, e.DecodeError(err))
		return
	}
	req, err := shared.JSONUnmarshal[Req](body)
	if err == nil && isNil(req) {
		err = ErrEmptyBody
	}
	if err != nil {
		Write(w, http.StatusBadRequest, e.DecodeError(err))
		return
	}
	res, err := e.Send(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	Write(w, http.StatusOK, res)
}

// Write encodes res as the Res union and sends it with status. It is what
// ServeHTTP writes with, exported so middleware in front of an Endpoint (a
// bearer-token check) can answer with the same union.
func Write[Res any](w http.ResponseWriter, status int, res Res) {
	body, err := shared.JSONMarshal[Res](res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// isNil reports whether a decoded value holds nothing: a nil union interface,
// or a nil pointer, map or slice.
func isNil[A any](v A) bool {
	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return true
	}
	switch rv.Kind() {
	case reflect.Interface, reflect.Pointer, reflect.Map, reflect.Slice:
		return rv.IsNil()
	}
	return false
}

// StatusError is a non-2xx answer the client could not turn into a Res: the
// status and the body as sent, for the caller to read.
type StatusError struct {
	Status int
	Body   []byte
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("rpc: HTTP %d: %s", e.Status, bytes.TrimSpace(e.Body))
}

// Client is the HTTP transport for a request union: it posts one Req to url
// and decodes one Res. decorate runs on every request before it is sent, for
// headers (see Bearer). A nil c is http.DefaultClient.
//
// A 2xx body is a Res. A 400 with a JSON body is also a Res, by the Endpoint's
// own rule (its DecodeError answer), so a client newer than its server sees
// the server's error variant rather than a status code. Anything else is a
// *StatusError.
func Client[Req, Res any](c *http.Client, url string, decorate ...func(*http.Request)) func(ctx context.Context, req Req) (Res, error) {
	if c == nil {
		c = http.DefaultClient
	}
	return func(ctx context.Context, req Req) (Res, error) {
		var zero Res
		body, err := shared.JSONMarshal[Req](req)
		if err != nil {
			return zero, fmt.Errorf("rpc: encode %T: %w", req, err)
		}
		hr, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return zero, err
		}
		hr.Header.Set("Content-Type", "application/json")
		hr.Header.Set("Accept", "application/json")
		for _, d := range decorate {
			d(hr)
		}
		resp, err := c.Do(hr)
		if err != nil {
			return zero, err
		}
		defer resp.Body.Close()
		out, err := io.ReadAll(resp.Body)
		if err != nil {
			return zero, err
		}
		isJSON := strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json")
		if !(resp.StatusCode/100 == 2 || (resp.StatusCode == http.StatusBadRequest && isJSON)) {
			return zero, &StatusError{Status: resp.StatusCode, Body: out}
		}
		res, err := shared.JSONUnmarshal[Res](out)
		if err != nil {
			return zero, fmt.Errorf("rpc: decode answer to %T: %w", req, err)
		}
		return res, nil
	}
}

// Bearer is a Client decorator that sets "Authorization: Bearer <token>".
func Bearer(token string) func(*http.Request) {
	return func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+token) }
}

// Op is a union variant that embeds f.Returns[R]: Ret names the answer type
// and the generated Perform hands the variant to a handler's typed method.
type Op[R any] interface {
	Ret() R
	Perform(ctx context.Context, h any) (any, error)
}

// Call performs op on h and returns the answer with the type the variant
// declared; R is inferred from op, so a caller never spells it. h is any
// value with the variant's Handle method: a server-side handler, or a client
// from NewReqClient, which is how the same call reaches a handler in-process
// or over HTTP.
func Call[R any](ctx context.Context, h any, op Op[R]) (R, error) {
	var zero R
	out, err := op.Perform(ctx, h)
	if err != nil {
		return zero, err
	}
	answer, _ := out.(R)
	return answer, nil
}
