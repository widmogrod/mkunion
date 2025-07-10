---
title: Generators Package
---

# x/generators - Code Generation Framework

The `x/generators` package powers mkunion's code generation capabilities. It provides a suite of generators that create union types, pattern matching functions, serialization code, and type introspection utilities from your Go source code.

## Overview

The generators package offers:
- **Union type generation** - Creates visitor patterns and constructors
- **Pattern matching** - Generates exhaustive match functions
- **JSON serialization** - Automatic marshalling/unmarshalling with generics support
- **Shape functions** - Runtime type introspection
- **Type registry** - Polymorphic JSON support
- **Extensible framework** - Easy to add custom generators

## Core Concepts

### Generator Interface

All generators implement a simple interface:

```go
type Generator interface {
    Generate() ([]byte, error)
}
```

### Generation Tags

Control code generation with Go tags:

```go
// Union type generation
//go:tag mkunion:"Vehicle"
type (
    Car   struct { Wheels int }
    Plane struct { Wings int }
)

// Pattern matching
//go:tag mkmatch:"Matcher"
type Matcher interface {
    // Match function signatures
}

// Shape generation
//go:tag shape:"+"
type MyType struct { /* ... */ }

// JSON serialization
//go:tag serde:"json"
type Config struct { /* ... */ }
```

## Getting Started

### Using the mkunion Tool

```bash
# Install mkunion
go install github.com/widmogrod/mkunion/cmd/mkunion@latest

# Generate code for current directory
mkunion watch ./...

# One-time generation
mkunion watch -g ./...

# Generate without running go generate
mkunion watch -G ./...
```

### Basic Union Type

```go
// vehicle.go
package example

//go:tag mkunion:"Vehicle"
type (
    Car struct {
        Model  string
        Wheels int
    }
    Plane struct {
        Model   string
        Engines int
    }
    Boat struct {
        Model string
        Sails int
    }
)
```

This generates:
- `vehicle_union_gen.go` - Union type and constructors
- `vehicle_shape_gen.go` - Shape functions for introspection
- `vehicle_serde_gen.go` - JSON marshalling/unmarshalling
- Pattern matching functions

## Generated Code Examples

### Union Type Code

```go
// Generated union type
type Vehicle interface {
    AcceptVehicle(visitor VehicleVisitor) any
}

// Generated visitor interface
type VehicleVisitor interface {
    VisitCar(v *Car) any
    VisitPlane(v *Plane) any
    VisitBoat(v *Boat) any
}

// Generated constructors
func MkVehicleCar(x *Car) Vehicle { return x }
func MkVehiclePlane(x *Plane) Vehicle { return x }
func MkVehicleBoat(x *Boat) Vehicle { return x }

// Accept methods
func (x *Car) AcceptVehicle(visitor VehicleVisitor) any {
    return visitor.VisitCar(x)
}
```

### Pattern Matching

```go
// Generated match functions
func MatchVehicleR1[T any](
    x Vehicle,
    f1 func(x *Car) T,
    f2 func(x *Plane) T,
    f3 func(x *Boat) T,
) T {
    switch v := x.(type) {
    case *Car:
        return f1(v)
    case *Plane:
        return f2(v)
    case *Boat:
        return f3(v)
    }
    panic("unreachable")
}

// Usage
fuelType := MatchVehicleR1(vehicle,
    func(c *Car) string { return "gasoline" },
    func(p *Plane) string { return "jet fuel" },
    func(b *Boat) string { return "diesel" },
)
```

### JSON Serialization

```go
// Generated JSON methods
func (x *Car) MarshalJSON() ([]byte, error) {
    return json.Marshal(struct {
        Type string `json:"$type"`
        Car  *Car   `json:"example.Car"`
    }{
        Type: "example.Car",
        Car:  x,
    })
}

func (x *Car) UnmarshalJSON(data []byte) error {
    // Implementation...
}

// Usage
jsonData, _ := json.Marshal(vehicle)
// {"$type":"example.Car","example.Car":{"Model":"Tesla","Wheels":4}}
```

### Shape Functions

```go
// Generated shape function
func CarShape() shape.Shape {
    return shape.MkStruct(
        shape.MkField("Model", shape.MkPrimitive(shape.Primitive_string)),
        shape.MkField("Wheels", shape.MkPrimitive(shape.Primitive_int)),
    )
}

// Usage for introspection
carShape := CarShape()
fields := shape.GetStructFields(carShape)
```

## Advanced Features

### Generic Union Types

```go
//go:tag mkunion:"Result[T]"
type (
    Success[T any] struct { Value T }
    Failure[T any] struct { Error error }
)

// Generated code handles generics
func MatchResultR1[T, R any](
    x Result[T],
    f1 func(x *Success[T]) R,
    f2 func(x *Failure[T]) R,
) R {
    // Implementation
}
```

### Custom Pattern Matching

```go
//go:tag mkmatch:"Calculator"
type Calculator interface {
    Calculate(left Expr, op string, right Expr) (float64, error)
}

// Generates specialized match functions based on interface
```

### Type Registry

The type registry enables polymorphic JSON marshalling:

```go
// Automatically generated registration
func init() {
    shared.TypeRegistry.Register("example.Car", func() any { return &Car{} })
    shared.TypeRegistry.Register("example.Plane", func() any { return &Plane{} })
}

// Enables unmarshalling to interface types
var vehicle Vehicle
err := json.Unmarshal(jsonData, &vehicle)
```

## Available Generators

### VisitorGenerator

Generates visitor pattern implementation:
- Creates visitor interface with Visit* methods
- Implements Accept method on each variant
- Generates Match functions for pattern matching

Configuration:
```go
gen := &VisitorGenerator{
    TypeName: "Vehicle",
    Types:    []string{"Car", "Plane", "Boat"},
    Package:  "example",
}
```

### SerdeJSONUnion

Generates JSON marshalling for union types:
- Handles generic type parameters
- Creates type-tagged JSON format
- Supports custom type registry

### ShapeTagged

Generates shape functions for type introspection:
- Creates *Shape() functions for each type
- Supports all Go types (structs, slices, maps, etc.)
- Handles recursive and generic types

### MkMatchGenerator

Generates pattern matching from interfaces:
- Based on `//go:tag mkmatch` annotations
- Creates exhaustive matching functions
- Supports multiple return values

## Customizing Generation

### Skip Imports and Package

```go
gen := &VisitorGenerator{
    // ... configuration
}
gen.SkipImportsAndPackage(true) // For generating snippets
```

### Custom Import Management

```go
gen.SetPkgMap(map[string]string{
    "github.com/widmogrod/mkunion/x/shape": "shape",
    "encoding/json": "json",
})
```

### Disable Type Registry

```go
//go:tag mkunion:"Vehicle,no-type-registry"
```

## Creating Custom Generators

Implement the Generator interface:

```go
type MyGenerator struct {
    TypeName string
    // ... configuration
}

func (g *MyGenerator) Generate() ([]byte, error) {
    var buf bytes.Buffer
    
    // Generate code
    fmt.Fprintf(&buf, "// Generated code for %s\n", g.TypeName)
    
    // Format code
    return format.Source(buf.Bytes())
}
```

### Integration with mkunion

Add your generator to the pipeline:

```go
// In your generation tool
generators := []Generator{
    &MyGenerator{TypeName: "Example"},
    // ... other generators
}

for _, gen := range generators {
    code, err := gen.Generate()
    // Write to file
}
```

## Best Practices

### 1. Consistent Naming

```go
// Good: Clear, descriptive names
//go:tag mkunion:"PaymentMethod"
type (
    CreditCard struct { /* ... */ }
    BankTransfer struct { /* ... */ }
)

// Avoid: Unclear abbreviations
//go:tag mkunion:"PM"
```

### 2. Group Related Types

```go
// Good: Related types in one union
//go:tag mkunion:"AuthEvent"
type (
    LoginAttempt struct { /* ... */ }
    LoginSuccess struct { /* ... */ }
    LoginFailure struct { /* ... */ }
    Logout struct { /* ... */ }
)
```

### 3. Use Meaningful Field Names

```go
// Good: Self-documenting fields
type UserCreated struct {
    UserID    string
    Email     string
    CreatedAt time.Time
}

// Avoid: Cryptic names
type UC struct {
    U string
    E string
    T time.Time
}
```

### 4. Document Generated Code

```go
// Vehicle represents different types of vehicles.
// This is a union type generated by mkunion.
//
//go:tag mkunion:"Vehicle"
type (
    // Car represents a road vehicle
    Car struct {
        Model  string
        Wheels int
    }
    // ... other variants
)
```

## Performance Considerations

1. **Generated Code Size**: Large unions generate more code
2. **Compilation Time**: Many generators increase build time
3. **Runtime Performance**: Generated code is optimized for performance
4. **Type Registry**: Small overhead for polymorphic JSON

## Troubleshooting

### Common Issues

1. **Import Cycles**: Use `SkipImportsAndPackage` for partial generation
2. **Missing Imports**: Ensure all types are properly imported
3. **Generic Constraints**: Some complex generics may not be supported
4. **Tag Syntax**: Verify correct tag format

### Debugging Generation

```bash
# Verbose output
mkunion watch -v ./...

# Check generated files
ls *_gen.go

# Verify imports
go mod tidy
```

## Integration with Build Tools

### go generate

```go
//go:generate mkunion
```

### Makefile

```makefile
generate:
    mkunion watch -g ./...

watch:
    mkunion watch ./...
```

### CI/CD

```yaml
- name: Generate code
  run: |
    go install github.com/widmogrod/mkunion/cmd/mkunion@latest
    mkunion watch -g ./...
    
- name: Check generated code
  run: |
    git diff --exit-code
```

## Future Enhancements

The generators framework is evolving to support:
- Protocol buffer generation
- OpenAPI schema generation
- GraphQL type generation
- Database schema generation
- More language targets (Rust, Swift, etc.)

## Further Reading

- [Getting Started Guide](../getting_started.md)
- [Union Types in Go](../examples/union_types.md)
- [x/shape Package](./shape.md)
- [Code Generation Best Practices](https://go.dev/blog/generate)