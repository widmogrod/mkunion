package testutil

import "github.com/widmogrod/mkunion/x/schema"

type MyVal1 bool
type MyVal2 = int

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
		Map     map[string]Example
		Any     any
		Alias1  MyVal1
		Alias2  MyVal2
		Ptr     *int
	}
)

type ExampleRecord[T any] struct {
	Data T
}

type ExampleChange[T any] struct {
	After ExampleRecord[T]
}
