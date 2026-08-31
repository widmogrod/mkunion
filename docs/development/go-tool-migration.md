---
title: Using mkunion as a Go tool dependency
---

# Using mkunion as a Go tool dependency

Since Go 1.24 a `go.mod` file can declare tools:

```
tool github.com/widmogrod/mkunion/cmd/mkunion
```

`go get -tool <pkg>` adds the line and pins the version. `go tool <name>`
builds the pinned version and runs it. `go mod tidy` keeps it up to date.

## For users of mkunion

Old way:

```bash
go install github.com/widmogrod/mkunion/cmd/mkunion@v1.26.1
mkunion watch -g ./...
```

New way:

```bash
go get -tool github.com/widmogrod/mkunion/cmd/mkunion@v1.26.1
go tool mkunion watch -g ./...
```

Why this is better:

- No install step and no `PATH` or `GOPATH` set-up.
- The mkunion version is pinned in `go.mod`. Every developer and every CI job
  runs the same generator, so generated files do not churn between machines.
- `go:generate` directives work without a binary on `PATH`:
  `//go:generate go tool mkunion -name=Example`.

`go install` still works. Use it if your toolchain is older than Go 1.24.

## What changed in this repository

- `go.mod` declares `cmd/mkunion`, `cmd/mkfunc` and `github.com/matryer/moq`
  as tools.
- CI no longer runs `go build -C cmd/mkunion .` or
  `go install github.com/matryer/moq@...`. It runs `go tool mkunion watch -g ./...`.
- `go:generate` directives in `example/` and `f/` use `go tool`.
- A root package (`doc.go`) was added. Without it, a module that points at a
  local checkout with `replace github.com/widmogrod/mkunion => ../..` fails
  with "cmd/mkunion imports github.com/widmogrod/mkunion: cannot find module
  providing package". `example/my-app` is such a module.

## Breaking change: OpenAI helpers moved

A tool dependency puts the tool's imports into the user's module graph.
`x/shape` used to import `github.com/sashabaranov/go-openai`, so every user of
the tool pulled the OpenAI client into their `go.sum`.

`ToOpenAIFunctionDefinition` now lives in `x/shapeopenai`:

```go
// before
import "github.com/widmogrod/mkunion/x/shape"
def := shape.ToOpenAIFunctionDefinition(name, desc, in)

// after
import "github.com/widmogrod/mkunion/x/shapeopenai"
def := shapeopenai.ToOpenAIFunctionDefinition(name, desc, in)
```

Nothing else moved. `x/shape` no longer depends on go-openai, so the tool
dependency now pulls only fsnotify, logrus, urfave/cli, golang.org/x/mod and
golang.org/x/sys.
