package mkunion

import (
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/shape"
	"testing"
)

func TestExtractInferenceForTree(t *testing.T) {
	out, err := InferFromFile("example/tree_example.go")
	assert.NoError(t, err)
	assert.Equal(t, "example", out.PackageName)
	assert.Equal(t,
		map[string][]Branching{
			"Branch": {
				{Lit: PtrStr("L")},
				{Lit: PtrStr("R")},
			},
			"Leaf": nil,
		},
		out.ForVariantType("Tree", []string{"Branch", "Leaf"}))

}

func TestExtractInferenceForWherePredicate(t *testing.T) {
	out, err := InferFromFile("example/where_predicate_example.go")
	assert.NoError(t, err)
	assert.Equal(t, "example", out.PackageName)
	assert.Equal(t,
		map[string][]Branching{
			"Eq":  nil,
			"And": nil,
			"Or":  nil,
			"Path": {
				{Lit: PtrStr("Condition")},
				{List: PtrStr("Then")},
				{Map: PtrStr("Y")},
			},
		},
		out.ForVariantType("WherePredicate", []string{"Eq", "And", "Or", "Path"}))
}

func TestAST(t *testing.T) {
	out, err := InferFromFile("example/ast/ast.go")
	assert.NoError(t, err)
	assert.Equal(t, "ast", out.PackageName)
	assert.Equal(t,
		map[string][]string{
			"Value":    {"Lit", "Accessor"},
			"Operator": {"Eq", "Gt", "And", "Or", "Not"},
		},
		out.possibleVariantTypes)
	assert.Equal(t, map[string]*shape.StructLike{
		"Lit": {
			Name:          "Lit",
			PkgName:       "ast",
			PkgImportName: "github.com/widmogrod/mkunion/example/ast",
			Fields: []*shape.FieldLike{
				{
					Name: "Value",
					Type: &shape.Any{},
				},
			},
		},
		"Accessor": {
			Name:          "Accessor",
			PkgName:       "ast",
			PkgImportName: "github.com/widmogrod/mkunion/example/ast",
			Fields: []*shape.FieldLike{
				{
					Name: "Path",
					Type: &shape.ListLike{Element: &shape.StringLike{}}},
			},
		},
		"Eq": {
			Name:          "Eq",
			PkgName:       "ast",
			PkgImportName: "github.com/widmogrod/mkunion/example/ast",
			Fields: []*shape.FieldLike{
				{
					Name: "L",
					Type: &shape.RefName{
						Name:          "Value",
						PkgName:       "ast",
						PkgImportName: "github.com/widmogrod/mkunion/example/ast",
					},
				},
				{
					Name: "R",
					Type: &shape.RefName{
						Name:          "Value",
						PkgName:       "ast",
						PkgImportName: "github.com/widmogrod/mkunion/example/ast",
					},
				},
			},
		},
		"Gt": {
			Name:          "Gt",
			PkgName:       "ast",
			PkgImportName: "github.com/widmogrod/mkunion/example/ast",
			Fields: []*shape.FieldLike{
				{
					Name: "L",
					Type: &shape.RefName{
						Name:          "Value",
						PkgName:       "ast",
						PkgImportName: "github.com/widmogrod/mkunion/example/ast",
					},
				},
				{
					Name: "R",
					Type: &shape.RefName{
						Name:          "Value",
						PkgName:       "ast",
						PkgImportName: "github.com/widmogrod/mkunion/example/ast",
					},
				},
			},
		},
		"And": {
			Name:          "And",
			PkgName:       "ast",
			PkgImportName: "github.com/widmogrod/mkunion/example/ast",
			Fields: []*shape.FieldLike{
				{
					Name: "List",
					Type: &shape.ListLike{
						Element: &shape.RefName{
							Name:          "Operator",
							PkgName:       "ast",
							PkgImportName: "github.com/widmogrod/mkunion/example/ast",
						},
					},
				},
			},
		},
		"Or": {
			Name:          "Or",
			PkgName:       "ast",
			PkgImportName: "github.com/widmogrod/mkunion/example/ast",
			Fields: []*shape.FieldLike{
				{
					Name: "List",
					Type: &shape.ListLike{
						Element: &shape.RefName{
							Name:          "Operator",
							PkgName:       "ast",
							PkgImportName: "github.com/widmogrod/mkunion/example/ast",
						},
					},
				},
			},
		},
		"Not": {
			Name:          "Not",
			PkgName:       "ast",
			PkgImportName: "github.com/widmogrod/mkunion/example/ast",
			Fields: []*shape.FieldLike{
				{
					Name: "Operator",
					Type: &shape.RefName{
						Name:          "Operator",
						PkgName:       "ast",
						PkgImportName: "github.com/widmogrod/mkunion/example/ast",
					},
				},
			},
		},
	}, out.shapes)
}
