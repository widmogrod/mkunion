// Package ast defines the AST for simple language
// AST can be created either by parser or by hand, it's up to implementer to decide how to create AST
// This package provides few examples of AST creation mostly by parsing JSON
// - ast_sugar.go
// - ast_human_friendly.go
// - ast_description_of_best_result
// Much more advance parser is also possible, but it's not implemented here
package ast

// Value represents how value can be represented in AST
// Lit - literal value like int or string
// Accessor represents field access like "a.b.c"
// Accessor assumes that interpreter will provide data to return value that exists under such path
//
//go:generate go run ../../cmd/mkunion/main.go -name=Value
type (
	Lit      struct{ Value any }
	Accessor struct{ Path []string }
)

// Operator represents binary operation like
// - Eq - ==
// - Gt - >
// - And - &&
// - Or - ||
// - Not - !
//
//go:generate go run ../../cmd/mkunion/main.go -name=Operator
type (
	Eq  struct{ L, R Value }
	Gt  struct{ L, R Value }
	And struct{ List []Operator }
	Or  struct{ List []Operator }
	Not struct{ Operator }
)
