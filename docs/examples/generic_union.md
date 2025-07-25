---
title: Union and generic types
---
# Union and generic types
MkUnion will generate generic unions for you.

For example, let's say you want to create a recursive **tree** data structure, that in its leaves will hold a value of `A` type.

## Declaration and generation

You can use `mkunion` to create a union type for the tree:

```go title="example/tree.go"
--8<-- "example/tree.go:tree-def"
```

After you run generation (as described in [getting started](../getting_started.md)), 
you have access to the same features as with non-generic unions.

When defining generic unions, you must follow these requirements:

1. **Type parameters must be specified in the tag**: The union tag must include all type parameters used by the variant types.
2. **Parameter names must match**: Type parameter names in the tag must match those used in variant types both by name and position.
3. **Same number of parameters**: Each variant type needs to have the same number of type parameters.

## Matching function

Let's define a higher-order function `ReduceTree` that will traverse leaves in `Tree` and produce a single value.

This function uses `MatchTreeR1` function that is generated automatically for you.

```go title="example/tree.go"
--8<-- "example/tree.go:reduce-tree"
```

## Example usage

You can use such function to sum all values in the tree, assuming that tree is of type `Tree[int]`:

```go title="example/tree_test.go"
--8<-- "example/tree_test.go:example-sum-values"
```

You can also reduce the tree to a complex structure, for example, to keep track of the order of values in the tree, along with the sum of all values in the tree.

```go title="example/tree_test.go"
--8<-- "example/tree_test.go:example-custom-agg"
```

## Either & Option types

For educational purposes, let's create two of the most popular union types in functional languages: `Option` and `Either`, with corresponding `Map` functions.

- The Either type is used to represent one of two possible values. Many times the left value holds the success value, and the right value holds an error value.
- The Option type is used to represent a value that may or may not be present, often replacing nulls in other languages.

```go title="f/datas.go"
--8<-- "f/datas.go:either"
--8<-- "f/datas.go:option"
--8<-- "f/datas.go:map-either"
--8<-- "f/datas.go:map-option"
```

In the example above, we define `MapEither` and `MapOption` functions that will apply the function `f` to the value inside the `Either` or `Option` type.

It would be preferable to have only one `Map` definition, but due to limitations of the Go type system, we need to define separate functions for each type.

I'm considering adding code generation for such behaviors in the future. This is not yet implemented due to a focus on validating core concepts.