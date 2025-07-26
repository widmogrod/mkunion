package generators

import (
	"github.com/stretchr/testify/assert"
	"github.com/widmogrod/mkunion/x/shape"
	"testing"
)

func TestSerdeSQLTagged_Generate_Struct(t *testing.T) {
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

	generator := NewSerdeSQLTagged(s)
	result, err := generator.Generate()

	assert.NoError(t, err)
	assert.Contains(t, result, "sql.Scanner")
	assert.Contains(t, result, "sql.Valuer")
	assert.Contains(t, result, "Scan(value interface{}) error")
	assert.Contains(t, result, "Value() (sql.Value, error)")
	assert.Contains(t, result, "encoding.Marshal")
	assert.Contains(t, result, "encoding.Unmarshal")
}

func TestSerdeSQLUnion_Generate(t *testing.T) {
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

	generator := NewSerdeSQLUnion(union)
	result, err := generator.Generate()

	assert.NoError(t, err)

	resultStr := string(result)
	assert.Contains(t, resultStr, "CREATE TABLE testunion")
	assert.Contains(t, resultStr, "TestUnionScanSQL")
	assert.Contains(t, resultStr, "TestUnionToSQL")
	assert.Contains(t, resultStr, "encoding.Marshal")
	assert.Contains(t, resultStr, "encoding.Unmarshal")
	assert.Contains(t, resultStr, "branch")
	assert.Contains(t, resultStr, "leaf")
}
