CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

mkunion is a Go code generation tool that implements strongly typed union types (algebraic data types) with:
- Exhaustive pattern matching
- JSON marshalling/unmarshalling with generics support
- TypeScript type generation for end-to-end type safety
- State machine modeling with mermaid diagram generation
- Shape inference and type introspection

## Development Commands

### Building mkunion
```bash
# Build the mkunion tool
cd cmd/mkunion && go build -o mkunion

# Install from latest release
go install github.com/widmogrod/mkunion/cmd/mkunion@latest
```

### Code Generation Workflow
```bash
# Generate union types and shapes, then automatically run go generate (run from project root)
mkunion watch ./...

# For one-time generation without watching (also runs go generate automatically)
mkunion watch -g ./...

# Skip running go generate after mkunion generation
mkunion watch -G ./...
# or
mkunion watch --dont-run-go-generate ./...

# Generate TypeScript types
mkunion shape-export --language typescript --output-dir ./output -i file.go
```

**Note:** As of the latest version, `mkunion watch` automatically runs `go generate ./...` after generating union types and shapes. This eliminates the need to run two separate commands. Use the `-G` flag if you want to skip the automatic `go generate` step.

### Testing
```bash
# Run all tests
go test -v ./...

# Run tests with race detector and coverage
go test -race -coverprofile=coverage.out -covermode=atomic ./...

# Run a single test
go test -v -run TestName ./path/to/package

# Run tests in short mode (skip long-running tests)
go test -short -v ./...

# Note: Some tests may fail if development environment is not bootstrapped
# Some tests require AWS services (localstack), Kafka, or OpenSearch to be running
```

### Development Setup
```bash
# Bootstrap development environment (sets up Docker services)
dev/bootstrap.sh

# The bootstrap script sets up:
# - Localstack (AWS services) at http://localhost:4566
# - Kafka at localhost:9092 (UI at http://localhost:9088)
# - OpenSearch at http://localhost:9200
# - Environment variables in .envrc
```

### Documentation
```bash
# Serve documentation locally at http://localhost:8088
dev/docs.sh run

# Build documentation
dev/docs.sh build
```

## Architecture and Code Structure

### Key Directories
- `cmd/mkunion/`: Main CLI tool for code generation
- `x/shape/`: Type introspection and representation system
- `x/generators/`: Code generators for unions, serde, shapes, pattern matching
- `x/machine/`: State machine framework with test-driven development support
- `x/storage/`: Storage abstractions with predicate queries
- `x/workflow/`: Workflow engine built on state machines
- `example/`: Examples demonstrating various features

### Code Generation Patterns

1. **Union Types**: Use `//go:tag mkunion:"UnionName"` to mark types for union generation
2. **Generated Files**: 
   - `*_union_gen.go` - Union type definitions and constructors
   - `*_shape_gen.go` - Shape definitions for type introspection
   - `*_serde_gen.go` - JSON marshalling/unmarshalling
   - `*_match_gen.go` - Pattern matching functions
   - `types_reg_gen.go` - Type registry for JSON marshalling

3. **Pattern Matching**: Use `MatchUnionNameR1()` for exhaustive matching with one return value

### State Machine Development

State machines use union types for states and commands with explicit dependencies:

```go
//go:tag mkunion:"State"
type (
    Initial struct{}
    Processing struct{ ID string }
    Complete struct{ Result string }
)

//go:tag mkunion:"Command"
type (
    StartCMD struct{ ID string }
    CompleteCMD struct{ Result string }
)

// Define dependencies explicitly
type Dependencies struct {
    DB *sql.DB
    Logger *log.Logger
}

// Transition function with dependencies
func Transition(ctx context.Context, deps Dependencies, cmd Command, state State) (State, error) {
    return MatchCommandR2(cmd,
        func(c *StartCMD) (State, error) {
            deps.Logger.Printf("Starting process %s", c.ID)
            // Use deps.DB for database operations
            return &Processing{ID: c.ID}, nil
        },
        func(c *CompleteCMD) (State, error) {
            deps.Logger.Printf("Completing process with result: %s", c.Result)
            return &Complete{Result: c.Result}, nil
        },
    )
}

// Create machine with explicit dependencies and optional state
func NewMachine(deps Dependencies, state State) *machine.Machine[Dependencies, Command, State] {
    // Use default initial state if none provided
    if state == nil {
        state = &Initial{}
    }
    return machine.NewMachine(deps, Transition, state)
}
```

Test state machines with the test suite for automatic mermaid diagram generation:
```go
// Create test dependencies
deps := Dependencies{
    DB: testDB,
    Logger: testLogger,
}

// Create test suite with dependency factory
suite := machine.NewTestSuite(func() *machine.Machine[Dependencies, Command, State] {
    return NewMachine(deps, nil) // nil uses default initial state
})
suite.Run(t)
suite.SelfDocumentStateDiagram(t, "filename.go")
```

## Important Notes

- Go version: 1.23.0 with toolchain 1.24.3
- Always run `mkunion watch -g ./...` to generate new files including go:generate tag
- The type registry can be disabled with `//go:tag mkunion:",no-type-registry"`
- When running tests that use AWS services, ensure the development environment is bootstrapped
- On Mac with Colima: ensure Colima is running before executing `dev/bootstrap.sh`