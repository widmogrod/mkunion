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

### Third pass: methods, an alias, and defaults

`program_fx.go` is the same again with the most ergonomic surface we can build
by hand. The handle becomes a receiver, the return type gets a short name, and
a generic method (Go 1.27) covers operations without a helper:

```go title="example/effect/program_fx.go"
--8<-- "example/effect/program_fx.go:greet-fx"
```

```go title="example/effect/program_fx.go"
--8<-- "example/effect/program_fx.go:fx-api"
```

`Defaults` lets a test handler override one method and inherit the rest, while
the compiler still checks that the result is a complete `EffectHandler`:

```go title="example/effect/program_fx.go"
--8<-- "example/effect/program_fx.go:defaults"
```

Everything in this file follows from the union and the `Result` methods. It is
the concrete shape a `//go:tag mkeffect:"Effect"` generator would emit.

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

## Why not plain dependency injection?

The `Fx` surface looks like interfaces and structs, because it is. The difference
is under the surface. With dependency injection, `fx.ReadFile(path)` runs a
method and the call is gone. With effects, it builds a value, `&ReadFile{Path:
path}`, and every such value passes through one door, `Run`, in order, as data.
Everything below follows from that. Each item has a test in
`example/effect/benefits_test.go`.

1. **One trace of everything.** `Trace` lists every operation of every body, in
   order. With DI you would wrap each interface by hand.
2. **Middleware for all effects at once.** `Retry`, `Count` and `FailEvery` each
   wrap a `Handler` once and cover every operation of every program.

    ```go title="example/effect/middleware.go"
    --8<-- "example/effect/middleware.go:middleware"
    ```

3. **Record and replay.** `Record` writes a tape of operations and answers.
   `Replay` answers from the tape and checks that the program still asks for the
   same things. Run once against `Live`, replay with no clock, files or network,
   get the same result. A golden test for any body, and drift fails loudly.
4. **Pause and resume.** `Replay(tape, live)` serves the recorded steps and hands
   the rest to a live handler. The test crashes `GreetFx` after two steps, resumes,
   and shows that only the third step touched the world. This is how durable
   workflow engines survive a crash, and what `x/workflow` in this repo does by
   hand with its own AST.

    ```go title="example/effect/recording.go"
    --8<-- "example/effect/recording.go:recording"
    ```

5. **Fault injection.** `FailEvery` fails every nth operation of any kind. A
   clock that jumps is a handler that embeds `Defaults` and overrides `HandleNow`.
   Deterministic, in-process, no mocks per test.
6. **The algebra is a mkunion union.** `Effect` has JSON, so a tape has JSON:
   `TapeToJSON` and `TapeFromJSON` round-trip a recording, and the test replays
   from the JSON alone. The same union exports to TypeScript with
   `mkunion shape-export --language typescript -i example/effect/ops.go`, so a
   browser can read or build a tape:

    ```typescript
    export type Effect = {
        "$type"?: "effect.Log",
        "effect.Log": Log
    } | {
        "$type"?: "effect.Now",
        "effect.Now": Now
    } | {
        "$type"?: "effect.ReadFile",
        "effect.ReadFile": ReadFile
    } | {
        "$type"?: "effect.Random",
        "effect.Random": Random
    }
    ```

## Beyond the happy path

The six tests above show that the door exists. The tests in
`example/effect/advanced_test.go` and `example/effect/observe_test.go` show what
walks through it when things go wrong. They state every expected tape, trace,
span and mailbox in full, so the data shape is visible in the test.

The program under test is `Notify`. It reads a name, mails a greeting, and logs
the receipt. `Send` is not idempotent: sending twice sends two.

```go title="example/effect/program_fx.go"
--8<-- "example/effect/program_fx.go:notify"
```

Adding `Send` to the union forced every handler, policy table and decoder in the
package to be updated before it compiled again. Nothing was forgotten.

### Retries are a policy per operation

`RetryWith` asks a policy function for each operation. The policy is an exhaustive
match, so the compiler asks "may this be retried?" for every new operation. A
`RetryBudget` caps retries across the whole run. Sleep is injected, so the tests
assert the backoff durations instead of waiting for them.

```go title="example/effect/middleware.go"
--8<-- "example/effect/middleware.go:retry"
```

The test also shows that middleware order is semantics. `Record(Retry(h))`
writes one committed answer per step. `Retry(Record(h))` writes every attempt,
failures included. Only the first tape can be resumed from.

### At-least-once, and idempotency keys minted by the run

The nasty fault is not "the call failed". It is "the mail server delivered, then
the answer was lost". A retry sends the mail again. `LoseAnswerAt` injects
exactly that fault, and the test shows two mails in the mailbox.

The fix: `StepKeys` gives every step a stable key, `run-1/3`, and the handler
passes it to the mail server as an idempotency key. All retries of one step
share the key. On resume, the replayed steps still count, so step 3 keeps its
key. With keys the mailbox holds one mail.

```go title="example/effect/middleware.go"
--8<-- "example/effect/middleware.go:step-keys"
```

```go title="example/effect/handlers.go"
--8<-- "example/effect/handlers.go:mailbox"
```

### Crash at every step

The tape is finite, so the crash points can be searched completely. The test
crashes `Notify` after step 0, 1, 2, 3 and 4, resumes each time from the
recorded facts, and asserts the same receipt and exactly one mail. `Log` has no
key, so it is at-least-once. That is a choice the test points out.

### Fault tools

```go title="example/effect/middleware.go"
--8<-- "example/effect/middleware.go:faults"
```

### Trace diff, policy, and spans

A typed trace can do three things spans cannot.

- **Diff two runs.** `notifyV2` in the test adds a config read and moves the
  clock read. The output is identical, so an output test passes. `DiffTraces`
  shows the change as a diff of operations.
- **Enforce a policy.** `Guard` refuses operations before they run. The test
  shows a dry-run policy: reads are fine, secrets are refused, nothing leaves the
  process. An exhaustive match, so a new operation has to be classified.
- **Feed OpenTelemetry.** `Spans` emits one span per operation from one place,
  with start, end, attributes and error. The test asserts the exact spans.

```go title="example/effect/observe.go"
--8<-- "example/effect/observe.go:guard"
```

### Seeded chaos

`Chaos` injects refusals and lost answers at random from a seed. The test runs
five hundred worlds. Without keys, chaos finds double delivery in about one world
in seven. With keys, every world is exactly-once or a clean failure, and a failing
world replays from its seed alone. That is deterministic simulation testing in a
unit test.

When a program is "call three services and return" and you only need swap-for-tests,
plain DI is enough. Effects earn their cost when you need the trace, the replay, the
resume, or the same middleware on every call: workflow engines, sagas, simulation
testing, audit logs, dry-run modes.

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
and `example/effect` uses them for `Chain.Then`, `Direct.Perform` and `Fx.Do`.

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
  cannot carry `Then` itself. A concrete wrapper such as `Chain` is needed, and
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
