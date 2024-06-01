# Getting started

### Install mkunion
Run in your terminal
```bash
go install github.com/widmogrod/mkunion/cmd/mkunion@latest
```

### Define your first union type
Create your first union. In our simple example we will represent different types of vehicles.
But in a more complex example you may want to represent different states of your application, model domain aggregates, or create your own DSL.
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

In above example you can see a few important concepts:

#### `//go:tag mkunion:"Vehicle"`

Tag are powerful and flexible way to add metadata to your code.
You may be familiar with tags when you work with JSON in golang

```go
type User struct {
	Name string `json:"name"`
}
```

Unfortunately Golang don't extend this feature to other parts of the language.

MkUnion defines `//go:tag` comment, following other idiomatic definitions `go:generate`, `go:embed` to allow to add metadata to struct type.
And MkUnion use it heavily to offer way of adding new behaviour to go types.

##### Tags supported by MkUnion

- `go:tag mkunion:"Vehicle"` - define union type
- `go:tag serde:"json"` - enable serialisation type (currently only JSON is supported), enbabled by default
- `go:tag shape:"-"` - disable shape generation for this type, useful in cases x/shared package cannot depend on other x packages, to avid circular dependencies
- `go:tag mkunion:",no-type-registry"` - if you want to disable generation type registry in a package, in one of go files above package declaration define tag
```go
//go:tag mkunion:",no-type-registry"
package example
```

#### `type (...)` convention

Union type is defined as a set of types in a single type declaration. You can think of it as "one of" type.
To make it more readable, as convention I decided to use `type (...)` declaration block, instead of individual `type` declaration.

### Generate code and watch for changes

Run in your terminal to generate union type for your code and watch for changes
```
mkunion watch ./...
```

To generate unions without watching for changes, you can run
```
mkunion watch -g ./...
```

Alternatively you can run `mkunion` command directly
```
mkunion -i example/vehicle.go
```

### Match over union type
When you run `mkunion` command, it will generate file alongside your original file with `union_gen.go` suffix (example [vehicle_union_gen.go](https://github.com/widmogrod/mkunion/tree/main/example/vehicle_union_gen.go))

You can use those function to do exhaustive matching on your union type.

For example, you can calculate fuel usage for different types of vehicles, with function that looks like this:

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

And as you can see, it leverage generics to make it easy to write.
No need to cast, or check type, or use `switch` statement.

#### matching functions `Match{Name}R1`
Where `Name` is the name of your union type.
Where `R0`, `R1`, `R2`, `R3` stands for number of return values.

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

MkUnion also generate JSON marshalling functions for you.
You just need to use `shared.JSONMarshal` and `shared.JSONUnmarshal` functions to marshal and unmarshal your union type.

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

You can notice that it has opinionated way of marshalling and unmarshalling your union type.
It uses `$type` field to store type information, and then store actual data in separate field, with corresponding name.

You can read more about it in [Marshaling union in JSON](./examples/json.md) section.