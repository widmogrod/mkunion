# Welcome to MkUnion
[![Go Reference](https://pkg.go.dev/badge/github.com/widmogrod/mkunion.svg)](https://pkg.go.dev/github.com/widmogrod/mkunion)
[![Go Report Card](https://goreportcard.com/badge/github.com/widmogrod/mkunion)](https://goreportcard.com/report/github.com/widmogrod/mkunion)
[![codecov](https://codecov.io/gh/widmogrod/mkunion/branch/main/graph/badge.svg?token=3Z3Z3Z3Z3Z)](https://codecov.io/gh/widmogrod/mkunion)


## About
Strongly typed **union type** in golang that supports generics*.

* with exhaustive _pattern matching_ support
* with _json marshalling_ including generics
* and as a bonus, can generate compatible TypeScript types for end-to-end type safety in your application

## Why
Historically, in languages like Go that lack native union types, developers have resorted to workarounds such as the Visitor pattern or `iota` with `switch` statements.

The Visitor pattern requires a lot of boilerplate code and manual crafting of the `Accept` method for each type in the union.
Using `iota` and `switch` statements is not type-safe and can lead to runtime errors, especially when a new type is added and not all `case` statements are updated.

On top of that, any data marshalling, like to/from JSON, requires additional, handcrafted code to make it work.

MkUnion solves all of these problems by generating opinionated and strongly typed, meaningful code for you.

## Example

```go title="example/vehicle.go"
package example

//go:generate mkunion

// union declaration
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

func CalculateFuelUsage(v Vehicle) int {
	// example of pattern matching over Vehicle union type
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

func ExampleToJSON() {
    var vehicle Vehicle = &Car{
        Color:  "black",
        Wheels: 4,
    }
    result, _ := shared.JSONMarshal(vehicle)
    fmt.Println(string(result))
    // Output: {"$type":"example.Car","example.Car":{"Color":"black","Wheels":4}}
}

func ExampleFromJSON() {
	input := []byte(`{"$type":"example.Car","example.Car":{"Color":"black","Wheels":4}}`)
	vehicle, _ := shared.JSONUnmarshal[Vehicle](input)
	fmt.Printf("%#v", vehicle)
	// Output: &example.Car{Color:"black", Wheels:4}
}
```


## Next

- Read [getting started](https://widmogrod.github.io/mkunion/getting_started/) to learn more.
- Or to understand better concepts jump and read [value proposition](./docs/value_proposition.md)

## Extended Packages

MkUnion includes powerful extended packages in the `x/` directory:

- **[State Machines](https://widmogrod.github.io/mkunion/x/machine/)** - Type-safe state machine framework with exhaustive pattern matching
- **[Storage](https://widmogrod.github.io/mkunion/x/storage/)** - Schemaless storage with union type support
- **[Workflows](https://widmogrod.github.io/mkunion/x/workflow/)** - Workflow orchestration engine
- **[Projections](https://widmogrod.github.io/mkunion/x/projection/)** - Event processing and stream analytics
- **[Code Generation](https://widmogrod.github.io/mkunion/x/generators/)** - Extensible code generation framework
- **[Type System](https://widmogrod.github.io/mkunion/x/shape/)** - Runtime type introspection

See the [full documentation](https://widmogrod.github.io/mkunion/x/readme/) for all available packages.