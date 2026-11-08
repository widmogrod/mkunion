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
// an Error variant is an answer (an answer that implements HTTPStatus() int
// picks its own 2xx, say 202 for an accepted job); 400 when the body is not a
// Req at all (not JSON, no $type, a $type the union lacks, empty); 413 when the
// body is over MaxBody; 500 only when the answer cannot be encoded.
// Authentication is not here: it is middleware in front of the Endpoint, and a
// request decorator on the client.
//
// Mount puts one or more Endpoints under a URL prefix in two transport modes
// that reach the same Dispatch:
//
//	POST /v1/series            {"$type":"series.Search","series.Search":{"query":"tower"}}   the $type picks the variant
//	POST /v1/series/Search     {"query":"tower"}                                              the URL picks the variant
//	GET  /v1/series/Search?q={"query":"tower"}                                                 Query endpoints only
//
// The Endpoint's Kind says what its requests do: a Query may be sent as GET; a
// Command is replayed, not re-run, when a repeat carries the same
// Idempotency-Key header.
package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"

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
	// Kind is what the requests do to the server; Mount reads it. The zero
	// value is Command, the careful one.
	Kind Kind
	// Names and ByName are the generated ReqNames() and NewReqByName: the
	// variants Mount routes to this Endpoint, and how it decodes a bare
	// payload. Without them an Endpoint serves only the union-body mode.
	Names  []string
	ByName func(name string) (Req, bool)

	replays replayCache // Command: answers by Idempotency-Key
}

// Kind is what an Endpoint's requests do to the server, so a transport can
// treat them differently without reading them.
type Kind int

const (
	// Command changes the server. POST only; a repeat with the same
	// Idempotency-Key header gets the first answer back without running again.
	Command Kind = iota
	// Query reads. It may also be sent as GET {prefix}/{Variant}?q=<payload>.
	Query
)

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

// ServeHTTP reads one Req from the body, in union form, and writes one Res.
// It is the single-union transport; Mount is the one with routes.
func (e *Endpoint[Req, Res]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, ok := e.readBody(w, r)
	if !ok {
		return
	}
	e.serveUnion(w, r, body)
}

// readBody reads the request body up to MaxBody, or answers 400/413 with
// DecodeError and reports false.
func (e *Endpoint[Req, Res]) readBody(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
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
		return nil, false
	}
	return body, true
}

// serveUnion decodes body as the Req union and serves it.
func (e *Endpoint[Req, Res]) serveUnion(w http.ResponseWriter, r *http.Request, body []byte) {
	req, err := shared.JSONUnmarshal[Req](body)
	if err == nil && isNil(req) {
		err = ErrEmptyBody
	}
	if err != nil {
		Write(w, http.StatusBadRequest, e.DecodeError(err))
		return
	}
	e.serve(w, r, req)
}

// serveVariant decodes body as the bare payload of the variant called name
// and serves it. An empty body is the zero variant, as a missing payload is
// in union form.
func (e *Endpoint[Req, Res]) serveVariant(w http.ResponseWriter, r *http.Request, name string, body []byte) {
	req, ok := e.ByName(name)
	if !ok {
		Write(w, http.StatusNotFound, e.DecodeError(fmt.Errorf("%w: %s", shared.ErrUnknownType, name)))
		return
	}
	if len(bytes.TrimSpace(body)) > 0 {
		if err := json.Unmarshal(body, req); err != nil {
			Write(w, http.StatusBadRequest, e.DecodeError(err))
			return
		}
	}
	e.serve(w, r, req)
}

// serve runs one decoded request and writes its answer. The *http.Request is
// in the context for RequestFrom; a Command with an Idempotency-Key is served
// from the replay cache when the key was seen.
func (e *Endpoint[Req, Res]) serve(w http.ResponseWriter, r *http.Request, req Req) {
	key := ""
	if e.Kind == Command {
		key = r.Header.Get("Idempotency-Key")
	}
	if key != "" {
		if e.replays.write(w, key) {
			return
		}
		rec := &recorder{header: http.Header{}}
		e.answer(rec, r, req)
		e.replays.store(key, rec)
		rec.replay(w)
		return
	}
	e.answer(w, r, req)
}

func (e *Endpoint[Req, Res]) answer(w http.ResponseWriter, r *http.Request, req Req) {
	ctx := context.WithValue(r.Context(), requestKey{}, r)
	res, err := e.Send(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	status := http.StatusOK
	if s, ok := any(res).(interface{ HTTPStatus() int }); ok {
		status = s.HTTPStatus()
	}
	Write(w, status, res)
}

type requestKey struct{}

// RequestFrom is the *http.Request an Endpoint is serving, for a handler that
// needs a header or the remote address; nil when the request did not come
// over HTTP (Send, or a test calling the handler directly).
func RequestFrom(ctx context.Context) *http.Request {
	r, _ := ctx.Value(requestKey{}).(*http.Request)
	return r
}

// HeadersFrom is the request headers, or an empty set off HTTP, so a handler
// can read a header without checking for nil.
func HeadersFrom(ctx context.Context) http.Header {
	if r := RequestFrom(ctx); r != nil {
		return r.Header
	}
	return http.Header{}
}

// recorder captures one answer so it can be sent twice: now, and to a repeat
// with the same Idempotency-Key.
type recorder struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (r *recorder) Header() http.Header         { return r.header }
func (r *recorder) WriteHeader(status int)      { r.status = status }
func (r *recorder) Write(b []byte) (int, error) { return r.body.Write(b) }

func (r *recorder) replay(w http.ResponseWriter) {
	for k, v := range r.header {
		w.Header()[k] = v
	}
	if r.status == 0 {
		r.status = http.StatusOK
	}
	w.WriteHeader(r.status)
	_, _ = w.Write(r.body.Bytes())
}

// maxReplays bounds the Idempotency-Key cache of one Endpoint.
const maxReplays = 1024

type replayCache struct {
	mu    sync.Mutex
	items map[string]*recorder
}

func (c *replayCache) store(key string, rec *recorder) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.items == nil || len(c.items) >= maxReplays {
		c.items = map[string]*recorder{}
	}
	c.items[key] = rec
}

// write replays the answer stored for key, with an Idempotent-Replayed
// header, and reports whether there was one.
func (c *replayCache) write(w http.ResponseWriter, key string) bool {
	c.mu.Lock()
	rec, ok := c.items[key]
	c.mu.Unlock()
	if !ok {
		return false
	}
	w.Header().Set("Idempotent-Replayed", "true")
	rec.replay(w)
	return true
}

// Mounted is an Endpoint seen without its type parameters, so Mount can take
// several of them. Only *Endpoint implements it.
type Mounted interface {
	kind() Kind
	names() []string
	owns(name string) bool
	read(w http.ResponseWriter, r *http.Request) ([]byte, bool)
	union(w http.ResponseWriter, r *http.Request, body []byte)
	variant(w http.ResponseWriter, r *http.Request, name string, body []byte)
}

func (e *Endpoint[Req, Res]) kind() Kind      { return e.Kind }
func (e *Endpoint[Req, Res]) names() []string { return e.Names }
func (e *Endpoint[Req, Res]) owns(name string) bool {
	if e.ByName == nil {
		return false
	}
	_, ok := e.ByName(name)
	return ok
}
func (e *Endpoint[Req, Res]) read(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	return e.readBody(w, r)
}
func (e *Endpoint[Req, Res]) union(w http.ResponseWriter, r *http.Request, body []byte) {
	e.serveUnion(w, r, body)
}
func (e *Endpoint[Req, Res]) variant(w http.ResponseWriter, r *http.Request, name string, body []byte) {
	e.serveVariant(w, r, name, body)
}

// Mount registers endpoints under prefix on mux, in both transport modes:
//
//	POST {prefix}            one union value; its $type picks the endpoint and the variant
//	POST {prefix}/{Variant}  the variant's bare payload; the URL picks the variant ("Search" or "series.Search")
//	GET  {prefix}/{Variant}  the same with the payload in ?q=, for Query endpoints; 405 for a Command
//
// Several endpoints share one prefix when a resource has more than one union
// (a query union and a command union); a body whose $type none of them owns is
// answered by the first, whose DecodeError says unknown type.
func Mount(mux *http.ServeMux, prefix string, endpoints ...Mounted) {
	if len(endpoints) == 0 {
		panic("rpc.Mount: no endpoints for " + prefix)
	}
	owner := func(name string) Mounted {
		for _, e := range endpoints {
			if e.owns(name) {
				return e
			}
		}
		return nil
	}
	mux.HandleFunc("POST "+prefix, func(w http.ResponseWriter, r *http.Request) {
		body, ok := endpoints[0].read(w, r)
		if !ok {
			return
		}
		var head struct {
			Type string `json:"$type"`
		}
		e := endpoints[0]
		if json.Unmarshal(body, &head) == nil && head.Type != "" {
			if o := owner(head.Type); o != nil {
				e = o
			}
		}
		e.union(w, r, body)
	})
	mux.HandleFunc(prefix+"/{variant}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("variant")
		e := owner(name)
		if e == nil {
			e = endpoints[0]
		}
		var body []byte
		switch r.Method {
		case http.MethodPost:
			var ok bool
			if body, ok = e.read(w, r); !ok {
				return
			}
		case http.MethodGet:
			if e.kind() != Query {
				w.Header().Set("Allow", http.MethodPost)
				http.Error(w, "a command is POST only", http.StatusMethodNotAllowed)
				return
			}
			body = []byte(r.URL.Query().Get("q"))
		default:
			w.Header().Set("Allow", "GET, POST")
			http.Error(w, r.Method+" is not allowed", http.StatusMethodNotAllowed)
			return
		}
		e.variant(w, r, name, body)
	})
}

// Names lists what every mounted endpoint accepts, in mount order: the
// catalogue a server can publish.
func Names(endpoints ...Mounted) []string {
	var out []string
	for _, e := range endpoints {
		out = append(out, e.names()...)
	}
	return out
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
		return post[Res](ctx, c, url, body, req, decorate)
	}
}

// post sends body to url and decodes the answer by the client status rule.
// req is only named in errors.
func post[Res any](ctx context.Context, c *http.Client, url string, body []byte, req any, decorate []func(*http.Request)) (Res, error) {
	var zero Res
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

// Bearer is a Client decorator that sets "Authorization: Bearer <token>".
func Bearer(token string) func(*http.Request) {
	return func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+token) }
}

// VariantClient is the transport for the per-variant routes Mount serves: it
// posts the variant's bare payload to prefix/{Variant} and decodes one Res,
// with the same status rule as Client. The variant name is read from the
// union encoding of req, so no name table is needed on the client.
func VariantClient[Req, Res any](c *http.Client, prefix string, decorate ...func(*http.Request)) func(ctx context.Context, req Req) (Res, error) {
	if c == nil {
		c = http.DefaultClient
	}
	return func(ctx context.Context, req Req) (Res, error) {
		var zero Res
		body, err := shared.JSONMarshal[Req](req)
		if err != nil {
			return zero, fmt.Errorf("rpc: encode %T: %w", req, err)
		}
		var envelope map[string]json.RawMessage
		if err := json.Unmarshal(body, &envelope); err != nil {
			return zero, fmt.Errorf("rpc: encode %T: %w", req, err)
		}
		var name string
		if err := json.Unmarshal(envelope["$type"], &name); err != nil || name == "" {
			return zero, fmt.Errorf("rpc: %T has no $type", req)
		}
		payload := envelope[name]
		if payload == nil {
			payload = json.RawMessage("{}")
		}
		return post[Res](ctx, c, prefix+"/"+name[strings.LastIndex(name, ".")+1:], payload, req, decorate)
	}
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
