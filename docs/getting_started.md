# Getting started

### Install mkunion
Run in your terminal
```bash
go install github.com/widmogrod/mkunion/cmd/mkunion@latest
```

### Define your first union type
Create your first union. In our simple example, we will represent different types of vehicles.
But in a more complex example, you may want to represent different states of your application, model domain aggregates, or create your own DSL.
```go title="example/vehicle.go"
package example

//go:tag mkunion:"Vehicle"
type (
	Car struct {
		Color  string
		Wheels int
	}
	Plane struct {
		Color   string
		Engines int
	}
	Boat struct {
		Color      string
		Propellers int
	}
)
```

In the above example, you can see a few important concepts:

#### `//go:tag mkunion:"Vehicle"`

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

- `go:tag mkunion:"Vehicle"` - defines a union type.
- `go:tag serde:"json"` - enables serialization type (currently only JSON is supported), enabled by default.
- `go:tag shape:"-"` - disables shape generation for this type, useful in cases where an x/shared package cannot depend on other x packages, to avoid circular dependencies.
- `go:tag mkunion:",no-type-registry"` - if you want to disable generation of the type registry in a package, define this tag in one of the Go files above the package declaration:
  ```go
  //go:tag mkunion:",no-type-registry"
  package example
  ```
- `go:tag mkmatch:""` - generate custom pattern matching function from interface definition
  ```go title="example/vehicle.go"
  //go:tag mkmatch:"MatchPairs"
  type MatchPairs[A, B Vehicle] interface {
      MatchCars(x, y *Car)
      MatchBoatAny(x *Boat, y any)
      Finally(x, y any)
  }
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

If you want to skip the automatic `go generate` step:
```
mkunion watch -G ./...
# or
mkunion watch --dont-run-go-generate ./...
```

Alternatively, you can run the `mkunion` command directly on specific files:
```
mkunion -i example/vehicle.go
```


#### Automatic `go generate` execution
As of the latest version, `mkunion watch` automatically runs `go generate ./...` after generating union types and shapes. This eliminates the need to run two separate commands.

If you need to skip the automatic `go generate` step (for example, if you want to run it manually with specific flags), use the `--dont-run-go-generate` flag:
```
mkunion watch --dont-run-go-generate ./...
# or use the short form
mkunion watch -G ./...
```

This automatic execution works well with extensions like `moq` that depend on union types being defined first.

### Match over union type
When you run the `mkunion` command, it will generate a file alongside your original file with the `union_gen.go` suffix (example [vehicle_union_gen.go](https://github.com/widmogrod/mkunion/tree/main/example/vehicle_union_gen.go)).

You can use these functions to do exhaustive matching on your union type.

For example, you can calculate fuel usage for different types of vehicles with a function that looks like this:

```go title="example/vehicle.go"
func CalculateFuelUsage(v Vehicle) int {
	return MatchVehicleR1(
		v,
		func(x *Car) int {
			return x.Wheels * 2
		},
		func(x *Plane) int {
			return x.Engines * 10
		},
		func(x *Boat) int {
			return x.Propellers * 5
		},
	)
}
```

And as you can see, it leverages generics to make it easy to write.
No need to cast, check types, or use `switch` statements.

#### matching functions `Match{Name}R1`
Where `Name` is the name of your union type.
Where `R0`, `R1`, `R2`, `R3` stand for the number of return values.

Example of `MatchVehicleR1` function signature:
```go
func MatchVehicleR1[T0 any](
	x Vehicle,
	f1 func(x *Car) T0,
	f2 func(x *Plane) T0,
	f3 func(x *Boat) T0,
) T0 {
	/* ... */
}
```

### JSON marshalling

MkUnion also generates JSON marshalling functions for you.
You just need to use the `shared.JSONMarshal` and `shared.JSONUnmarshal` functions to marshal and unmarshal your union type.

Example:

```go
func ExampleVehicleFromJSON() {
    vehicle := &Car{
        Color:  "black",
        Wheels: 4,
    }
    result, _ := shared.JSONMarshal[Vehicle](vehicle)
    fmt.Println(string(result))
    // Output: {"$type":"example.Car","example.Car":{"Color":"black","Wheels":4}}
}

func ExampleVehicleToJSON() {
    input := []byte(`{"$type":"example.Car","example.Car":{"Color":"black","Wheels":4}}`)
    vehicle, _ := shared.JSONUnmarshal[Vehicle](input)
    fmt.Printf("%#v", vehicle)
    // Output: &example.Car{Color:"black", Wheels:4}
}
```

You can notice that it has an opinionated way of marshalling and unmarshalling your union type.
It uses the `$type` field to store type information, and then stores the actual data in a separate field with the corresponding name.

You can read more about it in the [Marshaling union in JSON](./examples/json.md) section.