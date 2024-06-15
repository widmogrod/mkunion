package testutil

import "github.com/widmogrod/mkunion/x/schema"

//go:tag mkunion:"Example"
type (
	ExampleOne struct {
		OneValue string
	}
	ExampleTwo struct {
		TwoData schema.Schema
		TwoNext Example
	}
	ExampleTree struct {
		Items   []Example
		Schemas []schema.Schema
	}
)

type ExampleRecord[T any] struct {
	Data T
}

type ExampleChange[T any] struct {
	After ExampleRecord[T]
}
