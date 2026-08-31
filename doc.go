// Package mkunion is the module root for mkunion.
//
// The code lives in sub-packages:
//   - cmd/mkunion  - the code generator command line tool
//   - x/shape      - type introspection and shape inference
//   - x/generators - the code generators
//
// This file exists so that the module has a root package. Without it a
// "replace github.com/widmogrod/mkunion => <local path>" directive cannot
// resolve the module when mkunion is used as a `go tool` dependency.
package mkunion
