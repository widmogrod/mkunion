---
title: Shape Package
---

# x/shape - Type Introspection and Representation

The `x/shape` package is the foundation of mkunion's type system. It provides a comprehensive framework for representing, introspecting, and transforming Go types at both compile-time and runtime. This enables powerful features like cross-language code generation, runtime type checking, and automatic serialization.

## Overview

The shape package offers:
- **Type representation** - Universal representation of Go types as shapes
- **Shape inference** - Extract shapes from Go code, reflection, or runtime values
- **Cross-language generation** - Generate TypeScript, JSON Schema, and more
- **Type introspection** - Query and analyze type structures
- **Registry system** - Runtime and compile-time type discovery
- **Validation support** - Extract validation rules from struct tags

## Core Concepts

### What is a Shape?

A shape is a data structure that represents the structure of a Go type. It captures:
- Type kind (struct, slice, map, primitive, etc.)
- Field names and types for structs
- Type parameters for generics
- Package information for imports
- Validation rules and documentation

### The Shape Union Type

```go
//go:tag mkunion:"Shape"
type (
    // Any represents interface{} or unknown types
    Any struct{}
    
    // RefName represents named type references
    RefName struct {
        Name          string      // Type name
        PkgName       string      // Package name
        PkgImportName string      // Import path
        Indexed       []Shape     // Generic type parameters
    }
    
    // PointerLike represents pointer types
    PointerLike struct {
        Type Shape
    }
    
    // PrimitiveLike represents built-in types
    PrimitiveLike struct {
        Kind PrimitiveKind // bool, string, int, etc.
    }
    
    // ListLike represents slices and arrays
    ListLike struct {
        Element Shape
        ArrayLen *int // nil for slices, value for arrays
    }
    
    // MapLike represents map types
    MapLike struct {
        Key   Shape
        Val   Shape
    }
    
    // StructLike represents struct types
    StructLike struct {
        Name       string
        PkgName    string
        PkgImportName string
        Fields     []StructField
        TypeParams []TypeParam
    }
    
    // UnionLike represents union types (mkunion tagged)
    UnionLike struct {
        Name       string
        PkgName    string
        PkgImportName string
        Variants   []Variant
        TypeParams []TypeParam
    }
)
```

## Getting Started

### Inferring Shapes

```go
import "github.com/widmogrod/mkunion/x/shape"

// From runtime values
type User struct {
    Name  string `json:"name" required:"true"`
    Email string `json:"email"`
    Age   int    `json:"age" enum:"18,21,25,30"`
}

userShape := shape.FromGo(User{})
// Returns: StructLike with fields and tags

// From reflection
userType := reflect.TypeOf(User{})
userShape := shape.FromGoReflect(userType, shape.WithTag("json"))

// From source code
shapes, err := shape.InferFromFile("user.go")
```

### Working with Shapes

```go
// Pattern matching on shapes
typeName := shape.MatchShapeR1(
    myShape,
    func(x *shape.Any) string { return "any" },
    func(x *shape.RefName) string { return x.Name },
    func(x *shape.PointerLike) string { return "*" + shape.Name(x.Type) },
    func(x *shape.PrimitiveLike) string { return x.Kind.String() },
    func(x *shape.ListLike) string { return "[]" + shape.Name(x.Element) },
    func(x *shape.MapLike) string { 
        return "map[" + shape.Name(x.Key) + "]" + shape.Name(x.Val)
    },
    func(x *shape.StructLike) string { return x.Name },
    func(x *shape.UnionLike) string { return x.Name },
)

// Check type properties
if shape.IsString(myShape) {
    // Handle string type
}

if shape.IsUnion(myShape) {
    // Handle union type
}

// Extract struct fields
if s, ok := shape.ToStructLike(myShape); ok {
    for _, field := range s.Fields {
        fmt.Printf("Field: %s, Type: %s\n", field.Name, shape.Name(field.Type))
    }
}
```

## Advanced Features

### Generic Type Support

```go
// Define a generic type
type Result[T any] struct {
    Value T
    Error error
}

// Infer shape with type parameter
resultShape := shape.FromGo(Result[string]{})

// Access type parameters
if s, ok := shape.ToStructLike(resultShape); ok {
    fmt.Printf("Type params: %v\n", s.TypeParams)
    // Output: [{T 0}]
}
```

### Validation Tag Support

```go
type Config struct {
    Port     int    `json:"port" required:"true" enum:"8080,8443,9000"`
    Hostname string `json:"hostname" required:"true"`
    Debug    bool   `json:"debug"`
}

// Shape inference captures validation tags
configShape := shape.FromGo(Config{}, shape.WithTag("json"))

// Access validation rules
if s, ok := shape.ToStructLike(configShape); ok {
    for _, field := range s.Fields {
        if field.Tags != nil {
            fmt.Printf("Field %s: required=%v, enum=%v\n",
                field.Name,
                field.Tags["required"],
                field.Tags["enum"])
        }
    }
}
```

### Cross-Language Generation

#### TypeScript Generation

```go
// Convert shape to TypeScript
tsCode := shape.ToTypeScript(userShape, shape.TSGenConfig{
    PackageImports: map[string]string{
        "github.com/example/types": "./types",
    },
})

// Output:
// export interface User {
//   name: string;
//   email: string;
//   age: number;
// }
```

#### JSON Schema Generation

```go
// Convert to JSON Schema
schema := shape.ToJSONSchema(userShape)

// Output:
// {
//   "type": "object",
//   "properties": {
//     "name": {"type": "string"},
//     "email": {"type": "string"},
//     "age": {"type": "integer", "enum": [18,21,25,30]}
//   },
//   "required": ["name"]
// }
```

### Shape Registry

```go
// Register shapes at runtime
shape.Register[User]()
shape.Register[Product]()

// Lookup by type name
userShape, found := shape.LookupShape("User")

// Compile-time lookup from disk
userShape, err := shape.LookupShapeOnDisk("User", shape.SearchOptions{
    CurrentPkgPath: "github.com/example/app",
    GoModPath:      "/path/to/go.mod",
})

// Find all shapes in a package
shapes, err := shape.LookupPkgShapeOnDisk("github.com/example/types")
```

## Shape Utilities

### Type Checking

```go
// Primitive type checks
shape.IsString(s)      // string type
shape.IsInt(s)         // int types
shape.IsFloat(s)       // float types
shape.IsBool(s)        // bool type
shape.IsBinary(s)      // []byte type

// Complex type checks
shape.IsPointer(s)     // pointer type
shape.IsList(s)        // slice or array
shape.IsMap(s)         // map type
shape.IsStruct(s)      // struct type
shape.IsUnion(s)       // union type (mkunion)
```

### Type Extraction

```go
// Get type name
name := shape.Name(s)

// Get package info
pkgName := shape.ToGoPkgName(s)
importPath := shape.ToGoPkgImportName(s)

// Extract all type references
refs := shape.ExtractRefs(s)
for _, ref := range refs {
    fmt.Printf("Referenced type: %s from %s\n", ref.Name, ref.PkgImportName)
}

// Get struct fields
fields := shape.GetStructFields(s)
for _, field := range fields {
    fmt.Printf("Field: %s, Type: %s\n", field.Name, shape.Name(field.Type))
}
```

### Shape Transformation

```go
// Walk and transform shapes
transformed := shape.MapShape(originalShape, func(s shape.Shape) shape.Shape {
    // Transform pointers to their underlying type
    if p, ok := shape.ToPointerLike(s); ok {
        return p.Type
    }
    return s
})

// Replace type references
replaced := shape.ReplaceRefs(originalShape, map[string]shape.Shape{
    "OldType": shape.MkRefName("NewType", "pkg", "github.com/example/pkg"),
})
```

## Integration Examples

### With Code Generation

```go
// Generate shape functions for types
func generateShapeFunction(typeName string, s shape.Shape) string {
    return fmt.Sprintf(`
func %sShape() shape.Shape {
    return %s
}`, typeName, shape.ToGoCode(s))
}
```

### With Validation

```go
// Extract validation rules from shape
func extractValidation(s shape.Shape) ValidationRules {
    rules := ValidationRules{}
    
    if structShape, ok := shape.ToStructLike(s); ok {
        for _, field := range structShape.Fields {
            if field.Tags["required"] == "true" {
                rules.Required = append(rules.Required, field.Name)
            }
            if enum := field.Tags["enum"]; enum != "" {
                rules.Enums[field.Name] = strings.Split(enum, ",")
            }
        }
    }
    
    return rules
}
```

### With OpenAPI Generation

```go
// Convert shapes to OpenAPI schemas
func toOpenAPISchema(s shape.Shape) openapi.Schema {
    return shape.MatchShapeR1(s,
        func(*shape.Any) openapi.Schema {
            return openapi.Schema{Type: "object"}
        },
        func(x *shape.PrimitiveLike) openapi.Schema {
            return openapi.Schema{Type: primitiveToOpenAPI(x.Kind)}
        },
        func(x *shape.StructLike) openapi.Schema {
            schema := openapi.Schema{
                Type:       "object",
                Properties: make(map[string]openapi.Schema),
            }
            for _, field := range x.Fields {
                schema.Properties[field.Name] = toOpenAPISchema(field.Type)
            }
            return schema
        },
        // ... other cases
    )
}
```

## Best Practices

### 1. Use Shape Inference Appropriately

```go
// For compile-time known types
shape := UserShape() // Generated function

// For runtime type discovery
shape := shape.FromGo(value)

// For source code analysis
shapes, _ := shape.InferFromFile("types.go")
```

### 2. Handle Shape Variants

```go
// Always handle all shape variants
shape.MatchShapeR1(s,
    func(*shape.Any) Result { /* handle any */ },
    func(*shape.RefName) Result { /* handle ref */ },
    func(*shape.PointerLike) Result { /* handle pointer */ },
    // ... handle all variants
)
```

### 3. Cache Shape Lookups

```go
var shapeCache = make(map[string]shape.Shape)

func getCachedShape(typeName string) shape.Shape {
    if s, ok := shapeCache[typeName]; ok {
        return s
    }
    s, _ := shape.LookupShapeOnDisk(typeName, options)
    shapeCache[typeName] = s
    return s
}
```

## Performance Considerations

1. **AST Parsing**: File inference parses Go source, cache results
2. **Reflection**: FromGoReflect is slower than generated shapes
3. **Registry Lookups**: Runtime lookups have overhead
4. **Deep Shapes**: Recursive types need cycle detection

## Troubleshooting

### Common Issues

1. **Cyclic Types**: Use depth limits for recursive types
2. **Missing Imports**: Ensure all referenced types are accessible
3. **Generic Inference**: Complex generics may need explicit handling
4. **Tag Parsing**: Verify struct tag syntax

### Debugging

```go
// Enable shape debugging
shape.Debug = true

// Pretty print shapes
fmt.Printf("Shape: %s\n", shape.ToString(s))

// Validate shape consistency
if err := shape.Validate(s); err != nil {
    log.Printf("Invalid shape: %v", err)
}
```

## API Stability

The shape package is fundamental to mkunion and has a stable API. Key functions like `FromGo`, `MatchShapeR1`, and the `Shape` union type are unlikely to change. New features are added in a backward-compatible manner.

## Further Reading

- [Type System Design](../development/type_system.md)
- [Code Generation](./generators.md)
- [TypeScript Integration](../examples/type_script.md)
- [JSON Schema](../examples/json_schema.md)