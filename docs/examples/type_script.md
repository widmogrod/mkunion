---
title: Exporting types to TypeScript
---

# Exporting types to TypeScript

`mkunion shape-export --language typescript` writes TypeScript for your Go
types, so a web front-end and a Go server share one definition of what goes
over the wire. The JSON is exactly what the generated Go serde reads and
writes ([Marshaling union as JSON](./json.md)), so nothing is translated on
the way.

```bash
mkunion shape-export --language typescript -o ../web/src/api api.go
```

Or, next to the union, so `go generate` keeps the TypeScript in step:

```go
//go:generate go tool mkunion shape-export --language typescript -o ../web/src/api api.go
```

The command writes one file per Go package, named after the package
(`api` → `api.ts`), plus `mkunion.ts` with what every file shares. The files
use only erasable syntax, so `tsc`, bundlers, Bun and Node's type stripping
all take them as they are. Relative imports have no extension
(`"./mkunion"`), which `moduleResolution: "bundler"` resolves; add
`allowImportingTsExtensions` or a bundler for `node16` resolution.

## What a package becomes

TypeScript has no decoder of its own: `JSON.parse` returns the value and the
types describe it. So every declaration says exactly what Go writes.

| Go | TypeScript |
|---|---|
| union `Request` | `type Request = { "$type": "api.Search"; "api.Search": Search } \| ...`, one object type per variant, both keys required |
| struct `Search` | `type Search = { query: string; ... }` |
| field `Query string \`json:"query"\`` | `query: string` |
| field `json:"query,omitempty"` | `query?: string`, and never `null`: Go leaves a nil out |
| field `json:"-"` or an embedded `f.Returns[R]` | left out |
| a struct with no fields left | `Record<string, never>`; Go writes `{}` |
| `*T` | `T \| null` |
| a union-typed field | `Union \| null`, since Go encodes a nil union as `null` |
| `[]T`, `[N]T` | `T[]`; `T[] \| null` when `encoding/json` writes the struct (see below) |
| `[]byte` | `mkunion.Base64 \| null`, a base64 string |
| `map[K]V` | `Record<string, V>`, which is what Go puts on the wire for any key type |
| `time.Time` | `mkunion.RFC3339`, a string with up to nine fractional digits; `new Date(s)` reads it |
| `time.Duration` | `number` |
| `any` | `unknown` |
| `type K string` | `type K = string` |
| a field with a `desc` tag | a `/** */` comment |

`$type` is required in the type because Go always writes it. The Go decoder
also accepts a missing `$type` (the first key that names a variant wins), and
a missing or `null` payload (the variant's zero value); the types do not
describe those forms, and a client that wants to read them normalises first.

Two encoders write Go structs and they disagree on nil slices and maps. The
generated serde, which writes every union variant, every struct that holds a
union and every type tagged `serde:"json"`, writes `[]` and `{}`; `encoding/json`,
which writes the remaining structs, writes `null`. The exporter follows the
encoder of each struct, so a slice is `T[]` in the first case and
`T[] | null` in the second. A hand-written `MarshalJSON` is not seen.

Every Go name is kept, `Error` included: a module is a namespace, so
`import * as Api from "./api"` reads as `Api.Error`, like `api.Error` in Go.
The generated types use two globals, `Record` and `Promise`; a Go type of
either name gets a trailing underscore. Another Go package is imported as a
namespace too (`import type * as other from "./other"`).

## Typed requests with `f.Returns`

When a variant embeds `f.Returns[R]` ([typed handlers](./typed_handler.md)),
the TypeScript knows the answer type too. For a union `Request` whose
variants answer with variants of `Response`:

```ts
export type RequestAnswer = { "api.GetInfo": Info; "api.Search": SearchResults; ... };
export type RequestPayload = { "api.GetInfo": GetInfo; "api.Search": Search; ... };
export type RequestOperation = keyof RequestAnswer;
export class RequestUnexpectedAnswer extends mkunion.UnexpectedAnswer<Response> {}

export function send<K extends RequestOperation>(
  type: K, payload: RequestPayload[K], via: (request: Request) => Promise<Response>,
): Promise<RequestAnswer[K]>;
```

`send` wraps the payload as the union, hands it to your transport, and
unwraps the answer the operation names. Any other variant, such as an error
variant, is thrown as `RequestUnexpectedAnswer` with the whole `Response` in
`answer`. The transport is yours: it does the HTTP, the typing is done.

```ts
import * as Api from "./api";

const results = await Api.send("api.Search", { query: "tower" }, async (request) => {
  const http = await fetch(endpoint, { method: "POST", body: JSON.stringify(request) });
  return (await http.json()) as Api.Response;
});
```

A wrong field name (`{ Query: "tower" }`), a missing required field, an
operation the union does not have, or reading a field the answer does not
have are all compile errors. When a package has several unions with
`f.Returns`, `send` has one overload per union.

## Not supported yet

- Inline struct fields (`Inner struct{...}`): name the type.
- Runtime validation: the types are declarations, and `(await http.json()) as
  Api.Response` is a cast. A generated type guard per union is the natural
  next step.
- A `null` payload or a missing `$type` on input, which Go reads as the zero
  value; the types describe what Go writes.
