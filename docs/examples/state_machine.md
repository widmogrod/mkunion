---
title: State machines and unions
---
# MkUnion and state machines in golang

This document will show how to use `mkunion` to manage application state on example of an Order Service. 
You will learn:

- how to model state machines in golang, and find similarities to "__clean architecture__"
- How to **test state machines** (with fuzzing), and as a bonus you will get mermaid diagrams for free
- How to **persist state in database** and how optimistic concurrency helps __resolve concurrency conflicts__
- How to **handle errors** in state machines, and build foundations for __self-healing__ systems


## Working example

As an driving example, we will use e-commerce inspired Order Service that can be in one of the following states:

- `Pending` - order is created, and is waiting for someone to process it
- `Processing` - order is being processed, an human is going to pick up items from warehouse and pack them
- `Cancelled` - order was cancelled, there can be many reason, one of them is that warehouse is out of stock.
- `Completed` - order is completed, and can be shipped to customer.

Such states, have rules that govern **transitions**, like order cannot be cancelled if it's already completed, and so on.

And we need to have also to trigger changes in state, like create order that pending for processing, or cancel order. We will call those triggers **commands**.

Some of those rules could change in future, and we want to be able to change them without rewriting whole application.
This also informs us that our design should be open for extensions.

Side note, if you want go strait to final code product, then into [example/state/](example/state/) directory and have fun exploring.

## Modeling commands and states

Our example can be represented as state machine that looks like this:
[simple_machine_test.go.state_diagram.mmd](example/state/simple_machine_test.go.state_diagram.mmd)
```mermaid
--8<-- "example/state/machine_test.go.state_diagram.mmd"
```

In this diagram, we can see that we have 5 states, and 6 commands that can trigger transitions between states shown as arrows.

Because this diagram is generated from code, it has names that represent types in golang that we use in implementation. 

For example `*state.CreateOrderCMD`:

- `state` it's a package name
- `CreateOrderCMD` is a struct name in that package.
- `CMD` suffix it's naming convention, that it's optional, but I find it makes code more readable.


Below is a code snippet that demonstrate complete model of **state** and **commands** of Order Service, that we talked about.

**Notice** that we use `mkunion` to group commands and states. (Look for `//go:tag mkunion:"Command"`)

This is one example how union types can be used in golang. 
Historically in golang it would be very hard to achieve such thing, and it would require a lot of boilerplate code.
Here interface that group those types is generated automatically.

```go title="example/state/model.go"
--8<-- "example/state/model.go"
```

## Modeling transitions
One thing that is missing is implementation of transitions between states. 
There are few ways to do it. I will show you how to do it using functional approach (think  `reduce` or `map` function).

Let's name function that we will build `Transition` and define it as:

```go
func Transition(ctx context.Context, dep Dependencies, cmd Command, state State) (State, error)
```

Our function has few arguments, let's break them down:

- `ctx` standard golang context, that is used to pass deadlines, and cancelation signals, etc.
- `dep` encapsulates dependencies like API clients, database connection, configuration, context etc.
   everything that is needed for complete production implementation.
- `cmd` it's a command that we want to apply to state, 
   and it has `Command` interface, that was generate by `mkunion` when it was used to group commands.
- `state` it's a state that we want to apply our command to and change it, 
   and it has `State` interface, that was generate similarly to `Command` interface.


Our function must return either new state, or error when something went wrong during transition, like network error, or validation error.

Below is snippet of implementation of `Transition` function for our Order Service:

```go title="example/state/machine.go"
--8<-- "example/state/machine.go:30:81"
// ...
// rest remove for brevity 
// ...
```

You can notice few patterns in this snippet:

- `Dependency` interface help us to keep, well  dependencies - well defined, which helps greatly in testability and readability of the code. 
- Use of generated function `MatchCommandR2` to exhaustively match all commands. 
  This is powerful, when new command is added, you can be sure that you will get compile time error, if you don't handle it.
- Validation of commands in done in transition function. Current implementation is simple, but you can use go-validate to make it more robust, or refactor code and introduce domain helper functions or methods to the types.
- Each command check state to which is being applied using `switch` statement, it ignore states that it does not care about. 
  Which means as implementation you have to focus only on small bit of the picture, and not worry about rest of the states. 
  This is also example where non-exhaustive use of `switch` statement is welcome.

Simple, isn't it? Simplicity also comes from fact that we don't have to worry about marshalling/unmarshalling data, working with database, those are things that will be done in other parts of the application, keeping this part clean and focused on business logic.

Note: Implementation for educational purposes is kept in one big function, 
but for large projects it may be better to split it into smaller functions, 
or define OrderService struct that conforms to visitor pattern interface, that was also generated for you:

```go  title="example/state/model_union_gen.go"
--8<-- "example/state/model_union_gen.go:11:17"
```

## Testing state machines & self-documenting
Before we go further, let's talk about testing our implementation.

Testing will help us not only ensure that our implementation is correct, but also will help us to document our state machine, 
and discover transition that we didn't think about, that should or shouldn't be possible.

Here is how you can test state machine, in declarative way, using `mkunion/x/machine` package:

```go title="example/state/machine_test.go"
--8<-- "example/state/machine_test.go:15:151"
```
Few things to notice in this test:

- We use standard go testing
- We use `machine.NewTestSuite` as an standard way to test state machines
- We start with describing **happy path**, and use `suite.Case` to define test case.
- But most importantly, we define test cases using `GivenCommand` and `ThenState` functions, that help in making test more readable, and hopefully self-documenting.
- You can see use of `ForkCase` command, that allow you to take a definition of a state declared in `ThenState` command, and apply new command to it, and expect new state.
- Less visible is use of `moq` to generate `DependencyMock` for dependencies, but still important to write more concise code.

I know it's subjective, but I find it very readable, and easy to understand, even for non-programmers.

## Generating state diagram from tests
Last bit is this line at the bottom:

```go title="example/state/machine_test.go"
if suite.AssertSelfDocumentStateDiagram(t, "machine_test.go") {
   suite.SelfDocumentStateDiagram(t, "machine_test.go")
}
```

This code takes all inputs provided in test suit and fuzzy them, apply commands to random states, and records result of those transitions.

 - `SelfDocumentStateDiagram` - produce two `mermaid` diagrams, that show all possible transitions that are possible in our state machine.
 - `AssertSelfDocumentStateDiagram` can be used to compare new generated diagrams to diagrams committed in repository, and fail test if they are different.
   You don't have to use it, but it's good practice to ensure that your state machine is well tested and don't regress without you noticing.


There are two diagrams that are generated.

One is a diagram of ONLY successful transitions, that you saw at the beginning of this post.

```mermaid 
--8<-- "example/state/machine_test.go.state_diagram.mmd"
```

Second is a diagram that includes commands that resulted in an errors:
```mermaid 
--8<-- "example/state/machine_test.go.state_diagram_with_errors.mmd"
```

Those diagrams are stored in the same directory as test file, and are prefixed with name used in `AssertSelfDocumentStateDiagram` function.
```
machine_test.go.state_diagram.mmd
machine_test.go.state_diagram_with_errors.mmd
```

## State machines builder

MkUnion provide `*machine.Machine[Dependency, Command, State]` struct that wires Transition, dependencies and state together.
It provide methods like:

- `Handle(ctx context.Context, cmd C) error` that apply command to state, and return error if something went wrong during transition.
- `State() S` that return current state of the machine
- `Dep() D` that return dependencies that machine was build with.


This standard helps build on top of it, for example testing library that we use in [Testing state machines & self-documenting](#testing-state-machines-self-documenting) leverage it.

Another good practice is that every package that defines state machine in the way described here, 
should provide `NewMachine` function that will return bootstrapped machine with package types, like so:

```go title="example/state/machine.go"
--8<-- "example/state/machine.go:9:11"
```


