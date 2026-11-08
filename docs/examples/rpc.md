# Typed handlers over HTTP: `x/rpc`

`x/rpc` moves one union pair over HTTP: the client sends one request variant, the server answers one answer variant, both in mkunion's JSON form. It is the wire for a union with [`f.Returns`](./typed_handler.md): the generated `DispatchX` is the server side, the generated `NewXClient` is the client side, and `x/rpc` connects them.

## Endpoint

An `Endpoint[Req, Res]` serves one request union. `Dispatch` runs a decoded request; `DecodeError` and `Error` turn a bad body or a `Dispatch` error into an answer variant, so an application error reaches the client as data, not as HTTP plumbing:

```go
ep := &rpc.Endpoint[series.SeriesQuery, series.SeriesAnswer]{
    Kind: rpc.Query,
    Dispatch: func(ctx context.Context, op series.SeriesQuery) (series.SeriesAnswer, error) {
        return series.DispatchSeriesQuery(ctx, handler, op)
    },
    DecodeError: series.Reject, // func(error) SeriesAnswer
    Error:       series.Fail,   // func(error) SeriesAnswer
    Names:       series.SeriesQueryNames(),
    ByName:      series.NewSeriesQueryByName,
}
mux := http.NewServeMux()
rpc.Mount(mux, "/v1/series", ep)
```

`ep.ServeHTTP` alone serves the union body at one URL. `ep.Send` is the same endpoint in-process, with no HTTP, so a test can swap the wire for the handler.

## Mount: two transport modes

`Mount` puts one or more endpoints under a prefix, and every route reaches the same `Dispatch`:

```
POST /v1/series          {"$type":"series.Search","series.Search":{"query":"tower"}}   the $type picks the variant
POST /v1/series/Search   {"query":"tower"}                                              the URL picks the variant
GET  /v1/series/Search?q={"query":"tower"}                                               Query endpoints only
```

The per-variant routes need the name table the generator emits for every handler union: `XNames()` lists the `$type` names in declaration order, and `NewXByName(name)` returns the zero variant for `"series.Search"` or `"Search"`. Without `Names` and `ByName` an endpoint serves only the union-body mode. A query union and a command union of one resource share a prefix; `rpc.Names(endpoints...)` is the catalogue of everything mounted.

## Kind

`Kind` says what the requests do to the server, so the transport can treat them differently without reading them:

- `rpc.Command` (the zero value) is POST only. A repeat with the same `Idempotency-Key` header gets the first answer back, marked `Idempotent-Replayed: true`, and the handler does not run twice.
- `rpc.Query` may also be sent as GET with the payload in `?q=`. GET on a Command is 405.

A handler that needs a header reads `rpc.HeadersFrom(ctx)`; `rpc.RequestFrom(ctx)` is the `*http.Request`, nil off HTTP.

## Client and Call

`rpc.Client[Req, Res](httpClient, url, decorate...)` posts the union body; `rpc.VariantClient` posts bare payloads to the per-variant routes. Either feeds the generated client, and `rpc.Call` gives the answer with the type the variant declared:

```go
client := series.NewSeriesQueryClient(rpc.Client[series.SeriesQuery, series.SeriesAnswer](nil, ts.URL+"/v1/series", rpc.Bearer(token)))
found, err := rpc.Call(ctx, client, &series.Search{Query: "tower"}) // found is *series.Found
```

The same `Call` works against the server-side handler, so a program written against the handler interface runs on either side.

## The HTTP status rule

- `200` for every answer, application errors included: an `Error` variant is an answer. An answer type that implements `HTTPStatus() int` picks its own 2xx, say `202` for an accepted job.
- `400` when the body is not a request at all (not JSON, no `$type`, a `$type` the union lacks, empty); the body is still a `DecodeError` answer, so a client newer than its server sees the error variant.
- `413` over `MaxBody` (1 MiB by default), `405` for GET on a Command, `500` only when the answer cannot be encoded.

The client follows the same rule: a 2xx or a 400 with a JSON body is an answer; anything else is a `*rpc.StatusError` with the status and body as sent. Authentication is not in the package: it is middleware in front of the endpoint, and a request decorator such as `rpc.Bearer` on the client.
