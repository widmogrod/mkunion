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

### Example 1: Union definition and pattern matching with JSON marshaling

```go title="example/shape.go"
package example

//go:tag mkunion:"Shape"
type (
    Circle struct {
        Radius float64
    }
    Rectangle struct {
        Width  float64
        Height float64
    }
    Square struct {
        Side float64
    }
)

func CalculateArea(s Shape) float64 {
    // example of pattern matching over Shape union type
    return MatchShapeR1(
        s,
        func(x *Circle) float64 {
            return math.Pi * x.Radius * x.Radius
        },
        func(x *Rectangle) float64 {
            return x.Width * x.Height
        },
        func(x *Square) float64 {
            return x.Side * x.Side
        },
    )
}

func ExampleToJSON() {
    var shape Shape = &Circle{
        Radius: 10,
    }
    result, _ := shared.JSONMarshal(shape)
    fmt.Println(string(result))
    // Output: {"$type":"example.Circle","example.Circle":{"Radius":10}}
}

func ExampleFromJSON() {
    input := []byte(`{"$type":"example.Circle","example.Circle":{"Radius":10}}`)
    shape, _ := shared.JSONUnmarshal[Shape](input)
    fmt.Printf("%#v", shape)
    // Output: &example.Circle{Radius:10}
```

### Example 2: Result Type for Error Handling

```go title="f/datas.go"
//go:tag mkunion:"Result[T, E]"
type (
    Ok[T any, E any] struct{ Value T }
    Err[T any, E any] struct{ Error E }
)
```

**Important:** Generic unions MUST specify their type parameters in the tag. The type parameter names in the tag must match those used in the variant types.

### Example 3: AST and Recursive Types

```go title="example/calculator_example.go"
//go:tag mkunion:"Calc"
type (
    Lit struct{ V int }
    Sum struct{ Left, Right Calc }
    Mul struct{ Left, Right Calc }
)
```

### Example 4: States for State Machines or Events for Event Sourcing

```go
//go:tag mkunion:"OrderState"
type (
    Draft struct{ Items []Item }
    Submitted struct{ OrderID string; Items []Item }
    Shipped struct{ OrderID string; TrackingNumber string }
    Delivered struct{ OrderID string; DeliveredAt time.Time }
)
```

### Example 5: HTTP API responses

```go
//go:tag mkunion:"APIResponse[T]"
type (
    Success[T any] struct{ Data T; Status int }
    ValidationError[T any] struct{ Errors []string }
    ServerError[T any] struct{ Message string; Code string }
)
```

### Example 6: Configuration type
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
- Learn more about [value proposition](https://widmogrod.github.io/mkunion/value_proposition/).
