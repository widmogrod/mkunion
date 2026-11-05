package f

// --8<-- [start:returns]

// Returns is a phantom marker: it has no fields and carries no data.
// Embed it in a union variant to declare the type of answer the variant
// expects. When at least one variant of a union embeds it, mkunion generates
// a typed handler for the union: one interface per variant, one interface
// for the whole union, and a HandleX function with one typed arm per variant.
//
//	//go:tag mkunion:"Query"
//	type (
//		GetUser   struct{ f.Returns[User]; ID string }
//		CountUser struct{ f.Returns[int] }
//		Touch     struct{ ID string }          // answers nothing: only an error
//	)
//
// A variant without Returns is handled, with an error-only method, but is not
// an operation a program can perform (see x/effect.Op).
//
// Because it is a phantom, serde, TypeScript export and JSON schema skip it.
type Returns[R any] struct{}

// Ret names the answer type. It is promoted to every variant that embeds
// Returns, so a generic function can infer R from the variant:
// func Perform[R any](op interface{ Ret() R }). It carries no value.
func (Returns[R]) Ret() R {
	var zero R
	return zero
}

// --8<-- [end:returns]
