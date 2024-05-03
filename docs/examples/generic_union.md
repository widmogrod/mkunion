---
title: Generic unions
---
# Generic unions
MkUnion will generate generic unions for you.

You only need to declare each variant type of the union with a type parameter,
and library will figure out the rest.

What is **IMPORTANT** is that each variant type (Branch, Leaf in this example) needs to have the same number of type parameters.

For example, let's say you want to create a recursive tree data structure, that in leaves will hold value of `A` type.

## Declaration and generation

You can use `mkunion` to create a union type for the tree:

```go title="example/tree.go"
package example

//go:generate go run ../cmd/mkunion/main.go

//go:tag mkunion:"Tree"
type (
	Branch[A any] struct{ L, R Tree[A] }
	Leaf[A any]   struct{ Value A }
)
```

After you run generation (as described in [getting started](../getting_started.md)), 
you have access to the same features as with non-generic unions.

## Matching function

Let's define higher order function `ReduceTree` that will travers leaves in `Tree` and produce a single value.

This function uses `MatchTreeR1` function that is generated automatically for you.

```go title="example/tree.go"
func ReduceTree[A, B any](x Tree[A], f func(A, B) B, init B) B {
	return MatchTreeR1(
		x,
		func(x *Branch[A]) B {
			return ReduceTree(x.R, f, ReduceTree(x.L, f, init))
		}, func(x *Leaf[A]) B {
			return f(x.Value, init)
		},
	)
}
```
## Example usage

You can use such function to sum all values in the tree, assuming that tree is of type `Tree[int]`:

```go title="example/tree_test.go"
func ExampleTreeSumValues() {
	tree := &Branch[int]{
		L: &Leaf[int]{Value: 1},
		R: &Branch[int]{
			L: &Branch[int]{
				L: &Leaf[int]{Value: 2},
				R: &Leaf[int]{Value: 3},
			},
			R: &Leaf[int]{Value: 4},
		},
	}

	result := ReduceTree(tree, func(x int, agg int) int {
		return agg + x
	}, 0)

	fmt.Println(result)
	// Output: 10
}
```

You can also reduce tree to complex structure, for example to keep track of order of values in the tree, along with sum of all values in the tree.

```go title="example/tree_test.go"
func ExampleTreeCustomReduction() {
	tree := &Branch[int]{
		L: &Leaf[int]{Value: 1},
		R: &Branch[int]{
			L: &Branch[int]{
				L: &Leaf[int]{Value: 2},
				R: &Leaf[int]{Value: 3},
			},
			R: &Leaf[int]{Value: 4},
		},
	}

	result := ReduceTree(tree, func(x int, agg orderAgg) orderAgg {
		return orderAgg{
			Order:  append(agg.Order, x),
			Result: agg.Result + x,
		}
	}, orderAgg{
		Order:  []int{},
		Result: 0,
	})
	fmt.Println(result.Order)
	fmt.Println(result.Result)
	// Output: [1 2 3 4]
	// 10
}
```

## Either & Option types

For educational purposes, let's create two most popular union types in functional languages: `Option` and `Either` with corresponding `Map` functions.

- Either type is used to represent one of two possible values. Many times left value holds success value, and right value holds error value.
- Option type is used to represent a value that may or may not be present. Replaces nulls in some languages.

```go title="f/datas.go"
//go:tag mkunion:"Either"
type (
	Left[A, B any]  struct{ Value A }
	Right[A, B any] struct{ Value B }
)

//go:tag mkunion:"Option"
type (
	Some[A any] struct{ Value A }
	None[A any] struct{}
)

func MapEither[A, B, C any](x Either[A, B], f func(A) C) Either[C, B] {
	return MatchEitherR1(
		x,
		func(x *Left[A, B]) Either[C, B] {
			return &Left[C, B]{Value: f(x.Value)}
		},
		func(x *Right[A, B]) Either[C, B] {
			return &Right[C, B]{Value: x.Value}
		},
	)
}

func MapOption[A, B any](x Option[A], f func(A) B) Option[B] {
	return MatchOptionR1(
		x,
		func(x *Some[A]) Option[B] {
			return &Some[B]{Value: f(x.Value)}
		},
		func(x *None[A]) Option[B] {
			return &None[B]{}
		},
	)
}
```

In above example, we define `MapEither` and `MapOption` functions that will apply function `f` to value inside `Either` or `Option` type.

It would be much better to have only one `Map` definition, but due to limitations of Go type system, we need to define separate functions for each type.

I'm considering adding code generation for such behaviours in the future. Not yet due to focus on validating core concepts.

