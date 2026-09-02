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

`Then` sequences programs. It is a pattern match over the variants.

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

## Direct style: the same program without the nesting

`Then` needs a callback for "what happens after the answer comes back", so
every step nests one level deeper. Go has no `do` notation. What Go does have,
since 1.23, is `iter.Pull`: a coroutine that can pause and resume.

`Proc` runs a plain Go body as a coroutine. Each `Do` pauses the body, hands the
operation to `Run` through an ordinary `Bind`, and resumes with the answer.

```go title="example/effect/program_do.go"
--8<-- "example/effect/program_do.go:greet-do"
```

```go title="example/effect/program_do.go"
--8<-- "example/effect/program_do.go:roll-do"
```

The program is still a value. Nothing runs before `Run`. Handlers, `Trace`,
`Then` and `Map` work unchanged, because `Proc` produces the same `Bind` chain
that `Perform` and `Then` produce, one step at a time. The tests check that
`GreetDo` and `Greet` leave the same trace.

`Do` unwinds the body on error, and the program fails with that error, the same
as a `Then` chain. `Attempt` returns the error instead, for bodies that want to
react in place:

```go title="example/effect/program_do.go"
--8<-- "example/effect/program_do.go:greet-or-guest"
```

The helpers are one line each and follow from the union, like the typed layer:

```go title="example/effect/program_do.go"
--8<-- "example/effect/program_do.go:do-helpers"
```

The core behind them:

```go title="example/effect/proc.go"
--8<-- "example/effect/proc.go:proc"
```

Costs: one coroutine per run (cheap, not a goroutine you schedule), and the
`e` parameter, because Go has no ambient context for the body to reach.
`Suspend` was added to `Eff` so that a `Proc` is built on demand and the body
does not start before `Run`.

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

Go 1.27 lets a method declare its own type parameters. The module is on Go 1.27,
and `example/effect` uses them for `Program.Then` and `Direct.Perform`.

One thing to know when a project moves to generic methods: mkunion parses source
with the `go/parser` of the Go version it runs under. A 1.26 parser rejects a
generic method with "method must have no type parameters" and the whole
`mkunion watch ./...` run fails. With `go 1.27` in `go.mod`, the go command picks
a 1.27 toolchain for `go tool mkunion` as well, so this just works.

Generic methods help in two places:

```go title="example/effect"
--8<-- "example/effect/eff.go:program-127"
```

```go title="example/effect"
--8<-- "example/effect/ops.go:direct-127"
```

Two limits stay:

- **Interfaces cannot have generic methods**, so a union (which is an interface)
  cannot carry `Then` itself. A concrete wrapper such as `Program` is needed, and
  the handler boundary stays `any`.
- **Data dependencies still nest.** Chaining flattens value transforms, but a step
  that needs two earlier values needs a nested closure, as `GreetChained` shows.
  `GreetDirect` removes nesting at the price of losing the program as a value.
  `Proc` (see "Direct style" above) removes nesting and keeps the value; it needs
  a coroutine, not generic methods.

So generic methods are ergonomics, not new power. Every generic method here has
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
