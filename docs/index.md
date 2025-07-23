---
title: Introduction
---

# Welcome to MkUnion
[![Go Reference](https://pkg.go.dev/badge/github.com/widmogrod/mkunion.svg)](https://pkg.go.dev/github.com/widmogrod/mkunion)
[![Go Report Card](https://goreportcard.com/badge/github.com/widmogrod/mkunion)](https://goreportcard.com/report/github.com/widmogrod/mkunion)
[![codecov](https://codecov.io/gh/widmogrod/mkunion/branch/main/graph/badge.svg?token=3Z3Z3Z3Z3Z)](https://codecov.io/gh/widmogrod/mkunion)


## About
Strongly typed **union type** in golang that supports generics*

* with exhaustive _pattern matching_ support
* with _json marshalling_ including generics
* and as a bonus, can generate compatible TypeScript types for end-to-end type safety in your application

## Why
Historically, in languages like Go that lack native union types, developers have resorted to workarounds such as the Visitor pattern or `iota` with `switch` statements.

The Visitor pattern requires a lot of boilerplate code and manual crafting of the `Accept` method for each type in the union.
Using `iota` and `switch` statements is not type-safe and can lead to runtime errors, especially when a new type is added and not all `case` statements are updated.

On top of that, any data marshalling, like to/from JSON, requires additional, handcrafted code to make it work.

MkUnion solves all of these problems by generating opinionated and strongly typed, meaningful code for you.

## Example

```go title="example/vehicle.go"
--8<-- "example/vehicle.go:vehicle-def"
--8<-- "example/vehicle.go:calculate-fuel"
--8<-- "example/vehicle_test.go:json"
```

Watch for changes in the file and generate code on the fly:
```sh
mkunion watch ./...

# or use -g flag to generate code without watching
mkunion watch -g ./...
```

## Next

- Read [getting started](./getting_started.md) to learn more.
- Learn more about [value proposition](./value_proposition.md)