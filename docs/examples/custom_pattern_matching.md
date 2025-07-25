---
title: Custom Pattern Matching
---
# Custom Pattern Matching

mkunion provides a powerful feature for creating custom pattern matching functions that can match on multiple union values simultaneously. This guide explains how to write and use custom pattern matching functions.

## Overview

While mkunion automatically generates pattern matching functions for individual union types (like `MatchTreeR1`, `MatchShapeR2`, etc.), you can also define custom pattern matching functions that work across multiple values or implement specialized matching logic.

Custom pattern matching is useful when you need to:
- Match on combinations of multiple union values
- Implement domain-specific matching patterns
- Create reusable matching logic for complex scenarios

## When to Use Custom Pattern Matching

### Matching Specific Combinations
Custom pattern matching is particularly valuable when you need to handle pairs of types but only care about certain combinations. If you need to match all possible combinations exhaustively, it's better to use nested `Match<Type>R<N>` functions.

### Simplifying Complex Type Assertions
Without custom pattern matching, matching two union values requires nested type assertions:

```go
// Without custom pattern matching - verbose and error-prone
if a, ok := v1.(*Circle); ok {
    if b, ok := v2.(*Rectangle); ok {
        // handle Circle-Rectangle combination
    } else if b, ok := v2.(*Square); ok {
        // handle Circle-Square combination
    }
}
// ... many more nested ifs
```

This approach has several disadvantages:
- Difficult to maintain as the number of types grows
- Easy to miss combinations
- No compile-time exhaustiveness checking
- Deeply nested code that's hard to read

Custom pattern matching solves these issues elegantly:

```go
// With custom pattern matching - clean and maintainable
MatchShapesR1(v1, v2,
    func(a *Circle, b *Circle) string { 
        return "Two circles" 
    },
    func(a *Rectangle, b any) string { 
        return "Rectangle meets another shape" 
    },
    func(a any, b any) string { 
        return "Other combination" 
    },
)
```

## Basic Syntax

To create a custom pattern matching function, use the `//go:tag mkmatch` annotation on an interface definition:

```go
//go:tag mkmatch
type MatchShapes[A, B Shape] interface {
    MatchCircles(x, y *Circle)
    MatchRectangleAny(x *Rectangle, y any)
    Finally(x, y any)
}
```

This generates functions like `MatchShapesR0`, `MatchShapesR1`, `MatchShapesR2`, and `MatchShapesR3` with 0 to 3 return values respectively.

### Custom Naming

You can also provide the function name in the tag, and mkunion will use that instead of interface name:

```go
//go:tag mkmatch:"MyShapeMatcher"
type MatchShapes[A, B Shape] interface {
    MatchCircles(x, y *Circle)
    MatchRectangleAny(x *Rectangle, y any)
    Finally(x, y any)
}
```

This generates `MyShapeMatcherR0`, `MyShapeMatcherR1`, etc., 

## Interface Definition Rules

When defining a match interface:

1. **Type Parameters**: The interface can have type parameters that constrain the input types
2. **Method Names**: Method names can be anything descriptive
3. **Method Parameters**: Each method must have the same number of parameters as type parameters
4. **Parameter Types**: Parameters can be:
   - Concrete types from the union (e.g., `*Circle`, `*Rectangle`)
   - The `any` type for wildcard matching
   - Other specific types for specialized matching

## Examples

### Example 1: Matching Shape Pairs

```go title="example/shape.go"
--8<-- "example/shape.go:shape-def"
--8<-- "example/shape.go:match-def"
--8<-- "example/shape.go:match-shapes"
```

### Example 2: Matching Tree Nodes

Notice that CombineTreeValues function match against TreePair type parameters.

```go title="example/tree.go"
--8<-- "example/tree.go:tree-def"
--8<-- "example/tree.go:match-def"
--8<-- "example/tree.go:match-use"
```

### Example 3: State Machine Transitions

Custom pattern matching is particularly useful for state machines:

```go title="example/transition.go"
--8<-- "example/transition.go:match-def"
--8<-- "example/transition.go:match-use"
```

## Best Practices

1. **Order Matters**: The generated function checks patterns in the order they appear in the interface. Put more specific patterns first.
2. **Use Wildcards Wisely**: The `any` type acts as a wildcard. Use it for catch-all cases or when you need to handle any type.
3. **Exhaustiveness**: Always include a catch-all pattern (typically with `any` parameters) to ensure all cases are handled.
4. **Naming Conventions**:
    - Use descriptive names for match methods
    - The interface name becomes the function prefix
    - Consider the domain when naming
5. **Type Safety**: The generated functions are fully type-safe and will panic if the patterns are not exhaustive.


## Limitations

1. Custom pattern matching functions are **limited to matching on up to 3 return values** (R0, R1, R2, R3).
2. The interface methods **must have parameters matching the type parameters in order**.
3. All methods in the **interface must have the same number of parameters**.

## Summary

Custom pattern matching in `mkunion` provides a powerful way to handle complex matching scenarios while maintaining type safety. 
By defining interfaces with the `//go:tag mkmatch:""` annotation, you can create specialized matching functions that work across multiple values and implement domain-specific logic.

This feature is particularly useful for:

- State machine implementations
- Complex data transformations
- Multi-value comparisons
- Domain-specific pattern matching logic

Combined with `mkunion`'s automatic union type generation and standard pattern matching, custom pattern matching completes a comprehensive toolkit for working with algebraic data types in Go.

## Next steps

- **[Union and generic types](./examples/generic_union.md)** - Learn about generic unions
- **[Marshaling union in JSON](./examples/json.md)** - Learn about marshaling and unmarshalling of union types in JSON
- **[State Machines and unions](./examples/state_machine.md)** - Learn about modeling state machines and how union type helps
