// Package ast defines the AST for simple language
// AST can be created either by parser or by hand, it's up to implementer to decide how to create AST
// This package provides few examples of AST creation mostly by parsing JSON
// - ast_sugar.go
// - ast_human_friendly.go
// - ast_description_of_best_result
// Much more advance parser is also possible, but it's not implemented here
package ast

import "github.com/widmogrod/mkunion/x/schema"

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
	And []Operator
	Or  []Operator
	Not struct{ Operator }
)

func init() {
	// Value transformations
	schema.RegisterTransformations(ValueSchemaTransformations())
	schema.RegisterRules(ValueSchemaRules())

	// Operator
	schema.RegisterTransformations(OperatorSchemaTransformations())
	schema.RegisterRules(OperatorSchemaRules())
}

func ValueSchemaTransformations() []schema.TransformFunc {
	return []schema.TransformFunc{
		schema.WrapStruct(&Lit{}, "Lit"),
		schema.WrapStruct(&Accessor{}, "Accessor"),
	}
}

func ValueSchemaRules() []schema.RuleMatcher {
	return []schema.RuleMatcher{
		schema.UnwrapStruct(&Lit{}, "Lit"),
		schema.UnwrapStruct(&Accessor{}, "Accessor"),
	}
}

func OperatorSchemaTransformations() []schema.TransformFunc {
	return []schema.TransformFunc{
		schema.WrapStruct(&Eq{}, "Eq"),
		schema.WrapStruct(&Gt{}, "Gt"),
		schema.WrapStruct(&And{}, "And"),
		schema.WrapStruct(&Or{}, "Or"),
		schema.WrapStruct(&Not{}, "Not"),
	}
}

func OperatorSchemaRules() []schema.RuleMatcher {
	return []schema.RuleMatcher{
		schema.UnwrapStruct(&Eq{}, "Eq"),
		schema.UnwrapStruct(&Gt{}, "Gt"),
		//schema.UnwrapStruct(&And{}, "And"),
		//schema.UnwrapStruct(&Or{}, "Or"),
		schema.UnwrapStruct(&Not{}, "Not"),
	}
}
