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
	
	// Assert the complete output structure to serve as documentation
	expected := `package test

import (
	"fmt"
	"strconv"
	"strings"
)

/*
GraphQL Schema for TestStruct:

type TestStruct {
  Name: String!
  Value: Float!
}
*/

/*
Example GraphQL Resolver for TestStruct:

// In your resolver file:
func (r *queryResolver) GetTestStruct(ctx context.Context, id string) (*TestStruct, error) {
    // Your resolver logic here
    return &TestStruct{}, nil
}

func (r *mutationResolver) CreateTestStruct(ctx context.Context, input TestStructInput) (*TestStruct, error) {
    // Your mutation logic here
    return &TestStruct{}, nil
}
*/

`
	
	assert.Equal(t, expected, result, "GraphQL Tagged generator should produce complete schema and resolver template for struct types")
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

	// Assert the complete output structure to serve as documentation  
	expected := `package test

import (
	"context"
	"fmt"
	"strings"
)

// Experimental GraphQL support - schema and resolver templates

/*
GraphQL Schema for TestUnion Union:

interface TestUnion {
  __typename: String!
}

union TestUnionUnion = Branch | Leaf

type Branch implements TestUnion {
  __typename: String!
  Value: String!
}

type Leaf implements TestUnion {
  __typename: String!
  Data: Float!
}

extend type Query {
  getTestUnion(id: ID!): TestUnion
  listTestUnions: [TestUnion!]!
}

extend type Mutation {
  createBranch(input: BranchInput!): Branch
  createLeaf(input: LeafInput!): Leaf
}

input BranchInput {
  Value: String!
}

input LeafInput {
  Data: Float!
}

*/

/*
Example GraphQL Resolvers for TestUnion Union:

// In your resolver file:

func (r *queryResolver) GetTestUnion(ctx context.Context, id string) (TestUnion, error) {
    // Your query logic here
    // Return appropriate variant based on data
    return nil, nil
}

func (r *queryResolver) ListTestUnions(ctx context.Context) ([]TestUnion, error) {
    // Your list logic here
    return nil, nil
}

func (r *mutationResolver) CreateBranch(ctx context.Context, input BranchInput) (*Branch, error) {
    // Your mutation logic here
    return &Branch{}, nil
}

func (r *mutationResolver) CreateLeaf(ctx context.Context, input LeafInput) (*Leaf, error) {
    // Your mutation logic here
    return &Leaf{}, nil
}

func (r *Resolver) TestUnion() TestUnionResolver {
    return &testunionResolver{r}
}

type testunionResolver struct{ *Resolver }

func (r *testunionResolver) __resolveType(obj interface{}) (string, error) {
    switch obj.(type) {
    case *Branch:
        return "Branch", nil
    case *Leaf:
        return "Leaf", nil
    default:
        return "", fmt.Errorf("unknown type")
    }
}
*/

`

	assert.Equal(t, expected, string(result), "GraphQL Union generator should produce complete schema with interface, union, mutations, queries, and comprehensive resolver templates that demonstrate the full GraphQL integration pattern for union types")
}
