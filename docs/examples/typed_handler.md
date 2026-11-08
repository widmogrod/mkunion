---
title: Typed handlers
---
# Typed handlers: a union of questions, each with its own answer

This document will show what `mkunion` generates for a union whose variants say what they answer with, and how to use it, with the example of a user directory.
You will learn:

- how to declare a union where **each variant says what it answers with**
- what `mkunion` **generates** from that: one interface per variant, one for the union, and an exhaustive `HandleQuery` function
- how a query is **data**, so it can arrive as JSON and be dispatched in a few lines
- when to reach for this, and when a plain `Match` is enough

The [effect system](./effect_system.md) example builds on this page. You do not need it to use what is here.

## Working example

A user directory can answer three questions and take one order:

- `GetUser` - one user by ID, or nothing
- `FindUsers` - every user whose name starts with a prefix
- `CountUsers` - how many users there are
- `DeleteUser` - remove one user; there is nothing to answer with

The three answers have three different types: `*User`, `[]User`, `int`. That is the whole problem. A union gives one interface for the questions, but `MatchQueryR1` gives one `T` for all arms. Somewhere, the answer type has to be tied to the question.

## The union says what each variant answers with

```go title="example/query/query.go"
--8<-- "example/query/query.go:model"
```

**Notice** the embedded `f.Returns[...]`. It has no fields and carries no data. It is a label: `GetUser` answers with `*User`, `CountUsers` with `int`. Serde, TypeScript export and JSON schema skip it.

**Notice** there is no option in the tag. The label is the switch. A union with at least one `f.Returns` gets the typed layer below; a union with none is a plain union, with `Match` functions and JSON as usual.

**Notice** `DeleteUser` has no label. A variant without `f.Returns` is still handled, with an error-only method. It has no answer type, so it cannot be an operation a program asks for (the [effect system](./effect_system.md) says why). If you forget the label on a variant that should answer, its handler method has no place to return the value, and the compiler tells you.

## What is generated

Four things, from `example/query/query_union_gen.go`. Nothing has a default: a handler is complete or it does not compile.

```go title="example/query/query_union_gen.go (generated)"
// One interface per variant, so a handler can be assembled from parts.
type (
	QueryGetUserHandler    interface { HandleGetUser(ctx context.Context, op *GetUser) (*User, error) }
	QueryFindUsersHandler  interface { HandleFindUsers(ctx context.Context, op *FindUsers) ([]User, error) }
	QueryCountUsersHandler interface { HandleCountUsers(ctx context.Context, op *CountUsers) (int, error) }
	QueryDeleteUserHandler interface { HandleDeleteUser(ctx context.Context, op *DeleteUser) error }
)

// The whole union. Add a variant and every QueryHandler stops compiling.
type QueryHandler interface {
	QueryGetUserHandler
	QueryFindUsersHandler
	QueryCountUsersHandler
	QueryDeleteUserHandler
}

// The function form: MatchQueryR2 with one typed arm per variant, all required.
func HandleQuery(ctx context.Context, op Query,
	onGetUser func(ctx context.Context, op *GetUser) (*User, error),
	onFindUsers func(ctx context.Context, op *FindUsers) ([]User, error),
	onCountUsers func(ctx context.Context, op *CountUsers) (int, error),
	onDeleteUser func(ctx context.Context, op *DeleteUser) error,
) (any, error)

// On every variant with f.Returns: hand the query to any value that has its method.
func (r *GetUser) Perform(ctx context.Context, h any) (any, error)
```

Why an interface with one method per variant, and not one generic method? Go interfaces cannot carry generic methods. So the per-variant answer type lives where the compiler can read it: in each method's signature, and on the variant itself, through the `Ret() R` method that `f.Returns[R]` gives it. `HandleQuery` returns `any` because its arms do not share a type; the typed answer is on the method you call directly.

## Implement the handler

```go title="example/query/query.go"
--8<-- "example/query/query.go:in-memory"
```

**Notice** the `var _ QueryHandler = (*InMemory)(nil)` line. Add a query to the union, and this line is where the compiler tells you. Every handler, in every package, gets the same message.

## Ask, and get the type you declared

```go title="example/query/query.go"
--8<-- "example/query/query.go:ask"
```

```go title="example/query/query_test.go"
--8<-- "example/query/query_test.go:typed-answers"
```

No `any` at the call site. `R` is inferred from the query's `Ret() R`, and a query can only have one. `Perform` hands the query to its own typed method, so the cast inside `Ask` cannot fail.

## Handlers for tests

A test handler is a struct with the methods the test needs. The per-variant interfaces mean it can stop there: `Perform` asks only for the one method it needs, and says which one is missing, by name. `HandleQuery` is the other way: closures, all of them, no struct:

```go title="example/query/query_test.go"
--8<-- "example/query/query_test.go:test-handlers"
```

## Queries over the wire

A query is a value in a union, and the union has JSON generated by `mkunion`. So a query can arrive as bytes, be decoded into the right variant, and be dispatched with `HandleQuery` and the handler's own methods:

```go title="example/query/query_test.go"
--8<-- "example/query/query_test.go:over-the-wire"
```

**Notice** the last case. A query the union does not have is refused by the decoder, before any handler runs. The same union exports to TypeScript with `mkunion shape-export`, so a client can build these requests with the compiler's help.

## Refusals as values

A handler returns `(answer, error)`. The `error` is for failures: the index is down, the request did not decode. A refusal is different: "no such key" is a legitimate answer, the client should get it typed, and over the wire it rides in the same answer union as success. Two forms declare that; the fixtures in `x/generators/testutils/rpc.go` show both.

**A per-operation answer union** (recommended: it exports to Swift and TypeScript). The variant answers with a union, and `mkunion` accepts it when every leaf of that union is a variant of the `answers=` union:

```go title="x/generators/testutils/rpc.go"
//go:tag mkunion:"Inquiry,answers=Reply"
type (
	// Scan answers with any ScanAnswer variant.
	Scan struct{ f.Returns[ScanAnswer] }
)

//go:tag mkunion:"ScanAnswer"
type (
	ScanPage    = Page
	ScanMissing = Missing
)
```

The handler returns `(ScanAnswer, error)`. `DispatchInquiry` lifts a `ScanAnswer` onto `Reply` through the generated `MatchScanAnswerR1`, so the compiler checks every leaf; `NewInquiryClient` asserts the `Reply` back to a `ScanAnswer`, or answers with `InquiryUnexpectedAnswer`. In Swift, `send(op)` is typed as the operation's answer enum: `ScanAnswer` gets an `init?(_: Reply)` conformance, so a client switches over `.scanPage` and `.scanMissing` and nothing else.

**`f.Result[*V, E]`** (Go only today: the Swift exporter stops on a generic type argument). It reads as "answers `V`, refuses `E`", where `V` is a variant and `E` is a union of variants of the answer union:

```go title="x/generators/testutils/rpc.go"
	Lookup struct {
		f.Returns[f.Result[*Page, LookupRefusal]]
		Key string
	}
```

The handler returns `(f.Result[*Page, LookupRefusal], error)` and builds its answer with `f.MkOk[LookupRefusal](&Page{...})` or `f.MkErr[*Page, LookupRefusal](&Missing{...})`. `DispatchInquiry` unwraps `Ok` to its `Reply` variant and lifts `Err` through `MatchLookupRefusalR1`; `NewInquiryClient` rebuilds the `f.Ok` or `f.Err` from the `Reply` it receives. A `(nil, nil)` answer is an error for both forms: `Lookup answered with <nil>, want f.Result[*Page,LookupRefusal]`.

**The alias trick.** A struct is declared in one union block, so `Missing` cannot be listed under both `Reply` and `LookupRefusal`. A type alias makes it a variant of the second union too, and `mkunion` follows the alias to the struct when it checks that each leaf is a `Reply` variant:

```go title="x/generators/testutils/rpc.go"
//go:tag mkunion:"LookupRefusal"
type (
	LookupMissing = Missing
	LookupRefused = Refused
)
```

**Notice** the `$type` caveat. On the wire the variant name is the alias, so `LookupRefusalToJSON` writes `"$type": "testutils.LookupMissing"` where `ReplyToJSON` writes `"$type": "testutils.Missing"` for the same value. Over RPC this does not show: the answer travels as a `Reply`, and the lift and the rebuild happen on either side of the transport. Marshal a sub-union directly and the two spellings differ.

**Notice** the line between failure and refusal is drawn by what the unions share. The endpoint turns a Go error into `Refused`; `Refused` is also a `LookupRefusal` variant, so a `Lookup` client sees `f.Err{Refused}`, not an error. Keep the error variant out of the refusal union if a client must tell them apart.

## When to use it

| You have | Use |
|---|---|
| Variants that all answer with the same type, such as commands that answer with the next `State` | a plain union and `MatchXR1` (see [state machines](./state_machine.md)) |
| Variants that each answer with their own type: queries, requests, RPC | `f.Returns` on each: one interface per variant, typed answers |
| Operations a program asks for and something else performs | `f.Returns` plus the [effect system](./effect_system.md) |

A union with `f.Returns` must not have type parameters; `mkunion` says so if it does.

## Next steps

- **[Effects and unions](./effect_system.md)** - programs as data, built on typed handlers
- **[Typed handlers over HTTP](./rpc.md)** - `x/rpc`: Endpoint, Mount, Kind and Call for a union with `f.Returns`
- **[Marshaling union in JSON](./json.md)** - the JSON the query endpoint above relies on
- **[TypeScript](./type_script.md)** - the same union on the client
