---
title: State Machine Best Practices
---

# State Machine Best Practices

This guide covers best practices, patterns, and techniques for building robust state machines with mkunion. Whether you're building simple state machines or complex distributed systems, these practices will help you create maintainable and scalable solutions.

## Best Practices

When building state machines with mkunion, following these practices will help you create maintainable and robust systems:

### File Organization

Organize your state machine code across files for better maintainability:

1. **`model.go`**: State and command definitions with other model types like value objects, etc.
   ```go title="example/state/model.go"
   --8<-- "example/state/model.go:commands"
   --8<-- "example/state/model.go:states"
   --8<-- "example/state/model.go:value-objects"
   ```

2. **`machine.go`**: Core state machine initialization, and most importantly transition logic:
   ```go title="example/state/machine.go"
   --8<-- "example/state/machine.go:dependency"
   --8<-- "example/state/machine.go:new-machine"
   --8<-- "example/state/machine.go:transition-fragment"
   // ... and so on
   ```
   
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
--8<-- "example/state/machine_test.go:moq-init"

// ... and some time later in assertion functions
--8<-- "example/state/machine_test.go:moq-usage"
```

Benefits of generating mocks:

- **Reduces boilerplate**: No need to manually write mock implementations
- **Type safety**: Generated mocks always match the interface
- **Easy maintenance**: Mocks automatically update when interface changes
- **Better test readability**: Focus on behavior, not mock implementation

### State Machine Composition

For complex systems, compose multiple state machines as a service layer:

```go
type OrderService struct {
    repo schemaless.Repository[State]
    deps Dependency
}
```

```go
type ECommerceService struct {
    orderService   *OrderService
    paymentService *PaymentService
}

func NewECommerceService(orderSvc *OrderService, paymentSvc *PaymentService) *ECommerceService {
    return &ECommerceService{
        orderService:   orderSvc,
        paymentService: paymentSvc,
    }
}

func (s *ECommerceService) ProcessOrder(ctx context.Context, orderCmd Command) error {
    // 1. Handle order command through order service
    newOrderState, err := s.orderService.HandleCommand(ctx, orderCmd)
    if err != nil {
        return fmt.Errorf("order processing failed: %w", err)
    }

    // 2. If order is confirmed, trigger payment through payment service
    if processing, ok := newOrderState.(*OrderProcessing); ok {
        paymentCmd := &InitiatePaymentCMD{
            OrderID: processing.Order.ID,
            Amount:  processing.Order.OrderAttr.Price,
        }
        _, err := s.paymentService.HandleCommand(ctx, paymentCmd)
        if err != nil {
            return fmt.Errorf("payment initiation failed: %w", err)
        }
    }

    return nil
}
```


**Key principles:**

- **Domain services**: Each domain encapsulates its repository, dependencies, and machine logic
- **Schemaless repositories**: Use `schemaless.Repository[StateType]` for type-safe state storage
- **Service composition**: Compose domain services, avoiding direct repository/machine access
- **Single responsibility**: Each service handles one domain's state machine lifecycle
- **Optimistic concurrency**: Built-in through `schemaless.Repository` version handling
- **No duplication**: State loading, machine creation, and saving logic exists once per domain

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
--8<-- "example/state/machine.go:error-recovery"
```

This approach preserves critical information needed for recovery 
without losing the context of what failed (look at `Transition(ctx, di, s.ProblemCommand, s.ProblemState)`)

### 4. Ignoring Concurrency

**Problem**: Misunderstanding the state machine concurrency model
```go
// Wrong: Sharing a machine instance across goroutines
sharedMachine := NewMachine(deps, currentState)
go sharedMachine.Handle(ctx, cmd1) // Goroutine 1
go sharedMachine.Handle(ctx, cmd2) // Goroutine 2 - DON'T DO THIS!
```

**Solution**: For handling concurrent updates to the same entity, see the [Optimistic Concurrency Control](#optimistic-concurrency-control) section below.

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

## Optimistic Concurrency Control

The `x/storage/schemaless` package provides built-in optimistic concurrency control using version fields. This ensures data consistency when multiple processes work with the same state.

```go title="example/state/machine_test.go"
--8<-- "example/state/machine_test.go:example-store-state"
```

**How It Works**:

1. Each record has a `Version` field that increments on updates
2. Updates specify the expected version in the record
3. If versions don't match, `ErrVersionConflict` is returned
4. Applications retry with the latest version

