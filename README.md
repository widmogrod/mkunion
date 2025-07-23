# Welcome to MkUnion
[![Go Reference](https://pkg.go.dev/badge/github.com/widmogrod/mkunion.svg)](https://pkg.go.dev/github.com/widmogrod/mkunion)
[![Go Report Card](https://goreportcard.com/badge/github.com/widmogrod/mkunion)](https://goreportcard.com/report/github.com/widmogrod/mkunion)
[![codecov](https://codecov.io/gh/widmogrod/mkunion/branch/main/graph/badge.svg?token=3Z3Z3Z3Z3Z)](https://codecov.io/gh/widmogrod/mkunion)

```bash
go install github.com/widmogrod/mkunion/cmd/mkunion@latest
```

```
mkunion watch -g ./...
```

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

## Examples

### Example 1: Union definition and patter matching with JSON marshaling

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

### Example 2: Result Type (Error Handling)

```go title="f/datas.go"
//go:tag mkunion:"Result"
type (
    Ok[T any] struct{ Value T }
    Err[T any] struct{ Error error }
)
```

### Example 3: AST (Recursive Types)

```go title="example/calculator_example.go"
//go:tag mkunion:"Calc"
type (
    Lit struct{ V int }
    Sum struct{ Left, Right Calc }
    Mul struct{ Left, Right Calc }
)
```

### Example 4: State Machine (Business Logic)

```go
//go:tag mkunion:"OrderState"
type (
    Draft struct{ Items []Item }
    Submitted struct{ OrderID string; Items []Item }
    Shipped struct{ OrderID string; TrackingNumber string }
    Delivered struct{ OrderID string; DeliveredAt time.Time }
)
```

### Example 5: HTTP API (Real-world)

```go
//go:tag mkunion:"APIResponse"
type (
    Success[T any] struct{ Data T; Status int }
    ValidationError[T any] struct{ Errors []string }
    ServerError[T any] struct{ Message string; Code string }
)
```

### Example 6: Configuration (Practical)
```go
//go:tag mkunion:"Config"  
type (
    FileConfig struct{ Path string }
    EnvConfig struct{ Prefix string }
    DefaultConfig struct{}
)
```


## Next

- Read [getting started](https://widmogrod.github.io/mkunion/getting_started/) to learn more.
