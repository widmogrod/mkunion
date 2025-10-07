# Getting started

### Install mkunion
Run in your terminal
```bash
go install github.com/widmogrod/mkunion/cmd/mkunion@v1.26.1
```

### Define your first union type
Create your first union. In our simple example, we will represent different geometric shapes.
But in a more complex example, you may want to represent different states of your application, model domain aggregates, or create your own DSL.
```go title="example/shape.go"
--8<-- "example/shape.go:shape-def"
```

In the above example, you can see a few important concepts:

#### `//go:tag mkunion:"Shape"`

Tags are a powerful and flexible way to add metadata to your code.
You may be familiar with tags when you work with JSON in Go:

```go
type User struct {
	Name string `json:"name"`
}
```

Unfortunately, Go doesn't extend this feature to other parts of the language.

MkUnion defines the `//go:tag` comment, following other idiomatic definitions like `go:generate` and `go:embed`, to allow adding metadata to struct types.
And MkUnion uses it heavily to offer a way of adding new behavior to Go types.

##### Tags supported by MkUnion

- `go:tag mkunion:"Shape"` - defines a union type. For generic unions, type parameters MUST be specified: `go:tag mkunion:"Result[T, E]"`.
- `go:tag serde:"json"` - enables serialization type (currently only JSON is supported), enabled by default.
- `go:tag shape:"-"` - disables shape generation for this type, useful in cases where an x/shared package cannot depend on other x packages, to avoid circular dependencies.
- `go:tag mkunion:",no-type-registry"` - if you want to disable generation of the type registry in a package, define this tag in one of the Go files above the package declaration:
  ```go
  //go:tag mkunion:",no-type-registry"
  package example
  ```
- `go:tag mkmatch` - generate custom pattern matching function from interface definition
  ```go title="example/shape.go"
  --8<-- "example/shape.go:match-def"
  ```

#### `type (...)` convention

A union type is defined as a set of types in a single type declaration. You can think of it as a "one of" type.
To make it more readable, as a convention, I decided to use the `type (...)` declaration block instead of individual `type` declarations.

### Generate code and watch for changes

Run in your terminal to generate union types for your code and watch for changes:
```
mkunion watch ./...
```

This command will:

1. Generate union types and shapes from your code
2. Automatically run `go generate ./...` to trigger any other code generators
3. Continue watching for file changes and repeat the process

To generate unions without watching for changes (one-time generation):
```
mkunion watch -g ./...
```

Alternatively, you can run the `mkunion` command directly on specific files:
```
mkunion -i example/shape.go
```


#### Automatic `go generate` execution
As of the latest version, `mkunion watch` automatically runs `go generate ./...` after generating union types and shapes. This eliminates the need to run two separate commands.

If you need to skip the automatic `go generate` step (for example, if you want to run it manually with specific flags), use the `--dont-run-go-generate` flag:
```
mkunion watch --dont-run-go-generate ./...
# or use the short form
mkunion watch -G ./...
```

This automatic execution works well with extensions like [moq](https://github.com/matryer/moq) that depend on union types being defined first.

### Match over union type
When you run the `mkunion` command, it will generate a file alongside your original file with the `union_gen.go` suffix (example [shape_union_gen.go](https://github.com/widmogrod/mkunion/tree/main/example/shape_union_gen.go)).

You can use these functions to do exhaustive matching on your union type.

For example, you can calculate the area of different shapes with a function that looks like this:
```go title="example/shape.go"
--8<-- "example/shape.go:calculate-area"
```

And as you can see, it leverages generics to make it easy to write.
No need to cast, check types, or use `switch` statements.

#### matching functions `Match{Name}R1`
Where `Name` is the name of your union type.
Where `R0`, `R1`, `R2`, `R3` stand for the number of return values.

Example of `MatchShapeR1` function signature:
```go
func MatchShapeR1[T0 any](
	x Shape,
	f1 func(x *Circle) T0,
	f2 func(x *Rectangle) T0,
	f3 func(x *Square) T0,
) T0 {
	/* ... */
}
```

### JSON marshalling

MkUnion also generates JSON marshalling functions for you.
You just need to use the `shared.JSONMarshal` and `shared.JSONUnmarshal` functions to marshal and unmarshal your union type.

Example:

```go title="example/shape_test.go"
--8<-- "example/shape_test.go:json"
```

You can notice that it has an opinionated way of marshalling and unmarshalling your union type.
It uses the `$type` field to store type information, and then stores the actual data in a separate field with the corresponding name.

You can read more about it in the [Marshaling union in JSON](./examples/json.md) section.

## Next steps

- **[Union and generic types](./examples/generic_union.md)** - Learn about generic unions
- **[Custom Pattern Matching](./examples/custom_pattern_matching.md)** - Learn about custom pattern matching
- **[Marshaling union in JSON](./examples/json.md)** - Learn about marshaling and unmarshalling of union types in JSON
