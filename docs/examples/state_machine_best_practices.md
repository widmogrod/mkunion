---
title: State Machine Best Practices
---

# State Machine Best Practices

This guide covers best practices, patterns, and techniques for building robust state machines with mkunion. Whether you're building simple state machines or complex distributed systems, these practices will help you create maintainable and scalable solutions.

## Best Practices

When building state machines with mkunion, following these practices will help you create maintainable and robust systems:

### File Organization

Organize your state machine code across files for better maintainability:

1. **`model.go`**: State and command definitions
   ```go title="example/state/model.go"
   --8<-- "example/state/model.go:commands"
   --8<-- "example/state/model.go:states"
   ```

2. **`machine.go`**: Core state machine logic
   ```go title="example/state/machine.go"
   --8<-- "example/state/machine.go:dependency"
   --8<-- "example/state/machine.go:new-machine"
   --8<-- "example/state/machine.go:transition-fragment"
   // ... and so on
   ```

3. **`machine_test.go`**: Tests and state diagrams
4. **`machine_database_test.go`**: Persistence examples
5. **Generated files** (created by `go generate`):
   - `*_union_gen.go` - Union type definitions from mkunion
   - `*_shape_gen.go` - Shape definitions for introspection
   - `machine_mock.go` - Mock implementation of Dependency interface from moq

### Naming Conventions

1. **States**: Use descriptive nouns that clearly indicate the state (e.g., `OrderPending`, `PaymentProcessing`)
2. **Commands**: Suffix with `CMD` for clarity (e.g., `CreateOrderCMD`, `CancelOrderCMD`)
3. **Packages**: Keep state machines in dedicated packages named after the domain (e.g., `order`, `payment`)

### State Design

1. **Keep States Focused**: Each state should represent one clear condition
2. **Immutable Data**: States should contain immutable data; create new states instead of modifying
3. **Minimal State Data**: Only store data that's essential for the state's identity
4. **Use Zero Values**: Design states so Go's zero values are meaningful defaults

### Command Validation

Centralizing validation in the Transition function provides significant benefits:

1. **Single source of truth**: All business rules and validation logic live in one place
2. **Atomic validation**: Commands are validated together with state checks, preventing invalid transitions
3. **Testability**: Easy to test all validation rules through the state machine tests
4. **Maintainability**: When rules change, you only update one location

#### Basic Validation

```go title="example/state/machine.go"
--8<-- "example/state/machine.go:create-order"
```

#### Advanced Validation with go-validate

For complex validation requirements demonstrate

- Structural validation is declarative (struct tags)
- Business rules are explicit and testable
- External validations are isolated in dependencies
- State validations ensure valid transitions
- All validation happens before any state change:

```go title="example/state/machine.go"
--8<-- "example/state/machine.go:advanced-handling"
```

This approach scales well because of separation of state from IO and business logic.

### Dependency Management

1. **Define Clear Interfaces**: Dependencies should be interfaces, not concrete types
2. **Keep Dependencies Minimal**: Only inject what's absolutely necessary
3. **Generate Mocks with moq**: Use `//go:generate moq` to automatically generate mocks

```go title="example/state/machine.go"
--8<-- "example/state/machine.go:dependency"
```

Running `go generate ./...` creates `machine_mock.go` with a `DependencyMock` type. This mock can then be used in tests:

```go title="example/state/machine_test.go"
--8<-- "example/state/machine_test.go:test-suite-example"
```

Benefits of using moq:

- **Reduces boilerplate**: No need to manually write mock implementations
- **Type safety**: Generated mocks always match the interface
- **Easy maintenance**: Mocks automatically update when interface changes
- **Better test readability**: Focus on behavior, not mock implementation

## Common Pitfalls

Avoid these common mistakes when implementing state machines:

### 1. State Explosion

**Problem**: Creating too many states for every minor variation
```go
// Bad: Too granular
type (
    OrderPendingWithOneItem struct{}
    OrderPendingWithTwoItems struct{}
    OrderPendingWithThreeItems struct{}
    // ... and so on
)
```

**Solution**: Use state data instead
```go
// Good: Single state with data
type OrderPending struct {
    Items []OrderItem
}
```

### 2. Circular Dependencies

**Problem**: States that can transition in circles without progress
```go
// Problematic: A -> B -> C -> A without any business value
```

**Solution**: Ensure each transition represents meaningful progress or explicitly document allowed cycles

### 3. Missing Error States

**Problem**: Not modeling error conditions as explicit states
```go
// Bad: Errors only in transition function
return nil, fmt.Errorf("payment failed")
```

**Solution**: Model error conditions as states when they need handling. Crucially, store both the command that failed and the previous valid state to enable recovery or debugging:

```go title="example/state/model.go"
--8<-- "example/state/model.go:states"
```

The error state pattern enables recovery:

```go title="example/state/machine.go"
--8<-- "example/state/machine.go:transition"
```

For a more advanced pattern with BaseState:

```go title="example/state/model_with_base.go"
--8<-- "example/state/model_with_base.go:base-state-pattern"
```

```go title="example/state/model_with_base.go"
--8<-- "example/state/model_with_base.go:states-with-base"
```

This approach preserves critical information needed for recovery without losing the context of what failed.

### 4. Ignoring Concurrency

**Problem**: Misunderstanding the state machine concurrency model
```go
// Wrong: Sharing a machine instance across goroutines
sharedMachine := NewMachine(deps, currentState)
go sharedMachine.Handle(ctx, cmd1) // Goroutine 1
go sharedMachine.Handle(ctx, cmd2) // Goroutine 2 - DON'T DO THIS!
```

**Solution**: State machines are designed to be created per request. This isolation prevents accidental state mutations:

```go
// Correct: Create a new machine instance for each operation
func ProcessCommand(ctx context.Context, deps Dependencies, cmd Command) error {
    // 1. Load current state from storage
    record, err := repo.Get(ctx, orderID)
    if err != nil {
        return err
    }
    
    // 2. Create a fresh machine instance with the current state
    machine := NewMachine(deps, record.State)
    
    // 3. Handle the command
    if err := machine.Handle(ctx, cmd); err != nil {
        return err
    }
    
    // 4. Save the new state (with optimistic concurrency control)
    record.Data = machine.State()
    return repo.Update(ctx, record)
}
```

This pattern ensures:

- Each request gets an isolated machine instance
- No shared mutable state between concurrent operations
- Failures don't affect other requests
- Retries start with a clean machine instance

For handling concurrent updates to the same entity, see the [Optimistic Concurrency Control](#optimistic-concurrency-control) section below.

### 5. Overloading Transitions

**Problem**: Putting too much business logic in transition functions
```go
// Bad: Transition function doing too much
func Transition(...) (State, error) {
    // Send emails
    // Update inventory
    // Calculate prices
    // Log to external systems
    // ... 200 lines later
}
```

**Solution**: Keep transitions focused on state changes; delegate side effects to dependencies

## Advanced Patterns

### State Machine Composition

For complex systems, compose multiple state machines as a service layer:

```go title="example/state/service.go"
--8<-- "example/state/service.go:service-definition"
```

```go title="example/state/service.go"
--8<-- "example/state/service.go:handle-command"
```

```go title="example/state/service.go"
--8<-- "example/state/service.go:service-composition"
```

Key principles:

- **Domain services**: Each domain encapsulates its repository, dependencies, and machine logic
- **Schemaless repositories**: Use `schemaless.Repository[StateType]` for type-safe state storage
- **Service composition**: Compose domain services, avoiding direct repository/machine access
- **Single responsibility**: Each service handles one domain's state machine lifecycle
- **Optimistic concurrency**: Built-in through `schemaless.Repository` version handling
- **No duplication**: State loading, machine creation, and saving logic exists once per domain

### Async Operations with Callbacks

Handle long-running operations without blocking using a state-first approach:

```go title="example/state/async_operations.go"
--8<-- "example/state/async_operations.go:async-states"
```

```go title="example/state/async_operations.go"
--8<-- "example/state/async_operations.go:async-commands"
```

```go title="example/state/async_operations.go"
--8<-- "example/state/async_operations.go:async-transition"
```

```go title="example/state/async_operations.go"
--8<-- "example/state/async_operations.go:async-service"
```

```go title="example/state/async_operations.go"
--8<-- "example/state/async_operations.go:background-processor"
```

**Key principles:**
- **Pure transitions**: No side effects in transition functions
- **State-first persistence**: Save state before triggering async work
- **Background processing**: Separate system handles async operations
- **Callback mechanism**: Async completion creates new commands
- **Timeout handling**: Built into state for automatic cleanup
- **No race conditions**: State is always consistent with async operation status

### Time-Based Transitions

Handle timeouts properly based on the operation context:

**Request-scoped operations**: `machine.Handle(ctx, cmd)` and the `Transition` function respect context cancellation for standard Go timeout handling.

**Long-running process timeouts**: Model timeouts as explicit states for operations that exceed request boundaries:

```go
//go:tag mkunion:"State"
type (
    // States that can timeout should track when timeout expires
    AwaitingApproval struct {
        OrderID              string
        ExpectedTimeoutAt    time.Time  // When this will timeout
        BaseState
    }
    
    // Use Error state with standardized timeout code
    ProcessError struct {
        Code          string  // "TIMEOUT", "API_ERROR", etc.
        Reason        string
        FailedCommand Command
        PreviousState State
        RetryCount    int
        BaseState
    }
)

// Background process finds and transitions timed-out states
func ProcessTimeouts(ctx context.Context, repo Repository) error {
    // Find all states that should timeout
    records, err := repo.FindRecords(
        predicate.Where(`
            Data["order.AwaitingApproval"].ExpectedTimeoutAt < :now
        `, predicate.ParamBinds{
            ":now": schema.MkInt(time.Now().Unix()),
        }),
    )
    
    for _, record := range records.Items {
        machine := NewMachine(deps, record.Data)
        err := machine.Handle(ctx, &ExpireTimeoutCMD{
            RunID: record.ID,
        })
        // Save updated state...
    }
}
```

#### 3. Benefits of State-Based Timeouts

This approach enables powerful querying and recovery:

```go
// Find all timeout errors for retry
timeoutErrors, _ := repo.FindRecords(
    predicate.Where(`Data["order.ProcessError"].Code = :code`, 
        predicate.ParamBinds{":code": schema.MkString("TIMEOUT")},
    ),
)

// Find long-waiting approvals
longWaiting, _ := repo.FindRecords(
    predicate.Where(`
        Type = :type 
        AND Data["order.AwaitingApproval"].ExpectedTimeoutAt > :soon
    `, predicate.ParamBinds{
        ":type": schema.MkString("process"),
        ":soon": schema.MkInt(time.Now().Add(1*time.Hour).Unix()),
    }),
)
```

!!! tip "Real Implementation"
    See the workflow engine implementation where:
    - [`Await` state](https://github.com/widmogrod/mkunion/blob/main/x/workflow/workflow_machine.go#L80) tracks `ExpectedTimeoutTimestamp`
    - [`Error` state](https://github.com/widmogrod/mkunion/blob/main/x/workflow/workflow_machine.go#L73) uses standardized error codes including `ProblemCallbackTimeout`
    - [Background timeout processor](https://github.com/widmogrod/mkunion/blob/main/example/my-app/background.go) and [task queue setup](https://github.com/widmogrod/mkunion/blob/main/example/my-app/server.go#L613) demonstrate how to process timeouts asynchronously

## Debugging and Observability

### State History Tracking

The mkunion state machine pattern leverages Change Data Capture (CDC) for automatic state history tracking. Since every state transition is persisted with versioning through optimistic concurrency control, you get a complete audit trail without modifying your state machine logic.

The `schemaless.Repository` creates an append log of all state changes with version numbers, providing ordering guarantees and enabling powerful history tracking capabilities. CDC processors consume this stream asynchronously to build history aggregates, analytics, and debugging tools - all without impacting state machine performance. The system automatically handles failures through persistent, replayable streams that survive crashes and allow processors to resume from their last position.

This approach integrates seamlessly with other mkunion patterns like retry processors and timeout handlers, creating a unified system where every state change is tracked, queryable, and analyzable.

!!! tip "Real Implementation"
    The example app demonstrates CDC integration with `taskRetry.RunCDC(ctx)` and `store.AppendLog()`. Detailed examples of building history processors, analytics pipelines, and debugging tools will be added in future updates.

### Metrics and Monitoring

Currently, metrics collection is the responsibility of the user. If you need Prometheus metrics or other monitoring, include them in your dependency interface and use them within your `Transition` function:

```go
type Dependencies interface {
    // Your business dependencies
    StockService() StockService
    
    // Metrics dependencies - user's responsibility to provide
    Metrics() *prometheus.Registry
    TransitionCounter() prometheus.Counter
}

func Transition(ctx context.Context, deps Dependencies, cmd Command, state State) (State, error) {
    // Manual metrics collection
    startTime := time.Now()
    defer func() {
        deps.TransitionCounter().Inc()
        // Record duration, state types, etc.
    }()
    
    // Your transition logic here
}
```

There's no automatic metrics injection - you must explicitly add metrics to your dependencies and instrument your transitions manually.

!!! note "Future Enhancement"
    Automatic metrics collection would be a valuable addition to `machine.Machine`. This could include built-in counters for transitions, error rates, and timing histograms without requiring manual instrumentation.

## Evolution and Versioning

### Backward Compatible Changes

When evolving state machines, maintain compatibility:

```go
// Version 1
//go:tag mkunion:"OrderState"
type (
    OrderCreated struct {
        ID    string
        Items []Item
    }
)

// Version 2 - Added field with default
//go:tag mkunion:"OrderState"
type (
    OrderCreated struct {
        ID       string
        Items    []Item
        Discount float64 `json:"discount,omitempty"` // New field
    }
)
```

### State Migration Strategies

Handle state structure changes:

```go
// Migration function
func MigrateOrderState(old []byte) (State, error) {
    // Try to unmarshal as current version
    current, err := shared.JSONUnmarshal[OrderState](old)
    if err == nil {
        return current, nil
    }
    
    // Try older version
    v1, err := shared.JSONUnmarshal[OrderStateV1](old)
    if err == nil {
        // Convert v1 to current
        return convertV1ToCurrent(v1), nil
    }
    
    return nil, fmt.Errorf("unknown state version")
}
```

### Deprecating States and Commands

Gracefully phase out old states:

```go
//go:tag mkunion:"OrderState"
type (
    // Deprecated: Use OrderPending instead
    OrderCreated struct {
        // ... fields
    }
    
    OrderPending struct {
        // New state structure
    }
)

func Transition(ctx context.Context, deps Dependencies, cmd Command, state State) (State, error) {
    // Handle deprecated state
    if old, ok := state.(*OrderCreated); ok {
        // Automatically migrate to new state
        state = &OrderPending{
            // Map old fields to new
        }
    }
    
    // Continue with normal processing
    // ...
}
```

## Performance Considerations

### Memory Optimization

1. **Reuse State Instances**: For states without data, use singletons
```go
var (
    pendingState = &Pending{}
    activeState  = &Active{}
)
```

2. **Lazy Loading**: Don't load unnecessary data in states
```go
type OrderDetails struct {
    ID       string
    // Don't embed full customer, just reference
    CustomerID string `json:"customer_id"`
}
```

### State Storage

mkunion uses state storage pattern where the current state of the state machine is persisted directly to the database. This approach:

- Stores the complete state after each transition
- Provides immediate access to current state without replay
- Supports optimistic concurrency control through versioning
- Works seamlessly with the `x/storage/schemaless` package

Example using typed repository:

```go title="example/state/machine_database_test.go"
--8<-- "example/state/machine_database_test.go:example-store-state"
```

### Concurrent Processing

When multiple processes might update the same state, use optimistic concurrency control provided by the schemaless repository:

```go title="example/state/service.go" 
--8<-- "example/state/service.go:handle-with-retry"
```

Key strategies:
1. **Optimistic Concurrency**: Use version checking to detect conflicts
2. **Retry Logic**: Implement exponential backoff for version conflicts  
3. **Partition by ID**: Process different entities in parallel safely

See the [Optimistic Concurrency Control](#optimistic-concurrency-control) section for detailed examples.

## Optimistic Concurrency Control

The `x/storage/schemaless` package provides built-in optimistic concurrency control using version fields. This ensures data consistency when multiple processes work with the same state.

### How It Works

1. Each record has a `Version` field that increments on updates
2. Updates specify the expected version in the record
3. If versions don't match, `ErrVersionConflict` is returned
4. Applications retry with the latest version

