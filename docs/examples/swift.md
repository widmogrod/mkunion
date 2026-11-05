---
title: Exporting types to Swift
---

# Exporting types to Swift

`mkunion shape-export --language swift` writes Swift for your Go types, so an
iOS or macOS app and a Go server share one definition of what goes over the
wire. The JSON is exactly what the generated Go serde reads and writes
([Marshaling union as JSON](./json.md)), so nothing is translated on the way.

```bash
mkunion shape-export --language swift -o ../App/Sources/Api api.go
```

Or, next to the union, so `go generate` keeps the Swift in step:

```go
//go:generate go tool mkunion shape-export --language swift -o ../App/Sources/Api api.go
```

The command writes one file per Go package, named after the package
(`api` → `Api.swift`), plus `MkunionJSON.swift` with what every file shares.
Add the directory to a target of your Swift package; the files need only
`Foundation`.

## What a package becomes

Each Go package is a namespace enum, so `api.Search` in Go is `Api.Search`
in Swift and the names stay the same even when a Go type is called `Error`.

| Go | Swift |
|---|---|
| union `Request` | `enum Request: Codable, Hashable, Sendable`, one case per variant, lowerCamel: `case search(Search)` |
| struct `Search` | `struct Search: Codable, Hashable, Sendable` with `public var` fields and a memberwise `init` whose defaults are Go's zero values |
| field `Query string \`json:"query"\`` | `public var query: String`, coded as `"query"` |
| field `json:"-"` or an embedded `f.Returns[R]` | left out |
| `*T` | `T?` |
| a union-typed field | `Union?`, since Go encodes a nil union as `null` |
| `[]T`, `[N]T` | `[T]` |
| `[]byte` | `Data`, base64 on the wire |
| `map[K]V` | `[String: V]`, which is what Go puts on the wire for any key type |
| `time.Time` | `Date`, RFC 3339 with up to nine fractional digits |
| `any` | `JSONValue` |
| `type K string` | `typealias K = String` |

A union's `init(from:)` follows the Go decoder: `$type` picks the variant and
a missing or `null` payload is the variant's zero value; without `$type` the
first key that names a variant wins; no `$type` and no variant key throws
`MkunionJSON.Error.noType`, an unknown `$type` throws
`MkunionJSON.Error.unknownType`. `encode(to:)` always writes both keys.

A struct's `init(from:)` reads every field with `decodeIfPresent`, so a key
that Go left out with `omitempty`, or wrote as `null`, is the zero value in
Swift too. Unknown keys are ignored, so the server can add fields first.

Every union also gets `payload<T>()`, which returns the variant's value when
it is a `T`.

## Dates

Go writes `time.Time` as RFC 3339 with nanoseconds; `JSONDecoder`'s built-in
ISO 8601 strategy does not read that. Use the coders mkunion writes:

```swift
let decoder = MkunionJSON.decoder()
let encoder = MkunionJSON.encoder()
```

They read any number of fractional digits (keeping three), write three, and
know Go's zero time, `MkunionJSON.zeroDate`, which is what a missing date
decodes to.

## Typed requests with `f.Returns`

When a variant embeds `f.Returns[R]` ([typed handlers](./typed_handler.md)),
the Swift knows the answer type too. For a union `Request` whose variants
answer with variants of `Response`:

```swift
public protocol RequestOperation: Codable, Hashable, Sendable {
    associatedtype Answer: Codable
    var asRequest: Request { get }
}
extension Api.Search: Api.RequestOperation {}   // Answer = SearchResults

public static func send<Op: RequestOperation>(
    _ op: Op, via transport: (Request) async throws -> Response
) async throws -> Op.Answer
```

`send` wraps the operation as the union, hands it to your transport, and
unwraps the answer the operation names. Any other variant, such as an error
variant, is thrown as `RequestUnexpectedAnswer` with the whole `Response` in
it. The transport is yours: it does the HTTP, the typing is done.

```swift
let results = try await Api.send(Api.Search(query: "tower")) { request in
    var http = URLRequest(url: endpoint)
    http.httpMethod = "POST"
    http.httpBody = try MkunionJSON.encoder().encode(request)
    let (data, _) = try await URLSession.shared.data(for: http)
    return try MkunionJSON.decoder().decode(Api.Response.self, from: data)
}
```

## Not supported yet

- Types with type parameters: the export stops with an error that names the
  type.
- Inline struct fields (`Inner struct{...}`): name the type.
- A `null` element inside a list or map is a decoding error; the element
  types are not Optional.
