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
   ```go
   //go:tag mkunion:"State"
   type (
       OrderPending struct { /* ... */ }
       OrderProcessing struct { /* ... */ }
   )
   
   //go:tag mkunion:"Command"
   type (
       CreateOrderCMD struct { /* ... */ }
       CancelOrderCMD struct { /* ... */ }
   )
   ```

2. **`machine.go`**: Core state machine logic
   ```go
   package order
   
   //go:generate mkunion watch -g .
   //go:generate moq -skip-ensure -out machine_mock.go . Dependency
   
   // Dependency interface - moq will generate DependencyMock from this
   type Dependency interface {
       TimeNow() *time.Time
       StockService() StockService
       PaymentService() PaymentService
   }
   
   // Common errors
   var (
       ErrInvalidTransition = errors.New("invalid state transition")
       ErrOrderNotFound     = errors.New("order not found")
   )
   
   // Machine constructor
   func NewMachine(deps Dependency, state State) *machine.Machine[Dependency, Command, State] {
       if state == nil {
           state = &OrderPending{} // Default initial state
       }
       return machine.NewMachine(deps, Transition, state)
   }
   
   // Transition function
   func Transition(ctx context.Context, deps Dependency, cmd Command, state State) (State, error) {
       // Implementation
   }
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

```go
func Transition(ctx context.Context, deps Dependencies, cmd Command, state State) (State, error) {
    return MatchCommandR2(cmd,
        func(c *CreateOrderCMD) (State, error) {
            // Validate command first
            if c.CustomerID == "" {
                return nil, fmt.Errorf("customer ID is required")
            }
            if len(c.Items) == 0 {
                return nil, fmt.Errorf("order must contain at least one item")
            }
            // Then check state
            // ...
        },
    )
}
```

#### Advanced Validation with go-validate

For complex validation requirements, combine the Transition function with validation libraries:

```go
import "github.com/go-playground/validator/v10"

//go:tag mkunion:"Command"
type (
    CreateOrderCMD struct {
        CustomerID string      `validate:"required,uuid"`
        Items      []OrderItem `validate:"required,min=1,dive"`
        Email      string      `validate:"required,email"`
        Phone      string      `validate:"omitempty,e164"`
    }
    
    OrderItem struct {
        SKU      string  `validate:"required,alphanum"`
        Quantity int     `validate:"required,min=1,max=100"`
        Price    float64 `validate:"required,min=0.01"`
    }
)

type Dependencies interface {
    Validator() *validator.Validate
    CustomerService() CustomerService
}

func Transition(ctx context.Context, deps Dependencies, cmd Command, state State) (State, error) {
    return MatchCommandR2(cmd,
        func(c *CreateOrderCMD) (State, error) {
            // 1. Structural validation with go-validate
            if err := deps.Validator().Struct(c); err != nil {
                return nil, fmt.Errorf("validation failed: %w", err)
            }
            
            // 2. Business rule validation
            totalAmount := 0.0
            for _, item := range c.Items {
                totalAmount += item.Price * float64(item.Quantity)
            }
            if totalAmount > 10000 {
                return nil, fmt.Errorf("order total %.2f exceeds maximum allowed", totalAmount)
            }
            
            // 3. External validation (e.g., customer exists)
            customer, err := deps.CustomerService().Get(ctx, c.CustomerID)
            if err != nil {
                return nil, fmt.Errorf("customer validation failed: %w", err)
            }
            if !customer.Active {
                return nil, fmt.Errorf("customer %s is not active", c.CustomerID)
            }
            
            // 4. State-based validation
            switch state.(type) {
            case nil, *OrderPending:
                // Valid initial states
            default:
                return nil, fmt.Errorf("cannot create order in current state %T", state)
            }
            
            // All validations passed, create new state
            return &OrderPending{
                OrderID:    generateOrderID(),
                CustomerID: c.CustomerID,
                Items:      c.Items,
                CreatedAt:  deps.TimeNow(),
            }, nil
        },
    )
}
```

This approach scales well because:
- Structural validation is declarative (struct tags)
- Business rules are explicit and testable
- External validations are isolated in dependencies
- State validations ensure valid transitions
- All validation happens before any state change

### Dependency Management

1. **Define Clear Interfaces**: Dependencies should be interfaces, not concrete types
2. **Keep Dependencies Minimal**: Only inject what's absolutely necessary
3. **Generate Mocks with moq**: Use `//go:generate moq` to automatically generate mocks

```go title="example/state/machine.go"
--8<-- "example/state/machine.go:imports"
```

Running `go generate ./...` creates `machine_mock.go` with a `DependencyMock` type (generated from the `Dependency` interface shown in the File Organization section above). This mock can then be used in tests:

```go
func TestStateMachine(t *testing.T) {
    // DependencyMock is generated from the Dependency interface
    dep := &DependencyMock{
        TimeNowFunc: func() *time.Time {
            now := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
            return &now
        },
        StockServiceFunc: func() StockService {
            return &StockServiceMock{
                ReserveFunc: func(items []Item) error {
                    return nil
                },
            }
        },
    }
    
    machine := NewMachine(dep, nil)
    // Test the machine...
}
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

```go
// Best practice: Error state with command and previous state
type OrderError struct {
    // Error metadata
    Reason       string
    FailedAt     time.Time
    RetryCount   int
    
    // Critical for debugging and recovery
    FailedCommand Command  // The exact command that failed
    PreviousState State    // State before the failed transition
}

// This pattern enables:
// 1. Perfect reproduction of the failure
// 2. Automatic retry with the same command
// 3. Debugging with full context
// 4. Recovery to previous valid state

// Or use a shared BaseState pattern like in workflow
type BaseState struct {
    ID        string
    Metadata  map[string]string
    UpdatedAt time.Time
}

type (
    PaymentPending struct {
        BaseState
        Amount float64
    }
    PaymentFailed struct {
        BaseState
        Reason string
        PreviousAmount float64  // Store critical data from previous state
    }
)

// Recovery becomes straightforward
func Transition(ctx context.Context, deps Dependencies, cmd Command, state State) (State, error) {
    return MatchCommandR2(cmd,
        func(c *RetryFailedCMD) (State, error) {
            switch s := state.(type) {
            case *OrderError:
                // Retry the exact command that failed
                return Transition(ctx, deps, s.FailedCommand, s.PreviousState)
                
            case *PaymentFailed:
                // Can access previous state data for retry
                if s.PreviousState != nil {
                    // Retry with original state
                    return processPayment(s.PreviousState)
                }
                // Or use BaseState data if using that pattern
                return &PaymentPending{
                    BaseState: s.BaseState,
                    Amount:    s.PreviousAmount,
                }, nil
            }
            return nil, fmt.Errorf("can only retry from error states")
        },
    )
}
```

This approach preserves critical information needed for recovery without losing the context of what failed.

!!! tip "Real Implementation Example"
    The order state machine example demonstrates this pattern perfectly. See how `OrderError` in [`example/state/model.go`](https://github.com/widmogrod/mkunion/blob/main/example/state/model.go#L47) stores both `ProblemCommand` and `ProblemState`. The retry logic in [`machine.go`](https://github.com/widmogrod/mkunion/blob/main/example/state/machine.go#L180) shows how to use this information to retry the exact failed operation.

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
    record.State = machine.State()
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

For complex systems, compose multiple state machines:

```go
// Order state machine
type OrderMachine struct {
    *machine.Machine[OrderDeps, OrderCommand, OrderState]
}

// Payment state machine  
type PaymentMachine struct {
    *machine.Machine[PaymentDeps, PaymentCommand, PaymentState]
}

// Composed e-commerce flow
type ECommerceMachine struct {
    Order   *OrderMachine
    Payment *PaymentMachine
}

func (e *ECommerceMachine) ProcessOrder(ctx context.Context, orderCmd OrderCommand) error {
    // Handle order state change
    if err := e.Order.Handle(ctx, orderCmd); err != nil {
        return err
    }
    
    // If order is confirmed, trigger payment
    if _, ok := e.Order.State().(*OrderConfirmed); ok {
        return e.Payment.Handle(ctx, &InitiatePaymentCMD{})
    }
    
    return nil
}
```

### Async Operations with Callbacks

Handle long-running operations without blocking:

```go
//go:tag mkunion:"AsyncState"
type (
    OperationPending struct {
        ID        string
        StartedAt time.Time
    }
    OperationComplete struct {
        ID       string
        Result   interface{}
        Duration time.Duration
    }
)

//go:tag mkunion:"AsyncCommand"
type (
    StartAsyncCMD struct {
        ID string
    }
    CompleteAsyncCMD struct {
        ID     string
        Result interface{}
    }
)

// Transition starts async operation
func Transition(ctx context.Context, deps AsyncDeps, cmd AsyncCommand, state AsyncState) (AsyncState, error) {
    return MatchAsyncCommandR2(cmd,
        func(c *StartAsyncCMD) (AsyncState, error) {
            // Start async operation
            go deps.AsyncWorker(c.ID, func(result interface{}, err error) {
                // Callback to complete
                completeCMD := &CompleteAsyncCMD{
                    ID:     c.ID,
                    Result: result,
                }
                deps.CommandQueue.Enqueue(completeCMD)
            })
            
            return &OperationPending{
                ID:        c.ID,
                StartedAt: time.Now(),
            }, nil
        },
        // ... handle completion
    )
}
```

### Time-Based Transitions

Implement timeouts and scheduled transitions:

```go
//go:tag mkunion:"TimerCommand"  
type (
    TimeoutCMD struct {
        Reason string
    }
)

// Set up timeout when entering a state
func SetupStateTimeouts(m *machine.Machine[Deps, Command, State], timeout time.Duration) {
    go func() {
        timer := time.NewTimer(timeout)
        defer timer.Stop()
        
        select {
        case <-timer.C:
            m.Handle(context.Background(), &TimeoutCMD{
                Reason: "operation timeout",
            })
        case <-m.Done():
            return
        }
    }()
}
```

## Debugging and Observability

### Structured Logging

Implement comprehensive logging for state transitions:

```go
type LoggingDependencies struct {
    Logger *slog.Logger
    // ... other deps
}

func Transition(ctx context.Context, deps LoggingDependencies, cmd Command, state State) (State, error) {
    // Log command received
    deps.Logger.Info("command received",
        "command_type", fmt.Sprintf("%T", cmd),
        "current_state", fmt.Sprintf("%T", state),
        "trace_id", ctx.Value("trace_id"),
    )
    
    startTime := time.Now()
    newState, err := performTransition(ctx, deps, cmd, state)
    duration := time.Since(startTime)
    
    if err != nil {
        deps.Logger.Error("transition failed",
            "error", err,
            "duration_ms", duration.Milliseconds(),
        )
        return nil, err
    }
    
    deps.Logger.Info("transition completed",
        "new_state", fmt.Sprintf("%T", newState),
        "duration_ms", duration.Milliseconds(),
    )
    
    return newState, nil
}
```

### State History Tracking

Keep a history of state transitions for debugging:

```go
type StateHistory struct {
    Transitions []TransitionRecord
    mu          sync.RWMutex
}

type TransitionRecord struct {
    From      string
    To        string
    Command   string
    Timestamp time.Time
    Error     error
}

func (h *StateHistory) Record(from, to State, cmd Command, err error) {
    h.mu.Lock()
    defer h.mu.Unlock()
    
    h.Transitions = append(h.Transitions, TransitionRecord{
        From:      fmt.Sprintf("%T", from),
        To:        fmt.Sprintf("%T", to),
        Command:   fmt.Sprintf("%T", cmd),
        Timestamp: time.Now(),
        Error:     err,
    })
}
```

### Metrics and Monitoring

Export metrics for monitoring:

```go
type MetricsDependencies struct {
    TransitionCounter   *prometheus.CounterVec
    TransitionDuration  *prometheus.HistogramVec
    StateGauge         *prometheus.GaugeVec
    // ... other deps
}

func instrumentedTransition(deps MetricsDependencies, transition TransitionFunc) TransitionFunc {
    return func(ctx context.Context, cmd Command, state State) (State, error) {
        timer := prometheus.NewTimer(deps.TransitionDuration.WithLabelValues(
            fmt.Sprintf("%T", cmd),
            fmt.Sprintf("%T", state),
        ))
        defer timer.ObserveDuration()
        
        newState, err := transition(ctx, cmd, state)
        
        labels := prometheus.Labels{
            "from_state": fmt.Sprintf("%T", state),
            "to_state":   fmt.Sprintf("%T", newState),
            "command":    fmt.Sprintf("%T", cmd),
            "status":     "success",
        }
        
        if err != nil {
            labels["status"] = "error"
        }
        
        deps.TransitionCounter.With(labels).Inc()
        
        return newState, err
    }
}
```

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
func MigrateOrderState(old json.RawMessage) (State, error) {
    // Try to unmarshal as current version
    var current OrderState
    if err := json.Unmarshal(old, &current); err == nil {
        return current, nil
    }
    
    // Try older version
    var v1 OrderStateV1
    if err := json.Unmarshal(old, &v1); err == nil {
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

```go title="example/machine/state_storage.go"
--8<-- "example/machine/state_storage.go:example"
```

### Concurrent Processing

When multiple processes might update the same state, use optimistic concurrency control provided by the schemaless repository:

```go title="example/machine/concurrent_processing.go" 
--8<-- "example/machine/concurrent_processing.go:retry-loop"
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

### Complete Example

```go title="example/machine/concurrent_processing.go"
--8<-- "example/machine/concurrent_processing.go:process-order"
```

### Batch Operations with Concurrency Control

When updating multiple records:

```go title="example/machine/concurrent_processing.go"
--8<-- "example/machine/concurrent_processing.go:batch-operations"
```

### Testing Concurrent Access

```go title="example/machine/concurrent_processing_test.go"
--8<-- "example/machine/concurrent_processing_test.go:concurrent-test"
```

### Retry Helper Function

The retry logic can be extracted into a reusable helper:

```go title="example/machine/concurrent_processing.go"
--8<-- "example/machine/concurrent_processing.go:retry-helper"
```