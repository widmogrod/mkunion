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

## When to use it

| You have | Use |
|---|---|
| Variants that all answer with the same type, such as commands that answer with the next `State` | a plain union and `MatchXR1` (see [state machines](./state_machine.md)) |
| Variants that each answer with their own type: queries, requests, RPC | `f.Returns` on each: one interface per variant, typed answers |
| Operations a program asks for and something else performs | `f.Returns` plus the [effect system](./effect_system.md) |

A union with `f.Returns` must not have type parameters; `mkunion` says so if it does.

## Next steps

- **[Effects and unions](./effect_system.md)** - programs as data, built on typed handlers
- **[Marshaling union in JSON](./json.md)** - the JSON the query endpoint above relies on
- **[TypeScript](./type_script.md)** - the same union on the client
