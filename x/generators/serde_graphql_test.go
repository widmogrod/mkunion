package generators

import (
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/shape"
	"testing"
)

func TestSerdeGraphQLTagged_Generate_Struct(t *testing.T) {
	s := &shape.StructLike{
		Name:          "TestStruct",
		PkgName:       "test",
		PkgImportName: "github.com/test/test",
		Fields: []*shape.FieldLike{
			{
				Name: "Name",
				Type: &shape.PrimitiveLike{Kind: &shape.StringLike{}},
			},
			{
				Name: "Value",
				Type: &shape.PrimitiveLike{Kind: &shape.NumberLike{}},
			},
		},
	}

	generator := NewSerdeGraphQLTagged(s)
	result, err := generator.Generate()

	assert.NoError(t, err)
	assert.Contains(t, result, "GraphQL Schema for TestStruct")
	assert.Contains(t, result, "type TestStruct")
	assert.Contains(t, result, "Name: String!")
	assert.Contains(t, result, "Value: Float!")
	assert.Contains(t, result, "Example GraphQL Resolver")
}

func TestSerdeGraphQLUnion_Generate(t *testing.T) {
	union := &shape.UnionLike{
		Name:          "TestUnion",
		PkgName:       "test",
		PkgImportName: "github.com/test/test",
		Variant: []shape.Shape{
			&shape.StructLike{
				Name:          "Branch",
				PkgName:       "test",
				PkgImportName: "github.com/test/test",
				Fields: []*shape.FieldLike{
					{
						Name: "Value",
						Type: &shape.PrimitiveLike{Kind: &shape.StringLike{}},
					},
				},
			},
			&shape.StructLike{
				Name:          "Leaf",
				PkgName:       "test",
				PkgImportName: "github.com/test/test",
				Fields: []*shape.FieldLike{
					{
						Name: "Data",
						Type: &shape.PrimitiveLike{Kind: &shape.NumberLike{}},
					},
				},
			},
		},
	}

	generator := NewSerdeGraphQLUnion(union)
	result, err := generator.Generate()

	assert.NoError(t, err)

	resultStr := string(result)
	assert.Contains(t, resultStr, "GraphQL Schema for TestUnion Union")
	assert.Contains(t, resultStr, "interface TestUnion")
	assert.Contains(t, resultStr, "union TestUnionUnion = Branch | Leaf")
	assert.Contains(t, resultStr, "type Branch implements TestUnion")
	assert.Contains(t, resultStr, "type Leaf implements TestUnion")
	assert.Contains(t, resultStr, "extend type Query")
	assert.Contains(t, resultStr, "extend type Mutation")
	assert.Contains(t, resultStr, "Example GraphQL Resolvers")
}
