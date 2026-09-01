---
title: Effect System (exploration)
---

This page explores an algebraic effect system built from mkunion unions.
The code lives in `example/effect`. It is an exploration, not a library: the
goal is to find out what unions, generics and code generation can and cannot
do for effects in Go.

## The idea

An effect system separates *what* a program asks for from *how* it is done.

- **Operations** are data: "read this file", "what time is it", "log this line".
- A **program** is data too: a chain of operations with continuations between them.
- A **handler** performs one operation and returns its answer.
- **Run** walks the program and calls the handler until the program is done.

The payoff: the same program runs against the real world in production and
against fixed data in tests, and every handler is checked by the compiler for
covering every operation.

## Operations are a union

Each variant declares its answer type with a phantom `Result` method. The method
is never called; it exists for the type checker.

```go title="example/effect/ops.go"
--8<-- "example/effect/ops.go:ops-def"
```

## Programs are a union

`Eff[Op, A]` is a program that performs operations of type `Op` and yields `A`.
It is generic over the operation union, so the core does not know about `Effect`.

```go title="example/effect/eff.go"
--8<-- "example/effect/eff.go:eff-def"
```

`Then` sequences programs. It is a pattern match over the three variants.

```go title="example/effect/eff.go"
--8<-- "example/effect/eff.go:then"
```

`Run` is a loop, not a recursion, so a program of a million steps does not
grow the Go stack. There is a test for that.

```go title="example/effect/eff.go"
--8<-- "example/effect/eff.go:run"
```

## The typed layer

The core moves answers around as `any`. The layer below ties each operation to
its answer type at compile time:

- `Perform(&Now{})` has type `Eff[Effect, time.Time]`. Go infers `R` from the `Result` method.
- `EffectHandler` has one typed method per operation.
- `HandlerOf` adapts it to the untyped `Handler`. `MatchEffectR2` makes it exhaustive,
  and the `answer` helper refuses to compile if a handler method returns the wrong type.

```go title="example/effect/ops.go"
--8<-- "example/effect/ops.go:typed-layer"
```

## Writing a program

```go title="example/effect/program.go"
--8<-- "example/effect/program.go:greet"
```

Recursion builds loops. The program below is lazy: each step is created inside
a continuation, so `Run` sees one `Bind` at a time.

```go title="example/effect/program.go"
--8<-- "example/effect/program.go:roll"
```

## Handlers

The same program, two meanings.

```go title="example/effect/handlers.go"
--8<-- "example/effect/handlers.go:live"
```

```go title="example/effect/handlers.go"
--8<-- "example/effect/handlers.go:fake"
```

Handlers are functions, so middleware is a function that returns a function.
`Trace` records every operation, in order, and is what the tests assert on.

## Findings

### What unions give us

1. **Exhaustive handlers.** `MatchEffectR2` takes one function per variant.
   Add an operation to `Effect` and every handler stops compiling until it
   handles the new case. This is the main win over a plain interface of methods.
2. **Programs as values.** `Eff` is a union, so a program can be inspected,
   traced, or replayed before or without running it. `Trace` shows the smallest
   version of this.
3. **Typed answers with no annotations.** The phantom `Result` method plus Go's
   inference from method sets means `Perform(&Now{})` knows it yields `time.Time`.

### What a derive-like generator could add

The typed layer in `ops.go` is mechanical. It follows from the union and the
`Result` methods. A tag such as `//go:tag mkeffect:"Effect"` could emit:

- `EffectOf[R]`, `Perform`, and `PerformDirect`,
- the `EffectHandler` interface with one typed method per variant,
- `HandlerOf` with the exhaustive match and the `answer` type check.

mkunion already has the plumbing for this. Custom tags are parsed and carried
into shapes for any tag name (`x/shape/tags.go`), and `mkmatch` shows the
pattern of a tag-driven generator (`x/generators/mkmatch_visitor.go`) hooked
through `InferredInfo.RunVisitorOnTaggedASTNodes`. The generated `EffectVisitor`
interface is almost the handler already; it lacks `context.Context`, `error`, and
typed answers.

### What Go 1.27 generic methods give us

Go 1.27 lets a method declare its own type parameters. The package
`example/effect/_go127` uses them. The directory name starts with an underscore,
so `go test ./...` and `mkunion watch ./...` skip it, and the module stays on its
current Go version. Run it explicitly:

```bash
GOTOOLCHAIN=go1.27.0 go test ./example/effect/_go127/
```

Why the underscore and not only a `//go:build go1.27` tag? mkunion parses source
with the `go/parser` of the Go version it was built with. A 1.26 parser rejects a
generic method with "method must have no type parameters" and the whole
`mkunion watch ./...` run fails. Keeping the file in a directory the tool skips
avoids that until the module moves to Go 1.27. (mkunion now skips underscore
directories in `./...` patterns, the same rule the go tool uses.)

Generic methods help in two places:

```go title="example/effect/_go127/effect127.go"
--8<-- "example/effect/_go127/effect127.go:program-127"
```

```go title="example/effect/_go127/effect127.go"
--8<-- "example/effect/_go127/effect127.go:direct-127"
```

Two limits stay:

- **Interfaces cannot have generic methods**, so a union (which is an interface)
  cannot carry `Then` itself. A concrete wrapper such as `Program` is needed, and
  the handler boundary stays `any`.
- **Data dependencies still nest.** Chaining flattens value transforms, but a step
  that needs two earlier values needs a nested closure, as `GreetChained` shows.
  Direct style (`GreetDirect`) removes nesting entirely, at the price of losing the
  program as a value.

So generic methods are ergonomics, not new power. Everything in the 1.27 file has
a package-level function equivalent that works today.

### What Go cannot express

- **Extensible effects.** There is no way to say "this program uses `Log` and
  `Clock` effects but not `FS`". Unions cannot nest another union as a variant,
  since mkunion has no support for interface-typed variants. A wrapper struct per
  sub-union is possible but is boilerplate, which is again generator territory.
- **Higher-kinded types.** `Eff` is generic over the operation type and the result
  type, but not over the "shape of computation". This is fine for an effect system;
  it means there is one `Eff`, not a family of monads.

### Generator bugs found on the way

The type registry generator produced code that does not compile for this package,
so the registry is disabled with `//go:tag mkunion:",no-type-registry"`:

1. It mistook the type parameter `Op` in `Trace[Op any](..., sink *[]Op)` for a
   package-level type and emitted `TypeRegistryStore[[]Op]`.
2. It emitted `JSONMarshallerRegister` calls for `Eff` although the union is
   tagged `noserde`, so the referenced `EffFromJSON` does not exist.

## Next steps

- **[Generic Unions](./generic_union.md)** - the mechanics `Eff[Op, A]` builds on
- **[Custom Pattern Matching](./custom_pattern_matching.md)** - the tag-driven generator pattern a derive step would follow
