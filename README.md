# Welcome to MkUnion
[![Go Reference](https://pkg.go.dev/badge/github.com/widmogrod/mkunion.svg)](https://pkg.go.dev/github.com/widmogrod/mkunion)
[![Go Report Card](https://goreportcard.com/badge/github.com/widmogrod/mkunion)](https://goreportcard.com/report/github.com/widmogrod/mkunion)
[![codecov](https://codecov.io/gh/widmogrod/mkunion/branch/main/graph/badge.svg?token=3Z3Z3Z3Z3Z)](https://codecov.io/gh/widmogrod/mkunion)

```bash
go install github.com/widmogrod/mkunion/cmd/mkunion@v1.26.0
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

```go
//go:tag mkunion:"Shape"
type (
    Circle struct{ Radius float64 }
    Rectangle struct{ Width, Height float64 }
    Triangle struct{ Base, Height float64 }
)

// Generated code provides:
area := MatchShapeR1(
    shape,
    func(c *Circle) float64 { return math.Pi * c.Radius * c.Radius },
    func(r *Rectangle) float64 { return r.Width * r.Height },
    func(t *Triangle) float64 { return 0.5 * t.Base * t.Height },
)
```

### Example 2: Result Type for Error Handling

```go
//go:tag mkunion:"Option[T]"
type (
	None[T any] struct{}
	Some[T any] struct{ Value T }
)

//go:tag mkunion:"Result[T, E]"
type (
	Ok[T, E any]  struct{ Value T }
	Err[T, E any] struct{ Error E }
)

type User struct{ Name string }

type APIError struct {
	Code    int
	Message string
}

// FetchResult combine unions for rich error handling
type FetchResult = Result[Option[User], APIError]

// handleFetch uses nested pattern matching to handle result
func handleFetch(result FetchResult) string {
	return MatchResultR1(result,
		func(ok *Ok[Option[User], APIError]) string {
			return MatchOptionR1(ok.Value,
				func(*None[User]) string { return "User not found" },
				func(some *Some[User]) string {
					return fmt.Sprintf("Found user: %s", some.Value.Name)
				},
			)
		},
		func(err *Err[Option[User], APIError]) string {
			return fmt.Sprintf("API error: %v", err.Error)
		},
	)
}

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
