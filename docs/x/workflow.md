---
title: Workflow Package
---

# x/workflow - Workflow Orchestration Engine

The `x/workflow` package provides a powerful workflow orchestration engine built on top of mkunion's state machine framework. It enables you to define, execute, and manage complex workflows with features like asynchronous operations, scheduling, error recovery, and conditional logic.

## Overview

The workflow package offers:
- **DSL for workflow definition** - Simple, expressive syntax for defining workflows
- **Asynchronous operations** - Built-in support for async functions with callbacks
- **Scheduling** - Cron-based scheduling and delayed execution
- **Error handling** - Automatic retry and recovery mechanisms
- **State persistence** - Durable workflow execution with resume capability
- **Variable management** - Workflow-scoped variables and data flow
- **Conditional branching** - If/then/else logic within workflows

## Core Concepts

### Workflow Definition

Workflows are defined using a DSL that compiles to Go structures:

```go
// DSL syntax (in string format)
flow hello_world(name) {
    var greeting = await greet(name) @timeout(30)
    if greeting.language == "es" {
        return concat("¡Hola, ", name, "!")
    } else {
        return concat("Hello, ", name, "!")
    }
}

// Equivalent Go structure
workflow := &Flow{
    Name: "hello_world",
    Arg:  "name",
    Body: []Expr{
        &Assign{
            VarOk: "greeting",
            Val: &Apply{
                Name: "greet",
                Args: []Reshaper{&GetValue{Path: "name"}},
                Await: &ApplyAwait{Timeout: 30},
            },
        },
        &Choose{
            If: &ChooseCondition{
                Condition: /* condition expression */,
                Then: []Expr{/* then branch */},
                Else: []Expr{/* else branch */},
            },
        },
    },
}
```

### State Machine Foundation

The workflow engine is built on `x/machine`, providing:
- Type-safe state transitions
- Command-based execution model
- Comprehensive testing utilities

```go
// Workflow states
//go:tag mkunion:"State"
type (
    NextOperation struct {
        FlowID    string
        RunID     string
        RunOption RunOption
        BaseInfo  BaseInfo
    }
    Done struct {
        Result   reshaper.Reshaper
        BaseInfo BaseInfo
    }
    Error struct {
        Code     string
        Reason   string
        BaseInfo BaseInfo
    }
    Await struct {
        CallbackID string
        BaseInfo   BaseInfo
    }
)
```

## Getting Started

### Basic Workflow

```go
// Define dependencies
deps := &workflow.DI{
    Functions: map[string]workflow.Function{
        "add": func(input *workflow.FunctionInput) (*workflow.FunctionOutput, error) {
            a := input.Args[0].(float64)
            b := input.Args[1].(float64)
            return &workflow.FunctionOutput{
                Result: a + b,
            }, nil
        },
    },
}

// Define a simple workflow
addWorkflow := &workflow.Flow{
    Name: "add_numbers",
    Arg:  "input",
    Body: []workflow.Expr{
        &workflow.Apply{
            Name: "add",
            Args: []reshaper.Reshaper{
                &reshaper.GetValue{Path: "input.a"},
                &reshaper.GetValue{Path: "input.b"},
            },
        },
        &workflow.End{
            Result: &reshaper.GetValue{Path: "$"},
        },
    },
}

// Create workflow machine
machine := workflow.NewMachine(deps, nil)

// Run the workflow
machine.Send(ctx, &workflow.Run{
    Flow:  addWorkflow,
    Input: map[string]any{"a": 5, "b": 3},
})

// Result will be 8
```

### Asynchronous Operations

```go
// Define async function
deps.Functions["fetch_data"] = func(input *workflow.FunctionInput) (*workflow.FunctionOutput, error) {
    // Return callback ID for async operation
    callbackID := input.CallbackID
    
    // Start async operation (e.g., HTTP request)
    go func() {
        // ... perform async work ...
        result := fetchDataFromAPI()
        
        // Send callback when complete
        machine.Send(ctx, &workflow.Callback{
            CallbackID: callbackID,
            Result:     result,
        })
    }()
    
    // Return immediately with no result
    return &workflow.FunctionOutput{}, nil
}

// Use in workflow with await
workflow := &workflow.Flow{
    Name: "async_example",
    Body: []workflow.Expr{
        &workflow.Assign{
            VarOk: "data",
            Val: &workflow.Apply{
                Name: "fetch_data",
                Await: &workflow.ApplyAwait{
                    Timeout: 60, // 60 second timeout
                },
            },
        },
        &workflow.End{
            Result: &reshaper.GetValue{Path: "data"},
        },
    },
}
```

### Conditional Logic

```go
workflow := &workflow.Flow{
    Name: "conditional_example",
    Arg:  "user",
    Body: []workflow.Expr{
        &workflow.Choose{
            If: &workflow.ChooseCondition{
                Condition: &reshaper.Compare{
                    Left:  &reshaper.GetValue{Path: "user.age"},
                    Op:    ">=",
                    Right: &reshaper.SetValue{Value: 18},
                },
                Then: []workflow.Expr{
                    &workflow.Assign{
                        VarOk: "status",
                        Val:   &reshaper.SetValue{Value: "adult"},
                    },
                },
                Else: []workflow.Expr{
                    &workflow.Assign{
                        VarOk: "status",
                        Val:   &reshaper.SetValue{Value: "minor"},
                    },
                },
            },
        },
        &workflow.End{
            Result: &reshaper.GetValue{Path: "status"},
        },
    },
}
```

## Advanced Features

### Scheduled Workflows

Execute workflows on a schedule using cron expressions:

```go
// Schedule a workflow to run every hour
machine.Send(ctx, &workflow.Run{
    Flow: myWorkflow,
    RunOption: &workflow.ScheduleRun{
        Interval: "0 * * * *", // Every hour
        Overlap:  workflow.ScheduleOverlapSkip,
    },
})

// Control scheduled workflows
machine.Send(ctx, &workflow.StopSchedule{
    RunID: scheduledRunID,
})

machine.Send(ctx, &workflow.ResumeSchedule{
    RunID: scheduledRunID,
})
```

### Error Handling and Recovery

```go
// Define retry policy in dependencies
deps := &workflow.DI{
    MaxRetries: func() int64 { return 3 },
}

// Workflow with error handling
workflow := &workflow.Flow{
    Name: "error_handling",
    Body: []workflow.Expr{
        &workflow.Apply{
            Name: "risky_operation",
            // Will automatically retry up to MaxRetries times
        },
        &workflow.Choose{
            If: &workflow.ChooseCondition{
                Condition: /* check for error */,
                Then: []workflow.Expr{
                    // Handle error case
                    &workflow.End{
                        Result: &reshaper.SetValue{Value: "failed"},
                    },
                },
            },
        },
    },
}

// Manual recovery attempt
machine.Send(ctx, &workflow.TryRecover{
    RunID:  erroredRunID,
    Reason: "Manual intervention completed",
})
```

### Variable Scoping

```go
workflow := &workflow.Flow{
    Name: "variable_example",
    Arg:  "input",
    Body: []workflow.Expr{
        // Assign to variable
        &workflow.Assign{
            VarOk: "temp",
            Val:   &reshaper.GetValue{Path: "input.value"},
        },
        // Use variable in computation
        &workflow.Assign{
            VarOk: "doubled",
            Val: &workflow.Apply{
                Name: "multiply",
                Args: []reshaper.Reshaper{
                    &reshaper.GetValue{Path: "temp"},
                    &reshaper.SetValue{Value: 2},
                },
            },
        },
        // Variables are workflow-scoped
        &workflow.End{
            Result: &reshaper.GetValue{Path: "doubled"},
        },
    },
}
```

## Workflow Patterns

### Sequential Processing

```go
// Process items one by one
&workflow.Flow{
    Name: "sequential_process",
    Arg:  "items",
    Body: []workflow.Expr{
        &workflow.Assign{
            VarOk: "results",
            Val:   &reshaper.SetValue{Value: []any{}},
        },
        // Process each item (pseudo-code for illustration)
        &workflow.ForEach{
            Items: &reshaper.GetValue{Path: "items"},
            Body: []workflow.Expr{
                &workflow.Apply{
                    Name: "process_item",
                    Args: []reshaper.Reshaper{
                        &reshaper.GetValue{Path: "$.item"},
                    },
                },
            },
        },
    },
}
```

### Saga Pattern

Implement distributed transactions with compensating actions:

```go
// Conceptual saga implementation
&workflow.Flow{
    Name: "booking_saga",
    Body: []workflow.Expr{
        // Step 1: Reserve flight
        &workflow.Assign{
            VarOk:   "flight",
            VarErr:  "flightErr",
            Val: &workflow.Apply{
                Name: "reserve_flight",
                Await: &workflow.ApplyAwait{Timeout: 30},
            },
        },
        // Check for errors and compensate if needed
        &workflow.Choose{
            If: &workflow.ChooseCondition{
                Condition: /* check flightErr */,
                Then: []workflow.Expr{
                    &workflow.End{
                        Result: &reshaper.SetValue{Value: "Failed to reserve flight"},
                    },
                },
            },
        },
        // Step 2: Reserve hotel with compensation
        &workflow.Assign{
            VarOk:   "hotel",
            VarErr:  "hotelErr",
            Val: &workflow.Apply{
                Name: "reserve_hotel",
                Await: &workflow.ApplyAwait{Timeout: 30},
            },
        },
        &workflow.Choose{
            If: &workflow.ChooseCondition{
                Condition: /* check hotelErr */,
                Then: []workflow.Expr{
                    // Compensate: Cancel flight
                    &workflow.Apply{
                        Name: "cancel_flight",
                        Args: []reshaper.Reshaper{
                            &reshaper.GetValue{Path: "flight.id"},
                        },
                    },
                    &workflow.End{
                        Result: &reshaper.SetValue{Value: "Failed, rolled back"},
                    },
                },
            },
        },
    },
}
```

## Integration with Storage

Persist workflow state for durability:

```go
// Create storage backend
storage := schemaless.NewInMemoryRepository()

// Wrap workflow machine with persistence
persistedMachine := &PersistedWorkflowMachine{
    Machine: workflow.NewMachine(deps, nil),
    Storage: storage,
}

// Workflow state is automatically persisted
// Can resume from storage after restart
```

## Testing Workflows

```go
// Create test suite
suite := machine.NewTestSuite(func() *machine.Machine[workflow.Dependency, workflow.Command, workflow.State] {
    return workflow.NewMachine(testDeps, nil)
})

// Define test scenarios
suite.Case("successful workflow", func(t *testing.T, m *machine.Machine[...]) {
    // Send run command
    m.Send(ctx, &workflow.Run{
        Flow:  testFlow,
        Input: testInput,
    })
    
    // Assert final state
    state := m.CurrentState()
    done, ok := state.(*workflow.Done)
    assert.True(t, ok)
    assert.Equal(t, expectedResult, done.Result)
})

// Generate state diagram
suite.SelfDocumentStateDiagram(t, "workflow_states.go")
```

## Best Practices

### 1. Keep Functions Pure

```go
// Good: Pure function
deps.Functions["add"] = func(input *FunctionInput) (*FunctionOutput, error) {
    return &FunctionOutput{
        Result: input.Args[0].(int) + input.Args[1].(int),
    }, nil
}

// Avoid: Side effects in functions
// Use async functions for I/O operations
```

### 2. Handle Timeouts

Always set timeouts for async operations:

```go
&workflow.Apply{
    Name: "external_api_call",
    Await: &workflow.ApplyAwait{
        Timeout: 30, // seconds
    },
}
```

### 3. Design for Idempotency

Ensure workflows can be safely retried:

```go
// Use unique IDs for operations
deps.GenerateRunID = func() string {
    return uuid.New().String()
}
```

### 4. Monitor Workflow State

Track workflow execution:

```go
// Check current state
state := machine.CurrentState()
switch s := state.(type) {
case *workflow.Done:
    log.Printf("Workflow completed: %v", s.Result)
case *workflow.Error:
    log.Printf("Workflow failed: %s - %s", s.Code, s.Reason)
case *workflow.Await:
    log.Printf("Waiting for callback: %s", s.CallbackID)
}
```

## Performance Considerations

1. **Async Operations**: Use for I/O-bound tasks to avoid blocking
2. **Variable Size**: Be mindful of data stored in workflow variables
3. **Persistence**: Choose appropriate storage backend for your scale
4. **Scheduling**: Use appropriate overlap policies for scheduled workflows

## Future Roadmap

The workflow package is evolving to support:
- Parallel execution (Fork/Join)
- Dynamic parallelism (Parallel For-Each)
- Enhanced saga pattern support
- Event-driven workflow triggers
- Visual workflow designer
- Workflow versioning and migration

## Troubleshooting

Common issues and solutions:

1. **Callback Timeout**: Ensure async functions send callbacks within timeout
2. **Variable Not Found**: Check variable names and scoping
3. **Function Not Found**: Register all functions in dependencies
4. **State Recovery**: Ensure storage backend is properly configured

## API Reference

Key types and functions:

- `Flow`: Workflow definition
- `Expr`: Workflow expressions (Assign, Apply, Choose, End)
- `State`: Workflow execution states
- `Command`: Operations on workflows
- `Function`: Workflow function signature
- `Dependency`: External dependencies interface

For detailed API documentation, see the [Go package documentation](https://pkg.go.dev/github.com/widmogrod/mkunion/x/workflow).