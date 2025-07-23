---
title: Value Proposition
---

# MkUnion Value Proposition

## What Are Sum Types?

In type theory, data types can be composed in two fundamental ways:

- **Product Types**: Combine multiple values simultaneously (structs in Go, "AND" relationship)
- **Sum Types**: Represent a choice between variants (missing in Go, "OR" relationship)

Sum types, also known as tagged unions or algebraic data types (ADTs), are a cornerstone of type-safe programming. They allow you to express "this OR that" relationships with compile-time guarantees that all possibilities are handled.

### The Mathematical Inspiration

MkUnion draws inspiration from category theory and algebraic data types found in functional programming languages. While Go's type system doesn't natively support true sum types, MkUnion provides a practical simulation that captures their most useful properties.

#### Theoretical Background

In languages with native sum types, these constructs are mathematical coproducts with specific properties:

- **Sum Types**: Represent "either/or" relationships between types
- **Pattern Matching**: Provides exhaustive case handling
- **Type Safety**: Compile-time guarantees about variant handling

MkUnion emulates these properties through code generation, creating Go interfaces and structs that approximate this behavior

#### Design Philosophy

MkUnion's design is inspired by algebraic principles, but implemented within Go's constraints:

**Conceptual goals** (not formal properties):

 - **Exhaustive handling**: All variants must be handled
 - **Type safety:** No runtime type assertions in generated code
 - **Zero-cost abstraction**: Minimal runtime overhead
 - **Composability**: Unions can be nested and combined

What MkUnion **provides**:

 - ✓ Compile-time exhaustiveness checking (via function signatures)
 - ✓ Type-safe variant access (no manual casting)
 - ✓ Automatic JSON marshalling/unmarshalling
 - ✓ Generated helper functions

What MkUnion **doesn't** provide:

 - ✗ True algebraic data types (Go lacks the type system)
 - ✗ Mathematical properties like semiring laws
 - ✗ Zero-overhead (interface dispatch has cost)
 - ✗ Language-level integration


#### Practical Benefits

MkUnion enables Go developers to use patterns inspired by functional programming:

```go
--8<-- "f/datas.go:result"
--8<-- "f/datas.go:option"

// These provide compile-time safety through exhaustive matching,
// though they're interface-based simulations, not true sum types
```

While languages like Haskell, Rust, and Swift have native sum types, MkUnion brings similar ergonomics to Go through code generation.

## The Problem: Union Types in Go

Go is a powerful language, but it lacks native support for union types (also known as sum types or algebraic data types). This limitation leads developers to use workarounds that have significant drawbacks.

### Traditional Go Approaches

#### 1. The Visitor Pattern

```go
type ShapeVisitor interface {
    VisitCircle(c *Circle)
    VisitRectangle(r *Rectangle)
    VisitTriangle(t *Triangle)
}

type Shape interface {
    Accept(v ShapeVisitor)
}

type Circle struct{ Radius float64 }
func (c *Circle) Accept(v ShapeVisitor) { v.VisitCircle(c) }

type Rectangle struct{ Width, Height float64 }
func (r *Rectangle) Accept(v ShapeVisitor) { v.VisitRectangle(r) }

type Triangle struct{ Base, Height float64 }
func (t *Triangle) Accept(v ShapeVisitor) { v.VisitTriangle(t) }
```


**Characteristics:**

   - **Pro**: 
      - Actually provides compile-time exhaustiveness - adding a new method to the interface requires all implementations to be updated
      - Well-understood pattern in the software engineering community
      - No external dependencies or code generation
   - **Con**:
      - Verbose with boilerplate Accept methods
      - Handling return values requires additional interface methods

#### 2. Iota and Switch Statements

```go title="example/iota.go"
--8<-- "example/iota.go:example"
```

**Characteristics:**

- **Pro**: 
    - Simple and straightforward
    - Familiar to C programmers
- **Con**: 
    - No compile-time exhaustiveness checking
    - Risk of nil pointer dereference if wrong field accessed
    - Requires careful coordination between type field and data fields
    - JSON marshalling requires custom implementation

!!! note
    Projects like [exhaustive](https://github.com/nishanths/exhaustive) with [golangci-lint](https://github.com/golangci/golangci-lint) can detect those situations

#### 3. Interface with Type Assertions

```go
type Shape interface {
    shape() // private method to seal interface
}

type Circle struct{ Radius float64 }
func (Circle) shape() {}

func ProcessShape(s Shape) {
    switch v := s.(type) {
    case *Circle:
        // Process circle
    case *Rectangle:
        // Process rectangle
    default:
        // Handle unknown types gracefully
    }
}
```

**Characteristics:**

- **Pro**: 
    - Most idiomatic Go approach
    - Clean and readable
    - Works well with Go's interface philosophy
    - Static analysis tools can check exhaustiveness
- **Con**: 
    - No compiler-enforced exhaustiveness
    - Requires discipline to handle all cases
    - Each type needs custom JSON marshalling

### How Other Languages Solve This

Let's see how languages with native sum types handle the same problem:

**Rust:**
```rust
enum Shape {
    Circle { radius: f64 },
    Rectangle { width: f64, height: f64 },
    Triangle { base: f64, height: f64 },
}

fn area(shape: &Shape) -> f64 {
    match shape {
        Shape::Circle { radius } => std::f64::consts::PI * radius * radius,
        Shape::Rectangle { width, height } => width * height,
        Shape::Triangle { base, height } => 0.5 * base * height,
        // Compiler error if you miss a case!
    }
}
```

**Haskell:**
```haskell
data Shape = Circle Double
           | Rectangle Double Double
           | Triangle Double Double

area :: Shape -> Double
area (Circle r) = pi * r * r
area (Rectangle w h) = w * h
area (Triangle b h) = 0.5 * b * h
-- Compiler warns about non-exhaustive patterns
```

**Swift:**
```swift
enum Shape {
    case circle(radius: Double)
    case rectangle(width: Double, height: Double)
    case triangle(base: Double, height: Double)
}

func area(of shape: Shape) -> Double {
    switch shape {
    case .circle(let radius):
        return .pi * radius * radius
    case .rectangle(let width, let height):
        return width * height
    case .triangle(let base, let height):
        return 0.5 * base * height
    // Compiler requires exhaustiveness
    }
}
```

**TypeScript:**
```typescript
type Shape = 
    | { kind: 'circle'; radius: number }
    | { kind: 'rectangle'; width: number; height: number }
    | { kind: 'triangle'; base: number; height: number };

function area(shape: Shape): number {
    switch (shape.kind) {
        case 'circle':
            return Math.PI * shape.radius * shape.radius;
        case 'rectangle':
            return shape.width * shape.height;
        case 'triangle':
            return 0.5 * shape.base * shape.height;
        // TypeScript's exhaustiveness checking with strictNullChecks
    }
}
```

Notice the pattern? All these languages provide:

- **Compile-time exhaustiveness checking**
- **Clean, readable syntax**
- **Type-safe variant access**
- **No runtime type assertions**

While Go's simplicity is admirable, the lack of sum types forces developers into error-prone patterns that these other languages avoid entirely.

## The MkUnion Approach

MkUnion generates strongly-typed union types with exhaustive pattern matching, addressing many of these challenges:

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

## Key Benefits

### 1. **Compile-Time Exhaustiveness Checking**

The generated `Match` functions require you to handle every possible case through their function signature. Add a new type to the union? The compiler will force you to update all Match function calls.

```go
// This won't compile if you miss a case!
result := MatchShapeR1(shape,
    func(c *Circle) string { return "circle" },
    func(r *Rectangle) string { return "rectangle" },
    // Compile error: missing Triangle handler
)
```

- Read more about it [Match over union type](getting_started.md#match-over-union-type) and advance topic [Custom Pattern Matching](examples/custom_pattern_matching.md)

### 2. **Reduced Boilerplate**

Tag your types and run the generator. MkUnion creates:

- Union interface with private discriminator
- Constructor functions
- Multiple match function variants (R0, R1, R2 for different return values)
- Visitor pattern implementation
- JSON marshalling/unmarshalling


### 3. **Automatic JSON Marshalling**

```go
shape := &Circle{Radius: 10}
json, _ := shared.JSONMarshal[Shape](shape)
// {"$type":"example.Circle","example.Circle":{"Radius":10}}

decoded, _ := shared.JSONUnmarshal[Shape](json)
// Returns the correct concrete type
```

- Read more about it [Marshaling union as JSON](examples/json.md)

### 4. **Generic Support**

MkUnion fully supports Go generics:

```go
//go:tag mkunion:"Result"
type (
    Success[T any] struct{ Value T }
    Failure[T any] struct{ Error error }
)

// Use with any type
var result Result[string] = &Success[string]{Value: "hello"}
```

 - Read more about it [Union and generic types](examples/generic_union.md)

### 5. **TypeScript Generation**

Generate TypeScript types for end-to-end type safety:

```bash
mkunion shape-export --language typescript --output-dir ./ts
```

Creates matching TypeScript discriminated unions that work seamlessly with your Go API.

 - Read more about it [End-to-End types between Go and TypeScript](examples/type_script.md)


## Conclusion

MkUnion provides a code generation approach to simulating algebraic data types in Go. It offers compile-time exhaustiveness checking and automatic JSON marshalling at the cost of added build complexity and deviation from idiomatic Go.

Whether MkUnion is right for your project depends on your specific needs, team expertise, and tolerance for code generation. 
For teams comfortable with these trade-offs who need exhaustive pattern matching, MkUnion can be a valuable tool. 
For others, traditional Go patterns with modern linting tools may be more appropriate.