# Package-Level Tags Example

This example demonstrates how to use package-level `go:tag` annotations with mkunion.

## Package-Level Tags

The `main.go` file shows how to declare package-level tags:

```go
//go:tag mkunion:",no-type-registry"
//go:tag version:"1.0.0,stable,production"
//go:tag module:"package-tags-example"
//go:tag author:"mkunion"
package main
```

These tags can be used to:
- Configure package-wide behavior (like disabling type registry)
- Store package metadata (version, author, etc.)
- Add custom annotations for build tools or documentation

## Extracting Package Tags

The example shows three ways to extract package-level tags:

### 1. Extract from specific file
```go
tags, err := shape.ExtractPackageTagsFromFile("main.go")
```

### 2. Extract from directory
```go
tags, err := shape.ExtractPackageTagsFromDir(".")
```

### 3. Using IndexedTypeWalker
```go
walker, err := shape.NewIndexTypeInDir(".")
tags := walker.PackageTags()
```

## Convenience Functions

The shape package provides helpful functions for working with package tags:

- `GetPackageTagValue(tags, "version", "unknown")` - Get tag value with fallback
- `HasPackageTagOption(tags, "mkunion", "no-type-registry")` - Check for specific options

## Running the Example

To run this example:

```bash
cd example/package_tags
go run main.go
```

This will generate the union types and demonstrate package-level tag extraction.