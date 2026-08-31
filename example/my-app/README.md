# Sample App with Go and React

This demo uses mkunion as a **Go tool dependency**. `go.mod` holds the line:

```
tool github.com/widmogrod/mkunion/cmd/mkunion
```

So there is nothing to install and nothing to put on your `PATH`.
`go tool mkunion` builds the pinned version and runs it.
Needs Go 1.24 or newer.

## Run it

```
npm install
npm start

export OPENAI_API_KEY=$(op read "op://Personal/OpenAI dev token/credential")

# generate Go unions, shapes and the TypeScript types in ./src/workflow
go tool mkunion watch -g ./...

go run *.go
```

## How the code generation is wired

`server.go` exports the TypeScript types with a plain `go:generate` line:

```go
//go:generate go tool mkunion shape-export --language=typescript -o ./src/workflow
```

`go tool mkunion watch -g ./...` runs mkunion first, then `go generate ./...`,
so the union types exist before the TypeScript export reads them.
