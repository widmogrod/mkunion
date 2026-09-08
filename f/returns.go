package f

// --8<-- [start:returns]

// Returns is a phantom marker: it has no fields and no methods.
// Embed it in a union variant to declare the type of answer the variant
// expects. mkunion reads the type argument and generates a typed handler
// for the union, one method per variant (see the `handler` union option).
//
//	//go:tag mkunion:"Query,handler"
//	type (
//		GetUser   struct{ f.Returns[User]; ID string }
//		CountUser struct{ f.Returns[int] }
//	)
//
// Because it is a phantom, serde, TypeScript export and JSON schema skip it.
type Returns[R any] struct{}

// --8<-- [end:returns]
