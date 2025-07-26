# Runtime Package Tags Example

This example demonstrates how static binaries can self-reflect on package-level tags without requiring access to source files.

## Package Tags

The example includes several package-level tags:

```go
//go:tag version:"2.0.0,stable,production"
//go:tag module:"runtime-tags-example"  
//go:tag author:"mkunion development team"
//go:tag license:"MIT"
package main
```

## Two Approaches

The example demonstrates two different approaches to accessing package tags:

### 1. Source File Access

```go
// Requires source files to be present
tags, err := shape.ExtractPackageTagsFromFile("main.go")
```

**Use cases:**
- Development and build tools
- Code analysis utilities
- Documentation generation

**Limitations:**
- Requires source files
- Fails in deployed binaries

### 2. Runtime Access (NEW)

```go
// Works even in static binaries without source files
tags := shape.GetRuntimePackageTags()
version := shape.GetRuntimePackageTagValue("version", "unknown")
```

**Use cases:**
- Binary self-identification
- Version and metadata reporting
- Feature flags and configuration
- License and attribution information

**Benefits:**
- Works in static binaries
- No source files required
- Perfect for deployment scenarios

## Running the Example

```bash
# Generate mkunion code
mkunion -i main.go

# Run with source files available
go run .

# Build static binary and test in clean environment
go build -o demo .
mkdir clean_env && cp demo clean_env/
cd clean_env && ./demo
```

## Key Features Demonstrated

1. **Compile-time embedding**: Package tags are automatically embedded in the type registry
2. **Runtime access**: Binaries can access their own metadata without source files
3. **Fallback behavior**: Source file access fails gracefully when files aren't available
4. **Complete metadata**: Version, author, license, and custom tags are all accessible

## Generated Code

When you run `mkunion -i main.go`, it generates `types_reg_gen.go` with embedded package tags:

```go
func init() {
    // Package tags embedded at compile time
    shared.PackageTagsStore(map[string]interface{}{
        "author":  shape.Tag{Value: "mkunion development team", Options: nil},
        "license": shape.Tag{Value: "MIT", Options: nil},
        "module":  shape.Tag{Value: "runtime-tags-example", Options: nil},
        "version": shape.Tag{Value: "2.0.0", Options: []string{"stable", "production"}},
    })
    // ... type registry entries
}
```

This allows the binary to self-reflect on its compile-time metadata at runtime.