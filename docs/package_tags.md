# Package-Level Tags

Package-level `go:tag` annotations allow you to configure mkunion behavior and store metadata at the package level. These tags are placed before the `package` declaration and affect the entire package.

## Syntax

Package-level tags use the same syntax as type-level tags:

```go
//go:tag tagname:"value,option1,option2"
//go:tag anothertag:"anothervalue"
package mypackage
```

## Built-in Package Tags

### mkunion tag

The `mkunion` tag at package level supports these options:

#### no-type-registry

Disables type registry generation for the entire package:

```go
//go:tag mkunion:",no-type-registry"
package mypackage
```

This is equivalent to adding `//go:tag mkunion:",no-type-registry"` to each union type in the package, but more convenient for packages that never need type registry.

## Custom Package Tags

You can define custom package tags for your own purposes:

```go
//go:tag version:"1.2.3,stable,production"
//go:tag author:"John Doe"
//go:tag license:"MIT"
//go:tag module:"mypackage"
//go:tag build:"debug,verbose"
package mypackage
```

Custom tags are useful for:
- Build tool configuration
- Documentation generation
- Package metadata
- Custom code generation tools

## Extracting Package Tags

mkunion provides several functions to extract package-level tags:

### ExtractPackageTagsFromFile

Extract tags from a specific Go file:

```go
package main

import (
    "fmt"
    "log"
    "github.com/widmogrod/mkunion/x/shape"
)

func main() {
    tags, err := shape.ExtractPackageTagsFromFile("main.go")
    if err != nil {
        log.Fatal(err)
    }
    
    for tagName, tag := range tags {
        fmt.Printf("%s: %s %v\n", tagName, tag.Value, tag.Options)
    }
}
```

### ExtractPackageTagsFromDir

Extract tags from all Go files in a directory (they should all have the same package-level tags):

```go
tags, err := shape.ExtractPackageTagsFromDir("./mypackage")
if err != nil {
    log.Fatal(err)
}
```

### Using IndexedTypeWalker

For advanced use cases, you can use the IndexedTypeWalker directly:

```go
walker, err := shape.NewIndexTypeInDir("./mypackage")
if err != nil {
    log.Fatal(err)
}

packageTags := walker.PackageTags()
```

## Convenience Functions

### GetPackageTagValue

Get a tag value with a fallback default:

```go
version := shape.GetPackageTagValue(tags, "version", "unknown")
// Returns the version tag value, or "unknown" if not found
```

### HasPackageTagOption

Check if a tag has a specific option:

```go
if shape.HasPackageTagOption(tags, "mkunion", "no-type-registry") {
    fmt.Println("Type registry is disabled")
}

if shape.HasPackageTagOption(tags, "build", "debug") {
    fmt.Println("Debug build enabled")
}
```

## Examples

### Basic Package Configuration

```go
//go:tag mkunion:",no-type-registry"
//go:tag version:"1.0.0"
package config

//go:tag mkunion:"Environment"
type (
    Development struct{ Debug bool }
    Production  struct{ OptLevel int }
    Testing     struct{ Coverage float64 }
)
```

### Package Metadata

```go
//go:tag module:"user-service"
//go:tag version:"2.1.0,stable,production"
//go:tag author:"Development Team"
//go:tag license:"Apache-2.0"
//go:tag description:"User management service types"
package userservice

//go:tag mkunion:"UserEvent"
type (
    UserCreated struct{ ID string; Email string }
    UserUpdated struct{ ID string; Changes map[string]any }
    UserDeleted struct{ ID string }
)
```

### Build Tool Integration

```go
//go:tag build:"generate-docs,strict-validation"
//go:tag swagger:"v3,yaml"
//go:tag openapi:"enabled,validate-examples"
package api

//go:tag mkunion:"APIResponse[T]"
type (
    Success[T any] struct{ Data T; Status int }
    Error[T any]   struct{ Message string; Code int }
)
```

### Extracting and Using Tags

```go
package main

import (
    "fmt"
    "log"
    "github.com/widmogrod/mkunion/x/shape"
)

func main() {
    // Extract package tags
    tags, err := shape.ExtractPackageTagsFromFile("api.go")
    if err != nil {
        log.Fatal(err)
    }
    
    // Get version info
    version := shape.GetPackageTagValue(tags, "version", "0.0.0")
    fmt.Printf("Package version: %s\n", version)
    
    // Check build options
    if shape.HasPackageTagOption(tags, "build", "generate-docs") {
        fmt.Println("Documentation generation enabled")
    }
    
    // Check if type registry is disabled
    if shape.HasPackageTagOption(tags, "mkunion", "no-type-registry") {
        fmt.Println("Type registry disabled for this package")
    } else {
        fmt.Println("Type registry enabled for this package")
    }
}
```

## Integration with mkunion CLI

The mkunion CLI tool automatically recognizes package-level tags when processing files. The `no-type-registry` option is particularly useful for packages that don't need JSON marshalling support.

When you run:

```bash
mkunion watch ./...
```

The tool will:
1. Extract package-level tags from each package
2. Apply package-wide configuration (like `no-type-registry`)
3. Generate appropriate code based on the configuration

## Best Practices

1. **Consistency**: Keep package-level tags consistent across all files in the same package
2. **Documentation**: Document your custom tags and their meanings
3. **Validation**: Use the extraction functions to validate tag presence in build scripts
4. **Naming**: Use descriptive tag names that clearly indicate their purpose

## Tag Syntax Support

Package-level tags support the same advanced syntax as type-level tags:

- **Generic syntax**: `//go:tag container:"Container[T]"`
- **Multiple options**: `//go:tag build:"debug,verbose,strict"`
- **Empty values**: `//go:tag mkunion:",no-type-registry"`
- **Complex types**: `//go:tag mapping:"Map[String, List[Option[T]]]"`

This makes package-level tags powerful and flexible for various use cases.