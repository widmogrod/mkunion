package generators

import (
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/shape"
	"testing"
)

func TestSerdeProtobufTagged_Generate_Struct(t *testing.T) {
	s := &shape.StructLike{
		Name:    "TestStruct",
		PkgName: "test",
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

	generator := NewSerdeProtobufTagged(s)
	result, err := generator.Generate()

	assert.NoError(t, err)
	assert.Contains(t, result, "proto.Message")
	assert.Contains(t, result, "Marshal()")
	assert.Contains(t, result, "Unmarshal(")
	assert.Contains(t, result, "proto.Marshal")
	assert.Contains(t, result, "proto.Unmarshal")
}

func TestSerdeProtobufUnion_Generate(t *testing.T) {
	union := &shape.UnionLike{
		Name:    "TestUnion",
		PkgName: "test",
		PkgImportName: "github.com/test/test",
		Variant: []shape.Shape{
			&shape.StructLike{
				Name:    "Branch",
				PkgName: "test",
				PkgImportName: "github.com/test/test",
				Fields: []*shape.FieldLike{
					{
						Name: "Value",
						Type: &shape.PrimitiveLike{Kind: &shape.StringLike{}},
					},
				},
			},
			&shape.StructLike{
				Name:    "Leaf",
				PkgName: "test",
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

	generator := NewSerdeProtobufUnion(union)
	result, err := generator.Generate()

	assert.NoError(t, err)
	
	resultStr := string(result)
	assert.Contains(t, resultStr, "TestUnionProtoType")
	assert.Contains(t, resultStr, "TestUnionProtoMessage")
	assert.Contains(t, resultStr, "TestUnionFromProtobuf")
	assert.Contains(t, resultStr, "TestUnionToProtobuf")
	assert.Contains(t, resultStr, "proto.Marshal")
	assert.Contains(t, resultStr, "proto.Unmarshal")
	assert.Contains(t, resultStr, "BRANCH")
	assert.Contains(t, resultStr, "LEAF")
}