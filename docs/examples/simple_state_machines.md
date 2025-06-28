---
title: Simple State Machine Examples
---

# Simple State Machine Examples

This document provides simple, easy-to-understand examples of state machines using mkunion. These examples are designed to help you grasp the core concepts before moving on to more complex scenarios.

## Traffic Light Example

A traffic light is a classic example of a state machine with three states: Red, Yellow, and Green.

### Model Definition

```go title="example/traffic/model.go"
--8<-- "example/traffic/model.go"
```

### Transition Function

```go title="example/traffic/traffic_light.go"
--8<-- "example/traffic/traffic_light.go:17:33"
```

### Testing

```go title="example/traffic/traffic_light_test.go"
--8<-- "example/traffic/traffic_light_test.go:11:29"
```

### Complete Test Suite

```go title="example/traffic/traffic_light_test.go"
--8<-- "example/traffic/traffic_light_test.go:31:55"
```

### Example Usage

The traffic light state machine can be used in applications:

```go title="example/traffic/traffic_light.go"
--8<-- "example/traffic/traffic_light.go:35:58"
```

## Key Concepts Demonstrated

The traffic light example illustrates fundamental state machine concepts:

1. **States without data**: Pure states that represent distinct conditions
2. **Simple transitions**: Clear, predictable state changes in response to commands
3. **Exhaustive matching**: Generated match functions ensure all states are handled
4. **Dependency injection**: Even simple examples follow the pattern for consistency
5. **Testability**: Easy to test with mkunion's testing framework

## Next Steps

- Review the [comprehensive Order Service example](state_machine.md) for a more complex scenario
- Learn about [testing strategies](state_machine.md#testing-state-machines--self-documenting) for state machines
- Explore [advanced patterns](state_machine.md#advanced-patterns) for composition and async operations