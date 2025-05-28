---
title: End-to-End types between Go and TypeScript
---

# End-to-End types between Go and TypeScript

MkUnion enables the generation of TypeScript definitions directly from your Go union types. This facilitates end-to-end type safety when building applications with a Go backend and a TypeScript frontend.

By using the `mkunion` tool, you can ensure that the data structures exchanged between your Go server and TypeScript client are consistent, reducing the likelihood of integration errors and improving developer experience.

The following snippet shows an example of Go code from which TypeScript definitions can be generated:

```go title="example/my-app/server.go"
--8<-- "example/my-app/server.go:34:55"
```

This generated TypeScript code can then be imported into your frontend project, providing compile-time checks and autocompletion for your API responses and requests.