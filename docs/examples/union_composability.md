---
title: Composability and Type Safety
---

Union types in mkunion are designed to be highly composable, allowing you to build sophisticated type-safe abstractions. 

This guide explores how to create and compose fundamental union types like `Option` and `Result`, demonstrating patterns that eliminate null pointer exceptions and provide exhaustive error handling.

## Core Union Types: Option and Result

Let's start by implementing two of the most popular union types from functional programming:

- **Option[T]**: Represents a value that may or may not be present, eliminating null references
- **Result[T, E]**: Represents either a success value (Ok) or an error value (Err), providing type-safe error handling

## Basic Implementation

```go title="example/datas.go"
--8<-- "example/datas.go:example"
```

The example shows a common pattern: an API fetch that might fail (Result) and might return no data (Option):

```go title="example/datas.go"
--8<-- "example/datas.go:fetch-type"
```

This type precisely captures three states:

1. **Success with data**: `Ok[Some[User]]`
2. **Success with no data**: `Ok[None[User]]`
3. **API failure**: `Err[APIError]`

### Handling Nested Unions

```go title="example/datas.go"
--8<-- "example/datas.go:fetch-handling"
```

### Creating Composed Values

```go title="example/datas_test.go"
--8<-- "example/datas_test.go:creating-fetch"
```

## Summary

Union type composition in mkunion provides:

- **Type Safety**: Impossible states are unrepresentable
- **Exhaustiveness**: The compiler ensures all cases are handled
- **Composability**: Simple types combine into sophisticated abstractions
- **Clarity**: Error paths and edge cases are explicit in types

By mastering Option and Result composition, you can build robust applications that handle errors gracefully and eliminate entire classes of runtime failures. 
The key is to start simple, build a library of helper functions, and gradually compose more sophisticated types as your domain requires.

## Next steps

- **[Phantom Types](./union_phanthom_types.md)** - Learn benefits of phantom types
