---
title: Effects and unions
---
# Effects: programs as data, with mkunion

This document shows how to build a small effect system in Go with `mkunion`, using the example of a Welcome Mail service.
You will learn:

- how to model the **operations** a program may perform as a union, and write programs as plain Go against them
- how the same program runs against the **real world** in production and against a **fake** in tests
- how to **record a run as a tape**, replay it without any I/O, and store it as JSON
- how to handle **failure** like a staff engineer: retry policies per operation, idempotency keys, crash and resume, seeded chaos
- how a typed **trace** can be diffed, guarded by a policy, and turned into spans

Side note: if you want to go straight to the final code, then go into the [example/effect/](https://github.com/widmogrod/mkunion/tree/main/example/effect) directory. The files are numbered in reading order in `doc.go`, and every part has a test file that shows the behaviour with real data.

## Working example

The Welcome Mail service does one small job: read a customer's name from a file, mail them a greeting with the current time, and log the receipt.

Four things can happen in that job. We will call them **operations**:

- `ReadFile` - read the content of a file
- `Now` - ask for the current time
- `Send` - deliver a message. This one is dangerous: send it twice and the customer gets two mails
- `Log` - write a line somewhere

(There is a fifth, `Random`, which we use to show loops.)

The trick that this whole document builds on is simple. A program **asks** for these operations. It never performs them. Something else, called a **handler**, performs them. In production the handler talks to the file system and the mail server. In a test it answers from a map. In a replay it answers from a tape.

## Part 1: the basics

### Operations are a union

Each operation is a struct in a `type (...)` block tagged with `mkunion`, exactly like a state or a command in the [state machine](./state_machine.md) example.

```go title="example/effect/ops.go"
--8<-- "example/effect/ops.go:ops-def"
```

**Notice** the `Result` methods. They are never called. Each one tells the compiler what type of answer an operation gets back: a `ReadFile` answers with `[]byte`, a `Send` answers with a receipt `string`. Go's type inference reads these methods, so later you can write `fx.Do(&Now{})` and get a `time.Time` without spelling it out.

### Programs are plain Go

A program is a function that gets a handle, `Fx`, and calls methods on it. It reads like any Go code:

```go title="example/effect/program.go"
--8<-- "example/effect/program.go:notify"
```

There is one thing to keep in mind. Calling `Notify("name.txt", to)` performs nothing. It returns a `Program[string]`, a value. The body runs later, when you hand the value to `Run` together with a handler. Part 2 explains how that works. For now, the surface:

```go title="example/effect/program.go"
--8<-- "example/effect/program.go:fx-api"
```

**Notice** that `Fx` has one method per operation, plus a generic `Do` and `Attempt`. `Do` stops the body when an operation fails, and the program fails with that error. `Attempt` hands the error to the body instead, for the cases where the body knows what to do:

```go title="example/effect/program.go"
--8<-- "example/effect/program.go:greet-or-guest"
```

Loops are loops. A program that performs a million operations is a million steps in `Run`, not a million stack frames:

```go title="example/effect/program.go"
--8<-- "example/effect/program.go:roll"
```

### Handlers give the program meaning

A handler answers each operation with its declared type. `EffectHandler` is that contract, and `HandlerOf` turns it into the function the core runs:

```go title="example/effect/ops.go"
--8<-- "example/effect/ops.go:typed-layer"
```

**Notice** two compile-time checks in `HandlerOf`. `MatchEffectR2` is exhaustive, so adding an operation to the union breaks every handler until it handles the new case. The little `answer` helper refuses to compile when a handler method returns a type other than the one the operation declared. That is the union doing its job. When `Send` was added to this example, the compiler pointed at every handler, policy and decoder that needed a decision.

For production there is `Live`:

```go title="example/effect/handlers.go"
--8<-- "example/effect/handlers.go:live"
```

For tests there is `Fake`, which answers from fixed data and remembers what was logged and sent:

```go title="example/effect/handlers.go"
--8<-- "example/effect/handlers.go:fake"
```

### Run it

```go
fake := &Fake{Clock: noon, Files: map[string]string{"name.txt": "Ada\n"}}
got, err := Run(ctx, HandlerOf(fake), Greet("name.txt"))
// got == "Hello Ada, it is 12:00PM"
// fake.Logs == []string{"Hello Ada, it is 12:00PM"}
```

Swap `HandlerOf(fake)` for `HandlerOf(live)` and the same program writes a real log line. The tests in `example/effect/part1_basics_test.go` do both.

### Testing on day one

Two habits pay off from the first test.

First, assert on the **trace**. `Trace` is a wrapper around any handler that records every operation, in order, as data:

```go title="example/effect/eff.go"
--8<-- "example/effect/eff.go:trace"
```

```go
var trace []Effect
got, err := Run(ctx, Trace(HandlerOf(fake), &trace), Greet("name.txt"))
// trace == []Effect{&ReadFile{Path: "name.txt"}, &Now{}, &Log{Msg: "Hello Ada, it is 12:00PM"}}
```

An output test says what came out. A trace test says what the program did to get there. Part 5 shows what else a trace can do.

Second, override one method at a time. `Defaults` is a handler with harmless answers. Embed it and override only what the test cares about. The compiler still checks that the result is a complete handler:

```go title="example/effect/handlers.go"
--8<-- "example/effect/handlers.go:defaults"
```

```go
type clockOnly struct {
	Defaults
	at time.Time
}

func (c clockOnly) HandleNow(context.Context, *Now) (time.Time, error) { return c.at, nil }
```

**Notice** that `Defaults` refuses to read files. A test cannot depend on a file by accident.

## Part 2: under the hood

You can use everything in Part 1 without reading this part. Read it when you want to know why this is not just dependency injection.

### A program is a union too

`Program[A]` is a short name for `Eff[Effect, A]`, and `Eff` is a generic union:

```go title="example/effect/eff.go"
--8<-- "example/effect/eff.go:eff-def"
```

A program is either finished (`Pure` with a value, `Fail` with an error), or it asks for one operation and says what to do with the answer (`Bind`), or it is built on demand (`Suspend`). That is the whole data model.

`Then` glues two programs together. It is a pattern match over the variants, so a new variant cannot be forgotten:

```go title="example/effect/eff.go"
--8<-- "example/effect/eff.go:then"
```

With `Perform` and `Then` you can build a program by hand. This is what `Fx` builds for you:

```go title="example/effect/part2_under_the_hood_test.go"
--8<-- "example/effect/part2_under_the_hood_test.go:greet-then"
```

The tests run every scenario against both versions of `Greet` and assert the same trace. Same data, two ways to write it.

| Style | Write it when | Trade |
|---|---|---|
| `Prog` + `Fx` (default) | Always, for business logic. It reads like Go. | Needs a coroutine per run. |
| `Perform` + `Then` | When you build programs from data, or write combinators such as `Map`. | Nests one level per step. |

### Run is a loop

```go title="example/effect/eff.go"
--8<-- "example/effect/eff.go:run"
```

**Notice** that `Run` never recurses. It takes the next node, asks the handler, and moves on. That is why a million-step program does not grow the stack. Also notice what happens on error: the error goes to the continuation, the same as an answer. That is how a plain Go body gets to unwind, and how `Attempt` gets to see the error.

### Proc is a coroutine

The last piece. How does a plain Go body become `Bind` nodes, one at a time? With `iter.Pull`, which Go has had since 1.23. It turns a function into a coroutine that can pause and resume.

```go title="example/effect/proc.go"
--8<-- "example/effect/proc.go:proc"
```

Every `Do` pauses the body and hands the operation out as a `Bind`. `Run` asks the handler, and the answer resumes the body. The body does not start before `Run`, because `Proc` returns a `Suspend`. The tests check that the body stays lazy, that a panic in the body is not swallowed, and that the coroutine is freed when a handler fails or the context is cancelled.

## Part 3: testing with tapes

From here on the program under test is `Notify`, whose `Send` must never happen twice.

### Record once, replay forever

Because every operation and every answer is data, a run can be written down. `Record` writes the tape. `Replay` answers from it and checks that the program still asks for the same things:

```go title="example/effect/recording.go"
--8<-- "example/effect/recording.go:recording"
```

```go
var tape []Step[Effect]
want, _ := Run(ctx, Record(HandlerOf(live), &tape), Notify("name.txt", to))
// tape == []Step[Effect]{
//     {Op: &ReadFile{Path: "name.txt"}, Answer: []byte("Ada\n")},
//     {Op: &Now{}, Answer: noon},
//     {Op: &Send{To: to, Msg: greeting}, Answer: "receipt-1"},
//     {Op: &Log{Msg: "sent receipt-1"}, Answer: Unit{}},
// }

got, _ := Run(ctx, Replay(tape, nil), Notify("name.txt", to))   // no clock, files or mail server
// got == want
```

A tape is a golden test that you did not have to write, and it guards against drift: a program that asks for a different file fails with "replay mismatch at step 1".

### A tape is JSON

The `Effect` union has JSON, generated by `mkunion`, so a tape has JSON too. Answers are decoded into the type each operation declared:

```go title="example/effect/recording_json.go"
--8<-- "example/effect/recording_json.go:tape-json"
```

The test in `example/effect/part3_testing_test.go` shows the exact JSON and replays from it alone. The same union exports to TypeScript with `mkunion shape-export --language typescript -i example/effect/ops.go`, so a browser can read or build a tape:

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
} | {
    "$type"?: "effect.Send",
    "effect.Send": Send
}
```

## Part 4: failure is normal

Handlers are functions. So middleware is a function that takes a handler and returns a handler. It wraps the handler once and sees every operation of every program. Every tool in this part is such a function, and each one is under twenty lines.

### Retry is a policy per operation, not one number

Reading a file may be retried. Sending a mail may not, unless you can prove it is safe. `RetryWith` asks a policy for each operation. The policy is an exhaustive match, so the compiler asks "may this be retried?" for every new operation. A `RetryBudget` caps retries across the whole run, and sleep is injected so tests can assert the backoff durations instead of waiting for them.

```go title="example/effect/middleware.go"
--8<-- "example/effect/middleware.go:retry"
```

```go
func strictPolicy(op Effect) RetryPolicy {
	return MatchEffectR1(op,
		func(*Log) RetryPolicy      { return RetryPolicy{Attempts: 2} },
		func(*Now) RetryPolicy      { return RetryPolicy{Attempts: 2} },
		func(*ReadFile) RetryPolicy { return RetryPolicy{Attempts: 3, Backoff: ExponentialBackoff(10 * time.Millisecond)} },
		func(*Random) RetryPolicy   { return RetryPolicy{Attempts: 2} },
		func(*Send) RetryPolicy     { return RetryPolicy{Attempts: 1} }, // not idempotent
	)
}
```

**Notice** that middleware order is semantics. `Record(Retry(h))` writes one committed answer per step. `Retry(Record(h))` writes every attempt, failures included. Only the first tape can be replayed. The test shows both tapes side by side.

### At-least-once, and idempotency keys

The nasty fault is not "the call failed". It is "the mail server delivered, then the answer was lost". A retry sends the mail again. `LoseAnswerAt` injects exactly that fault, and the test shows two identical mails in the mailbox.

The fix comes from the run itself. `StepKeys` gives every step a stable key, `run-1/3`. The handler passes it to the mail server as an idempotency key. All retries of one step share the key. On resume, replayed steps still count, so step 3 keeps its key.

```go title="example/effect/middleware.go"
--8<-- "example/effect/middleware.go:step-keys"
```

```go title="example/effect/handlers.go"
--8<-- "example/effect/handlers.go:mailbox"
```

With keys the mailbox holds one mail, and the retry receives the receipt of the first delivery. This is how durable workflow engines make activities safe.

### Crash at every step, then resume

`Replay(tape, live)` answers the recorded steps from the tape and hands everything after them to the live handler. That is resume after a crash. And because the tape is finite, the crash points can be searched completely: crash after step 0, 1, 2, 3 and 4, resume each time, and assert the same receipt and exactly one mail.

```go
first := StepKeys(Record(CrashAfter(HandlerOf(live), k, crash), &tape), "run-1")
_, err := Run(ctx, first, Notify("name.txt", to))             // dies after k steps
facts := tape[:k]                                            // the crashed step is not a fact

second := StepKeys(Replay(facts, HandlerOf(live)), "run-1")
got, err := Run(ctx, second, Notify("name.txt", to))         // replays k steps, performs the rest
// got == "receipt-1"; mail.Sent has one entry with key "run-1/3"
```

**Notice** the choice the test points out: `Log` has no key, so it is at-least-once. Not every operation needs exactly-once, and the policy is written down per operation.

### Fault tools and seeded chaos

```go title="example/effect/middleware.go"
--8<-- "example/effect/middleware.go:faults"
```

`Chaos` injects refusals and lost answers at random from a seed. The test runs five hundred worlds. Without keys, chaos finds double delivery in about one world in seven. With keys, every world is exactly-once or a clean failure, and a failing world replays from its seed alone. That is deterministic simulation testing in a unit test.

## Part 5: seeing what happened

A trace of typed operations can do three things log lines and spans cannot.

**Diff two runs.** The test has a "version two" of `Notify` that adds a config read and moves the clock read. The output is identical, so an output test passes. The trace diff shows the change:

```
+ &effect.ReadFile{Path:"config.txt"}
+ &effect.Now{}
  &effect.ReadFile{Path:"name.txt"}
- &effect.Now{}
  &effect.Send{To:"ada@example.com", Msg:"Hello Ada, it is 12:00PM"}
  &effect.Log{Msg:"sent receipt-1"}
```

**Enforce a policy.** `Guard` refuses operations before they run. Authorization and dry-run are the same function: an exhaustive match, so a new operation has to be classified.

```go title="example/effect/observe.go"
--8<-- "example/effect/observe.go:guard"
```

**Feed your tracing backend.** `Spans` emits one span per operation, with start, end, attributes and error, from one place. With dependency injection you would instrument each client.

```go title="example/effect/observe.go"
--8<-- "example/effect/observe.go:spans"
```

## When to use this, and when not

The `Fx` surface looks like interfaces and structs, because it is. The difference is under the surface. With dependency injection, `fx.ReadFile(path)` runs a method and the call is gone. With effects, it builds a value and that value passes through one door, `Run`, in order, as data. Everything in Parts 3 to 5 follows from that one door.

When a program is "call three services and return" and you only need swap-for-tests, plain dependency injection is enough, and cheaper. Effects earn their cost when you need the tape, the resume, the same middleware on every call, or a trace you can diff: workflow engines, sagas, simulation testing, audit logs, dry-run modes. If you never need those, delete Parts 2 to 5 and keep the interfaces. Nothing in Part 1 has to change.

## Notes for contributors

A few things learned while building this, for whoever extends it.

- **Go 1.27 generic methods** make `fx.Do(&Now{})` possible: a method with its own type parameter, with `R` inferred from the operation. Interfaces still cannot carry generic methods, so the union interface `Eff` cannot have a `Then` method, and the handler boundary stays `any` with typed wrappers at the edges.
- **The typed layer is mechanical.** `EffectOf`, `Perform`, `EffectHandler`, `HandlerOf`, the `Fx` methods, `Defaults` and `answerFromJSON` all follow from the union and the `Result` methods. A `//go:tag mkeffect:"Effect"` generator could emit them. `mkunion` already carries custom tags into shapes and has the `mkmatch` pattern for a tag-driven generator.
- **Extensible effects are not expressible.** Go cannot say "this program uses `Log` and `Now` but not `Send`" as a type built on the fly. Two workable models: small unions wrapped into one app union with a lift function, or capability interfaces on `Fx` (`interface{ Clock; FS }`) with one app union underneath. Neither is in this example yet.
- **Generator bugs found on the way.** The type registry generator produced code that does not compile for this package, so the registry is off with `//go:tag mkunion:",no-type-registry"`. It mistook the type parameter `Op` in `Trace[Op any](..., sink *[]Op)` for a package type, and it ignored `noserde` on `Eff`.

## Next steps

- **[State Machines](./state_machine.md)** - the other way this repository turns behaviour into data
- **[Generic Unions](./generic_union.md)** - the mechanics `Eff[Op, A]` builds on
- **[Custom Pattern Matching](./custom_pattern_matching.md)** - the tag-driven generator pattern a `mkeffect` step would follow
