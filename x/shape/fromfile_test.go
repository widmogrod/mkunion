package shape

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInferFromFile(t *testing.T) {
	inferred, err := InferFromFile("testasset/type_example.go")
	if err != nil {
		t.Fatal(err)
	}

	union := inferred.RetrieveUnion("Example")
	assert.Equal(t, "Example", union.Name)
	assert.Equal(t, "testasset", union.PkgName)
	assert.Equal(t, "github.com/widmogrod/mkunion/x/shape/testasset", union.PkgImportName)
	assert.Equal(t, 2, len(union.Variant))

	expected := &UnionLike{
		Name:          "Example",
		PkgName:       "testasset",
		PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
		Variant: []Shape{
			&StructLike{
				Name:          "A",
				PkgName:       "testasset",
				PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				Fields: []*FieldLike{
					{
						Name:      "Name",
						Type:      &StringLike{},
						Desc:      nil,
						Guard:     nil,
						IsPointer: false,
						Tags: map[string]FieldTag{
							"json": {
								Value: "name",
							},
						},
					},
				},
			},
			&StructLike{
				Name:          "B",
				PkgName:       "testasset",
				PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				Fields: []*FieldLike{
					{
						Name:      "Age",
						Type:      &NumberLike{},
						Desc:      nil,
						Guard:     nil,
						IsPointer: false,
						Tags: map[string]FieldTag{
							"json": {
								Value: "age",
							},
						},
					},
					{
						Name: "A",
						Type: &RefName{
							Name:          "A",
							PkgName:       "testasset",
							PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
						},
						Desc:      nil,
						Guard:     nil,
						IsPointer: true,
						Tags:      nil,
					},
					{
						Name: "T",
						Type: &RefName{
							Name:          "Time",
							PkgName:       "time",
							PkgImportName: "time",
						},
						Desc:      nil,
						Guard:     nil,
						IsPointer: true,
						Tags:      nil,
					},
				},
			},
		},
	}

	assert.Equal(t, expected, &union)
}
