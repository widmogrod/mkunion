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
	assert.Equal(t, 10, len(union.Variant))

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
						Desc:      ptr("Name of the person"),
						Guard:     nil,
						IsPointer: false,
						Tags: map[string]FieldTag{
							"json": {
								Value: "name",
							},
							"desc": {
								Value: "Name of the person",
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
			&StringLike{
				Named: &Named{
					Name:          "C",
					PkgName:       "testasset",
					PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				},
			},
			&NumberLike{
				Named: &Named{
					Name:          "D",
					PkgName:       "testasset",
					PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				},
			},
			&NumberLike{
				Named: &Named{
					Name:          "E",
					PkgName:       "testasset",
					PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				},
			},
			&BooleanLike{
				Named: &Named{
					Name:          "F",
					PkgName:       "testasset",
					PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				},
			},
			&Any{
				Named: &Named{
					Name:          "G",
					PkgName:       "testasset",
					PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				},
			},
			&MapLike{
				Key: &StringLike{},
				Val: &RefName{
					Name:          "Example",
					PkgName:       "testasset",
					PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				},
				KeyIsPointer: false,
				ValIsPointer: false,
				Named: &Named{
					Name:          "H",
					PkgName:       "testasset",
					PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				},
			},
			&ListLike{
				Element: &RefName{
					Name:          "Example",
					PkgName:       "testasset",
					PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				},
				ElementIsPointer: false,
				Named: &Named{
					Name:          "I",
					PkgName:       "testasset",
					PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				},
			},
			&ListLike{
				Element:          &StringLike{},
				ElementIsPointer: false,
				ArrayLen:         ptr(2),
				Named: &Named{
					Name:          "J",
					PkgName:       "testasset",
					PkgImportName: "github.com/widmogrod/mkunion/x/shape/testasset",
				},
			},
		},
	}

	assert.Equal(t, expected, &union)
}